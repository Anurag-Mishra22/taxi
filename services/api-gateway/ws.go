package main

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/Anurag-Mishra22/taxi/services/api-gateway/grpc_clients"
	"github.com/Anurag-Mishra22/taxi/shared/contracts"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/proto/driver"
	"time"
)

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user ID provided")
		return
	}

	// Add connection to manager
	connManager.Add(userID, conn)
	if appMetrics != nil {
		appMetrics.WebSocketConnectionsActive.Inc()
	}
	defer func() {
		connManager.Remove(userID)
		if appMetrics != nil {
			appMetrics.WebSocketConnectionsActive.Dec()
		}
	}()

	// Initialize queue consumers
	queues := []string{
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignQueue,
		messaging.NotifyPaymentSessionCreatedQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Printf("Received message: %s", message)
	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user ID provided")
		return
	}

	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Println("No package slug provided")
		return
	}

	// Add connection to manager
	connManager.Add(userID, conn)
	if appMetrics != nil {
		appMetrics.WebSocketConnectionsActive.Inc()
	}

	ctx := r.Context()

	driverService, err := grpc_clients.NewDriverServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Closing connections
	defer func() {
		connManager.Remove(userID)
		if appMetrics != nil {
			appMetrics.WebSocketConnectionsActive.Dec()
		}

		grpcStart := time.Now()
		driverService.Client.UnregisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})
		if appMetrics != nil {
			appMetrics.GRPCRequestDuration.WithLabelValues("UnregisterDriver").Observe(time.Since(grpcStart).Seconds())
			appMetrics.GRPCRequestsTotal.WithLabelValues("UnregisterDriver", "success").Inc()
		}

		driverService.Close()

		log.Println("Driver unregistered: ", userID)
	}()

	grpcStart := time.Now()
	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("Error registering driver: %v", err)
		if appMetrics != nil {
			appMetrics.GRPCRequestDuration.WithLabelValues("RegisterDriver").Observe(time.Since(grpcStart).Seconds())
			appMetrics.GRPCRequestsTotal.WithLabelValues("RegisterDriver", "error").Inc()
		}
		return
	}
	if appMetrics != nil {
		appMetrics.GRPCRequestDuration.WithLabelValues("RegisterDriver").Observe(time.Since(grpcStart).Seconds())
		appMetrics.GRPCRequestsTotal.WithLabelValues("RegisterDriver", "success").Inc()
	}

	if err := connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverData.Driver,
	}); err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}

	// Initialize queue consumers
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		type driverMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		var driverMsg driverMessage
		if err := json.Unmarshal(message, &driverMsg); err != nil {
			log.Printf("Error unmarshaling driver message: %v", err)
			continue
		}

		// Handle the different message type
		switch driverMsg.Type {
		case contracts.DriverCmdLocation:
			// Handle driver location update in the future
			continue
		case contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline:
			// Forward the message to RabbitMQ
			if err := rb.PublishMessage(ctx, driverMsg.Type, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    driverMsg.Data,
			}); err != nil {
				log.Printf("Error publishing message to RabbitMQ: %v", err)
				if appMetrics != nil {
					appMetrics.RecordMessagePublished(driverMsg.Type, driverMsg.Type, "error")
				}
			} else {
				if appMetrics != nil {
					appMetrics.RecordMessagePublished(driverMsg.Type, driverMsg.Type, "success")
				}
			}
		default:
			log.Printf("Unknown message type: %s", driverMsg.Type)
		}
	}
}