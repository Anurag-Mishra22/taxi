package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Anurag-Mishra22/taxi/shared/env"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	"github.com/Anurag-Mishra22/taxi/shared/tracing"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	appMetrics  *metrics.Metrics
)

func main() {
	log.Println("Starting API Gateway")

	// Initialize Metrics
	appMetrics = metrics.InitMetrics("api-gateway")
	log.Println("Metrics initialized for api-gateway")

	// Start metrics server on port 9090
	metricsServer := metrics.NewMetricsServer(9090)
	if err := metricsServer.Start(); err != nil {
		log.Printf("Failed to start metrics server: %v", err)
	}
	log.Printf("Metrics server started on port %d", metricsServer.Port())

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)
	defer metricsServer.Stop(ctx)

	mux := http.NewServeMux()

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	mux.Handle("POST /trip/preview", tracing.WrapHandlerFunc(metricsMiddleware(enableCORS(handleTripPreview), "POST", "/trip/preview"), "/trip/preview"))
	mux.Handle("POST /trip/start", tracing.WrapHandlerFunc(metricsMiddleware(enableCORS(handleTripStart), "POST", "/trip/start"), "/trip/start"))
	mux.Handle("/ws/drivers", tracing.WrapHandlerFunc(metricsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq)
	}, "GET", "/ws/drivers"), "/ws/drivers"))
	mux.Handle("/ws/riders", tracing.WrapHandlerFunc(metricsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq)
	}, "GET", "/ws/riders"), "/ws/riders"))
	mux.Handle("/webhook/stripe", tracing.WrapHandlerFunc(metricsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleStripeWebhook(w, r, rabbitmq)
	}, "POST", "/webhook/stripe"), "/webhook/stripe"))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server listening on %s", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error starting the server: %v", err)

	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Could not stop the server gracefully: %v", err)
			server.Close()
		}
	}
}
