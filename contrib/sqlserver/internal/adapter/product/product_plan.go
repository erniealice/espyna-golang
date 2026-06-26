//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProductPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductPlanRepository(dbOps, tableName), nil
	})
}

// SQLServerProductPlanRepository implements product_plan CRUD using SQL Server.
type SQLServerProductPlanRepository struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerProductPlanRepository creates a new SQL Server product_plan repository.
func NewSQLServerProductPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) productplanpb.ProductPlanDomainServiceServer {
	if tableName == "" {
		tableName = "product_plan"
	}
	return &SQLServerProductPlanRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerProductPlanRepository) CreateProductPlan(ctx context.Context, req *productplanpb.CreateProductPlanRequest) (*productplanpb.CreateProductPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product_plan data is required")
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
		return nil, fmt.Errorf("failed to create product_plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pp := &productplanpb.ProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productplanpb.CreateProductPlanResponse{Data: []*productplanpb.ProductPlan{pp}}, nil
}

func (r *SQLServerProductPlanRepository) ReadProductPlan(ctx context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product_plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pp := &productplanpb.ProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productplanpb.ReadProductPlanResponse{Data: []*productplanpb.ProductPlan{pp}}, nil
}

func (r *SQLServerProductPlanRepository) UpdateProductPlan(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) (*productplanpb.UpdateProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_plan ID is required")
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
		return nil, fmt.Errorf("failed to update product_plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pp := &productplanpb.ProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productplanpb.UpdateProductPlanResponse{Data: []*productplanpb.ProductPlan{pp}}, nil
}

func (r *SQLServerProductPlanRepository) DeleteProductPlan(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) (*productplanpb.DeleteProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product_plan ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product_plan: %w", err)
	}
	return &productplanpb.DeleteProductPlanResponse{Success: true}, nil
}

func (r *SQLServerProductPlanRepository) ListProductPlans(ctx context.Context, req *productplanpb.ListProductPlansRequest) (*productplanpb.ListProductPlansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product_plans: %w", err)
	}
	var pps []*productplanpb.ProductPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pp := &productplanpb.ProductPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pp); err != nil {
			continue
		}
		pps = append(pps, pp)
	}
	return &productplanpb.ListProductPlansResponse{Data: pps}, nil
}

func (r *SQLServerProductPlanRepository) GetProductPlanListPageData(ctx context.Context, req *productplanpb.GetProductPlanListPageDataRequest) (*productplanpb.GetProductPlanListPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductPlanListPageData not yet implemented — Phase 2")
}

func (r *SQLServerProductPlanRepository) GetProductPlanItemPageData(ctx context.Context, req *productplanpb.GetProductPlanItemPageDataRequest) (*productplanpb.GetProductPlanItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetProductPlanItemPageData not yet implemented — Phase 2")
}
