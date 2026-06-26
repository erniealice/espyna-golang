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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PlanGroup, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres plan_group repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPlanGroupRepository(dbOps, tableName), nil
	})
}

type PostgresPlanGroupRepository struct {
	pb.UnimplementedPlanGroupDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresPlanGroupRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.PlanGroupDomainServiceServer {
	if tableName == "" {
		tableName = "plan_group"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresPlanGroupRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresPlanGroupRepository) CreatePlanGroup(ctx context.Context, req *pb.CreatePlanGroupRequest) (*pb.CreatePlanGroupResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan group data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan group: %w", err)
	}
	item, err := planGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreatePlanGroupResponse{Data: []*pb.PlanGroup{item}, Success: true}, nil
}

func (r *PostgresPlanGroupRepository) ReadPlanGroup(ctx context.Context, req *pb.ReadPlanGroupRequest) (*pb.ReadPlanGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan group ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan group: %w", err)
	}
	item, err := planGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadPlanGroupResponse{Data: []*pb.PlanGroup{item}, Success: true}, nil
}

func (r *PostgresPlanGroupRepository) UpdatePlanGroup(ctx context.Context, req *pb.UpdatePlanGroupRequest) (*pb.UpdatePlanGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan group ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan group: %w", err)
	}
	item, err := planGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePlanGroupResponse{Data: []*pb.PlanGroup{item}, Success: true}, nil
}

func (r *PostgresPlanGroupRepository) DeletePlanGroup(ctx context.Context, req *pb.DeletePlanGroupRequest) (*pb.DeletePlanGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan group ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete plan group: %w", err)
	}
	return &pb.DeletePlanGroupResponse{Success: true}, nil
}

func (r *PostgresPlanGroupRepository) ListPlanGroups(ctx context.Context, req *pb.ListPlanGroupsRequest) (*pb.ListPlanGroupsResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListPlanGroupsResponse{Data: items, Success: true}, nil
}

func (r *PostgresPlanGroupRepository) GetPlanGroupListPageData(ctx context.Context, req *pb.GetPlanGroupListPageDataRequest) (*pb.GetPlanGroupListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginatePlanGroup(all, req.GetPagination())
	return &pb.GetPlanGroupListPageDataResponse{PlanGroupList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresPlanGroupRepository) GetPlanGroupItemPageData(ctx context.Context, req *pb.GetPlanGroupItemPageDataRequest) (*pb.GetPlanGroupItemPageDataResponse, error) {
	if req == nil || req.PlanGroupId == "" {
		return nil, fmt.Errorf("plan group ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PlanGroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan group: %w", err)
	}
	item, err := planGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetPlanGroupItemPageDataResponse{PlanGroup: item, Success: true}, nil
}

func (r *PostgresPlanGroupRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.PlanGroup, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan groups: %w", err)
	}
	var items []*pb.PlanGroup
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.PlanGroup{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func planGroupFromResult(result any) (*pb.PlanGroup, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.PlanGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginatePlanGroup(all []*pb.PlanGroup, p *commonpb.PaginationRequest) (int32, []*pb.PlanGroup, *commonpb.PaginationResponse) {
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
