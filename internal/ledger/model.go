package ledger

import "time"

// Entry represents one line of a wholesale shipment record.
type Entry struct {
	ID           string
	CustomerID   string
	CustomerName string
	ProductName  string
	UnitPrice    float64
	Quantity     float64
	Unit         string
	Amount       float64 // UnitPrice * Quantity, stored for fast aggregation
	EntryDate    time.Time
	IsSettled    bool
	SettledAt    *time.Time
	Notes        string
	// IdempotencyKey, when set, makes Create safe to retry. A duplicate
	// insert with the same key returns the existing row's ID instead of
	// inserting a second row.
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CustomerSummary aggregates outstanding amounts for a single customer.
type CustomerSummary struct {
	CustomerID    string
	CustomerName  string
	TotalAmount   float64
	SettledAmount float64
	PendingAmount float64
	EntryCount    int
}
