package product

import "time"

// ProductResponse is the HTTP response shape for a single product.
type ProductResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Aliases        []string  `json:"aliases"`
	DefaultUnit    string    `json:"default_unit"`
	ReferencePrice float64   `json:"reference_price,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// NewProductResponse builds the response DTO from the domain model.
func NewProductResponse(p *Product) ProductResponse {
	aliases := p.Aliases
	if aliases == nil {
		aliases = []string{}
	}
	return ProductResponse{
		ID:             p.ID,
		Name:           p.Name,
		Aliases:        aliases,
		DefaultUnit:    p.DefaultUnit,
		ReferencePrice: p.ReferencePrice,
		CreatedAt:      p.CreatedAt,
	}
}
