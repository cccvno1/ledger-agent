package customer

import "time"

// CustomerResponse is the HTTP response shape for a single customer.
type CustomerResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Aliases   []string  `json:"aliases"`
	CreatedAt time.Time `json:"created_at"`
}

// NewCustomerResponse builds the response DTO from the domain model.
func NewCustomerResponse(c *Customer) CustomerResponse {
	aliases := c.Aliases
	if aliases == nil {
		aliases = []string{}
	}
	return CustomerResponse{
		ID:        c.ID,
		Name:      c.Name,
		Aliases:   aliases,
		CreatedAt: c.CreatedAt,
	}
}
