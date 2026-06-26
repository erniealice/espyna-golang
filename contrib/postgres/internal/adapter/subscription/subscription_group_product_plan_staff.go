//go:build postgresql

package subscription

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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionGroupProductPlanStaff, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_group_product_plan_staff repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionGroupProductPlanStaffRepository(dbOps, tableName), nil
	})
}

type PostgresSubscriptionGroupProductPlanStaffRepository struct {
	pb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresSubscriptionGroupProductPlanStaffRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SubscriptionGroupProductPlanStaffDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_group_product_plan_staff"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSubscriptionGroupProductPlanStaffRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) CreateSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription group product plan staff data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupProductPlanStaffResponse{Data: []*pb.SubscriptionGroupProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) ReadSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.ReadSubscriptionGroupProductPlanStaffRequest) (*pb.ReadSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Data: []*pb.SubscriptionGroupProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) UpdateSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.UpdateSubscriptionGroupProductPlanStaffRequest) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSubscriptionGroupProductPlanStaffResponse{Data: []*pb.SubscriptionGroupProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) DeleteSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.DeleteSubscriptionGroupProductPlanStaffRequest) (*pb.DeleteSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription group product plan staff: %w", err)
	}
	return &pb.DeleteSubscriptionGroupProductPlanStaffResponse{Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) ListSubscriptionGroupProductPlanStaffs(ctx context.Context, req *pb.ListSubscriptionGroupProductPlanStaffsRequest) (*pb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupProductPlanStaffsResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) GetSubscriptionGroupProductPlanStaffListPageData(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanStaffListPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginateSubscriptionGroupProductPlanStaff(all, req.GetPagination())
	return &pb.GetSubscriptionGroupProductPlanStaffListPageDataResponse{SubscriptionGroupProductPlanStaffList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) GetSubscriptionGroupProductPlanStaffItemPageData(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanStaffItemPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffItemPageDataResponse, error) {
	if req == nil || req.SubscriptionGroupProductPlanStaffId == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SubscriptionGroupProductPlanStaffId)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetSubscriptionGroupProductPlanStaffItemPageDataResponse{SubscriptionGroupProductPlanStaff: item, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.SubscriptionGroupProductPlanStaff, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription group product plan staffs: %w", err)
	}
	var items []*pb.SubscriptionGroupProductPlanStaff
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.SubscriptionGroupProductPlanStaff{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func subscriptionGroupProductPlanStaffFromResult(result any) (*pb.SubscriptionGroupProductPlanStaff, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.SubscriptionGroupProductPlanStaff{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateSubscriptionGroupProductPlanStaff(all []*pb.SubscriptionGroupProductPlanStaff, p *commonpb.PaginationRequest) (int32, []*pb.SubscriptionGroupProductPlanStaff, *commonpb.PaginationResponse) {
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
