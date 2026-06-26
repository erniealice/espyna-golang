//go:build sqlserver

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.AssetTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver asset_transaction repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerAssetTransactionRepository(dbOps, tableName), nil
	})
}

var assetTransactionSortableSQLCols = []string{
	"id", "asset_id", "transaction_type", "transaction_date", "transaction_date_string",
	"amount", "description", "reference_number",
	"performed_by", "depreciation_run_id", "depreciation_period_start_date",
	"asset_revaluation_id",
	"active", "date_created", "date_modified",
}

var assetTransactionSortSpec = espynahttp.SortSpec{AllowedCols: assetTransactionSortableSQLCols}

// SQLServerAssetTransactionRepository implements asset_transaction CRUD operations
// using SQL Server. AssetTransaction rows are append-only: no UpdateAssetTransaction
// or DeleteAssetTransaction business flow — corrections are offsetting entries.
// The Update/Delete methods delegate to dbOps for admin-level corrections only.
type SQLServerAssetTransactionRepository struct {
	assettxpb.UnimplementedAssetTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerAssetTransactionRepository creates a new SQL Server asset_transaction repository.
func NewSQLServerAssetTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) assettxpb.AssetTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "asset_transaction"
	}
	return &SQLServerAssetTransactionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateAssetTransaction inserts a new asset_transaction row.
func (r *SQLServerAssetTransactionRepository) CreateAssetTransaction(ctx context.Context, req *assettxpb.CreateAssetTransactionRequest) (*assettxpb.CreateAssetTransactionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("asset_transaction data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_transaction protobuf: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_transaction JSON: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset_transaction: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_transaction result: %w", err)
	}

	tx := &assettxpb.AssetTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_transaction result: %w", err)
	}

	return &assettxpb.CreateAssetTransactionResponse{
		Data:    []*assettxpb.AssetTransaction{tx},
		Success: true,
	}, nil
}

// ReadAssetTransaction retrieves a single asset_transaction row by ID.
func (r *SQLServerAssetTransactionRepository) ReadAssetTransaction(ctx context.Context, req *assettxpb.ReadAssetTransactionRequest) (*assettxpb.ReadAssetTransactionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("asset_transaction ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset_transaction: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("asset_transaction with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset_transaction result: %w", err)
	}

	tx := &assettxpb.AssetTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset_transaction result: %w", err)
	}

	return &assettxpb.ReadAssetTransactionResponse{
		Data:    []*assettxpb.AssetTransaction{tx},
		Success: true,
	}, nil
}

// ListAssetTransactions lists asset_transaction rows using SQL Server operations.
func (r *SQLServerAssetTransactionRepository) ListAssetTransactions(ctx context.Context, req *assettxpb.ListAssetTransactionsRequest) (*assettxpb.ListAssetTransactionsResponse, error) {
	if err := espynahttp.ValidateSortColumns(assetTransactionSortSpec, req.GetSort(), "asset_transaction"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list asset_transactions: %w", err)
	}

	var txs []*assettxpb.AssetTransaction
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		tx := &assettxpb.AssetTransaction{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, tx); err != nil {
			continue
		}
		txs = append(txs, tx)
	}

	return &assettxpb.ListAssetTransactionsResponse{
		Data:    txs,
		Success: true,
	}, nil
}

// GetAssetTransactionListPageData retrieves asset_transactions with pagination metadata.
func (r *SQLServerAssetTransactionRepository) GetAssetTransactionListPageData(
	ctx context.Context,
	req *assettxpb.GetAssetTransactionListPageDataRequest,
) (*assettxpb.GetAssetTransactionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset_transaction list page data request is required")
	}

	limit := int32(50)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
		}
	}

	listResp, err := r.ListAssetTransactions(ctx, &assettxpb.ListAssetTransactionsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list asset_transactions for page data: %w", err)
	}
	txs := listResp.GetData()

	totalItems := int32(len(txs))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &assettxpb.GetAssetTransactionListPageDataResponse{
		AssetTransactionList: txs,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetAssetTransactionItemPageData retrieves a single asset_transaction via
// composition over ReadAssetTransaction.
func (r *SQLServerAssetTransactionRepository) GetAssetTransactionItemPageData(
	ctx context.Context,
	req *assettxpb.GetAssetTransactionItemPageDataRequest,
) (*assettxpb.GetAssetTransactionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get asset_transaction item page data request is required")
	}
	if req.AssetTransactionId == "" {
		return nil, fmt.Errorf("asset_transaction ID is required")
	}

	rr, err := r.ReadAssetTransaction(ctx, &assettxpb.ReadAssetTransactionRequest{Data: &assettxpb.AssetTransaction{Id: req.AssetTransactionId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("asset_transaction with ID '%s' not found", req.AssetTransactionId)
	}

	return &assettxpb.GetAssetTransactionItemPageDataResponse{
		AssetTransaction: rr.GetData()[0],
		Success:          true,
	}, nil
}

// NewAssetTransactionRepository creates a new SQL Server asset_transaction repository (old-style constructor).
func NewAssetTransactionRepository(db *sql.DB, tableName string) assettxpb.AssetTransactionDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerAssetTransactionRepository(dbOps, tableName)
}
