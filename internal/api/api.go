package api

import (
	"context"
	"crypto"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/grassrootseconomics/ethutils"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	APIOpts struct {
		Debug         bool
		EnableMetrics bool
		EnableDocs    bool
		ListenAddress string
		Build         string
		VerifyingKey  crypto.PublicKey
		SigningKey    crypto.PrivateKey
		Store         store.Store
		Logg          *slog.Logger
		ChainProvider *ethutils.Provider
		Worker        *worker.WorkerContainer
		BannedTokens  []string
	}

	API struct {
		listenAddress string
		build         string
		signingKey    crypto.PrivateKey
		verifyingKey  crypto.PublicKey
		store         store.Store
		logg          *slog.Logger
		chainProvider *ethutils.Provider
		router        *echo.Echo
		worker        *worker.WorkerContainer
		BannedTokens  map[string]struct{}
	}
)

const (
	apiVersion         = "/api/v2"
	maxBodySize        = "1M"
	allowedContentType = "application/json"
)

func New(o APIOpts) *API {
	api := &API{
		build:         o.Build,
		signingKey:    o.SigningKey,
		verifyingKey:  o.VerifyingKey,
		listenAddress: o.ListenAddress,
		logg:          o.Logg,
		store:         o.Store,
		chainProvider: o.ChainProvider,
		worker:        o.Worker,
		BannedTokens:  make(map[string]struct{}, len(o.BannedTokens)),
	}

	for _, addr := range o.BannedTokens {
		api.BannedTokens[addr] = struct{}{}
	}

	customValidator := validator.New(validator.WithRequiredStructEnabled())
	router := echo.New()
	router.HideBanner = true
	router.HidePort = true
	router.Validator = &Validator{
		ValidatorProvider: customValidator,
	}
	router.HTTPErrorHandler = api.customHTTPErrorHandler

	router.Use(middleware.Recover())
	router.Use(middleware.BodyLimit(maxBodySize))
	router.Use(middleware.ContextTimeout(util.SLATimeout))
	if o.Debug {
		router.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogStatus:   true,
			LogURI:      true,
			LogError:    true,
			HandleError: true,
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
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
	if o.EnableMetrics {
		router.GET("/metrics", api.metricsHandler)
	}

	if o.EnableDocs {
		router.GET("/docs", api.docsHandler)
	}

	authGroup := router.Group("/auth")
	authGroup.POST("/login", api.loginHandler)

	apiGroup := router.Group(apiVersion)
	apiGroup.Use(echojwt.WithConfig(api.apiJWTAuthConfig()))
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

	api.router = router
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
