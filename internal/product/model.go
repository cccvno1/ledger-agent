package product

import "time"

// Product represents a catalog item with fuzzy-match aliases.
type Product struct {
	ID             string
	Name           string
	Aliases        []string
	DefaultUnit    string
	ReferencePrice float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
