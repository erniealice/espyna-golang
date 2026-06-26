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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionGroup, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_group repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionGroupRepository(dbOps, tableName), nil
	})
}

type PostgresSubscriptionGroupRepository struct {
	pb.UnimplementedSubscriptionGroupDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresSubscriptionGroupRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SubscriptionGroupDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_group"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSubscriptionGroupRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresSubscriptionGroupRepository) CreateSubscriptionGroup(ctx context.Context, req *pb.CreateSubscriptionGroupRequest) (*pb.CreateSubscriptionGroupResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription group data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription group: %w", err)
	}
	item, err := subscriptionGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupResponse{Data: []*pb.SubscriptionGroup{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupRepository) ReadSubscriptionGroup(ctx context.Context, req *pb.ReadSubscriptionGroupRequest) (*pb.ReadSubscriptionGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group: %w", err)
	}
	item, err := subscriptionGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadSubscriptionGroupResponse{Data: []*pb.SubscriptionGroup{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupRepository) UpdateSubscriptionGroup(ctx context.Context, req *pb.UpdateSubscriptionGroupRequest) (*pb.UpdateSubscriptionGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription group: %w", err)
	}
	item, err := subscriptionGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSubscriptionGroupResponse{Data: []*pb.SubscriptionGroup{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupRepository) DeleteSubscriptionGroup(ctx context.Context, req *pb.DeleteSubscriptionGroupRequest) (*pb.DeleteSubscriptionGroupResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription group: %w", err)
	}
	return &pb.DeleteSubscriptionGroupResponse{Success: true}, nil
}

func (r *PostgresSubscriptionGroupRepository) ListSubscriptionGroups(ctx context.Context, req *pb.ListSubscriptionGroupsRequest) (*pb.ListSubscriptionGroupsResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupsResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupRepository) GetSubscriptionGroupListPageData(ctx context.Context, req *pb.GetSubscriptionGroupListPageDataRequest) (*pb.GetSubscriptionGroupListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginateSubscriptionGroup(all, req.GetPagination())
	return &pb.GetSubscriptionGroupListPageDataResponse{SubscriptionGroupList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresSubscriptionGroupRepository) GetSubscriptionGroupItemPageData(ctx context.Context, req *pb.GetSubscriptionGroupItemPageDataRequest) (*pb.GetSubscriptionGroupItemPageDataResponse, error) {
	if req == nil || req.SubscriptionGroupId == "" {
		return nil, fmt.Errorf("subscription group ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SubscriptionGroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group: %w", err)
	}
	item, err := subscriptionGroupFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetSubscriptionGroupItemPageDataResponse{SubscriptionGroup: item, Success: true}, nil
}

func (r *PostgresSubscriptionGroupRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.SubscriptionGroup, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription groups: %w", err)
	}
	var items []*pb.SubscriptionGroup
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.SubscriptionGroup{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func subscriptionGroupFromResult(result any) (*pb.SubscriptionGroup, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.SubscriptionGroup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateSubscriptionGroup(all []*pb.SubscriptionGroup, p *commonpb.PaginationRequest) (int32, []*pb.SubscriptionGroup, *commonpb.PaginationResponse) {
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
