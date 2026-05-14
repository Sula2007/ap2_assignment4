package domain

type PaymentRequest struct {
OrderID       string
Amount        int64
CustomerEmail string
}

type PaymentResponse struct {
TransactionID string
Status        string
}
