//go:build postgresql

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
	accountgrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account_group"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.AccountGroup, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres account_group repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresAccountGroupRepository(dbOps, tableName), nil
	})
}

// PostgresAccountGroupRepository implements account_group CRUD operations using PostgreSQL.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_account_group_active ON account_group(active)
//   - CREATE INDEX idx_account_group_element ON account_group(element)
//   - CREATE INDEX idx_account_group_date_created ON account_group(date_created DESC)
//
// TODO Phase 2: Implement GetAccountGroupListPageData with CTE + search/pagination
// TODO Phase 2: Implement GetAccountGroupItemPageData with child accounts count
type PostgresAccountGroupRepository struct {
	accountgrouppb.UnimplementedAccountGroupDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresAccountGroupRepository creates a new PostgreSQL account_group repository.
func NewPostgresAccountGroupRepository(dbOps interfaces.DatabaseOperation, tableName string) accountgrouppb.AccountGroupDomainServiceServer {
	if tableName == "" {
		tableName = "account_group"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresAccountGroupRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateAccountGroup creates a new account_group using common PostgreSQL operations.
func (r *PostgresAccountGroupRepository) CreateAccountGroup(ctx context.Context, req *accountgrouppb.CreateAccountGroupRequest) (*accountgrouppb.CreateAccountGroupResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("account_group data is required")
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
		return nil, fmt.Errorf("failed to create account_group: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	accountGroup := &accountgrouppb.AccountGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountGroup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accountgrouppb.CreateAccountGroupResponse{
		Data: []*accountgrouppb.AccountGroup{accountGroup},
	}, nil
}

// ReadAccountGroup retrieves an account_group by ID using common PostgreSQL operations.
func (r *PostgresAccountGroupRepository) ReadAccountGroup(ctx context.Context, req *accountgrouppb.ReadAccountGroupRequest) (*accountgrouppb.ReadAccountGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_group ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read account_group: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	accountGroup := &accountgrouppb.AccountGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountGroup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accountgrouppb.ReadAccountGroupResponse{
		Data: []*accountgrouppb.AccountGroup{accountGroup},
	}, nil
}

// UpdateAccountGroup updates an account_group using common PostgreSQL operations.
func (r *PostgresAccountGroupRepository) UpdateAccountGroup(ctx context.Context, req *accountgrouppb.UpdateAccountGroupRequest) (*accountgrouppb.UpdateAccountGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_group ID is required")
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
		return nil, fmt.Errorf("failed to update account_group: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	accountGroup := &accountgrouppb.AccountGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountGroup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &accountgrouppb.UpdateAccountGroupResponse{
		Data: []*accountgrouppb.AccountGroup{accountGroup},
	}, nil
}

// DeleteAccountGroup soft-deletes an account_group using common PostgreSQL operations.
func (r *PostgresAccountGroupRepository) DeleteAccountGroup(ctx context.Context, req *accountgrouppb.DeleteAccountGroupRequest) (*accountgrouppb.DeleteAccountGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_group ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete account_group: %w", err)
	}

	return &accountgrouppb.DeleteAccountGroupResponse{
		Success: true,
	}, nil
}

// ListAccountGroups lists account_groups using common PostgreSQL operations.
func (r *PostgresAccountGroupRepository) ListAccountGroups(ctx context.Context, req *accountgrouppb.ListAccountGroupsRequest) (*accountgrouppb.ListAccountGroupsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list account_groups: %w", err)
	}

	var accountGroups []*accountgrouppb.AccountGroup
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		accountGroup := &accountgrouppb.AccountGroup{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountGroup); err != nil {
			continue
		}
		accountGroups = append(accountGroups, accountGroup)
	}

	return &accountgrouppb.ListAccountGroupsResponse{
		Data: accountGroups,
	}, nil
}

// GetAccountGroupListPageData - TODO Phase 2: implement CTE query with search and pagination.
func (r *PostgresAccountGroupRepository) GetAccountGroupListPageData(ctx context.Context, req *accountgrouppb.GetAccountGroupListPageDataRequest) (*accountgrouppb.GetAccountGroupListPageDataResponse, error) {
	// TODO Phase 2: CTE with account count per group, search by name, pagination
	return nil, fmt.Errorf("GetAccountGroupListPageData not yet implemented — Phase 2")
}

// GetAccountGroupItemPageData - TODO Phase 2: implement with account list under this group.
func (r *PostgresAccountGroupRepository) GetAccountGroupItemPageData(ctx context.Context, req *accountgrouppb.GetAccountGroupItemPageDataRequest) (*accountgrouppb.GetAccountGroupItemPageDataResponse, error) {
	// TODO Phase 2: fetch group + all child accounts
	return nil, fmt.Errorf("GetAccountGroupItemPageData not yet implemented — Phase 2")
}
