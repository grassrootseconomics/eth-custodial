package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	APIOpts struct {
		Debug         bool
		EnableMetrics bool
		ListenAddress string
		Logg          *slog.Logger
	}

	API struct {
		listenAddress string
		logg          *slog.Logger
		router        *echo.Echo
	}
)

const (
	apiVersion         = "/api/v2"
	maxBodySize        = "1M"
	allowedContentType = "application/json"
	slaTimeout         = 15 * time.Second

	authorizationHeader = "X-GE-KEY"
)

func New(o APIOpts) *API {
	api := &API{
		listenAddress: o.ListenAddress,
		logg:          o.Logg,
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
	router.Use(middleware.ContextTimeout(slaTimeout))
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

	apiGroup := router.Group(apiVersion)
	// TODO: Use JWT handler and bypass with Key whenever present
	apiGroup.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Skipper: func(c echo.Context) bool {
			_, err := c.Cookie("_ge_auth")
			return !(err != nil)
		},
		KeyLookup: "header:" + authorizationHeader,
		Validator: func(auth string, c echo.Context) (bool, error) {
			return auth == "xd", nil
		},
	}))

	apiGroup.POST("/transfer", api.transferHandler)

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
