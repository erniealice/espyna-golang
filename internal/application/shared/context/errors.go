package context

import "errors"

// Context-specific errors
var (
	// ErrUserNotFoundInContext indicates no valid user ID was found in the context
	ErrUserNotFoundInContext = errors.New("user not found in context")
)
