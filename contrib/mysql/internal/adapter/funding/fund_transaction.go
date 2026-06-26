//go:build mysql

package funding

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	fundtransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund_transaction"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.FundTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql fund_transaction repository requires *sql.DB, got %T", conn)
		}
		// FundTransaction.workspace_id is nullable (fund-global events have no workspace
		// attribution). WorkspaceAwareOperations handles this correctly.
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLFundTransactionRepository(dbOps, tableName), nil
	})
}

// MySQLFundTransactionRepository implements fund_transaction CRUD operations using MySQL 8.0+.
//
// Append-only semantics: FundTransaction is conceptually an append-only event log.
// This adapter exposes a standard UpdateFundTransaction to satisfy the service interface,
// but the use-case layer MUST enforce that only status transitions are permitted
// (DRAFT → POSTED → VOIDED). Direct field mutation violates the event-sourcing contract.
type MySQLFundTransactionRepository struct {
	fundtransactionpb.UnimplementedFundTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLFundTransactionRepository creates a new MySQL fund_transaction repository.
func NewMySQLFundTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) fundtransactionpb.FundTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "fund_transaction"
	}
	return &MySQLFundTransactionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateFundTransaction creates a new fund_transaction record.
func (r *MySQLFundTransactionRepository) CreateFundTransaction(ctx context.Context, req *fundtransactionpb.CreateFundTransactionRequest) (*fundtransactionpb.CreateFundTransactionResponse, error) {
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
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLFundTransactionRepository) ReadFundTransaction(ctx context.Context, req *fundtransactionpb.ReadFundTransactionRequest) (*fundtransactionpb.ReadFundTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_transaction ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fund_transaction: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
// IMPORTANT: FundTransaction is append-only. This method exists to satisfy the service
// interface but the use-case layer MUST restrict mutations to status transitions only
// (DRAFT → POSTED → VOIDED). Any other field mutation violates the event-sourcing contract.
func (r *MySQLFundTransactionRepository) UpdateFundTransaction(ctx context.Context, req *fundtransactionpb.UpdateFundTransactionRequest) (*fundtransactionpb.UpdateFundTransactionResponse, error) {
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
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
func (r *MySQLFundTransactionRepository) DeleteFundTransaction(ctx context.Context, req *fundtransactionpb.DeleteFundTransactionRequest) (*fundtransactionpb.DeleteFundTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_transaction ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete fund_transaction: %w", err)
	}
	return &fundtransactionpb.DeleteFundTransactionResponse{Success: true}, nil
}

// ListFundTransactions lists fund_transactions matching optional filters.
func (r *MySQLFundTransactionRepository) ListFundTransactions(ctx context.Context, req *fundtransactionpb.ListFundTransactionsRequest) (*fundtransactionpb.ListFundTransactionsResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
