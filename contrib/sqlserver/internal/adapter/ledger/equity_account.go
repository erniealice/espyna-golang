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
	equityaccountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_account"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.EquityAccount, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver equity_account repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerEquityAccountRepository(dbOps, tableName), nil
	})
}

// SQLServerEquityAccountRepository implements equity_account CRUD using SQL Server.
type SQLServerEquityAccountRepository struct {
	equityaccountpb.UnimplementedEquityAccountDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerEquityAccountRepository creates a new SQL Server equity_account repository.
func NewSQLServerEquityAccountRepository(dbOps interfaces.DatabaseOperation, tableName string) equityaccountpb.EquityAccountDomainServiceServer {
	if tableName == "" {
		tableName = "equity_account"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerEquityAccountRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerEquityAccountRepository) CreateEquityAccount(ctx context.Context, req *equityaccountpb.CreateEquityAccountRequest) (*equityaccountpb.CreateEquityAccountResponse, error) {
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
	return &equityaccountpb.CreateEquityAccountResponse{Data: []*equityaccountpb.EquityAccount{equityAccount}}, nil
}

func (r *SQLServerEquityAccountRepository) ReadEquityAccount(ctx context.Context, req *equityaccountpb.ReadEquityAccountRequest) (*equityaccountpb.ReadEquityAccountResponse, error) {
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
	return &equityaccountpb.ReadEquityAccountResponse{Data: []*equityaccountpb.EquityAccount{equityAccount}}, nil
}

func (r *SQLServerEquityAccountRepository) UpdateEquityAccount(ctx context.Context, req *equityaccountpb.UpdateEquityAccountRequest) (*equityaccountpb.UpdateEquityAccountResponse, error) {
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
	return &equityaccountpb.UpdateEquityAccountResponse{Data: []*equityaccountpb.EquityAccount{equityAccount}}, nil
}

func (r *SQLServerEquityAccountRepository) DeleteEquityAccount(ctx context.Context, req *equityaccountpb.DeleteEquityAccountRequest) (*equityaccountpb.DeleteEquityAccountResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("equity_account ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete equity_account: %w", err)
	}
	return &equityaccountpb.DeleteEquityAccountResponse{Success: true}, nil
}

func (r *SQLServerEquityAccountRepository) ListEquityAccounts(ctx context.Context, req *equityaccountpb.ListEquityAccountsRequest) (*equityaccountpb.ListEquityAccountsResponse, error) {
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
	return &equityaccountpb.ListEquityAccountsResponse{Data: equityAccounts}, nil
}

func (r *SQLServerEquityAccountRepository) GetEquityAccountListPageData(ctx context.Context, req *equityaccountpb.GetEquityAccountListPageDataRequest) (*equityaccountpb.GetEquityAccountListPageDataResponse, error) {
	return nil, fmt.Errorf("GetEquityAccountListPageData not yet implemented — Phase 2")
}

func (r *SQLServerEquityAccountRepository) GetEquityAccountItemPageData(ctx context.Context, req *equityaccountpb.GetEquityAccountItemPageDataRequest) (*equityaccountpb.GetEquityAccountItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetEquityAccountItemPageData not yet implemented — Phase 2")
}
