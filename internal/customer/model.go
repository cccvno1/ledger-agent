package customer

import "time"

// Customer represents a wholesale buyer.
type Customer struct {
	ID        string
	Name      string
	Aliases   []string
	CreatedAt time.Time
}
