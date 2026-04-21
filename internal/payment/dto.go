package payment

import (
	"strings"
	"time"

	"github.com/cccvno1/goplate/pkg/errkit"
)

// CreateRequest is the HTTP request body for POST /api/v1/payments.
type CreateRequest struct {
	CustomerID  string  `json:"customer_id"`
	Amount      float64 `json:"amount"`
	PaymentDate string  `json:"payment_date"` // "2006-01-02"
	Notes       string  `json:"notes"`
}

// Validate checks field format.
func (r *CreateRequest) Validate() error {
	if strings.TrimSpace(r.CustomerID) == "" {
		return errkit.New(errkit.InvalidInput, "customer_id is required")
	}
	if r.Amount <= 0 {
		return errkit.New(errkit.InvalidInput, "amount must be greater than 0")
	}
	if r.PaymentDate == "" {
		return errkit.New(errkit.InvalidInput, "payment_date is required")
	}
	if _, err := time.Parse("2006-01-02", r.PaymentDate); err != nil {
		return errkit.New(errkit.InvalidInput, "payment_date must be in YYYY-MM-DD format")
	}
	return nil
}

// ToInput converts the request DTO to service input.
func (r *CreateRequest) ToInput() CreateInput {
	d, _ := time.Parse("2006-01-02", r.PaymentDate)
	return CreateInput{
		CustomerID:  strings.TrimSpace(r.CustomerID),
		Amount:      r.Amount,
		PaymentDate: d,
		Notes:       r.Notes,
	}
}

// PaymentResponse is the HTTP response shape for a single payment.
type PaymentResponse struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customer_id"`
	Amount      float64   `json:"amount"`
	PaymentDate string    `json:"payment_date"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewPaymentResponse builds the response DTO from the domain model.
func NewPaymentResponse(p *Payment) PaymentResponse {
	return PaymentResponse{
		ID:          p.ID,
		CustomerID:  p.CustomerID,
		Amount:      p.Amount,
		PaymentDate: p.PaymentDate.Format("2006-01-02"),
		Notes:       p.Notes,
		CreatedAt:   p.CreatedAt,
	}
}
