//go:build sqlserver

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EquityTransaction, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver equity_transaction repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEquityTransactionRepository(dbOps, tableName), nil
	})
}

// SQLServerEquityTransactionRepository implements equity_transaction CRUD using SQL Server.
type SQLServerEquityTransactionRepository struct {
	equitytransactionpb.UnimplementedEquityTransactionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerEquityTransactionRepository creates a new SQL Server equity_transaction repository.
func NewSQLServerEquityTransactionRepository(dbOps interfaces.DatabaseOperation, tableName string) equitytransactionpb.EquityTransactionDomainServiceServer {
	if tableName == "" {
		tableName = "equity_transaction"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerEquityTransactionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerEquityTransactionRepository) CreateEquityTransaction(ctx context.Context, req *equitytransactionpb.CreateEquityTransactionRequest) (*equitytransactionpb.CreateEquityTransactionResponse, error) {
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
	return &equitytransactionpb.CreateEquityTransactionResponse{Data: []*equitytransactionpb.EquityTransaction{equityTransaction}}, nil
}

func (r *SQLServerEquityTransactionRepository) ReadEquityTransaction(ctx context.Context, req *equitytransactionpb.ReadEquityTransactionRequest) (*equitytransactionpb.ReadEquityTransactionResponse, error) {
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
	return &equitytransactionpb.ReadEquityTransactionResponse{Data: []*equitytransactionpb.EquityTransaction{equityTransaction}}, nil
}

func (r *SQLServerEquityTransactionRepository) UpdateEquityTransaction(ctx context.Context, req *equitytransactionpb.UpdateEquityTransactionRequest) (*equitytransactionpb.UpdateEquityTransactionResponse, error) {
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
	return &equitytransactionpb.UpdateEquityTransactionResponse{Data: []*equitytransactionpb.EquityTransaction{equityTransaction}}, nil
}

func (r *SQLServerEquityTransactionRepository) DeleteEquityTransaction(ctx context.Context, req *equitytransactionpb.DeleteEquityTransactionRequest) (*equitytransactionpb.DeleteEquityTransactionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_transaction ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete equity_transaction: %w", err)
	}
	return &equitytransactionpb.DeleteEquityTransactionResponse{Success: true}, nil
}

func (r *SQLServerEquityTransactionRepository) ListEquityTransactions(ctx context.Context, req *equitytransactionpb.ListEquityTransactionsRequest) (*equitytransactionpb.ListEquityTransactionsResponse, error) {
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
	return &equitytransactionpb.ListEquityTransactionsResponse{Data: equityTransactions}, nil
}

func (r *SQLServerEquityTransactionRepository) GetEquityTransactionListPageData(ctx context.Context, req *equitytransactionpb.GetEquityTransactionListPageDataRequest) (*equitytransactionpb.GetEquityTransactionListPageDataResponse, error) {
	return nil, fmt.Errorf("GetEquityTransactionListPageData not yet implemented — Phase 2")
}

func (r *SQLServerEquityTransactionRepository) GetEquityTransactionItemPageData(ctx context.Context, req *equitytransactionpb.GetEquityTransactionItemPageDataRequest) (*equitytransactionpb.GetEquityTransactionItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetEquityTransactionItemPageData not yet implemented — Phase 2")
}
