package customer

import (
	"net/http"
	"strings"

	"github.com/cccvno1/goplate/pkg/errkit"
	"github.com/cccvno1/goplate/pkg/httpkit"
)

// Handler handles HTTP requests for the customer feature.
type Handler struct {
	svc *Service
}

// NewHandler creates a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/customers.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	customers, err := h.svc.ListAll(r.Context())
	if err != nil {
		httpkit.Error(w, err)
		return
	}

	resp := make([]CustomerResponse, len(customers))
	for i, c := range customers {
		resp[i] = NewCustomerResponse(c)
	}
	httpkit.JSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/customers/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusOK, NewCustomerResponse(c))
}

// Create handles POST /api/v1/customers.
// Idempotent: returns existing customer if name already exists.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := httpkit.Decode(r, &req); err != nil {
		httpkit.Error(w, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpkit.Error(w, errkit.New(errkit.InvalidInput, "name is required"))
		return
	}
	c, err := h.svc.Create(r.Context(), CreateInput{Name: strings.TrimSpace(req.Name)})
	if err != nil {
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusCreated, NewCustomerResponse(c))
}
