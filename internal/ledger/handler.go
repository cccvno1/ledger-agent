package ledger

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

// Handler handles HTTP requests for the ledger feature.
type Handler struct {
	svc *Service
}

// NewHandler creates a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List handles GET /api/v1/entries.
// Query params: customer_id, date_from (YYYY-MM-DD), date_to (YYYY-MM-DD), is_settled (true/false).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	in := ListInput{
		CustomerID: q.Get("customer_id"),
	}
	if df := q.Get("date_from"); df != "" {
		t, err := time.Parse("2006-01-02", df)
		if err == nil {
			in.DateFrom = &t
		}
	}
	if dt := q.Get("date_to"); dt != "" {
		t, err := time.Parse("2006-01-02", dt)
		if err == nil {
			in.DateTo = &t
		}
	}
	if is := q.Get("is_settled"); is != "" {
		b, err := strconv.ParseBool(is)
		if err == nil {
			in.IsSettled = &b
		}
	}

	entries, err := h.svc.List(r.Context(), in)
	if err != nil {
		httpkit.Error(w, err)
		return
	}

	resp := make([]EntryResponse, len(entries))
	for i, e := range entries {
		resp[i] = NewEntryResponse(e)
	}
	httpkit.JSON(w, http.StatusOK, resp)
}

// Create handles POST /api/v1/entries.
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

	e, err := h.svc.Create(r.Context(), req.ToInput())
	if err != nil {
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusCreated, NewEntryResponse(e))
}

// Update handles PUT /api/v1/entries/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRequest
	if err := httpkit.Decode(r, &req); err != nil {
		httpkit.Error(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpkit.Error(w, err)
		return
	}

	e, err := h.svc.Update(r.Context(), req.ToInput(id))
	if err != nil {
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusOK, NewEntryResponse(e))
}

// Delete handles DELETE /api/v1/entries/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Delete(r.Context(), DeleteInput{ID: id}); err != nil {
		httpkit.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Summary handles GET /api/v1/customers/{customer_id}/summary.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customer_id")
	summaries, err := h.svc.SummaryByCustomer(r.Context(), customerID)
	if err != nil {
		httpkit.Error(w, err)
		return
	}

	resp := make([]SummaryResponse, len(summaries))
	for i, s := range summaries {
		resp[i] = NewSummaryResponse(s)
	}
	httpkit.JSON(w, http.StatusOK, resp)
}

// Settle handles POST /api/v1/customers/{customer_id}/settle.
func (h *Handler) Settle(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customer_id")
	if err := h.svc.SettleByCustomer(r.Context(), customerID); err != nil {
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusOK, map[string]string{"status": "settled"})
}

// Dashboard handles GET /api/v1/dashboard.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Dashboard(r.Context())
	if err != nil {
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusOK, NewDashboardResponse(out))
}
