package chat

import "github.com/cccvno1/goplate/pkg/errkit"

var errMissingMessage = errkit.New(errkit.InvalidInput, "message is required")
