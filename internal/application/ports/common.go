package ports

import (
	"fmt"
)

// ApplicationError represents a standardized application-level error
type ApplicationError struct {
	Code    string
	Message string
	Err     error
	Context map[string]any
}

func (e *ApplicationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (details: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewApplicationError creates a new ApplicationError
func NewApplicationError(code, message string, err error, ctx map[string]any) *ApplicationError {
	return &ApplicationError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: ctx,
	}
}
