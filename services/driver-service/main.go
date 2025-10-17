package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"github.com/Anurag-Mishra22/taxi/shared/env"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	"github.com/Anurag-Mishra22/taxi/shared/tracing"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9092"

func main() {
	// Initialize Metrics
	appMetrics := metrics.InitMetrics("driver-service")
	log.Println("Metrics initialized for driver-service")

	// Start metrics server on port 9090
	metricsServer := metrics.NewMetricsServer(9090)
	if err := metricsServer.Start(); err != nil {
		log.Printf("Failed to start metrics server: %v", err)
	}
	log.Printf("Metrics server started on port %d", metricsServer.Port())

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
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

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

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

	svc := NewService(appMetrics)

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// Starting the gRPC server with metrics and tracing
	grpcOpts := []grpcserver.ServerOption{
		grpcserver.ChainUnaryInterceptor(
			metrics.UnaryServerInterceptor(appMetrics),
		),
	}
	grpcOpts = append(grpcOpts, tracing.WithTracingInterceptors()...)
	grpcServer := grpcserver.NewServer(grpcOpts...)
	NewGrpcHandler(grpcServer, svc, appMetrics)

	consumer := NewTripConsumer(rabbitmq, svc, appMetrics)
	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to the message: %v", err)
		}
	}()

	log.Printf("Starting gRPC server Driver service on port %s", lis.Addr().String())

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
