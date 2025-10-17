package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"github.com/Anurag-Mishra22/taxi/shared/contracts"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
	metrics  *metrics.Metrics
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service, m *metrics.Metrics) *tripConsumer {
	return &tripConsumer{
		rabbitmq: rabbitmq,
		service:  service,
		metrics:  m,
	}
}

func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		// Start timer for ENTIRE message processing
		start := time.Now()
		
		// Unmarshal outer wrapper
		var tripEvent contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			if c.metrics != nil {
				c.metrics.RecordMessageConsumed(messaging.FindAvailableDriversQueue, "error", time.Since(start), msg.RoutingKey)
			}
			return err
		}

		// Unmarshal inner payload
		var payload messaging.TripEventData
		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			if c.metrics != nil {
				c.metrics.RecordMessageConsumed(messaging.FindAvailableDriversQueue, "error", time.Since(start), msg.RoutingKey)
			}
			return err
		}

		log.Printf("driver received message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			// Start timer for JUST the matching logic (after unmarshaling)
			matchStart := time.Now()
			
			// Call handler (this does the actual driver matching)
			err := c.handleFindAndNotifyDrivers(ctx, payload)
			
			// Record matching duration (pure business logic time)
			if c.metrics != nil {
				c.metrics.DriverMatchDuration.Observe(time.Since(matchStart).Seconds())
			}
			
			// Record total message processing time
			status := "success"
			if err != nil {
				status = "error"
			}
			if c.metrics != nil {
				c.metrics.RecordMessageConsumed(messaging.FindAvailableDriversQueue, status, time.Since(start), msg.RoutingKey)
			}
			
			return err
		}

		log.Printf("unknown trip event: %+v", payload)

		if c.metrics != nil {
			c.metrics.RecordMessageConsumed(messaging.FindAvailableDriversQueue, "success", time.Since(start), msg.RoutingKey)
		}

		return nil
	})
}

func (c *tripConsumer) handleFindAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	suitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug)

	log.Printf("Found %d suitable drivers for package '%s'", len(suitableIDs), payload.Trip.SelectedFare.PackageSlug)

	if len(suitableIDs) == 0 {
		// Notify the driver that no drivers are available
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("Failed to publish message to exchange: %v", err)
			return err
		}

		return nil
	}

	// Get a random index from the matching drivers
	randomIndex := rand.Intn(len(suitableIDs))

	suitableDriverID := suitableIDs[randomIndex]

	marshalledEvent, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Notify the driver about a potential trip
	if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data:    marshalledEvent,
	}); err != nil {
		log.Printf("Failed to publish message to exchange: %v", err)
		return err
	}

	return nil
}
