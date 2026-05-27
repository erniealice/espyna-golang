//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.CollectionPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver collection_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerCollectionPlanRepository(dbOps, tableName), nil
	})
}

// SQLServerCollectionPlanRepository implements collection_plan CRUD using SQL Server.
type SQLServerCollectionPlanRepository struct {
	collectionplanpb.UnimplementedCollectionPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerCollectionPlanRepository creates a new SQL Server collection_plan repository.
func NewSQLServerCollectionPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionplanpb.CollectionPlanDomainServiceServer {
	if tableName == "" {
		tableName = "collection_plan"
	}
	return &SQLServerCollectionPlanRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerCollectionPlanRepository) CreateCollectionPlan(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) (*collectionplanpb.CreateCollectionPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection_plan data is required")
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
		return nil, fmt.Errorf("failed to create collection_plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cp := &collectionplanpb.CollectionPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionplanpb.CreateCollectionPlanResponse{Data: []*collectionplanpb.CollectionPlan{cp}}, nil
}

func (r *SQLServerCollectionPlanRepository) ReadCollectionPlan(ctx context.Context, req *collectionplanpb.ReadCollectionPlanRequest) (*collectionplanpb.ReadCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cp := &collectionplanpb.CollectionPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionplanpb.ReadCollectionPlanResponse{Data: []*collectionplanpb.CollectionPlan{cp}}, nil
}

func (r *SQLServerCollectionPlanRepository) UpdateCollectionPlan(ctx context.Context, req *collectionplanpb.UpdateCollectionPlanRequest) (*collectionplanpb.UpdateCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_plan ID is required")
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
		return nil, fmt.Errorf("failed to update collection_plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cp := &collectionplanpb.CollectionPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionplanpb.UpdateCollectionPlanResponse{Data: []*collectionplanpb.CollectionPlan{cp}}, nil
}

func (r *SQLServerCollectionPlanRepository) DeleteCollectionPlan(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) (*collectionplanpb.DeleteCollectionPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_plan ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection_plan: %w", err)
	}
	return &collectionplanpb.DeleteCollectionPlanResponse{Success: true}, nil
}

func (r *SQLServerCollectionPlanRepository) ListCollectionPlans(ctx context.Context, req *collectionplanpb.ListCollectionPlansRequest) (*collectionplanpb.ListCollectionPlansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection_plans: %w", err)
	}
	var cps []*collectionplanpb.CollectionPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		cp := &collectionplanpb.CollectionPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
			continue
		}
		cps = append(cps, cp)
	}
	return &collectionplanpb.ListCollectionPlansResponse{Data: cps}, nil
}

func (r *SQLServerCollectionPlanRepository) GetCollectionPlanListPageData(ctx context.Context, req *collectionplanpb.GetCollectionPlanListPageDataRequest) (*collectionplanpb.GetCollectionPlanListPageDataResponse, error) {
	return nil, fmt.Errorf("GetCollectionPlanListPageData not yet implemented — Phase 2")
}

func (r *SQLServerCollectionPlanRepository) GetCollectionPlanItemPageData(ctx context.Context, req *collectionplanpb.GetCollectionPlanItemPageDataRequest) (*collectionplanpb.GetCollectionPlanItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetCollectionPlanItemPageData not yet implemented — Phase 2")
}
