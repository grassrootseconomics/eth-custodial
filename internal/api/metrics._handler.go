package api

import (
	"github.com/VictoriaMetrics/metrics"
	"github.com/labstack/echo/v4"
)

func (a *API) metricsHandler(c echo.Context) error {
	metrics.WritePrometheus(c.Response(), true)
	return nil
}
