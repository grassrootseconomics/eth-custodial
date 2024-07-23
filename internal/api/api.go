package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type (
	APIOpts struct {
		Debug         bool
		EnableMetrics bool
		ListenAddress string
		Logg          *slog.Logger
	}

	API struct {
		logg   *slog.Logger
		server *http.Server
	}
)

const (
	apiVersion         = "/v2"
	maxBodySize        = 1 << 20
	allowedContentType = "application/json"
)

func New(o APIOpts) *API {
	r := chi.NewRouter()

	if o.Debug {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestSize(maxBodySize))
	r.Use(middleware.AllowContentType(allowedContentType))

	metricsHandler := newMetricshandler((o.EnableMetrics))
	r.Get("/metrics", metricsHandler.handle)

	return &API{
		logg: o.Logg,
		server: &http.Server{
			ReadTimeout: 30 * time.Second,
			Addr:        o.ListenAddress,
			Handler:     r,
		},
	}
}

func (a *API) Start() error {
	a.logg.Info("starting API HTTP server", "listen_address", a.server.Addr)
	return a.server.ListenAndServe()
}

func (a *API) Stop(ctx context.Context) error {
	a.logg.Info("shutting down API server")
	return a.server.Shutdown(ctx)
}
