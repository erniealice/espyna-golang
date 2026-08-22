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
	var params *interfaces.ListParams
	if req.GetFilters() != nil {
		params = &interfaces.ListParams{Filters: req.GetFilters()}
	}
	items, _, err := r.listAll(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupWorkspaceUsersResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) GetSubscriptionGroupWorkspaceUserListPageData(ctx context.Context, req *pb.GetSubscriptionGroupWorkspaceUserListPageDataRequest) (*pb.GetSubscriptionGroupWorkspaceUserListPageDataResponse, error) {
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

func (r *PostgresSubscriptionGroupWorkspaceUserRepository) listAll(ctx context.Context, params *interfaces.ListParams) ([]*pb.SubscriptionGroupWorkspaceUser, *commonpb.PaginationResponse, error) {
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list subscription group workspace users: %w", err)
	}
	var items []*pb.SubscriptionGroupWorkspaceUser
	for i, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal subscription_group_workspace_user row %d: %w", i, err)
		}
		item := &pb.SubscriptionGroupWorkspaceUser{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal subscription_group_workspace_user row %d to proto: %w", i, err)
		}
		items = append(items, item)
	}
	return items, listResult.Pagination, nil
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
