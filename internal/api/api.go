package api

import (
	"context"
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
		APIKey        string
		Debug         bool
		EnableMetrics bool
		EnableDocs    bool
		ListenAddress string
		Store         store.Store
		Logg          *slog.Logger
		ChainProvider *ethutils.Provider
		Worker        *worker.WorkerContainer
	}

	API struct {
		apiKey        string
		listenAddress string
		store         store.Store
		logg          *slog.Logger
		chainProvider *ethutils.Provider
		router        *echo.Echo
		worker        *worker.WorkerContainer
	}
)

const (
	apiVersion         = "/api/v2"
	maxBodySize        = "1M"
	allowedContentType = "application/json"
)

func New(o APIOpts) *API {
	api := &API{
		apiKey:        o.APIKey,
		listenAddress: o.ListenAddress,
		logg:          o.Logg,
		store:         o.Store,
		chainProvider: o.ChainProvider,
		worker:        o.Worker,
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

	apiGroup := router.Group(apiVersion)

	// serviceGroup := apiGroup.Group("/service")
	apiGroup.Use(middleware.KeyAuthWithConfig(api.serviceAPIAuthConfig()))

	apiGroup.GET("/system", api.systemInfoHandler)
	apiGroup.POST("/account/create", api.accountCreateHandler)
	apiGroup.GET("/account/status/:address", api.accountStatusHandler)
	apiGroup.GET("/account/otx/:address", api.getOTXByAddressHandler)
	apiGroup.GET("/otx/track/:trackingId", api.trackOTXHandler)
	apiGroup.POST("/token/transfer", api.transferHandler)
	apiGroup.POST("/pool/quote", api.poolQuoteHandler)
	apiGroup.POST("/pool/swap", api.poolSwapHandler)
	apiGroup.POST("/pool/deposit", api.poolDepositHandler)

	userGroup := apiGroup.Group("/user")
	userGroup.Use(echojwt.WithConfig(api.userAPIJWTAuthConfig()))
	userGroup.GET("/test", api.testRestircted)

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
