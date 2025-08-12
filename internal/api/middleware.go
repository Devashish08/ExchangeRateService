
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/metrics"

	"github.com/go-chi/chi/v5"
)

// responseWriterInterceptor is a custom ResponseWriter that captures the status code.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

// newResponseWriterInterceptor creates a new interceptor.
// It defaults the status code to 200 OK, as this is the default for http.ResponseWriter.
func newResponseWriterInterceptor(w http.ResponseWriter) *responseWriterInterceptor {
	return &responseWriterInterceptor{w, http.StatusOK}
}

// WriteHeader captures the status code before calling the underlying ResponseWriter.
func (wri *responseWriterInterceptor) WriteHeader(code int) {
	wri.statusCode = code
	wri.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			interceptor := newResponseWriterInterceptor(w)

			next.ServeHTTP(interceptor, r)

			duration := time.Since(start)

			routePattern := chi.RouteContext(r.Context()).RoutePattern()
			if routePattern == "" {
				routePattern = "unmatched"
			}

			statusCodeStr := strconv.Itoa(interceptor.statusCode)

			m.HttpRequestsTotal.WithLabelValues(r.Method, routePattern, statusCodeStr).Inc()
			m.HttpRequestDuration.WithLabelValues(r.Method, routePattern, statusCodeStr).Observe(duration.Seconds())
		})
	}
}
