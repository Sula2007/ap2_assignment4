package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"notification-service/internal/domain"
	"notification-service/internal/provider"
	"notification-service/internal/repository"
)

type NotificationUsecase interface {
	HandlePaymentEvent(event domain.PaymentEvent) error
}

type notificationUsecase struct {
	store      repository.IdempotencyStore
	sender     provider.EmailSender
	maxRetries int
}

func NewNotificationUsecase(store repository.IdempotencyStore, sender provider.EmailSender, maxRetries int) NotificationUsecase {
	return &notificationUsecase{
		store:      store,
		sender:     sender,
		maxRetries: maxRetries,
	}
}

func (u *notificationUsecase) HandlePaymentEvent(event domain.PaymentEvent) error {
	ctx := context.Background()

	already, err := u.store.HasProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if already {
		log.Printf("[Notification] Duplicate event %s – skipping.", event.EventID)
		return nil
	}

	subject := fmt.Sprintf("Payment confirmed for Order #%s", event.OrderID)
	body := fmt.Sprintf("Hello,\n\nYour payment of $%.2f for Order #%s has been confirmed.\n\nThank you!",
		float64(event.Amount)/100.0, event.OrderID)

	if err := u.sendWithRetry(event.CustomerEmail, subject, body); err != nil {
		return fmt.Errorf("send notification failed: %w", err)
	}

	const idempotencyTTL = 72 * time.Hour
	if err := u.store.MarkProcessed(ctx, event.EventID, idempotencyTTL); err != nil {
		return fmt.Errorf("mark processed failed: %w", err)
	}

	return nil
}

func (u *notificationUsecase) sendWithRetry(to, subject, body string) error {
	var lastErr error
	for attempt := 0; attempt <= u.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Printf("[Notification] Retry %d/%d after %s", attempt, u.maxRetries, backoff)
			time.Sleep(backoff)
		}

		if err := u.sender.Send(to, subject, body); err != nil {
			lastErr = err
			log.Printf("[Notification] Send attempt %d failed: %v", attempt+1, err)
			continue
		}

		return nil
	}
	return fmt.Errorf("all %d attempts failed: %w", u.maxRetries+1, lastErr)
}
