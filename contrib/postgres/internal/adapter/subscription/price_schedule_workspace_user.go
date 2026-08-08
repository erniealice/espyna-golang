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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule_workspace_user"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PriceScheduleWorkspaceUser, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres price_schedule_workspace_user repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPriceScheduleWorkspaceUserRepository(dbOps, tableName), nil
	})
}

type PostgresPriceScheduleWorkspaceUserRepository struct {
	pb.UnimplementedPriceScheduleWorkspaceUserDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresPriceScheduleWorkspaceUserRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.PriceScheduleWorkspaceUserDomainServiceServer {
	if tableName == "" {
		tableName = "price_schedule_workspace_user"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresPriceScheduleWorkspaceUserRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) CreatePriceScheduleWorkspaceUser(ctx context.Context, req *pb.CreatePriceScheduleWorkspaceUserRequest) (*pb.CreatePriceScheduleWorkspaceUserResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price schedule workspace user data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price schedule workspace user: %w", err)
	}
	item, err := priceScheduleWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreatePriceScheduleWorkspaceUserResponse{Data: []*pb.PriceScheduleWorkspaceUser{item}, Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) ReadPriceScheduleWorkspaceUser(ctx context.Context, req *pb.ReadPriceScheduleWorkspaceUserRequest) (*pb.ReadPriceScheduleWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule workspace user ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price schedule workspace user: %w", err)
	}
	item, err := priceScheduleWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadPriceScheduleWorkspaceUserResponse{Data: []*pb.PriceScheduleWorkspaceUser{item}, Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) UpdatePriceScheduleWorkspaceUser(ctx context.Context, req *pb.UpdatePriceScheduleWorkspaceUserRequest) (*pb.UpdatePriceScheduleWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule workspace user ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price schedule workspace user: %w", err)
	}
	item, err := priceScheduleWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePriceScheduleWorkspaceUserResponse{Data: []*pb.PriceScheduleWorkspaceUser{item}, Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) DeletePriceScheduleWorkspaceUser(ctx context.Context, req *pb.DeletePriceScheduleWorkspaceUserRequest) (*pb.DeletePriceScheduleWorkspaceUserResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule workspace user ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete price schedule workspace user: %w", err)
	}
	return &pb.DeletePriceScheduleWorkspaceUserResponse{Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) ListPriceScheduleWorkspaceUsers(ctx context.Context, req *pb.ListPriceScheduleWorkspaceUsersRequest) (*pb.ListPriceScheduleWorkspaceUsersResponse, error) {
	var params *interfaces.ListParams
	if req.GetFilters() != nil {
		params = &interfaces.ListParams{Filters: req.GetFilters()}
	}
	items, _, err := r.listAll(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.ListPriceScheduleWorkspaceUsersResponse{Data: items, Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) GetPriceScheduleWorkspaceUserListPageData(ctx context.Context, req *pb.GetPriceScheduleWorkspaceUserListPageDataRequest) (*pb.GetPriceScheduleWorkspaceUserListPageDataResponse, error) {
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
	return &pb.GetPriceScheduleWorkspaceUserListPageDataResponse{PriceScheduleWorkspaceUserList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) GetPriceScheduleWorkspaceUserItemPageData(ctx context.Context, req *pb.GetPriceScheduleWorkspaceUserItemPageDataRequest) (*pb.GetPriceScheduleWorkspaceUserItemPageDataResponse, error) {
	if req == nil || req.PriceScheduleWorkspaceUserId == "" {
		return nil, fmt.Errorf("price schedule workspace user ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PriceScheduleWorkspaceUserId)
	if err != nil {
		return nil, fmt.Errorf("failed to read price schedule workspace user: %w", err)
	}
	item, err := priceScheduleWorkspaceUserFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetPriceScheduleWorkspaceUserItemPageDataResponse{PriceScheduleWorkspaceUser: item, Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) listAll(ctx context.Context, params *interfaces.ListParams) ([]*pb.PriceScheduleWorkspaceUser, *commonpb.PaginationResponse, error) {
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list price schedule workspace users: %w", err)
	}
	var items []*pb.PriceScheduleWorkspaceUser
	for i, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal price_schedule_workspace_user row %d: %w", i, err)
		}
		item := &pb.PriceScheduleWorkspaceUser{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal price_schedule_workspace_user row %d to proto: %w", i, err)
		}
		items = append(items, item)
	}
	return items, listResult.Pagination, nil
}

func priceScheduleWorkspaceUserFromResult(result any) (*pb.PriceScheduleWorkspaceUser, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.PriceScheduleWorkspaceUser{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}
