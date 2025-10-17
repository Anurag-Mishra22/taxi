package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Anurag-Mishra22/taxi/services/payment-service/internal/domain"
	"github.com/Anurag-Mishra22/taxi/services/payment-service/pkg/types"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"

	"github.com/google/uuid"
)

type paymentService struct {
	paymentProcessor domain.PaymentProcessor
	metrics          *metrics.Metrics
}

// NewPaymentService creates a new instance of the payment service
func NewPaymentService(paymentProcessor domain.PaymentProcessor, m *metrics.Metrics) domain.Service {
	return &paymentService{
		paymentProcessor: paymentProcessor,
		metrics:          m,
	}
}

// CreatePaymentSession creates a new payment session for a trip
func (s *paymentService) CreatePaymentSession(
	ctx context.Context,
	tripID string,
	userID string,
	driverID string,
	amount int64,
	currency string,
) (*types.PaymentIntent, error) {
	metadata := map[string]string{
		"trip_id":   tripID,
		"user_id":   userID,
		"driver_id": driverID,
	}

	sessionID, err := s.paymentProcessor.CreatePaymentSession(ctx, amount, currency, metadata)
	if err != nil {
		if s.metrics != nil {
			s.metrics.PaymentErrors.WithLabelValues("session_creation_failed").Inc()
		}
		return nil, fmt.Errorf("failed to create payment session: %w", err)
	}

	if s.metrics != nil {
		s.metrics.RecordPayment("created", "stripe", currency, float64(amount))
	}

	paymentIntent := &types.PaymentIntent{
		ID:              uuid.New().String(),
		TripID:          tripID,
		UserID:          userID,
		DriverID:        driverID,
		Amount:          amount,
		Currency:        currency,
		StripeSessionID: sessionID,
		CreatedAt:       time.Now(),
	}

	return paymentIntent, nil
}
