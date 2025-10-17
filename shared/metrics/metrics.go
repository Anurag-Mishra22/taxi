package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// HTTP Metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec
	HTTPActiveRequests  *prometheus.GaugeVec

	// gRPC Metrics
	GRPCRequestsTotal   *prometheus.CounterVec
	GRPCRequestDuration *prometheus.HistogramVec
	GRPCActiveRequests  *prometheus.GaugeVec

	// Database Metrics
	DBConnectionsActive   prometheus.Gauge
	DBConnectionsIdle     prometheus.Gauge
	DBQueryDuration       *prometheus.HistogramVec
	DBQueriesTotal        *prometheus.CounterVec
	DBConnectionErrors    prometheus.Counter

	// RabbitMQ Metrics
	MessagesPublishedTotal   *prometheus.CounterVec
	MessagesConsumedTotal    *prometheus.CounterVec
	MessageProcessingDuration *prometheus.HistogramVec
	MessageProcessingErrors   *prometheus.CounterVec
	MessageRetries            *prometheus.CounterVec
	QueueSize                 *prometheus.GaugeVec
	DeadLetterQueueSize       prometheus.Gauge

	// Business Metrics (Trip Service Specific)
	TripsCreatedTotal      *prometheus.CounterVec
	TripsFareCalculated    *prometheus.CounterVec
	TripsCancelled         *prometheus.CounterVec
	FareCalculationDuration prometheus.Histogram
	ActiveTrips            prometheus.Gauge

	// Business Metrics (Driver Service Specific)
	DriversOnline          prometheus.Gauge
	DriversRegisteredTotal prometheus.Counter
	DriverMatchDuration    prometheus.Histogram

	// Business Metrics (Payment Service Specific)
	PaymentsProcessedTotal *prometheus.CounterVec
	PaymentAmount          *prometheus.HistogramVec
	PaymentErrors          *prometheus.CounterVec

	// API Gateway Metrics
	WebSocketConnectionsActive prometheus.Gauge
	WebSocketMessagesTotal     *prometheus.CounterVec

	// System Metrics
	ServiceUptime       prometheus.Counter
	PanicRecoveryTotal  *prometheus.CounterVec
	MemoryAllocBytes    prometheus.Gauge
	GoroutinesActive    prometheus.Gauge

	// External API Metrics
	ExternalAPICallsTotal    *prometheus.CounterVec
	ExternalAPICallDuration  *prometheus.HistogramVec
	ExternalAPICircuitBreaker *prometheus.GaugeVec
}

var (
	// Global metrics instance - can be accessed by all services
	AppMetrics *Metrics
)

