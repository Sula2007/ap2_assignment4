package domain

import "time"

type Order struct {
ID            string
CustomerID    string
CustomerEmail string
ItemName      string
Amount        int64
Status        string
CreatedAt     time.Time
}

type PaymentRequest struct {
OrderID       string
Amount        int64
CustomerEmail string
}

type PaymentResponse struct {
TransactionID string
Status        string
}
