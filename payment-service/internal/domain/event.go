package domain

import "time"

// PaymentEvent is the message published to RabbitMQ after a successful payment.
type PaymentEvent struct {
	EventID       string    `json:"event_id"`
	OrderID       string    `json:"order_id"`
	Amount        int64     `json:"amount"`
	CustomerEmail string    `json:"customer_email"`
	Status        string    `json:"status"`
	OccurredAt    time.Time `json:"occurred_at"`
}
