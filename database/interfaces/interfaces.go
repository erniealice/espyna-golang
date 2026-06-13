// Package interfaces re-exports internal database operation interfaces for use by contrib sub-modules.
package interfaces

import (
	internal "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
)

// Core database operation types
type (
	DatabaseOperation = internal.DatabaseOperation
	TransactionAware  = internal.TransactionAware
	ListParams        = internal.ListParams
	ListResult        = internal.ListResult
)

// Query types
type (
	QueryBuilder       = internal.QueryBuilder
	QueryFilter        = internal.QueryFilter
	QueryCondition     = internal.QueryCondition
	OrderByClause      = internal.OrderByClause
	CompositeKeyQuery  = internal.CompositeKeyQuery
	SimpleQueryBuilder = internal.SimpleQueryBuilder
)

var (
	NewQueryBuilder      = internal.NewQueryBuilder
	NewCompositeKeyQuery = internal.NewCompositeKeyQuery
)

// Transaction types
type (
	Transaction        = internal.Transaction
	TransactionManager = internal.TransactionManager
	TransactionState   = internal.TransactionState
	TransactionOptions = internal.TransactionOptions
)

// Transaction state constants
const (
	TransactionStatePending    = internal.TransactionStatePending
	TransactionStateCommitted  = internal.TransactionStateCommitted
	TransactionStateRolledBack = internal.TransactionStateRolledBack
)

var DefaultTransactionOptions = internal.DefaultTransactionOptions
