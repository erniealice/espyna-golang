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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionGroupWorkspaceUser, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_group_workspace_user repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionGroupWorkspaceUserRepository(dbOps, tableName), nil
	})
}

type PostgresSubscriptionGroupWorkspaceUserRepository struct {
	pb.UnimplementedSubscriptionGroupWorkspaceUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresSubscriptionGroupWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SubscriptionGroupWorkspaceUserDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_group_workspace_user"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSubscriptionGroupWorkspaceUserRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) CreateSubscriptionGroupWorkspaceUser(ctx context.Context, req *pb.CreateSubscriptionGroupWorkspaceUserRequest) (*pb.CreateSubscriptionGroupWorkspaceUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription group workspace user data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription group workspace user: %w", err)
	}
	item, err := subscriptionGroupWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupWorkspaceUserResponse{Data: []*pb.SubscriptionGroupWorkspaceUser{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) ReadSubscriptionGroupWorkspaceUser(ctx context.Context, req *pb.ReadSubscriptionGroupWorkspaceUserRequest) (*pb.ReadSubscriptionGroupWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group workspace user ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group workspace user: %w", err)
	}
	item, err := subscriptionGroupWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadSubscriptionGroupWorkspaceUserResponse{Data: []*pb.SubscriptionGroupWorkspaceUser{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) UpdateSubscriptionGroupWorkspaceUser(ctx context.Context, req *pb.UpdateSubscriptionGroupWorkspaceUserRequest) (*pb.UpdateSubscriptionGroupWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group workspace user ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription group workspace user: %w", err)
	}
	item, err := subscriptionGroupWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSubscriptionGroupWorkspaceUserResponse{Data: []*pb.SubscriptionGroupWorkspaceUser{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) DeleteSubscriptionGroupWorkspaceUser(ctx context.Context, req *pb.DeleteSubscriptionGroupWorkspaceUserRequest) (*pb.DeleteSubscriptionGroupWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group workspace user ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription group workspace user: %w", err)
	}
	return &pb.DeleteSubscriptionGroupWorkspaceUserResponse{Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) ListSubscriptionGroupWorkspaceUsers(ctx context.Context, req *pb.ListSubscriptionGroupWorkspaceUsersRequest) (*pb.ListSubscriptionGroupWorkspaceUsersResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupWorkspaceUsersResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) GetSubscriptionGroupWorkspaceUserListPageData(ctx context.Context, req *pb.GetSubscriptionGroupWorkspaceUserListPageDataRequest) (*pb.GetSubscriptionGroupWorkspaceUserListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginateSubscriptionGroupWorkspaceUser(all, req.GetPagination())
	return &pb.GetSubscriptionGroupWorkspaceUserListPageDataResponse{SubscriptionGroupWorkspaceUserList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) GetSubscriptionGroupWorkspaceUserItemPageData(ctx context.Context, req *pb.GetSubscriptionGroupWorkspaceUserItemPageDataRequest) (*pb.GetSubscriptionGroupWorkspaceUserItemPageDataResponse, error) {
	if req == nil || req.SubscriptionGroupWorkspaceUserId == "" {
		return nil, fmt.Errorf("subscription group workspace user ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SubscriptionGroupWorkspaceUserId)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group workspace user: %w", err)
	}
	item, err := subscriptionGroupWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetSubscriptionGroupWorkspaceUserItemPageDataResponse{SubscriptionGroupWorkspaceUser: item, Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.SubscriptionGroupWorkspaceUser, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription group workspace users: %w", err)
	}
	var items []*pb.SubscriptionGroupWorkspaceUser
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.SubscriptionGroupWorkspaceUser{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func subscriptionGroupWorkspaceUserFromResult(result any) (*pb.SubscriptionGroupWorkspaceUser, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.SubscriptionGroupWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateSubscriptionGroupWorkspaceUser(all []*pb.SubscriptionGroupWorkspaceUser, p *commonpb.PaginationRequest) (int32, []*pb.SubscriptionGroupWorkspaceUser, *commonpb.PaginationResponse) {
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
