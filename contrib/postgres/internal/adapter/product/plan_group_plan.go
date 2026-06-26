//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PlanGroupPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres plan_group_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPlanGroupPlanRepository(dbOps, tableName), nil
	})
}

type PostgresPlanGroupPlanRepository struct {
	pb.UnimplementedPlanGroupPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresPlanGroupPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.PlanGroupPlanDomainServiceServer {
	if tableName == "" {
		tableName = "plan_group_plan"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresPlanGroupPlanRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresPlanGroupPlanRepository) CreatePlanGroupPlan(ctx context.Context, req *pb.CreatePlanGroupPlanRequest) (*pb.CreatePlanGroupPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan group plan data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan group plan: %w", err)
	}
	item, err := planGroupPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreatePlanGroupPlanResponse{Data: []*pb.PlanGroupPlan{item}, Success: true}, nil
}

func (r *PostgresPlanGroupPlanRepository) ReadPlanGroupPlan(ctx context.Context, req *pb.ReadPlanGroupPlanRequest) (*pb.ReadPlanGroupPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan group plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan group plan: %w", err)
	}
	item, err := planGroupPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadPlanGroupPlanResponse{Data: []*pb.PlanGroupPlan{item}, Success: true}, nil
}

func (r *PostgresPlanGroupPlanRepository) UpdatePlanGroupPlan(ctx context.Context, req *pb.UpdatePlanGroupPlanRequest) (*pb.UpdatePlanGroupPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan group plan ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan group plan: %w", err)
	}
	item, err := planGroupPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePlanGroupPlanResponse{Data: []*pb.PlanGroupPlan{item}, Success: true}, nil
}

func (r *PostgresPlanGroupPlanRepository) DeletePlanGroupPlan(ctx context.Context, req *pb.DeletePlanGroupPlanRequest) (*pb.DeletePlanGroupPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan group plan ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete plan group plan: %w", err)
	}
	return &pb.DeletePlanGroupPlanResponse{Success: true}, nil
}

func (r *PostgresPlanGroupPlanRepository) ListPlanGroupPlans(ctx context.Context, req *pb.ListPlanGroupPlansRequest) (*pb.ListPlanGroupPlansResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListPlanGroupPlansResponse{Data: items, Success: true}, nil
}

func (r *PostgresPlanGroupPlanRepository) GetPlanGroupPlanListPageData(ctx context.Context, req *pb.GetPlanGroupPlanListPageDataRequest) (*pb.GetPlanGroupPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginatePlanGroupPlan(all, req.GetPagination())
	return &pb.GetPlanGroupPlanListPageDataResponse{PlanGroupPlanList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresPlanGroupPlanRepository) GetPlanGroupPlanItemPageData(ctx context.Context, req *pb.GetPlanGroupPlanItemPageDataRequest) (*pb.GetPlanGroupPlanItemPageDataResponse, error) {
	if req == nil || req.PlanGroupPlanId == "" {
		return nil, fmt.Errorf("plan group plan ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PlanGroupPlanId)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan group plan: %w", err)
	}
	item, err := planGroupPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetPlanGroupPlanItemPageDataResponse{PlanGroupPlan: item, Success: true}, nil
}

func (r *PostgresPlanGroupPlanRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.PlanGroupPlan, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan group plans: %w", err)
	}
	var items []*pb.PlanGroupPlan
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.PlanGroupPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func planGroupPlanFromResult(result any) (*pb.PlanGroupPlan, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.PlanGroupPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginatePlanGroupPlan(all []*pb.PlanGroupPlan, p *commonpb.PaginationRequest) (int32, []*pb.PlanGroupPlan, *commonpb.PaginationResponse) {
	limit, page := int32(50), int32(1)
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if off := p.GetOffset(); off != nil && off.Page > 0 {
			page = off.Page
		}
	}
	total := int32(len(all))
	start := (page - 1) * limit
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	totalPages := int32(0)
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return page, all[start:end], &commonpb.PaginationResponse{
		TotalItems:  total,
		CurrentPage: &page,
		TotalPages:  &totalPages,
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
}
