package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Anurag-Mishra22/taxi/services/payment-service/internal/events"
	"github.com/Anurag-Mishra22/taxi/services/payment-service/internal/infrastructure/stripe"
	"github.com/Anurag-Mishra22/taxi/services/payment-service/internal/service"
	"github.com/Anurag-Mishra22/taxi/services/payment-service/pkg/types"
	"github.com/Anurag-Mishra22/taxi/shared/env"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	"github.com/Anurag-Mishra22/taxi/shared/tracing"
)

var GrpcAddr = env.GetString("GRPC_ADDR", ":9004")

func main() {
	// Initialize Metrics
	appMetrics := metrics.InitMetrics("payment-service")
	log.Println("Metrics initialized for payment-service")

	// Start metrics server on port 9090
	metricsServer := metrics.NewMetricsServer(9090)
	if err := metricsServer.Start(); err != nil {
		log.Printf("Failed to start metrics server: %v", err)
	}
	log.Printf("Metrics server started on port %d", metricsServer.Port())

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName: "payment-service",
		Environment: env.GetString("ENVIRONMENT", "development"),
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
	
	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	// Setup graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	appURL := env.GetString("APP_URL", "http://localhost:3000")

	// Stripe config
	stripeCfg := &types.PaymentConfig{
		StripeSecretKey: env.GetString("STRIPE_SECRET_KEY", ""),
		SuccessURL:      env.GetString("STRIPE_SUCCESS_URL", appURL+"?payment=success"),
		CancelURL:       env.GetString("STRIPE_CANCEL_URL", appURL+"?payment=cancel"),
	}

	if stripeCfg.StripeSecretKey == "" {
		log.Fatalf("STRIPE_SECRET_KEY is not set")
		return
	}

	// Stripe processor
	paymentProcessor := stripe.NewStripeClient(stripeCfg, appMetrics)

	// Service
	svc := service.NewPaymentService(paymentProcessor, appMetrics)

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// Trip Consumer
	tripConsumer := events.NewTripConsumer(rabbitmq, svc, appMetrics)
	go tripConsumer.Listen()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down payment service...")
}
