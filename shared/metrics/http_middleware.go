package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default status
		size:           0,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// HTTPMetricsMiddleware returns a middleware that instruments HTTP handlers with Prometheus metrics
func HTTPMetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract endpoint from request (you can customize this based on your routing)
			endpoint := r.URL.Path
			method := r.Method

			// Increment active requests
			metrics.HTTPActiveRequests.WithLabelValues(method, endpoint).Inc()
			defer metrics.HTTPActiveRequests.WithLabelValues(method, endpoint).Dec()

			// Wrap response writer to capture status code and size
			rw := newResponseWriter(w)

			// Start timer
			start := time.Now()

			// Call next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Record metrics
			statusCode := strconv.Itoa(rw.statusCode)
			metrics.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
			metrics.HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(rw.size))
		})
	}
}

// HTTPMetricsHandler is a convenience function that wraps a single handler
func HTTPMetricsHandler(metrics *Metrics, endpoint string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := r.Method

		// Increment active requests
		metrics.HTTPActiveRequests.WithLabelValues(method, endpoint).Inc()
		defer metrics.HTTPActiveRequests.WithLabelValues(method, endpoint).Dec()

		// Wrap response writer
		rw := newResponseWriter(w)

		// Start timer
		start := time.Now()

		// Call handler
		handler(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Record metrics
		statusCode := strconv.Itoa(rw.statusCode)
		metrics.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
		metrics.HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(rw.size))
	}
}

// RecoveryMiddleware wraps handlers with panic recovery and metrics
func RecoveryMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					metrics.PanicRecoveryTotal.WithLabelValues("http_handler").Inc()
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}