package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"payment-service/internal/domain"
)

const (
	ExchangeName    = "payments"
	ExchangeType    = "direct"
	QueueName       = "payment.completed"
	RoutingKey      = "payment.completed"
	DLXExchangeName = "payments.dlx"
	DLQName         = "payment.completed.dlq"
	MaxRetries      = 3
)

type rabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(url string) (Publisher, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[Publisher] RabbitMQ not ready, retrying in 3s (%d/10)...", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err = ch.ExchangeDeclare(DLXExchangeName, "direct", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare DLX exchange: %w", err)
	}
	if _, err = ch.QueueDeclare(DLQName, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare DLQ: %w", err)
	}
	if err = ch.QueueBind(DLQName, RoutingKey, DLXExchangeName, false, nil); err != nil {
		return nil, fmt.Errorf("bind DLQ: %w", err)
	}

	if err = ch.ExchangeDeclare(ExchangeName, ExchangeType, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    DLXExchangeName,
		"x-dead-letter-routing-key": RoutingKey,
		"x-message-ttl":             int32(60000),
		"x-max-delivery-count":      int32(MaxRetries),
	}
	if _, err = ch.QueueDeclare(QueueName, true, false, false, false, queueArgs); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}
	if err = ch.QueueBind(QueueName, RoutingKey, ExchangeName, false, nil); err != nil {
		return nil, fmt.Errorf("bind queue: %w", err)
	}

	return &rabbitMQPublisher{conn: conn, channel: ch}, nil
}

func (p *rabbitMQPublisher) PublishPaymentEvent(event domain.PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(ctx,
		ExchangeName,
		RoutingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.EventID,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	log.Printf("[Publisher] Published event %s for order %s", event.EventID, event.OrderID)
	return nil
}

func (p *rabbitMQPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}
