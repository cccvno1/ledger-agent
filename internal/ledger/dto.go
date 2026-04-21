package ledger

import (
	"strings"
	"time"

	"github.com/cccvno1/goplate/pkg/errkit"
)

// CreateRequest is the HTTP request body for POST /api/v1/entries.
type CreateRequest struct {
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	ProductName  string  `json:"product_name"`
	UnitPrice    float64 `json:"unit_price"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	EntryDate    string  `json:"entry_date"` // "2006-01-02"
	Notes        string  `json:"notes"`
}

// Validate checks field format.
func (r *CreateRequest) Validate() error {
	if strings.TrimSpace(r.CustomerID) == "" {
		return errkit.New(errkit.InvalidInput, "customer_id is required")
	}
	if strings.TrimSpace(r.CustomerName) == "" {
		return errkit.New(errkit.InvalidInput, "customer_name is required")
	}
	if strings.TrimSpace(r.ProductName) == "" {
		return errkit.New(errkit.InvalidInput, "product_name is required")
	}
	if r.UnitPrice <= 0 {
		return errkit.New(errkit.InvalidInput, "unit_price must be greater than 0")
	}
	if r.Quantity <= 0 {
		return errkit.New(errkit.InvalidInput, "quantity must be greater than 0")
	}
	if r.EntryDate == "" {
		return errkit.New(errkit.InvalidInput, "entry_date is required")
	}
	if _, err := time.Parse("2006-01-02", r.EntryDate); err != nil {
		return errkit.New(errkit.InvalidInput, "entry_date must be in YYYY-MM-DD format")
	}
	return nil
}

// ToInput converts the request DTO to service input.
func (r *CreateRequest) ToInput() CreateInput {
	d, _ := time.Parse("2006-01-02", r.EntryDate)
	return CreateInput{
		CustomerID:   strings.TrimSpace(r.CustomerID),
		CustomerName: strings.TrimSpace(r.CustomerName),
		ProductName:  strings.TrimSpace(r.ProductName),
		UnitPrice:    r.UnitPrice,
		Quantity:     r.Quantity,
		Unit:         r.Unit,
		EntryDate:    d,
		Notes:        r.Notes,
	}
}

// UpdateRequest is the HTTP request body for PUT /api/v1/entries/{id}.
// All fields are optional; only non-zero values trigger an update.
type UpdateRequest struct {
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	EntryDate   string  `json:"entry_date"` // "2006-01-02", optional
	Notes       string  `json:"notes"`
}

// Validate checks field format for the update request.
func (r *UpdateRequest) Validate() error {
	if r.EntryDate != "" {
		if _, err := time.Parse("2006-01-02", r.EntryDate); err != nil {
			return errkit.New(errkit.InvalidInput, "entry_date must be in YYYY-MM-DD format")
		}
	}
	if r.UnitPrice != 0 && r.UnitPrice <= 0 {
		return errkit.New(errkit.InvalidInput, "unit_price must be greater than 0")
	}
	if r.Quantity != 0 && r.Quantity <= 0 {
		return errkit.New(errkit.InvalidInput, "quantity must be greater than 0")
	}
	return nil
}

// ToInput converts the request DTO to service input.
func (r *UpdateRequest) ToInput(id string) UpdateInput {
	var entryDate time.Time
	if r.EntryDate != "" {
		entryDate, _ = time.Parse("2006-01-02", r.EntryDate)
	}
	return UpdateInput{
		ID:          id,
		ProductName: r.ProductName,
		UnitPrice:   r.UnitPrice,
		Quantity:    r.Quantity,
		Unit:        r.Unit,
		EntryDate:   entryDate,
		Notes:       r.Notes,
	}
}

// EntryResponse is the HTTP response shape for a single ledger entry.
type EntryResponse struct {
	ID           string     `json:"id"`
	CustomerID   string     `json:"customer_id"`
	CustomerName string     `json:"customer_name"`
	ProductName  string     `json:"product_name"`
	UnitPrice    float64    `json:"unit_price"`
	Quantity     float64    `json:"quantity"`
	Unit         string     `json:"unit"`
	Amount       float64    `json:"amount"`
	EntryDate    string     `json:"entry_date"`
	IsSettled    bool       `json:"is_settled"`
	SettledAt    *time.Time `json:"settled_at,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// NewEntryResponse builds the response DTO from the domain model.
func NewEntryResponse(e *Entry) EntryResponse {
	return EntryResponse{
		ID:           e.ID,
		CustomerID:   e.CustomerID,
		CustomerName: e.CustomerName,
		ProductName:  e.ProductName,
		UnitPrice:    e.UnitPrice,
		Quantity:     e.Quantity,
		Unit:         e.Unit,
		Amount:       e.Amount,
		EntryDate:    e.EntryDate.Format("2006-01-02"),
		IsSettled:    e.IsSettled,
		SettledAt:    e.SettledAt,
		Notes:        e.Notes,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

// SummaryResponse is the HTTP response shape for a customer summary.
type SummaryResponse struct {
	CustomerID    string  `json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	TotalAmount   float64 `json:"total_amount"`
	SettledAmount float64 `json:"settled_amount"`
	PendingAmount float64 `json:"pending_amount"`
	EntryCount    int     `json:"entry_count"`
}

// NewSummaryResponse builds the response DTO from the domain model.
func NewSummaryResponse(s *CustomerSummary) SummaryResponse {
	return SummaryResponse{
		CustomerID:    s.CustomerID,
		CustomerName:  s.CustomerName,
		TotalAmount:   s.TotalAmount,
		SettledAmount: s.SettledAmount,
		PendingAmount: s.PendingAmount,
		EntryCount:    s.EntryCount,
	}
}

// DashboardResponse is the HTTP response for GET /api/v1/dashboard.
type DashboardResponse struct {
	TotalPending     float64 `json:"total_pending"`
	TotalCustomers   int     `json:"total_customers"`
	EntriesThisMonth int     `json:"entries_this_month"`
	AmountThisMonth  float64 `json:"amount_this_month"`
}

// NewDashboardResponse builds the response DTO from service output.
func NewDashboardResponse(out *DashboardOutput) DashboardResponse {
	return DashboardResponse{
		TotalPending:     out.TotalPending,
		TotalCustomers:   out.TotalCustomers,
		EntriesThisMonth: out.EntriesThisMonth,
		AmountThisMonth:  out.AmountThisMonth,
	}
}
