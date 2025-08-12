package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds Prometheus collectors for the service.
type Metrics struct {
	HttpRequestsTotal     *prometheus.CounterVec
	HttpRequestDuration   *prometheus.HistogramVec
	ProviderRequestsTotal *prometheus.CounterVec
}

// NewMetrics creates and registers Prometheus metrics with the provided registry.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	return &Metrics{
		HttpRequestsTotal: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "path"},
		),
		HttpRequestDuration: promauto.With(reg).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		ProviderRequestsTotal: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "provider_requests_total",
				Help: "Total number of requests to external providers.",
			},
			[]string{"provider", "status"},
		),
	}
}
