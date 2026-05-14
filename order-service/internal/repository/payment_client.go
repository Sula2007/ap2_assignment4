package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"order-service/internal/domain"
)

type PaymentClient interface {
	AuthorizePayment(req *domain.PaymentRequest) (*domain.PaymentResponse, error)
}

type paymentHTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewPaymentHTTPClient(baseURL string) PaymentClient {
	return &paymentHTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type paymentRequest struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
}

type paymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

func (c *paymentHTTPClient) AuthorizePayment(req *domain.PaymentRequest) (*domain.PaymentResponse, error) {
	body, err := json.Marshal(paymentRequest{
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		CustomerEmail: req.CustomerEmail,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/v1/payments", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("payment service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment service returned status: %d", resp.StatusCode)
	}

	var result paymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &domain.PaymentResponse{
		TransactionID: result.TransactionID,
		Status:        result.Status,
	}, nil
}