// InitMetrics initializes Prometheus metrics with service name
func InitMetrics(serviceName string) *Metrics {
	labels := prometheus.Labels{"service": serviceName}

	metrics := &Metrics{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "http",
				Name:        "requests_total",
				Help:        "Total number of HTTP requests",
				ConstLabels: labels,
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "http",
				Name:        "request_duration_seconds",
				Help:        "HTTP request duration in seconds",
				ConstLabels: labels,
				Buckets:     []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "http",
				Name:        "response_size_bytes",
				Help:        "HTTP response size in bytes",
				ConstLabels: labels,
				Buckets:     prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		HTTPActiveRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "http",
				Name:        "active_requests",
				Help:        "Number of active HTTP requests",
				ConstLabels: labels,
			},
			[]string{"method", "endpoint"},
		),

		// gRPC Metrics
		GRPCRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "grpc",
				Name:        "requests_total",
				Help:        "Total number of gRPC requests",
				ConstLabels: labels,
			},
			[]string{"method", "status"},
		),
		GRPCRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "grpc",
				Name:        "request_duration_seconds",
				Help:        "gRPC request duration in seconds",
				ConstLabels: labels,
				Buckets:     []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"method"},
		),
		GRPCActiveRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "grpc",
				Name:        "active_requests",
				Help:        "Number of active gRPC requests",
				ConstLabels: labels,
			},
			[]string{"method"},
		),

		// Database Metrics
		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "database",
				Name:        "connections_active",
				Help:        "Number of active database connections",
				ConstLabels: labels,
			},
		),
		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "database",
				Name:        "connections_idle",
				Help:        "Number of idle database connections",
				ConstLabels: labels,
			},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "database",
				Name:        "query_duration_seconds",
				Help:        "Database query duration in seconds",
				ConstLabels: labels,
				Buckets:     []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"operation", "collection"},
		),
		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "database",
				Name:        "queries_total",
				Help:        "Total number of database queries",
				ConstLabels: labels,
			},
			[]string{"operation", "collection", "status"},
		),
		DBConnectionErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "database",
				Name:        "connection_errors_total",
				Help:        "Total number of database connection errors",
				ConstLabels: labels,
			},
		),

		// RabbitMQ Metrics
		MessagesPublishedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "rabbitmq",
				Name:        "messages_published_total",
				Help:        "Total number of messages published to RabbitMQ",
				ConstLabels: labels,
			},
			[]string{"exchange", "routing_key", "status"},
		),
		MessagesConsumedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "rabbitmq",
				Name:        "messages_consumed_total",
				Help:        "Total number of messages consumed from RabbitMQ",
				ConstLabels: labels,
			},
			[]string{"queue", "status"},
		),
		MessageProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "rabbitmq",
				Name:        "message_processing_duration_seconds",
				Help:        "Message processing duration in seconds",
				ConstLabels: labels,
				Buckets:     []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"queue", "routing_key"},
		),
		MessageProcessingErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "rabbitmq",
				Name:        "message_processing_errors_total",
				Help:        "Total number of message processing errors",
				ConstLabels: labels,
			},
			[]string{"queue", "routing_key", "error_type"},
		),
		MessageRetries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "rabbitmq",
				Name:        "message_retries_total",
				Help:        "Total number of message retries",
				ConstLabels: labels,
			},
			[]string{"queue", "routing_key"},
		),
		QueueSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "rabbitmq",
				Name:        "queue_size",
				Help:        "Current size of RabbitMQ queue",
				ConstLabels: labels,
			},
			[]string{"queue"},
		),
		DeadLetterQueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "rabbitmq",
				Name:        "dead_letter_queue_size",
				Help:        "Current size of dead letter queue",
				ConstLabels: labels,
			},
		),

		// Business Metrics - Trip Service
		TripsCreatedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "trips_created_total",
				Help:        "Total number of trips created",
				ConstLabels: labels,
			},
			[]string{"package_type", "status"},
		),
		TripsFareCalculated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "fares_calculated_total",
				Help:        "Total number of fares calculated",
				ConstLabels: labels,
			},
			[]string{"package_type"},
		),
		TripsCancelled: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "trips_cancelled_total",
				Help:        "Total number of trips cancelled",
				ConstLabels: labels,
			},
			[]string{"reason"},
		),
		FareCalculationDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "fare_calculation_duration_seconds",
				Help:        "Fare calculation duration in seconds",
				ConstLabels: labels,
				Buckets:     []float64{.001, .005, .01, .025, .05, .1},
			},
		),
		ActiveTrips: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "active_trips",
				Help:        "Number of currently active trips",
				ConstLabels: labels,
			},
		),

		// Business Metrics - Driver Service
		DriversOnline: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "drivers_online",
				Help:        "Number of drivers currently online",
				ConstLabels: labels,
			},
		),
		DriversRegisteredTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "drivers_registered_total",
				Help:        "Total number of drivers registered",
				ConstLabels: labels,
			},
		),
		DriverMatchDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "driver_match_duration_seconds",
				Help:        "Driver matching duration in seconds",
				ConstLabels: labels,
				Buckets:     []float64{.1, .25, .5, 1, 2, 5, 10, 30},
			},
		),

		// Business Metrics - Payment Service
		PaymentsProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "payments_processed_total",
				Help:        "Total number of payments processed",
				ConstLabels: labels,
			},
			[]string{"status", "payment_method"},
		),
		PaymentAmount: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "payment_amount_cents",
				Help:        "Payment amount distribution in cents",
				ConstLabels: labels,
				Buckets:     []float64{100, 500, 1000, 2000, 5000, 10000, 20000, 50000},
			},
			[]string{"currency"},
		),
		PaymentErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "business",
				Name:        "payment_errors_total",
				Help:        "Total number of payment processing errors",
				ConstLabels: labels,
			},
			[]string{"error_type"},
		),

		// API Gateway Metrics
		WebSocketConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "websocket",
				Name:        "connections_active",
				Help:        "Number of active WebSocket connections",
				ConstLabels: labels,
			},
		),
		WebSocketMessagesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "websocket",
				Name:        "messages_total",
				Help:        "Total number of WebSocket messages",
				ConstLabels: labels,
			},
			[]string{"type", "direction"},
		),

		// System Metrics
		ServiceUptime: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "system",
				Name:        "uptime_seconds_total",
				Help:        "Total uptime of the service in seconds",
				ConstLabels: labels,
			},
		),
		PanicRecoveryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "system",
				Name:        "panic_recovery_total",
				Help:        "Total number of panic recoveries",
				ConstLabels: labels,
			},
			[]string{"location"},
		),

		// External API Metrics
		ExternalAPICallsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "external_api",
				Name:        "calls_total",
				Help:        "Total number of external API calls",
				ConstLabels: labels,
			},
			[]string{"api_name", "status"},
		),
		ExternalAPICallDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "external_api",
				Name:        "call_duration_seconds",
				Help:        "External API call duration in seconds",
				ConstLabels: labels,
				Buckets:     []float64{.1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"api_name"},
		),
		ExternalAPICircuitBreaker: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   "ride_sharing",
				Subsystem:   "external_api",
				Name:        "circuit_breaker_state",
				Help:        "Circuit breaker state (0=closed, 1=open, 2=half-open)",
				ConstLabels: labels,
			},
			[]string{"api_name"},
		),
	}

	// Set global metrics instance
	AppMetrics = metrics

	// Start uptime counter
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			metrics.ServiceUptime.Inc()
		}
	}()

	return metrics
}

