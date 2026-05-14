package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"payment-service/internal/usecase"
)

type PaymentHandler struct {
	usecase usecase.PaymentUsecase
}

func NewPaymentHandler(uc usecase.PaymentUsecase) *PaymentHandler {
	return &PaymentHandler{usecase: uc}
}

type createPaymentRequest struct {
	OrderID       string `json:"order_id" binding:"required"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
}

type createPaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

type getPaymentResponse struct {
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.usecase.AuthorizePayment(req.OrderID, req.Amount, req.CustomerEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, createPaymentResponse{
		TransactionID: payment.TransactionID,
		Status:        payment.Status,
	})
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	orderID := c.Param("order_id")

	payment, err := h.usecase.GetPaymentByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, getPaymentResponse{
		OrderID:       payment.OrderID,
		TransactionID: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     payment.CreatedAt.Format(time.RFC3339),
	})
}
