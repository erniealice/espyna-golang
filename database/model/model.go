// Package model re-exports internal database model types for use by contrib sub-modules.
package model

import (
	internal "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/model"
)

// Base model
type BaseModel = internal.BaseModel

// Transaction error types
type (
	TransactionErrorCode    = internal.TransactionErrorCode
	TransactionError        = internal.TransactionError
	TransactionErrorHandler = internal.TransactionErrorHandler
)

// Transaction error code constants
const (
	TransactionErrorCodeGeneral        = internal.TransactionErrorCodeGeneral
	TransactionErrorCodeBeginFailed    = internal.TransactionErrorCodeBeginFailed
	TransactionErrorCodeCommitFailed   = internal.TransactionErrorCodeCommitFailed
	TransactionErrorCodeRollbackFailed = internal.TransactionErrorCodeRollbackFailed
	TransactionErrorCodeConflict       = internal.TransactionErrorCodeConflict
	TransactionErrorCodeTimeout        = internal.TransactionErrorCodeTimeout
	TransactionErrorCodeDeadlock       = internal.TransactionErrorCodeDeadlock
	TransactionErrorCodeInvalidState   = internal.TransactionErrorCodeInvalidState
	TransactionErrorCodeNotSupported   = internal.TransactionErrorCodeNotSupported
	TransactionErrorCodeContextMissing = internal.TransactionErrorCodeContextMissing
)

// Transaction error constructors
var (
	NewTransactionError          = internal.NewTransactionError
	NewTransactionErrorWithCause = internal.NewTransactionErrorWithCause
	NewTransactionErrorWithID    = internal.NewTransactionErrorWithID
	WrapTransactionError         = internal.WrapTransactionError
	IsTransactionError           = internal.IsTransactionError
	GetTransactionError          = internal.GetTransactionError
	IsRetryableTransactionError  = internal.IsRetryableTransactionError
	NewTransactionErrorHandler   = internal.NewTransactionErrorHandler
)

// Database error types
type DatabaseError = internal.DatabaseError

var (
	NewDatabaseError          = internal.NewDatabaseError
	NewDatabaseErrorWithCause = internal.NewDatabaseErrorWithCause
	WrapDatabaseError         = internal.WrapDatabaseError
	IsDatabaseError           = internal.IsDatabaseError
	GetDatabaseError          = internal.GetDatabaseError
)

// Validation types
type (
	ValidationError  = internal.ValidationError
	ValidationErrors = internal.ValidationErrors
)

var ValidateRequired = internal.ValidateRequired
