package domain

// PageRequest holds pagination parameters.
type PageRequest struct {
	Page int
	Size int
}

// PageResponse holds pagination metadata for list responses.
type PageResponse struct {
	Page       int
	Size       int
	Total      int
	TotalPages int
}
