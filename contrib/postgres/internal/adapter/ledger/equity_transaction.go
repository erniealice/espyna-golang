package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EquityTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres equity_transaction repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEquityTransactionRepository(dbOps, tableName), nil
	})
}

// PostgresEquityTransactionRepository implements equity_transaction CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_equity_transaction_equity_account_id ON equity_transaction(equity_account_id)
//   - CREATE INDEX idx_equity_transaction_transaction_date ON equity_transaction(transaction_date DESC)
//   - CREATE INDEX idx_equity_transaction_active ON equity_transaction(active)
//   - CREATE INDEX idx_equity_transaction_transaction_type ON equity_transaction(transaction_type)
//
// TODO Phase 2: Implement GetEquityTransactionListPageData with equity_account filter and pagination
// TODO Phase 2: Implement GetEquityTransactionItemPageData with account context
type PostgresEquityTransactionRepository struct {
	equitytransactionpb.UnimplementedEquityTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresEquityTransactionRepository creates a new PostgreSQL equity_transaction repository.
func NewPostgresEquityTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) equitytransactionpb.EquityTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "equity_transaction"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEquityTransactionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEquityTransaction creates a new equity_transaction using common PostgreSQL operations.
func (r *PostgresEquityTransactionRepository) CreateEquityTransaction(ctx context.Context, req *equitytransactionpb.CreateEquityTransactionRequest) (*equitytransactionpb.CreateEquityTransactionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("equity_transaction data is required")
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
		return nil, fmt.Errorf("failed to create equity_transaction: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	equityTransaction := &equitytransactionpb.EquityTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityTransaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &equitytransactionpb.CreateEquityTransactionResponse{
		Data: []*equitytransactionpb.EquityTransaction{equityTransaction},
	}, nil
}

// ReadEquityTransaction retrieves an equity_transaction by ID using common PostgreSQL operations.
func (r *PostgresEquityTransactionRepository) ReadEquityTransaction(ctx context.Context, req *equitytransactionpb.ReadEquityTransactionRequest) (*equitytransactionpb.ReadEquityTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_transaction ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read equity_transaction: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	equityTransaction := &equitytransactionpb.EquityTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityTransaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &equitytransactionpb.ReadEquityTransactionResponse{
		Data: []*equitytransactionpb.EquityTransaction{equityTransaction},
	}, nil
}

// UpdateEquityTransaction updates an equity_transaction using common PostgreSQL operations.
func (r *PostgresEquityTransactionRepository) UpdateEquityTransaction(ctx context.Context, req *equitytransactionpb.UpdateEquityTransactionRequest) (*equitytransactionpb.UpdateEquityTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_transaction ID is required")
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
		return nil, fmt.Errorf("failed to update equity_transaction: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	equityTransaction := &equitytransactionpb.EquityTransaction{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityTransaction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &equitytransactionpb.UpdateEquityTransactionResponse{
		Data: []*equitytransactionpb.EquityTransaction{equityTransaction},
	}, nil
}

// DeleteEquityTransaction soft-deletes an equity_transaction using common PostgreSQL operations.
func (r *PostgresEquityTransactionRepository) DeleteEquityTransaction(ctx context.Context, req *equitytransactionpb.DeleteEquityTransactionRequest) (*equitytransactionpb.DeleteEquityTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_transaction ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete equity_transaction: %w", err)
	}

	return &equitytransactionpb.DeleteEquityTransactionResponse{
		Success: true,
	}, nil
}

// ListEquityTransactions lists equity_transactions using common PostgreSQL operations.
func (r *PostgresEquityTransactionRepository) ListEquityTransactions(ctx context.Context, req *equitytransactionpb.ListEquityTransactionsRequest) (*equitytransactionpb.ListEquityTransactionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list equity_transactions: %w", err)
	}

	var equityTransactions []*equitytransactionpb.EquityTransaction
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		equityTransaction := &equitytransactionpb.EquityTransaction{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityTransaction); err != nil {
			continue
		}
		equityTransactions = append(equityTransactions, equityTransaction)
	}

	return &equitytransactionpb.ListEquityTransactionsResponse{
		Data: equityTransactions,
	}, nil
}

// GetEquityTransactionListPageData - TODO Phase 2: CTE with equity_account join, date range filter, pagination.
func (r *PostgresEquityTransactionRepository) GetEquityTransactionListPageData(ctx context.Context, req *equitytransactionpb.GetEquityTransactionListPageDataRequest) (*equitytransactionpb.GetEquityTransactionListPageDataResponse, error) {
	// TODO Phase 2: CTE with equity_account name join, filter by account/date range, sort by transaction_date DESC
	return nil, fmt.Errorf("GetEquityTransactionListPageData not yet implemented — Phase 2")
}

// GetEquityTransactionItemPageData - TODO Phase 2: implement with equity_account context.
func (r *PostgresEquityTransactionRepository) GetEquityTransactionItemPageData(ctx context.Context, req *equitytransactionpb.GetEquityTransactionItemPageDataRequest) (*equitytransactionpb.GetEquityTransactionItemPageDataResponse, error) {
	// TODO Phase 2: fetch transaction + parent equity_account details
	return nil, fmt.Errorf("GetEquityTransactionItemPageData not yet implemented — Phase 2")
}
