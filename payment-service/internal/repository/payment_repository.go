package repository

import (
	"database/sql"

	"payment-service/internal/domain"
)

type PaymentRepository interface {
	Create(payment *domain.Payment) error
	FindByOrderID(orderID string) (*domain.Payment, error)
}

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(payment *domain.Payment) error {
	query := `INSERT INTO payments (id, order_id, transaction_id, amount, status, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(query, payment.ID, payment.OrderID, payment.TransactionID,
		payment.Amount, payment.Status, payment.CreatedAt)
	return err
}

func (r *paymentRepository) FindByOrderID(orderID string) (*domain.Payment, error) {
	query := `SELECT id, order_id, transaction_id, amount, status, created_at 
	          FROM payments WHERE order_id = $1`

	var payment domain.Payment
	err := r.db.QueryRow(query, orderID).Scan(
		&payment.ID, &payment.OrderID, &payment.TransactionID,
		&payment.Amount, &payment.Status, &payment.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &payment, nil
}