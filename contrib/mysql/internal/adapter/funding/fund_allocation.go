//go:build mysql

package funding

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	fundallocationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/funding/fund_allocation"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.FundAllocation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql fund_allocation repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLFundAllocationRepository(dbOps, tableName), nil
	})
}

// MySQLFundAllocationRepository implements fund_allocation CRUD operations using MySQL 8.0+.
type MySQLFundAllocationRepository struct {
	fundallocationpb.UnimplementedFundAllocationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLFundAllocationRepository creates a new MySQL fund_allocation repository.
func NewMySQLFundAllocationRepository(dbOps interfaces.DatabaseOperation, tableName string) fundallocationpb.FundAllocationDomainServiceServer {
	if tableName == "" {
		tableName = "fund_allocation"
	}
	return &MySQLFundAllocationRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateFundAllocation creates a new fund_allocation record.
func (r *MySQLFundAllocationRepository) CreateFundAllocation(ctx context.Context, req *fundallocationpb.CreateFundAllocationRequest) (*fundallocationpb.CreateFundAllocationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("fund_allocation data is required")
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
		return nil, fmt.Errorf("failed to create fund_allocation: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	alloc := &fundallocationpb.FundAllocation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, alloc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundallocationpb.CreateFundAllocationResponse{Data: []*fundallocationpb.FundAllocation{alloc}}, nil
}

// ReadFundAllocation retrieves a fund_allocation by ID.
func (r *MySQLFundAllocationRepository) ReadFundAllocation(ctx context.Context, req *fundallocationpb.ReadFundAllocationRequest) (*fundallocationpb.ReadFundAllocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_allocation ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fund_allocation: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	alloc := &fundallocationpb.FundAllocation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, alloc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundallocationpb.ReadFundAllocationResponse{Data: []*fundallocationpb.FundAllocation{alloc}}, nil
}

// UpdateFundAllocation updates an existing fund_allocation record.
func (r *MySQLFundAllocationRepository) UpdateFundAllocation(ctx context.Context, req *fundallocationpb.UpdateFundAllocationRequest) (*fundallocationpb.UpdateFundAllocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_allocation ID is required")
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
		return nil, fmt.Errorf("failed to update fund_allocation: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	alloc := &fundallocationpb.FundAllocation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, alloc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &fundallocationpb.UpdateFundAllocationResponse{Data: []*fundallocationpb.FundAllocation{alloc}}, nil
}

// DeleteFundAllocation soft-deletes a fund_allocation.
func (r *MySQLFundAllocationRepository) DeleteFundAllocation(ctx context.Context, req *fundallocationpb.DeleteFundAllocationRequest) (*fundallocationpb.DeleteFundAllocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fund_allocation ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete fund_allocation: %w", err)
	}
	return &fundallocationpb.DeleteFundAllocationResponse{Success: true}, nil
}

// ListFundAllocations lists fund_allocations matching optional filters.
func (r *MySQLFundAllocationRepository) ListFundAllocations(ctx context.Context, req *fundallocationpb.ListFundAllocationsRequest) (*fundallocationpb.ListFundAllocationsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list fund_allocations: %w", err)
	}
	var allocs []*fundallocationpb.FundAllocation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		alloc := &fundallocationpb.FundAllocation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, alloc); err != nil {
			continue
		}
		allocs = append(allocs, alloc)
	}
	return &fundallocationpb.ListFundAllocationsResponse{Data: allocs}, nil
}
