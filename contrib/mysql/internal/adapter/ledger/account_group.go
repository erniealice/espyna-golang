//go:build mysql

package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	accountgrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account_group"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.AccountGroup, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql account_group repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLAccountGroupRepository(dbOps, tableName), nil
	})
}

// MySQLAccountGroupRepository implements account_group CRUD operations using MySQL 8.0+.
type MySQLAccountGroupRepository struct {
	accountgrouppb.UnimplementedAccountGroupDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLAccountGroupRepository creates a new MySQL account_group repository.
func NewMySQLAccountGroupRepository(dbOps interfaces.DatabaseOperation, tableName string) accountgrouppb.AccountGroupDomainServiceServer {
	if tableName == "" {
		tableName = "account_group"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLAccountGroupRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *MySQLAccountGroupRepository) CreateAccountGroup(ctx context.Context, req *accountgrouppb.CreateAccountGroupRequest) (*accountgrouppb.CreateAccountGroupResponse, error) {
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
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	accountGroup := &accountgrouppb.AccountGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountGroup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accountgrouppb.CreateAccountGroupResponse{Data: []*accountgrouppb.AccountGroup{accountGroup}}, nil
}

func (r *MySQLAccountGroupRepository) ReadAccountGroup(ctx context.Context, req *accountgrouppb.ReadAccountGroupRequest) (*accountgrouppb.ReadAccountGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_group ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read account_group: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	accountGroup := &accountgrouppb.AccountGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountGroup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accountgrouppb.ReadAccountGroupResponse{Data: []*accountgrouppb.AccountGroup{accountGroup}}, nil
}

func (r *MySQLAccountGroupRepository) UpdateAccountGroup(ctx context.Context, req *accountgrouppb.UpdateAccountGroupRequest) (*accountgrouppb.UpdateAccountGroupResponse, error) {
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
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	accountGroup := &accountgrouppb.AccountGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, accountGroup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &accountgrouppb.UpdateAccountGroupResponse{Data: []*accountgrouppb.AccountGroup{accountGroup}}, nil
}

func (r *MySQLAccountGroupRepository) DeleteAccountGroup(ctx context.Context, req *accountgrouppb.DeleteAccountGroupRequest) (*accountgrouppb.DeleteAccountGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("account_group ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete account_group: %w", err)
	}
	return &accountgrouppb.DeleteAccountGroupResponse{Success: true}, nil
}

func (r *MySQLAccountGroupRepository) ListAccountGroups(ctx context.Context, req *accountgrouppb.ListAccountGroupsRequest) (*accountgrouppb.ListAccountGroupsResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		ag := &accountgrouppb.AccountGroup{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ag); err != nil {
			continue
		}
		accountGroups = append(accountGroups, ag)
	}
	return &accountgrouppb.ListAccountGroupsResponse{Data: accountGroups}, nil
}

func (r *MySQLAccountGroupRepository) GetAccountGroupListPageData(ctx context.Context, req *accountgrouppb.GetAccountGroupListPageDataRequest) (*accountgrouppb.GetAccountGroupListPageDataResponse, error) {
	return nil, fmt.Errorf("GetAccountGroupListPageData not yet implemented — Phase 2")
}

func (r *MySQLAccountGroupRepository) GetAccountGroupItemPageData(ctx context.Context, req *accountgrouppb.GetAccountGroupItemPageDataRequest) (*accountgrouppb.GetAccountGroupItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetAccountGroupItemPageData not yet implemented — Phase 2")
}
