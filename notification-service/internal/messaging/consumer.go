package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"
)

const (
	exchangeName    = "payments"
	exchangeType    = "direct"
	queueName       = "payment.completed"
	routingKey      = "payment.completed"
	dlxExchangeName = "payments.dlx"
	dlqName         = "payment.completed.dlq"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	uc      usecase.NotificationUsecase
}

func NewConsumer(url string, uc usecase.NotificationUsecase) (*Consumer, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[Consumer] RabbitMQ not ready, retrying in 3s (%d/10)...", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err = ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("set QoS: %w", err)
	}

	if err = ch.ExchangeDeclare(dlxExchangeName, "direct", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare DLX: %w", err)
	}
	if _, err = ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare DLQ: %w", err)
	}
	if err = ch.QueueBind(dlqName, routingKey, dlxExchangeName, false, nil); err != nil {
		return nil, fmt.Errorf("bind DLQ: %w", err)
	}

	if err = ch.ExchangeDeclare(exchangeName, exchangeType, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    dlxExchangeName,
		"x-dead-letter-routing-key": routingKey,
		"x-message-ttl":             int32(60000),
		"x-max-delivery-count":      int32(3),
	}
	if _, err = ch.QueueDeclare(queueName, true, false, false, false, queueArgs); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}
	if err = ch.QueueBind(queueName, routingKey, exchangeName, false, nil); err != nil {
		return nil, fmt.Errorf("bind queue: %w", err)
	}

	return &Consumer{conn: conn, channel: ch, uc: uc}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		queueName,
		"notification-consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Printf("[Consumer] Waiting for messages on queue '%s'...", queueName)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			c.handleDelivery(msg)
		}
	}
}

func (c *Consumer) handleDelivery(msg amqp.Delivery) {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Consumer] Failed to unmarshal message: %v", err)
		_ = msg.Nack(false, false)
		return
	}

	if err := c.uc.HandlePaymentEvent(event); err != nil {
		log.Printf("[Consumer] Processing failed for event %s: %v", event.EventID, err)

		deaths, _ := msg.Headers["x-death"].([]interface{})
		retries := len(deaths)

		if retries >= 3 {
			_ = msg.Nack(false, false)
		} else {
			_ = msg.Nack(false, true)
		}
		return
	}

	_ = msg.Ack(false)
}

func (c *Consumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}
