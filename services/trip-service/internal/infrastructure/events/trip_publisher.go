package events

import (
	"context"
	"encoding/json"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/domain"
	"github.com/Anurag-Mishra22/taxi/shared/contracts"
	"github.com/Anurag-Mishra22/taxi/shared/messaging"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	"time"
)

type TripEventPublisher struct {
	rabbitmq *messaging.RabbitMQ
	metrics  *metrics.Metrics
}

func NewTripEventPublisher(rabbitmq *messaging.RabbitMQ, m *metrics.Metrics) *TripEventPublisher {
	return &TripEventPublisher{
		rabbitmq: rabbitmq,
		metrics:  m,
	}
}

func (p *TripEventPublisher) PublishTripCreated(ctx context.Context, trip *domain.TripModel) error {
	start := time.Now()
	payload := messaging.TripEventData{
		Trip: trip.ToProto(),
	}

	tripEventJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = p.rabbitmq.PublishMessage(ctx, contracts.TripEventCreated, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    tripEventJSON,
	})

	if p.metrics != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		p.metrics.RecordMessagePublished("trip", contracts.TripEventCreated, status)
		if err == nil {
			p.metrics.MessageProcessingDuration.WithLabelValues("trip", contracts.TripEventCreated).Observe(time.Since(start).Seconds())
		}
	}

	return err
}
