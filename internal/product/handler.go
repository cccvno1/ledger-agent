package product

import (
	"net/http"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

// Handler handles HTTP requests for the product feature.
type Handler struct {
	svc *Service
}

// NewHandler creates a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/products.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.ListAll(r.Context())
	if err != nil {
		httpkit.Error(w, err)
		return
	}

	resp := make([]ProductResponse, len(products))
	for i, p := range products {
		resp[i] = NewProductResponse(p)
	}
	httpkit.JSON(w, http.StatusOK, resp)
}
