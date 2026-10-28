package httpapi

import (
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds Prometheus metrics for the HTTP layer.
type Metrics struct {
	// RequestDuration is a histogram of request durations in seconds.
	RequestDuration *prometheus.HistogramVec
	// RequestCount is a counter of total requests by method, path, and status.
	RequestCount *prometheus.CounterVec
}

// NewMetrics creates new Prometheus metrics and registers them.
func NewMetrics() *Metrics {
	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "secunda",
		Name:      "request_duration_seconds",
		Help:      "Request duration in seconds.",
		Buckets:   prometheus.ExponentialBuckets(0.005, 2, 10), // 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.28s, 2.56s
	}, []string{"method", "path", "status"})

	requestCount := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "secunda",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	prometheus.MustRegister(requestDuration, requestCount)

	return &Metrics{
		RequestDuration: requestDuration,
		RequestCount:    requestCount,
	}
}

// PrometheusMiddleware is a Chi middleware that records Prometheus metrics.
// It should be added after RequestLogger (or replace it) in the middleware chain.
func PrometheusMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture the status code.
			wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			status := strStatus(wrapped.Status())

			// Record histogram.
			metrics.RequestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)

			// Record counter.
			metrics.RequestCount.WithLabelValues(r.Method, r.URL.Path, status).Add(1)
		})
	}
}

// strStatus converts an integer HTTP status code to a string label.
func strStatus(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500 && status < 600:
		return "5xx"
	default:
		return "other"
	}
}