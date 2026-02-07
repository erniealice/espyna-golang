package model

import (
	"fmt"
)

// TransactionErrorCode represents specific transaction error codes
type TransactionErrorCode string

const (
	// TransactionErrorCodeGeneral indicates a general transaction error
	TransactionErrorCodeGeneral TransactionErrorCode = "TRANSACTION_GENERAL"

	// TransactionErrorCodeBeginFailed indicates transaction begin failed
	TransactionErrorCodeBeginFailed TransactionErrorCode = "TRANSACTION_BEGIN_FAILED"

	// TransactionErrorCodeCommitFailed indicates transaction commit failed
	TransactionErrorCodeCommitFailed TransactionErrorCode = "TRANSACTION_COMMIT_FAILED"

	// TransactionErrorCodeRollbackFailed indicates transaction rollback failed
	TransactionErrorCodeRollbackFailed TransactionErrorCode = "TRANSACTION_ROLLBACK_FAILED"

	// TransactionErrorCodeConflict indicates a transaction conflict (concurrent modification)
	TransactionErrorCodeConflict TransactionErrorCode = "TRANSACTION_CONFLICT"

	// TransactionErrorCodeTimeout indicates transaction timeout
	TransactionErrorCodeTimeout TransactionErrorCode = "TRANSACTION_TIMEOUT"

	// TransactionErrorCodeDeadlock indicates a transaction deadlock
	TransactionErrorCodeDeadlock TransactionErrorCode = "TRANSACTION_DEADLOCK"

	// TransactionErrorCodeInvalidState indicates invalid transaction state
	TransactionErrorCodeInvalidState TransactionErrorCode = "TRANSACTION_INVALID_STATE"

	// TransactionErrorCodeNotSupported indicates transactions are not supported
	TransactionErrorCodeNotSupported TransactionErrorCode = "TRANSACTION_NOT_SUPPORTED"

	// TransactionErrorCodeContextMissing indicates missing transaction context
	TransactionErrorCodeContextMissing TransactionErrorCode = "TRANSACTION_CONTEXT_MISSING"
)

// TransactionError represents a transaction-specific error with context preservation
type TransactionError struct {
	// Code is the specific transaction error code
	Code TransactionErrorCode

	// Message is the human-readable error message
	Message string

	// TransactionID is the ID of the transaction that failed (if available)
	TransactionID string

	// Operation is the operation that was being performed when the error occurred
	Operation string

	// Cause is the underlying error that caused this transaction error
	Cause error

	// Context provides additional context about the error
	Context map[string]any

	// Retryable indicates if this error condition can be retried
	Retryable bool
}

// Error implements the error interface
func (e *TransactionError) Error() string {
	if e.TransactionID != "" {
		return fmt.Sprintf("transaction error [%s] in transaction %s during %s: %s",
			e.Code, e.TransactionID, e.Operation, e.Message)
	}
	return fmt.Sprintf("transaction error [%s] during %s: %s",
		e.Code, e.Operation, e.Message)
}

// Unwrap returns the underlying cause error for error wrapping support
func (e *TransactionError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if this error can be retried
func (e *TransactionError) IsRetryable() bool {
	return e.Retryable
}

// WithContext adds context information to the error
func (e *TransactionError) WithContext(key string, value any) *TransactionError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// GetContext retrieves context information from the error
func (e *TransactionError) GetContext(key string) (any, bool) {
	if e.Context == nil {
		return nil, false
	}
	value, exists := e.Context[key]
	return value, exists
}

// NewTransactionError creates a new transaction error
func NewTransactionError(code TransactionErrorCode, message string, operation string) *TransactionError {
	return &TransactionError{
		Code:      code,
		Message:   message,
		Operation: operation,
		Context:   make(map[string]any),
		Retryable: isRetryableError(code),
	}
}

// NewTransactionErrorWithCause creates a new transaction error with an underlying cause
func NewTransactionErrorWithCause(code TransactionErrorCode, message string, operation string, cause error) *TransactionError {
	return &TransactionError{
		Code:      code,
		Message:   message,
		Operation: operation,
		Cause:     cause,
		Context:   make(map[string]any),
		Retryable: isRetryableError(code),
	}
}

// NewTransactionErrorWithID creates a new transaction error with transaction ID
func NewTransactionErrorWithID(code TransactionErrorCode, message string, operation string, transactionID string) *TransactionError {
	return &TransactionError{
		Code:          code,
		Message:       message,
		Operation:     operation,
		TransactionID: transactionID,
		Context:       make(map[string]any),
		Retryable:     isRetryableError(code),
	}
}

// WrapTransactionError wraps an existing error as a transaction error
func WrapTransactionError(err error, code TransactionErrorCode, operation string) *TransactionError {
	if err == nil {
		return nil
	}

	// If it's already a TransactionError, preserve the original
	if txErr, ok := err.(*TransactionError); ok {
		// Create a new error that wraps the original
		return &TransactionError{
			Code:          code,
			Message:       fmt.Sprintf("wrapped transaction error: %s", txErr.Message),
			Operation:     operation,
			TransactionID: txErr.TransactionID,
			Cause:         txErr,
			Context:       make(map[string]any),
			Retryable:     txErr.Retryable,
		}
	}

	return NewTransactionErrorWithCause(code, err.Error(), operation, err)
}

// isRetryableError determines if an error code represents a retryable condition
func isRetryableError(code TransactionErrorCode) bool {
	switch code {
	case TransactionErrorCodeConflict, TransactionErrorCodeTimeout, TransactionErrorCodeDeadlock:
		return true
	case TransactionErrorCodeBeginFailed, TransactionErrorCodeCommitFailed, TransactionErrorCodeRollbackFailed:
		// These might be retryable depending on the underlying cause
		return true
	case TransactionErrorCodeInvalidState, TransactionErrorCodeNotSupported, TransactionErrorCodeContextMissing:
		return false
	default:
		return false
	}
}

// IsTransactionError checks if an error is a transaction error
func IsTransactionError(err error) bool {
	_, ok := err.(*TransactionError)
	return ok
}

// GetTransactionError extracts a transaction error from an error chain
func GetTransactionError(err error) (*TransactionError, bool) {
	if err == nil {
		return nil, false
	}

	if txErr, ok := err.(*TransactionError); ok {
		return txErr, true
	}

	// Check if it's wrapped
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return GetTransactionError(unwrapper.Unwrap())
	}

	return nil, false
}

