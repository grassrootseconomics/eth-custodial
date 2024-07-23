package api

import (
	"net/http"

	"github.com/VictoriaMetrics/metrics"
)

type metricsHandler struct {
	enabled bool
}

func newMetricshandler(enabled bool) *metricsHandler {
	return &metricsHandler{enabled: enabled}
}

func (m *metricsHandler) handle(w http.ResponseWriter, r *http.Request) {
	if !m.enabled {
		http.Error(w, "Metrics are disabled", http.StatusNotFound)
		return
	}
	metrics.WritePrometheus(w, true)
}
