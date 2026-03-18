
package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Account, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres account repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresAccountRepository(dbOps, tableName), nil
	})
}

// PostgresAccountRepository implements account CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_account_active ON account(active)
//   - CREATE INDEX idx_account_account_group_id ON account(account_group_id)
//   - CREATE INDEX idx_account_element ON account(element)
//   - CREATE INDEX idx_account_date_created ON account(date_created DESC)
//   - CREATE INDEX idx_account_code ON account(code)
//
// TODO Phase 2: Implement GetAccountListPageData with CTE + search/pagination
// TODO Phase 2: Implement GetAccountItemPageData with enriched data
// TODO Phase 2: Implement GetAccountTreePageData for hierarchical CoA display
type PostgresAccountRepository struct {
	accountpb.UnimplementedAccountDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresAccountRepository creates a new PostgreSQL account repository.
func NewPostgresAccountRepository(dbOps interfaces.DatabaseOperation, tableName string) accountpb.AccountDomainServiceServer {
	if tableName == "" {
		tableName = "account"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresAccountRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccount creates a new account using common PostgreSQL operations.
func (r *PostgresAccountRepository) CreateAccount(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("account data is required")
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
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	account := &accountpb.Account{}
	if err := protojson.Unmarshal(resultJSON, account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accountpb.CreateAccountResponse{
		Data: []*accountpb.Account{account},
	}, nil
}

// ReadAccount retrieves an account by ID using common PostgreSQL operations.
func (r *PostgresAccountRepository) ReadAccount(ctx context.Context, req *accountpb.ReadAccountRequest) (*accountpb.ReadAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read account: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	account := &accountpb.Account{}
	if err := protojson.Unmarshal(resultJSON, account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accountpb.ReadAccountResponse{
		Data: []*accountpb.Account{account},
	}, nil
}

// UpdateAccount updates an account using common PostgreSQL operations.
func (r *PostgresAccountRepository) UpdateAccount(ctx context.Context, req *accountpb.UpdateAccountRequest) (*accountpb.UpdateAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
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
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	account := &accountpb.Account{}
	if err := protojson.Unmarshal(resultJSON, account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accountpb.UpdateAccountResponse{
		Data: []*accountpb.Account{account},
	}, nil
}

// DeleteAccount soft-deletes an account using common PostgreSQL operations.
func (r *PostgresAccountRepository) DeleteAccount(ctx context.Context, req *accountpb.DeleteAccountRequest) (*accountpb.DeleteAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete account: %w", err)
	}

	return &accountpb.DeleteAccountResponse{
		Success: true,
	}, nil
}

// ListAccounts lists accounts using common PostgreSQL operations.
func (r *PostgresAccountRepository) ListAccounts(ctx context.Context, req *accountpb.ListAccountsRequest) (*accountpb.ListAccountsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var accounts []*accountpb.Account
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		account := &accountpb.Account{}
		if err := protojson.Unmarshal(resultJSON, account); err != nil {
			continue
		}
		accounts = append(accounts, account)
	}

	return &accountpb.ListAccountsResponse{
		Data: accounts,
	}, nil
}

// GetAccountListPageData - TODO Phase 2: implement CTE query with search, pagination, and sorting.
func (r *PostgresAccountRepository) GetAccountListPageData(ctx context.Context, req *accountpb.GetAccountListPageDataRequest) (*accountpb.GetAccountListPageDataResponse, error) {
	// TODO Phase 2: CTE query with enriched account_group data, search by code/name, pagination, sort by code ASC
	return nil, fmt.Errorf("GetAccountListPageData not yet implemented — Phase 2")
}

// GetAccountItemPageData - TODO Phase 2: implement CTE query for single account with enriched data.
func (r *PostgresAccountRepository) GetAccountItemPageData(ctx context.Context, req *accountpb.GetAccountItemPageDataRequest) (*accountpb.GetAccountItemPageDataResponse, error) {
	// TODO Phase 2: CTE query with account_group join, child accounts, recent journal lines
	return nil, fmt.Errorf("GetAccountItemPageData not yet implemented — Phase 2")
}

// GetAccountTreePageData - TODO Phase 2: implement recursive CTE for hierarchical CoA display.
func (r *PostgresAccountRepository) GetAccountTreePageData(ctx context.Context, req *accountpb.GetAccountTreePageDataRequest) (*accountpb.GetAccountTreePageDataResponse, error) {
	// TODO Phase 2: recursive CTE (WITH RECURSIVE) to build account tree grouped by element/classification
	return nil, fmt.Errorf("GetAccountTreePageData not yet implemented — Phase 2")
}
