//go:build sqlserver

// Package funding provides SQL Server adapters for the funding domain entities.
package funding

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	fundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Fund, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver fund repository requires *sql.DB, got %T", conn)
		}
		// Fund is a global entity (no workspace_id column). We use the base
		// SQLServerOperations directly — NOT WorkspaceAwareOperations — so the
		// workspace-injection logic never runs against this table.
		dbOps := sqlserverCore.NewSQLServerOperations(db)
		return NewSQLServerFundRepository(dbOps, tableName), nil
	})
}

// SQLServerFundRepository implements fund CRUD operations using SQL Server.
type SQLServerFundRepository struct {
	fundpb.UnimplementedFundDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerFundRepository creates a new SQL Server fund repository.
func NewSQLServerFundRepository(dbOps interfaces.DatabaseOperation, tableName string) fundpb.FundDomainServiceServer {
	if tableName == "" {
		tableName = "fund"
	}
	return &SQLServerFundRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateFund creates a new fund record.
func (r *SQLServerFundRepository) CreateFund(ctx context.Context, req *fundpb.CreateFundRequest) (*fundpb.CreateFundResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("fund data is required")
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
		return nil, fmt.Errorf("failed to create fund: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	fund := &fundpb.Fund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fund); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundpb.CreateFundResponse{Data: []*fundpb.Fund{fund}}, nil
}

// ReadFund retrieves a fund by ID.
func (r *SQLServerFundRepository) ReadFund(ctx context.Context, req *fundpb.ReadFundRequest) (*fundpb.ReadFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fund: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	fund := &fundpb.Fund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fund); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundpb.ReadFundResponse{Data: []*fundpb.Fund{fund}}, nil
}

// UpdateFund updates an existing fund record.
func (r *SQLServerFundRepository) UpdateFund(ctx context.Context, req *fundpb.UpdateFundRequest) (*fundpb.UpdateFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund ID is required")
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
		return nil, fmt.Errorf("failed to update fund: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	fund := &fundpb.Fund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fund); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundpb.UpdateFundResponse{Data: []*fundpb.Fund{fund}}, nil
}

// DeleteFund soft-deletes a fund.
func (r *SQLServerFundRepository) DeleteFund(ctx context.Context, req *fundpb.DeleteFundRequest) (*fundpb.DeleteFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete fund: %w", err)
	}
	return &fundpb.DeleteFundResponse{Success: true}, nil
}

// ListFunds lists funds matching optional filters.
func (r *SQLServerFundRepository) ListFunds(ctx context.Context, req *fundpb.ListFundsRequest) (*fundpb.ListFundsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list funds: %w", err)
	}
	var funds []*fundpb.Fund
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		fund := &fundpb.Fund{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fund); err != nil {
			continue
		}
		funds = append(funds, fund)
	}
	return &fundpb.ListFundsResponse{Data: funds}, nil
}
