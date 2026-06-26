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
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListPriceScheduleWorkspaceUsersResponse{Data: items, Success: true}, nil
}

func (r *PostgresPriceScheduleWorkspaceUserRepository) GetPriceScheduleWorkspaceUserListPageData(ctx context.Context, req *pb.GetPriceScheduleWorkspaceUserListPageDataRequest) (*pb.GetPriceScheduleWorkspaceUserListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginatePriceScheduleWorkspaceUser(all, req.GetPagination())
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

func (r *PostgresPriceScheduleWorkspaceUserRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.PriceScheduleWorkspaceUser, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list price schedule workspace users: %w", err)
	}
	var items []*pb.PriceScheduleWorkspaceUser
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.PriceScheduleWorkspaceUser{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
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

func paginatePriceScheduleWorkspaceUser(all []*pb.PriceScheduleWorkspaceUser, p *commonpb.PaginationRequest) (int32, []*pb.PriceScheduleWorkspaceUser, *commonpb.PaginationResponse) {
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
