package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/domain"
	"github.com/Anurag-Mishra22/taxi/shared/contracts"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"

	"github.com/rabbitmq/amqp091-go"
)

type paymentConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
	metrics  *metrics.Metrics
}

func NewPaymentConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService, m *metrics.Metrics) *paymentConsumer {
	return &paymentConsumer{
		rabbitmq: rabbitmq,
		service:  service,
		metrics:  m,
	}
}

func (c *paymentConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.NotifyPaymentSuccessQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		start := time.Now()
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}
		var payload messaging.PaymentStatusUpdateData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal payload: %v", err)
			return err
		}

		log.Printf("Trip has been completed and payed.")

		err := c.service.UpdateTrip(
			ctx,
			payload.TripID,
			"payed",
			nil,
		)

		if c.metrics != nil {
			status := "success"
			if err != nil {
				status = "error"
			}
			c.metrics.RecordMessageConsumed(messaging.NotifyPaymentSuccessQueue, status, time.Since(start), msg.RoutingKey)
		}

		return err
	})
}
