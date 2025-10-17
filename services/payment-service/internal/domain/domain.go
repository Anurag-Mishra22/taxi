package domain

import (
	"context"

	"github.com/Anurag-Mishra22/taxi/services/payment-service/pkg/types"
)

type Service interface {
	CreatePaymentSession(ctx context.Context, tripID, userID, driverID string, amount int64, currency string) (*types.PaymentIntent, error)
}

type PaymentProcessor interface {
	CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (string, error)
}
