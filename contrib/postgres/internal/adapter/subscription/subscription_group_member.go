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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionGroupMember, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_group_member repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionGroupMemberRepository(dbOps, tableName), nil
	})
}

type PostgresSubscriptionGroupMemberRepository struct {
	pb.UnimplementedSubscriptionGroupMemberDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresSubscriptionGroupMemberRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SubscriptionGroupMemberDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_group_member"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSubscriptionGroupMemberRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresSubscriptionGroupMemberRepository) CreateSubscriptionGroupMember(ctx context.Context, req *pb.CreateSubscriptionGroupMemberRequest) (*pb.CreateSubscriptionGroupMemberResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription group member data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription group member: %w", err)
	}
	item, err := subscriptionGroupMemberFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupMemberResponse{Data: []*pb.SubscriptionGroupMember{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) ReadSubscriptionGroupMember(ctx context.Context, req *pb.ReadSubscriptionGroupMemberRequest) (*pb.ReadSubscriptionGroupMemberResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group member ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group member: %w", err)
	}
	item, err := subscriptionGroupMemberFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadSubscriptionGroupMemberResponse{Data: []*pb.SubscriptionGroupMember{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) UpdateSubscriptionGroupMember(ctx context.Context, req *pb.UpdateSubscriptionGroupMemberRequest) (*pb.UpdateSubscriptionGroupMemberResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group member ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription group member: %w", err)
	}
	item, err := subscriptionGroupMemberFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSubscriptionGroupMemberResponse{Data: []*pb.SubscriptionGroupMember{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) DeleteSubscriptionGroupMember(ctx context.Context, req *pb.DeleteSubscriptionGroupMemberRequest) (*pb.DeleteSubscriptionGroupMemberResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group member ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription group member: %w", err)
	}
	return &pb.DeleteSubscriptionGroupMemberResponse{Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) ListSubscriptionGroupMembers(ctx context.Context, req *pb.ListSubscriptionGroupMembersRequest) (*pb.ListSubscriptionGroupMembersResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupMembersResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) GetSubscriptionGroupMemberListPageData(ctx context.Context, req *pb.GetSubscriptionGroupMemberListPageDataRequest) (*pb.GetSubscriptionGroupMemberListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginateSubscriptionGroupMember(all, req.GetPagination())
	return &pb.GetSubscriptionGroupMemberListPageDataResponse{SubscriptionGroupMemberList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) GetSubscriptionGroupMemberItemPageData(ctx context.Context, req *pb.GetSubscriptionGroupMemberItemPageDataRequest) (*pb.GetSubscriptionGroupMemberItemPageDataResponse, error) {
	if req == nil || req.SubscriptionGroupMemberId == "" {
		return nil, fmt.Errorf("subscription group member ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SubscriptionGroupMemberId)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group member: %w", err)
	}
	item, err := subscriptionGroupMemberFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetSubscriptionGroupMemberItemPageDataResponse{SubscriptionGroupMember: item, Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.SubscriptionGroupMember, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription group members: %w", err)
	}
	var items []*pb.SubscriptionGroupMember
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.SubscriptionGroupMember{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func subscriptionGroupMemberFromResult(result any) (*pb.SubscriptionGroupMember, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.SubscriptionGroupMember{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateSubscriptionGroupMember(all []*pb.SubscriptionGroupMember, p *commonpb.PaginationRequest) (int32, []*pb.SubscriptionGroupMember, *commonpb.PaginationResponse) {
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