// Helper functions to make metrics recording easier

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordGRPCRequest records a gRPC request
func (m *Metrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	m.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordDBQuery records a database query
func (m *Metrics) RecordDBQuery(operation, collection, status string, duration time.Duration) {
	m.DBQueriesTotal.WithLabelValues(operation, collection, status).Inc()
	m.DBQueryDuration.WithLabelValues(operation, collection).Observe(duration.Seconds())
}

// RecordMessagePublished records a published message
func (m *Metrics) RecordMessagePublished(exchange, routingKey, status string) {
	m.MessagesPublishedTotal.WithLabelValues(exchange, routingKey, status).Inc()
}

// RecordMessageConsumed records a consumed message
func (m *Metrics) RecordMessageConsumed(queue, status string, duration time.Duration, routingKey string) {
	m.MessagesConsumedTotal.WithLabelValues(queue, status).Inc()
	m.MessageProcessingDuration.WithLabelValues(queue, routingKey).Observe(duration.Seconds())
}

// RecordTripCreated records a new trip creation
func (m *Metrics) RecordTripCreated(packageType, status string) {
	m.TripsCreatedTotal.WithLabelValues(packageType, status).Inc()
}

// RecordPayment records a payment transaction
func (m *Metrics) RecordPayment(status, paymentMethod, currency string, amount float64) {
	m.PaymentsProcessedTotal.WithLabelValues(status, paymentMethod).Inc()
	m.PaymentAmount.WithLabelValues(currency).Observe(amount)
}