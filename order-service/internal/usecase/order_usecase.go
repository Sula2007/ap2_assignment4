package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"order-service/internal/domain"
	"order-service/internal/repository"
)

type OrderUsecase interface {
	CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error)
	GetOrder(id string) (*domain.Order, error)
	CancelOrder(id string) error
}

type orderUsecase struct {
	orderRepo     repository.OrderRepository
	paymentClient repository.PaymentClient
	cache         repository.OrderCache
	cacheTTL      time.Duration
}

func NewOrderUsecase(
	orderRepo repository.OrderRepository,
	paymentClient repository.PaymentClient,
	cache repository.OrderCache,
	cacheTTL time.Duration,
) OrderUsecase {
	return &orderUsecase{
		orderRepo:     orderRepo,
		paymentClient: paymentClient,
		cache:         cache,
		cacheTTL:      cacheTTL,
	}
}

func (u *orderUsecase) CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	order := &domain.Order{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     "Pending",
		CreatedAt:  time.Now(),
	}

	if err := u.orderRepo.Create(order); err != nil {
		return nil, err
	}

	paymentResp, err := u.paymentClient.AuthorizePayment(&domain.PaymentRequest{
		OrderID: order.ID,
		Amount:  amount,
	})

	if err != nil {
		u.orderRepo.UpdateStatus(order.ID, "Failed")
		u.cache.Delete(context.Background(), order.ID)
		return nil, errors.New("payment service unavailable")
	}

	if paymentResp.Status == "Authorized" {
		u.orderRepo.UpdateStatus(order.ID, "Paid")
		order.Status = "Paid"
	} else {
		u.orderRepo.UpdateStatus(order.ID, "Failed")
		order.Status = "Failed"
	}

	u.cache.Set(context.Background(), order, u.cacheTTL)

	return order, nil
}

func (u *orderUsecase) GetOrder(id string) (*domain.Order, error) {
	ctx := context.Background()

	cached, err := u.cache.Get(ctx, id)
	if err == nil && cached != nil {
		return cached, nil
	}

	order, err := u.orderRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}

	u.cache.Set(ctx, order, u.cacheTTL)

	return order, nil
}

func (u *orderUsecase) CancelOrder(id string) error {
	order, err := u.orderRepo.FindByID(id)
	if err != nil {
		return err
	}
	if order == nil {
		return errors.New("order not found")
	}
	if order.Status == "Paid" {
		return errors.New("cannot cancel paid order")
	}
	if order.Status != "Pending" {
		return errors.New("only pending orders can be cancelled")
	}

	if err := u.orderRepo.UpdateStatus(id, "Cancelled"); err != nil {
		return err
	}

	u.cache.Delete(context.Background(), id)

	return nil
}
