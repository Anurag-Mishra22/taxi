package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/infrastructure/events"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/infrastructure/grpc"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/infrastructure/repository"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/service"
	"github.com/Anurag-Mishra22/taxi/shared/db"
	"github.com/Anurag-Mishra22/taxi/shared/env"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	"github.com/Anurag-Mishra22/taxi/shared/tracing"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9093"

func main() {
	// Initialize Metrics
	appMetrics := metrics.InitMetrics("trip-service")
	log.Println("Metrics initialized for trip-service")

	// Start metrics server on port 9090
	metricsServer := metrics.NewMetricsServer(9090)
	if err := metricsServer.Start(); err != nil {
		log.Printf("Failed to start metrics server: %v", err)
	}
	log.Printf("Metrics server started on port %d", metricsServer.Port())

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "trip-service",
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

	// Initialize MongoDB
	mongoClient, err := db.NewMongoClient(ctx, db.NewMongoDefaultConfig())
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB, err: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	mongoDb := db.GetDatabase(mongoClient, db.NewMongoDefaultConfig())

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	mongoDBRepo := repository.NewMongoRepository(mongoDb, appMetrics)
	svc := service.NewService(mongoDBRepo, appMetrics)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	publisher := events.NewTripEventPublisher(rabbitmq, appMetrics)

	// Start driver consumer
	driverConsumer := events.NewDriverConsumer(rabbitmq, svc, appMetrics)
	go driverConsumer.Listen()

	// Start payment consumer
	paymentConsumer := events.NewPaymentConsumer(rabbitmq, svc, appMetrics)
	go paymentConsumer.Listen()

	// Starting the gRPC server with metrics and tracing
	grpcOpts := []grpcserver.ServerOption{
		grpcserver.ChainUnaryInterceptor(
			metrics.UnaryServerInterceptor(appMetrics),
		),
	}
	grpcOpts = append(grpcOpts, tracing.WithTracingInterceptors()...)
	grpcServer := grpcserver.NewServer(grpcOpts...)
	grpc.NewGRPCHandler(grpcServer, svc, publisher, appMetrics)

	log.Printf("Starting gRPC server Trip service on port %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	<-ctx.Done()
	log.Println("Shutting down the server...")
	grpcServer.GracefulStop()
}
