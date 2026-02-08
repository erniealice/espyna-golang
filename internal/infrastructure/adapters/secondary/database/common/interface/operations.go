package interfaces

import (
	"context"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListParams contains standardized parameters for list operations
// matching the proto request patterns (search, filters, sort, pagination)
type ListParams struct {
	Search     *commonpb.SearchRequest
	Filters    *commonpb.FilterRequest
	Sort       *commonpb.SortRequest
	Pagination *commonpb.PaginationRequest
}

// ListResult contains the results of a list operation with pagination metadata
type ListResult struct {
	Data       []map[string]any
	Pagination *commonpb.PaginationResponse
	Total      int32
}

// DatabaseOperation defines the common database operations interface
type DatabaseOperation interface {
	Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error)
	Read(ctx context.Context, tableName string, id string) (map[string]any, error)
	Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error)
	Delete(ctx context.Context, tableName string, id string) error
	List(ctx context.Context, tableName string, params *ListParams) (*ListResult, error)

	// Query-based operations for composite keys and complex queries
	Query(ctx context.Context, tableName string, query QueryBuilder) ([]map[string]any, error)
	QueryOne(ctx context.Context, tableName string, query QueryBuilder) (map[string]any, error)
}

// TransactionAware extends DatabaseOperation with transaction-aware behavior
// Repositories can optionally implement this interface for automatic transaction participation
type TransactionAware interface {
	DatabaseOperation

	// WithTransaction returns a transaction-aware version of this repository
	// If no transaction is active in context, returns the original repository
	WithTransaction(ctx context.Context) DatabaseOperation

	// SupportsTransactions indicates if this repository can participate in transactions
	SupportsTransactions() bool
}
