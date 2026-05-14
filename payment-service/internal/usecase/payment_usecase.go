package usecase

import (
	"errors"
	"fmt"
	"time"

	"payment-service/internal/domain"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
)

type PaymentUsecase interface {
	AuthorizePayment(orderID string, amount int64, customerEmail string) (*domain.Payment, error)
	GetPaymentByOrderID(orderID string) (*domain.Payment, error)
}

type paymentUsecase struct {
	repo      repository.PaymentRepository
	publisher messaging.Publisher
}

func NewPaymentUsecase(repo repository.PaymentRepository, publisher messaging.Publisher) PaymentUsecase {
	return &paymentUsecase{repo: repo, publisher: publisher}
}

func (u *paymentUsecase) AuthorizePayment(orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	status := "Authorized"
	if amount > 100000 {
		status = "Declined"
	}

	now := time.Now()
	payment := &domain.Payment{
		ID:            fmt.Sprintf("%d", now.UnixNano()),
		OrderID:       orderID,
		TransactionID: fmt.Sprintf("txn_%d", now.UnixNano()),
		Amount:        amount,
		Status:        status,
		CreatedAt:     now,
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	if status == "Authorized" {
		event := domain.PaymentEvent{
			EventID:       fmt.Sprintf("evt_%d", now.UnixNano()),
			OrderID:       orderID,
			Amount:        amount,
			CustomerEmail: customerEmail,
			Status:        status,
			OccurredAt:    now,
		}
		if err := u.publisher.PublishPaymentEvent(event); err != nil {
			fmt.Printf("[PaymentUsecase] WARNING: failed to publish event: %v\n", err)
		}
	}

	return payment, nil
}

func (u *paymentUsecase) GetPaymentByOrderID(orderID string) (*domain.Payment, error) {
	payment, err := u.repo.FindByOrderID(orderID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, fmt.Errorf("payment not found for order: %s", orderID)
	}
	return payment, nil
}
