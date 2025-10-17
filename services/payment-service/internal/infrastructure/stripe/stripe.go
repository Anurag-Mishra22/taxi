package stripe

import (
	"context"
	"fmt"
	"github.com/Anurag-Mishra22/taxi/services/payment-service/internal/domain"
	"github.com/Anurag-Mishra22/taxi/services/payment-service/pkg/types"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

type stripeClient struct {
	config  *types.PaymentConfig
	metrics *metrics.Metrics
}

func NewStripeClient(config *types.PaymentConfig, m *metrics.Metrics) domain.PaymentProcessor {
	stripe.Key = config.StripeSecretKey

	return &stripeClient{
		config:  config,
		metrics: m,
	}
}

func (s *stripeClient) CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (string, error) {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.ExternalAPICallDuration.WithLabelValues("stripe").Observe(time.Since(start).Seconds())
		}
	}()
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(s.config.SuccessURL),
		CancelURL:  stripe.String(s.config.CancelURL),
		Metadata: metadata,
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Ride Payment"),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
	}

	result, err := session.New(params)
	if err != nil {
		if s.metrics != nil {
			s.metrics.ExternalAPICallsTotal.WithLabelValues("stripe", "error").Inc()
		}
		return "", fmt.Errorf("failed to create a payment session on stripe: %w", err)
	}

	if s.metrics != nil {
		s.metrics.ExternalAPICallsTotal.WithLabelValues("stripe", "success").Inc()
	}

	return result.ID, nil
}
