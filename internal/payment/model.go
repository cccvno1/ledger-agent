package payment

import "time"

// Payment represents a customer's payment against their outstanding balance.
type Payment struct {
	ID          string
	CustomerID  string
	Amount      float64
	PaymentDate time.Time
	Notes       string
	CreatedAt   time.Time
}
