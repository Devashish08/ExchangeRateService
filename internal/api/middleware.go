package api

import (
	"net/http"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/metrics"
	"github.com/go-chi/chi/v5"
)

// MetricsMiddleware records request count and duration labeled by method and route.
func MetricsMiddleware(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)

			duration := time.Since(start)
			routePattern := chi.RouteContext(r.Context()).RoutePattern()
			m.HttpRequestsTotal.WithLabelValues(r.Method, routePattern).Inc()
			m.HttpRequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration.Seconds())
		})
	}
}
