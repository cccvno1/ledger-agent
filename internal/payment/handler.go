package payment

import (
	"net/http"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

// Handler handles HTTP requests for the payment feature.
type Handler struct {
	svc *Service
}

// NewHandler creates a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/payments?customer_id=.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	payments, err := h.svc.ListByCustomer(r.Context(), customerID)
	if err != nil {
		httpkit.Error(w, err)
		return
	}

	resp := make([]PaymentResponse, len(payments))
	for i, p := range payments {
		resp[i] = NewPaymentResponse(p)
	}
	httpkit.JSON(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/payments.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpkit.Decode(r, &req); err != nil {
		httpkit.Error(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpkit.Error(w, err)
		return
	}

	p, err := h.svc.Create(r.Context(), req.ToInput())
	if err != nil {
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusCreated, NewPaymentResponse(p))
}
