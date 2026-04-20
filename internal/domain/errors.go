package domain

import "github.com/cccvno1/goplate/pkg/errkit"

// Standard business error codes.
var (
	ErrNotFound     = errkit.New(errkit.NotFound, "not found")
	ErrInvalidInput = errkit.New(errkit.InvalidInput, "invalid input")
	ErrConflict     = errkit.New(errkit.Conflict, "conflict")
	ErrUnauthorized = errkit.New(errkit.Unauthorized, "unauthorized")
	ErrForbidden    = errkit.New(errkit.Forbidden, "forbidden")
	ErrInternal     = errkit.New(errkit.Internal, "internal error")
)
