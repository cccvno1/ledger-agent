---
applyTo: "**/handler.go,**/handler_test.go,**/dto.go"
description: "HTTP transport conventions. Handlers parse, delegate, respond — no business logic."
---

# HTTP Transport Layer

## Handler Struct

```go
type Handler struct {
    svc *Service
}

func NewHandler(svc *Service) *Handler {
    return &Handler{svc: svc}
}
```

- One `Handler` per feature, holds reference to `*Service`.
- Handlers are registered in `wire.go` via the feature's `Wire` function.

## Handler Method Pattern

```go
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
    out, err := h.svc.Create(r.Context(), req.ToInput())
    if err != nil {
        httpkit.Error(w, err)
        return
    }
    httpkit.JSON(w, http.StatusCreated, NewCreateResponse(out))
}
```

- Three steps only: **parse** → **delegate** → **respond**.
- Never call store directly — always go through service.
- Never put business logic in handlers.
- Pass `r.Context()` to service calls.
- Always convert service output to a **response DTO** before sending — never pass
  domain models directly to `httpkit.JSON`. This prevents leaking internal fields
  (e.g., password hashes, internal flags) to API consumers.

## Request / Response DTOs (dto.go)

```go
// --- Request DTO ---
type CreateRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *CreateRequest) Validate() error {
    if r.Name == "" {
        return errkit.New(errkit.InvalidInput, "name is required")
    }
    if !strings.Contains(r.Email, "@") {
        return errkit.New(errkit.InvalidInput, "invalid email format")
    }
    return nil
}

func (r *CreateRequest) ToInput() CreateInput {
    return CreateInput{Name: r.Name, Email: r.Email}
}

// --- Response DTO ---
type CreateResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func NewCreateResponse(u *User) CreateResponse {
    return CreateResponse{ID: u.ID, Name: u.Name, Email: u.Email}
}
```

- DTOs are separate from domain models.
- `Validate()` checks field format (required, shape, length) — returns `errkit.InvalidInput` on failure.
- `ToInput()` converts request → service input. When the handler provides non-body
  data (user ID from header, path params), `ToInput` accepts those as arguments:
  `req.ToInput(userID)`.
- Response DTO constructor (`NewCreateResponse`) selects which domain fields are
  exposed. **Never expose internal-only fields** (password hashes, internal status flags).
- JSON field names are snake_case.
- Call `Validate()` in the handler after `Decode`, before calling the service.
- Validate field format in DTO, validate business rules in service.

## Error Responses

- Use `httpkit.Error(w, err)` — it projects `errkit.Code` → HTTP status.
- Never write raw status codes: `w.WriteHeader(404)`.
- Error response shape: `{"error": {"code": "not_found", "message": "..."}}`.

## Route Registration

Feature routes are registered in the feature's `wire.go`, NOT in `router.go`.
`router.go` is reserved for infrastructure routes (health check, readiness, metrics).

```go
func Wire(mux *http.ServeMux, db *sql.DB) {
    store := NewStore(db)
    svc := NewService(db, store)
    h := NewHandler(svc)
    mux.HandleFunc("GET /users/{id}", h.Get)
    mux.HandleFunc("POST /users", h.Create)
}
```

## Testing

- Use `httptest.NewRequest` + `httptest.NewRecorder`.
- Test the full handler: input → service → response status + body.
- Do not mock `http.ResponseWriter`.
