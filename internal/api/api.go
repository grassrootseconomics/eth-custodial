package api

import (
	"context"
	"crypto"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"
	"github.com/grassrootseconomics/eth-custodial/internal/gas"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/grassrootseconomics/ethutils"
	"github.com/jackc/pgx/v5"
	"github.com/kamikazechaser/jrpc"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/riverqueue/river"
)

type (
	APIOpts struct {
		Debug         bool
		EnableMetrics bool
		EnableDocs    bool
		JRPC          bool
		// TODO: temporary patch for prod because poolIndex doesn't exisit in the entry point registry
		Prod           bool
		ListenAddress  string
		Build          string
		Registry       map[string]common.Address
		CORS           []string
		VerifyingKey   crypto.PublicKey
		SigningKey     crypto.PrivateKey
		Store          store.Store
		Logg           *slog.Logger
		ChainProvider  *ethutils.Provider
		GasOracle      gas.GasOracle
		QueueClient    *river.Client[pgx.Tx]
		BannedTokens   []string
		PretiumAddress string
	}

	API struct {
		listenAddress  string
		build          string
		prod           bool
		registry       map[string]common.Address
		signingKey     crypto.PrivateKey
		verifyingKey   crypto.PublicKey
		store          store.Store
		gasOracle      gas.GasOracle
		logg           *slog.Logger
		chainProvider  *ethutils.Provider
		router         *echo.Echo
		queueClient    *river.Client[pgx.Tx]
		bannedTokens   map[string]struct{}
		pretiumAddress string
	}
)

const (
	apiVersion         = "/api/v2"
	jRPCPath           = "/jrpc"
	maxBodySize        = "1M"
	allowedContentType = "application/json"
)

func New(o APIOpts) *API {
	api := &API{
		prod:           o.Prod,
		build:          o.Build,
		registry:       o.Registry,
		signingKey:     o.SigningKey,
		verifyingKey:   o.VerifyingKey,
		listenAddress:  o.ListenAddress,
		logg:           o.Logg,
		store:          o.Store,
		gasOracle:      o.GasOracle,
		chainProvider:  o.ChainProvider,
		queueClient:    o.QueueClient,
		bannedTokens:   make(map[string]struct{}, len(o.BannedTokens)),
		pretiumAddress: o.PretiumAddress,
	}

	for _, addr := range o.BannedTokens {
		api.bannedTokens[addr] = struct{}{}
	}

	customValidator := validator.New(validator.WithRequiredStructEnabled())
	router := echo.New()
	router.HideBanner = true
	router.HidePort = true
	router.Validator = &Validator{
		ValidatorProvider: customValidator,
	}
	router.HTTPErrorHandler = api.customHTTPErrorHandler

	corsConfig := middleware.CORSConfig{
		AllowOrigins:     o.CORS,
		AllowCredentials: true,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		MaxAge:           86400,
	}

	router.Use(middleware.Recover())
	router.Use(middleware.BodyLimit(maxBodySize))
	router.Use(middleware.ContextTimeout(util.SLATimeout))
	if o.Debug {
		// All frontend development must happen on localhost:3000
		corsConfig.AllowOrigins = append(corsConfig.AllowOrigins, "http://localhost:3000")

		router.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogStatus:   true,
			LogURI:      true,
			LogError:    true,
			HandleError: true,
			LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
				if v.Error == nil {
					o.Logg.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
						slog.String("uri", v.URI),
						slog.Int("status", v.Status),
					)
				} else {
					o.Logg.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
						slog.String("uri", v.URI),
						slog.Int("status", v.Status),
						slog.String("err", v.Error.Error()),
					)
				}
				return nil
			},
		}))
	}
	router.Use(middleware.CORSWithConfig(corsConfig))

	if o.EnableMetrics {
		router.GET("/metrics", api.metricsHandler)
	}

	if o.EnableDocs {
		router.GET("/docs", api.docsHandler)
	}

	authGroup := router.Group("/auth")
	authGroup.POST("/login", api.loginHandler)
	authGroup.POST("/logout", api.logoutHandler)

	apiGroup := router.Group(apiVersion)
	apiGroup.Use(echojwt.WithConfig(api.apiJWTAuthConfig()))
	apiGroup.Use(api.authStatusMiddleware())

	if o.JRPC {
		api.logg.Debug("registering supported eth namespace RPC handlers")
		j := jrpc.Endpoint(apiGroup, jRPCPath)
		j.Method("eth_sendTransaction", api.methodEthSendTransaction)
	}

	apiGroup.GET("/system", api.systemInfoHandler)
	apiGroup.POST("/account/create", api.accountCreateHandler)
	apiGroup.GET("/account/status/:address", api.accountStatusHandler)
	apiGroup.GET("/account/otx/:address", api.getOTXByAddressHandler)
	apiGroup.GET("/otx/track/:trackingId", api.trackOTXHandler)
	apiGroup.POST("/token/transfer", api.transferHandler)
	apiGroup.POST("/token/sweep", api.sweepHandler)
	apiGroup.POST("/pool/quote", api.poolQuoteHandler)
	apiGroup.POST("/pool/swap", api.poolSwapHandler)
	apiGroup.POST("/pool/deposit", api.poolDepositHandler)
	apiGroup.POST("/contracts/erc20", api.contractsERC20Handler)
	apiGroup.POST("/contracts/erc20-demurrage", api.contractsDemurrageERC20Handler)
	apiGroup.POST("/contracts/pool", api.contractsPoolHandler)

	api.router = router
	api.logg.Debug("API initialized", "listen_address", api.router)
	return api
}

func (a *API) Start() error {
	a.logg.Info("starting API HTTP server", "listen_address", a.listenAddress)
	return a.router.Start(a.listenAddress)
}

func (a *API) Stop(ctx context.Context) error {
	a.logg.Info("shutting down API server")
	return a.router.Shutdown(ctx)
}
