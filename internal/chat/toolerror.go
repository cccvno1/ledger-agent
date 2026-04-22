package chat

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cccvno1/goplate/pkg/errkit"
)

// ToolErrorCode enumerates structured failure codes that tools may return.
// Codes are stable identifiers the LLM can branch on; messages are user-facing.
type ToolErrorCode string

const (
	// CodeInternal is reserved for unexpected failures (bugs, panics).
	CodeInternal ToolErrorCode = "internal_error"
	// CodeInvalidArgs indicates the LLM passed malformed/missing parameters.
	CodeInvalidArgs ToolErrorCode = "invalid_args"
	// CodeTimeout means the tool exceeded its execution budget.
	CodeTimeout ToolErrorCode = "timeout"
	// CodePhaseMismatch is returned when a tool is called in a phase that does
	// not allow it. Used by Phase B (policy layer).
	CodePhaseMismatch ToolErrorCode = "phase_mismatch"
	// CodeRefNotFound is returned when a context-ref (e.g. "last_saved.0")
	// cannot be resolved. Used by Phase E.
	CodeRefNotFound ToolErrorCode = "ref_not_found"
	// CodeNotFound — generic "entity does not exist".
	CodeNotFound ToolErrorCode = "not_found"
	// CodeConstraint — domain-level validation failed (e.g. amount exceeds pending).
	CodeConstraint ToolErrorCode = "constraint_violation"
	// CodeTokenInvalid — operation token is missing/expired/wrong session.
	CodeTokenInvalid ToolErrorCode = "token_invalid"
)

// ToolError is the structured failure payload returned by tools.
// It satisfies the error interface so tools can simply `return nil, err`.
//
// Two design contracts:
//  1. The Registry serialises *ToolError into a deterministic JSON envelope
//     before handing it back to the planner; the LLM sees a stable shape.
//  2. Recoverable indicates whether the LLM should retry/branch (true) or
//     surface the failure to the user verbatim (false).
type ToolError struct {
	Code        ToolErrorCode  `json:"code"`
	Message     string         `json:"message"`
	Recoverable bool           `json:"recoverable"`
	Hint        string         `json:"hint,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
}

// Error implements error. The string form is intentionally compact; the
// rich form is available via JSON encoding.
func (e *ToolError) Error() string {
	if e == nil {
		return "<nil ToolError>"
	}
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (hint: %s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// MarshalJSON wraps ToolError under a top-level `error` field so the LLM can
// distinguish failure envelopes from successful payloads at a glance.
func (e *ToolError) MarshalJSON() ([]byte, error) {
	type alias ToolError
	return json.Marshal(struct {
		Error *alias `json:"error"`
	}{(*alias)(e)})
}

// NewToolError constructs a ToolError. Hint and Context are optional and may
// be supplied via the With* helpers for readability at call sites.
func NewToolError(code ToolErrorCode, msg string) *ToolError {
	return &ToolError{Code: code, Message: msg, Recoverable: code != CodeInternal}
}

// WithHint sets the recovery hint and returns the receiver for chaining.
func (e *ToolError) WithHint(h string) *ToolError {
	e.Hint = h
	return e
}

// WithContext attaches structured key-value context (e.g. pending=278.50).
func (e *ToolError) WithContext(kv map[string]any) *ToolError {
	e.Context = kv
	return e
}

// WithRecoverable overrides the default recoverability.
func (e *ToolError) WithRecoverable(v bool) *ToolError {
	e.Recoverable = v
	return e
}

// AsToolError unwraps an arbitrary error chain looking for a *ToolError.
// Returns nil if none is present.
func AsToolError(err error) *ToolError {
	var te *ToolError
	if errors.As(err, &te) {
		return te
	}
	return nil
}

// FromDomainError converts a domain-layer error into a *ToolError when it
// carries a recognised errkit.Code. Returns nil for unknown errors so the
// caller can fall through to its own classification.
func FromDomainError(err error) *ToolError {
	if err == nil {
		return nil
	}
	if te := AsToolError(err); te != nil {
		return te
	}
	if e := errkit.AsError(err); e != nil {
		switch e.Code {
		case errkit.InvalidInput:
			return NewToolError(CodeInvalidArgs, e.Error()).WithRecoverable(true)
		case errkit.NotFound:
			return NewToolError(CodeNotFound, e.Error()).WithRecoverable(true)
		case errkit.Conflict:
			return NewToolError(CodeConstraint, e.Error()).WithRecoverable(true)
		default:
			return NewToolError(CodeInternal, e.Error()).WithRecoverable(false)
		}
	}
	return nil
}