// IsRetryableTransactionError checks if an error is a retryable transaction error
func IsRetryableTransactionError(err error) bool {
	txErr, ok := GetTransactionError(err)
	if !ok {
		return false
	}
	return txErr.IsRetryable()
}

// TransactionErrorHandler provides utilities for handling transaction errors
type TransactionErrorHandler struct{}

// NewTransactionErrorHandler creates a new transaction error handler
func NewTransactionErrorHandler() *TransactionErrorHandler {
	return &TransactionErrorHandler{}
}

// HandleError processes a transaction error and determines the appropriate response
func (h *TransactionErrorHandler) HandleError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// If it's already a TransactionError, return as-is
	if IsTransactionError(err) {
		return err
	}

	// Wrap the error as a transaction error
	return WrapTransactionError(err, TransactionErrorCodeGeneral, operation)
}

// ShouldRetry determines if an operation should be retried based on the error
func (h *TransactionErrorHandler) ShouldRetry(err error, attemptCount int, maxRetries int) bool {
	if err == nil || attemptCount >= maxRetries {
		return false
	}

	return IsRetryableTransactionError(err)
}

// GetRetryDelay calculates the delay before retrying based on attempt count
func (h *TransactionErrorHandler) GetRetryDelay(attemptCount int) int64 {
	// Exponential backoff: 100ms, 200ms, 400ms, etc.
	delay := int64(100) // Start with 100ms
	for i := 0; i < attemptCount; i++ {
		delay *= 2
	}
	// Cap at 5 seconds
	if delay > 5000 {
		delay = 5000
	}
	return delay
}

// DatabaseError represents a database operation error
type DatabaseError struct {
	// Message is the human-readable error message
	Message string `json:"message"`

	// Code is the specific database error code
	Code string `json:"code"`

	// HTTPStatus is the HTTP status code to return
	HTTPStatus int `json:"http_status"`

	// Context provides additional context about the error
	Context map[string]any `json:"context,omitempty"`

	// Cause is the underlying error that caused this database error
	Cause error `json:"-"`
}

// Error implements the error interface
func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error [%s]: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error for error wrapping support
func (e *DatabaseError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *DatabaseError) WithContext(key string, value any) *DatabaseError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// GetContext retrieves context information from the error
func (e *DatabaseError) GetContext(key string) (any, bool) {
	if e.Context == nil {
		return nil, false
	}
	value, exists := e.Context[key]
	return value, exists
}

// NewDatabaseError creates a new database error
func NewDatabaseError(message, code string, httpStatus int) *DatabaseError {
	return &DatabaseError{
		Message:    message,
		Code:       code,
		HTTPStatus: httpStatus,
		Context:    make(map[string]any),
	}
}

// NewDatabaseErrorWithCause creates a new database error with an underlying cause
func NewDatabaseErrorWithCause(message, code string, httpStatus int, cause error) *DatabaseError {
	return &DatabaseError{
		Message:    message,
		Code:       code,
		HTTPStatus: httpStatus,
		Cause:      cause,
		Context:    make(map[string]any),
	}
}

// WrapDatabaseError wraps an existing error as a database error
func WrapDatabaseError(err error, code string, httpStatus int) *DatabaseError {
	if err == nil {
		return nil
	}

	// If it's already a DatabaseError, preserve the original
	if dbErr, ok := err.(*DatabaseError); ok {
		// Create a new error that wraps the original
		return &DatabaseError{
			Message:    fmt.Sprintf("wrapped database error: %s", dbErr.Message),
			Code:       code,
			HTTPStatus: httpStatus,
			Cause:      dbErr,
			Context:    make(map[string]any),
		}
	}

	return NewDatabaseErrorWithCause(err.Error(), code, httpStatus, err)
}

// IsDatabaseError checks if an error is a database error
func IsDatabaseError(err error) bool {
	_, ok := err.(*DatabaseError)
	return ok
}

// GetDatabaseError extracts a database error from an error chain
func GetDatabaseError(err error) (*DatabaseError, bool) {
	if err == nil {
		return nil, false
	}

	if dbErr, ok := err.(*DatabaseError); ok {
		return dbErr, true
	}

	// Check if it's wrapped
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return GetDatabaseError(unwrapper.Unwrap())
	}

	return nil, false
}
