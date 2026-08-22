//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
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
	var params *interfaces.ListParams
	if req.GetFilters() != nil {
		params = &interfaces.ListParams{Filters: req.GetFilters()}
	}
	items, _, err := r.listAll(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupMembersResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupMemberRepository) GetSubscriptionGroupMemberListPageData(ctx context.Context, req *pb.GetSubscriptionGroupMemberListPageDataRequest) (*pb.GetSubscriptionGroupMemberListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	params := &interfaces.ListParams{
		Search:     req.GetSearch(),
		Filters:    req.GetFilters(),
		Sort:       req.GetSort(),
		Pagination: req.GetPagination(),
	}
	items, pagination, err := r.listAll(ctx, params)
	if err != nil {
		return nil, err
	}
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

func (r *PostgresSubscriptionGroupMemberRepository) listAll(ctx context.Context, params *interfaces.ListParams) ([]*pb.SubscriptionGroupMember, *commonpb.PaginationResponse, error) {
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list subscription group members: %w", err)
	}
	var items []*pb.SubscriptionGroupMember
	for i, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal subscription_group_member row %d: %w", i, err)
		}
		item := &pb.SubscriptionGroupMember{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal subscription_group_member row %d to proto: %w", i, err)
		}
		items = append(items, item)
	}
	return items, listResult.Pagination, nil
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
