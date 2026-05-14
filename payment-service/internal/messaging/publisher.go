package messaging

import "payment-service/internal/domain"

// Publisher is the port that hides the messaging infrastructure from the use-case layer.
type Publisher interface {
	PublishPaymentEvent(event domain.PaymentEvent) error
	Close() error
}
