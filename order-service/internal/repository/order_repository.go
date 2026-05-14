package repository

import (
	"database/sql"

	"order-service/internal/domain"
)

type OrderRepository interface {
	Create(order *domain.Order) error
	FindByID(id string) (*domain.Order, error)
	UpdateStatus(id, status string) error
}

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(order *domain.Order) error {
	query := `INSERT INTO orders (id, customer_id, item_name, amount, status, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(query, order.ID, order.CustomerID, order.ItemName,
		order.Amount, order.Status, order.CreatedAt)
	return err
}

func (r *orderRepository) FindByID(id string) (*domain.Order, error) {
	query := `SELECT id, customer_id, item_name, amount, status, created_at 
	          FROM orders WHERE id = $1`

	var order domain.Order
	err := r.db.QueryRow(query, id).Scan(
		&order.ID, &order.CustomerID, &order.ItemName,
		&order.Amount, &order.Status, &order.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) UpdateStatus(id, status string) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}