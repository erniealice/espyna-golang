//go:build postgresql

package funding

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	fundtransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund_transaction"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.FundTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres fund_transaction repository requires *sql.DB, got %T", conn)
		}
		// FundTransaction.workspace_id is nullable (fund-global events have no
		// workspace attribution). WorkspaceAwareOperations handles this correctly:
		// when the table has a workspace_id column and the context carries a
		// workspace, it injects the filter on List and validates ownership on
		// Read/Update/Delete — NULL workspace_id rows are treated as not-found
		// for cross-workspace safety. Fund-global inserts (no workspace context)
		// pass through the workspace injection unchanged.
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresFundTransactionRepository(dbOps, tableName), nil
	})
}

// PostgresFundTransactionRepository implements fund_transaction CRUD operations using PostgreSQL.
//
// Append-only semantics: FundTransaction is conceptually an append-only event log
// (architecture.md §3.10). This adapter exposes a standard UpdateFundTransaction method
// to satisfy the service interface, but the use-case layer MUST enforce that only
// `status` transitions are permitted (DRAFT → POSTED → VOIDED). Direct field mutation
// is a violation of the event-sourcing contract. Corrections must be made by inserting
// a *_REVERSAL row (reverses_id set) followed by a corrected row — never by mutating
// an existing row.
type PostgresFundTransactionRepository struct {
	fundtransactionpb.UnimplementedFundTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresFundTransactionRepository creates a new PostgreSQL fund_transaction repository.
func NewPostgresFundTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) fundtransactionpb.FundTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "fund_transaction"
	}
	return &PostgresFundTransactionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateFundTransaction creates a new fund_transaction record.
func (r *PostgresFundTransactionRepository) CreateFundTransaction(ctx context.Context, req *fundtransactionpb.CreateFundTransactionRequest) (*fundtransactionpb.CreateFundTransactionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("fund_transaction data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create fund_transaction: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	tx := &fundtransactionpb.FundTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundtransactionpb.CreateFundTransactionResponse{Data: []*fundtransactionpb.FundTransaction{tx}}, nil
}

// ReadFundTransaction retrieves a fund_transaction by ID.
func (r *PostgresFundTransactionRepository) ReadFundTransaction(ctx context.Context, req *fundtransactionpb.ReadFundTransactionRequest) (*fundtransactionpb.ReadFundTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_transaction ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fund_transaction: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	tx := &fundtransactionpb.FundTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundtransactionpb.ReadFundTransactionResponse{Data: []*fundtransactionpb.FundTransaction{tx}}, nil
}

// UpdateFundTransaction updates an existing fund_transaction record.
//
// IMPORTANT: FundTransaction is append-only (architecture.md §3.10). This method
// exists to satisfy the service interface but the use-case layer MUST restrict
// mutations to status transitions only (DRAFT → POSTED → VOIDED). Any other field
// mutation violates the event-sourcing contract.
func (r *PostgresFundTransactionRepository) UpdateFundTransaction(ctx context.Context, req *fundtransactionpb.UpdateFundTransactionRequest) (*fundtransactionpb.UpdateFundTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_transaction ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update fund_transaction: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	tx := &fundtransactionpb.FundTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundtransactionpb.UpdateFundTransactionResponse{Data: []*fundtransactionpb.FundTransaction{tx}}, nil
}

// DeleteFundTransaction soft-deletes a fund_transaction.
func (r *PostgresFundTransactionRepository) DeleteFundTransaction(ctx context.Context, req *fundtransactionpb.DeleteFundTransactionRequest) (*fundtransactionpb.DeleteFundTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_transaction ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete fund_transaction: %w", err)
	}
	return &fundtransactionpb.DeleteFundTransactionResponse{Success: true}, nil
}

// ListFundTransactions lists fund_transactions matching optional filters.
func (r *PostgresFundTransactionRepository) ListFundTransactions(ctx context.Context, req *fundtransactionpb.ListFundTransactionsRequest) (*fundtransactionpb.ListFundTransactionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list fund_transactions: %w", err)
	}
	var txs []*fundtransactionpb.FundTransaction
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		tx := &fundtransactionpb.FundTransaction{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, tx); err != nil {
			continue
		}
		txs = append(txs, tx)
	}
	return &fundtransactionpb.ListFundTransactionsResponse{Data: txs}, nil
}
