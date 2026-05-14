package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"order-service/internal/usecase"
)

type OrderHandler struct {
	usecase usecase.OrderUsecase
}

func NewOrderHandler(uc usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{usecase: uc}
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required,gt=0"`
}

type orderResponse struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	ItemName   string `json:"item_name"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.usecase.CreateOrder(req.CustomerID, req.ItemName, req.Amount)
	if err != nil {
		if err.Error() == "payment service unavailable" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, orderResponse{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		ItemName:   order.ItemName,
		Amount:     order.Amount,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt.Format(time.RFC3339),
	})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.usecase.GetOrder(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, orderResponse{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		ItemName:   order.ItemName,
		Amount:     order.Amount,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt.Format(time.RFC3339),
	})
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	err := h.usecase.CancelOrder(id)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled successfully"})
}