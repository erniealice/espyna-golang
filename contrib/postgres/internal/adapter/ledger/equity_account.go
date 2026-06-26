//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	equityaccountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_account"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EquityAccount, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres equity_account repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEquityAccountRepository(dbOps, tableName), nil
	})
}

// PostgresEquityAccountRepository implements equity_account CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_equity_account_active ON equity_account(active)
//   - CREATE INDEX idx_equity_account_equity_type ON equity_account(equity_type)
//   - CREATE INDEX idx_equity_account_date_created ON equity_account(date_created DESC)
//
// TODO Phase 2: Implement GetEquityAccountListPageData with balance calculation and pagination
// TODO Phase 2: Implement GetEquityAccountItemPageData with transaction history summary
type PostgresEquityAccountRepository struct {
	equityaccountpb.UnimplementedEquityAccountDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresEquityAccountRepository creates a new PostgreSQL equity_account repository.
func NewPostgresEquityAccountRepository(dbOps interfaces.DatabaseOperation, tableName string) equityaccountpb.EquityAccountDomainServiceServer {
	if tableName == "" {
		tableName = "equity_account"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEquityAccountRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateEquityAccount creates a new equity_account using common PostgreSQL operations.
func (r *PostgresEquityAccountRepository) CreateEquityAccount(ctx context.Context, req *equityaccountpb.CreateEquityAccountRequest) (*equityaccountpb.CreateEquityAccountResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("equity_account data is required")
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
		return nil, fmt.Errorf("failed to create equity_account: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	equityAccount := &equityaccountpb.EquityAccount{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityAccount); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &equityaccountpb.CreateEquityAccountResponse{
		Data: []*equityaccountpb.EquityAccount{equityAccount},
	}, nil
}

// ReadEquityAccount retrieves an equity_account by ID using common PostgreSQL operations.
func (r *PostgresEquityAccountRepository) ReadEquityAccount(ctx context.Context, req *equityaccountpb.ReadEquityAccountRequest) (*equityaccountpb.ReadEquityAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_account ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read equity_account: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	equityAccount := &equityaccountpb.EquityAccount{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityAccount); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &equityaccountpb.ReadEquityAccountResponse{
		Data: []*equityaccountpb.EquityAccount{equityAccount},
	}, nil
}

// UpdateEquityAccount updates an equity_account using common PostgreSQL operations.
func (r *PostgresEquityAccountRepository) UpdateEquityAccount(ctx context.Context, req *equityaccountpb.UpdateEquityAccountRequest) (*equityaccountpb.UpdateEquityAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_account ID is required")
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
		return nil, fmt.Errorf("failed to update equity_account: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	equityAccount := &equityaccountpb.EquityAccount{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityAccount); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &equityaccountpb.UpdateEquityAccountResponse{
		Data: []*equityaccountpb.EquityAccount{equityAccount},
	}, nil
}

// DeleteEquityAccount soft-deletes an equity_account using common PostgreSQL operations.
func (r *PostgresEquityAccountRepository) DeleteEquityAccount(ctx context.Context, req *equityaccountpb.DeleteEquityAccountRequest) (*equityaccountpb.DeleteEquityAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_account ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete equity_account: %w", err)
	}

	return &equityaccountpb.DeleteEquityAccountResponse{
		Success: true,
	}, nil
}

// ListEquityAccounts lists equity_accounts using common PostgreSQL operations.
func (r *PostgresEquityAccountRepository) ListEquityAccounts(ctx context.Context, req *equityaccountpb.ListEquityAccountsRequest) (*equityaccountpb.ListEquityAccountsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list equity_accounts: %w", err)
	}

	var equityAccounts []*equityaccountpb.EquityAccount
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		equityAccount := &equityaccountpb.EquityAccount{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, equityAccount); err != nil {
			continue
		}
		equityAccounts = append(equityAccounts, equityAccount)
	}

	return &equityaccountpb.ListEquityAccountsResponse{
		Data: equityAccounts,
	}, nil
}

// GetEquityAccountListPageData - TODO Phase 2: CTE with computed balance from equity_transactions.
func (r *PostgresEquityAccountRepository) GetEquityAccountListPageData(ctx context.Context, req *equityaccountpb.GetEquityAccountListPageDataRequest) (*equityaccountpb.GetEquityAccountListPageDataResponse, error) {
	// TODO Phase 2: CTE with SUM(equity_transaction.amount) per account for running balance
	return nil, fmt.Errorf("GetEquityAccountListPageData not yet implemented — Phase 2")
}

// GetEquityAccountItemPageData - TODO Phase 2: implement with transaction history.
func (r *PostgresEquityAccountRepository) GetEquityAccountItemPageData(ctx context.Context, req *equityaccountpb.GetEquityAccountItemPageDataRequest) (*equityaccountpb.GetEquityAccountItemPageDataResponse, error) {
	// TODO Phase 2: fetch account + recent transactions + running balance
	return nil, fmt.Errorf("GetEquityAccountItemPageData not yet implemented — Phase 2")
}
