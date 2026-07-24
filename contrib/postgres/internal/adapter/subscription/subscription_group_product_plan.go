//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionGroupProductPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_group_product_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionGroupProductPlanRepository(dbOps, tableName), nil
	})
}

type PostgresSubscriptionGroupProductPlanRepository struct {
	pb.UnimplementedSubscriptionGroupProductPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresSubscriptionGroupProductPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SubscriptionGroupProductPlanDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_group_product_plan"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSubscriptionGroupProductPlanRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresSubscriptionGroupProductPlanRepository) CreateSubscriptionGroupProductPlan(ctx context.Context, req *pb.CreateSubscriptionGroupProductPlanRequest) (*pb.CreateSubscriptionGroupProductPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription group product plan data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription group product plan: %w", err)
	}
	item, err := subscriptionGroupProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupProductPlanResponse{Data: []*pb.SubscriptionGroupProductPlan{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanRepository) ReadSubscriptionGroupProductPlan(ctx context.Context, req *pb.ReadSubscriptionGroupProductPlanRequest) (*pb.ReadSubscriptionGroupProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group product plan: %w", err)
	}
	item, err := subscriptionGroupProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadSubscriptionGroupProductPlanResponse{Data: []*pb.SubscriptionGroupProductPlan{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanRepository) UpdateSubscriptionGroupProductPlan(ctx context.Context, req *pb.UpdateSubscriptionGroupProductPlanRequest) (*pb.UpdateSubscriptionGroupProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription group product plan: %w", err)
	}
	item, err := subscriptionGroupProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSubscriptionGroupProductPlanResponse{Data: []*pb.SubscriptionGroupProductPlan{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanRepository) DeleteSubscriptionGroupProductPlan(ctx context.Context, req *pb.DeleteSubscriptionGroupProductPlanRequest) (*pb.DeleteSubscriptionGroupProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription group product plan: %w", err)
	}
	return &pb.DeleteSubscriptionGroupProductPlanResponse{Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanRepository) ListSubscriptionGroupProductPlans(ctx context.Context, req *pb.ListSubscriptionGroupProductPlansRequest) (*pb.ListSubscriptionGroupProductPlansResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupProductPlansResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanRepository) GetSubscriptionGroupProductPlanListPageData(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanListPageDataRequest) (*pb.GetSubscriptionGroupProductPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginateSubscriptionGroupProductPlan(all, req.GetPagination())
	return &pb.GetSubscriptionGroupProductPlanListPageDataResponse{SubscriptionGroupProductPlanList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanRepository) GetSubscriptionGroupProductPlanItemPageData(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanItemPageDataRequest) (*pb.GetSubscriptionGroupProductPlanItemPageDataResponse, error) {
	if req == nil || req.SubscriptionGroupProductPlanId == "" {
		return nil, fmt.Errorf("subscription group product plan ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SubscriptionGroupProductPlanId)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group product plan: %w", err)
	}
	item, err := subscriptionGroupProductPlanFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetSubscriptionGroupProductPlanItemPageDataResponse{SubscriptionGroupProductPlan: item, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.SubscriptionGroupProductPlan, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription group product plans: %w", err)
	}
	var items []*pb.SubscriptionGroupProductPlan
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.SubscriptionGroupProductPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func subscriptionGroupProductPlanFromResult(result any) (*pb.SubscriptionGroupProductPlan, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.SubscriptionGroupProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateSubscriptionGroupProductPlan(all []*pb.SubscriptionGroupProductPlan, p *commonpb.PaginationRequest) (int32, []*pb.SubscriptionGroupProductPlan, *commonpb.PaginationResponse) {
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
