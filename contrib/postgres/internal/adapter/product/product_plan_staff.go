//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ProductPlanStaff, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_plan_staff repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresProductPlanStaffRepository(dbOps, tableName), nil
	})
}

type PostgresProductPlanStaffRepository struct {
	pb.UnimplementedProductPlanStaffDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresProductPlanStaffRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ProductPlanStaffDomainServiceServer {
	if tableName == "" {
		tableName = "product_plan_staff"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresProductPlanStaffRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresProductPlanStaffRepository) CreateProductPlanStaff(ctx context.Context, req *pb.CreateProductPlanStaffRequest) (*pb.CreateProductPlanStaffResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product plan staff data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create product plan staff: %w", err)
	}
	item, err := productPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateProductPlanStaffResponse{Data: []*pb.ProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresProductPlanStaffRepository) ReadProductPlanStaff(ctx context.Context, req *pb.ReadProductPlanStaffRequest) (*pb.ReadProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan staff ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product plan staff: %w", err)
	}
	item, err := productPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadProductPlanStaffResponse{Data: []*pb.ProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresProductPlanStaffRepository) UpdateProductPlanStaff(ctx context.Context, req *pb.UpdateProductPlanStaffRequest) (*pb.UpdateProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan staff ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product plan staff: %w", err)
	}
	item, err := productPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateProductPlanStaffResponse{Data: []*pb.ProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresProductPlanStaffRepository) DeleteProductPlanStaff(ctx context.Context, req *pb.DeleteProductPlanStaffRequest) (*pb.DeleteProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan staff ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product plan staff: %w", err)
	}
	return &pb.DeleteProductPlanStaffResponse{Success: true}, nil
}

func (r *PostgresProductPlanStaffRepository) ListProductPlanStaffs(ctx context.Context, req *pb.ListProductPlanStaffsRequest) (*pb.ListProductPlanStaffsResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListProductPlanStaffsResponse{Data: items, Success: true}, nil
}

func (r *PostgresProductPlanStaffRepository) GetProductPlanStaffListPageData(ctx context.Context, req *pb.GetProductPlanStaffListPageDataRequest) (*pb.GetProductPlanStaffListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginateProductPlanStaff(all, req.GetPagination())
	return &pb.GetProductPlanStaffListPageDataResponse{ProductPlanStaffList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresProductPlanStaffRepository) GetProductPlanStaffItemPageData(ctx context.Context, req *pb.GetProductPlanStaffItemPageDataRequest) (*pb.GetProductPlanStaffItemPageDataResponse, error) {
	if req == nil || req.ProductPlanStaffId == "" {
		return nil, fmt.Errorf("product plan staff ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ProductPlanStaffId)
	if err != nil {
		return nil, fmt.Errorf("failed to read product plan staff: %w", err)
	}
	item, err := productPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetProductPlanStaffItemPageDataResponse{ProductPlanStaff: item, Success: true}, nil
}

func (r *PostgresProductPlanStaffRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ProductPlanStaff, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product plan staffs: %w", err)
	}
	var items []*pb.ProductPlanStaff
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ProductPlanStaff{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func productPlanStaffFromResult(result any) (*pb.ProductPlanStaff, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ProductPlanStaff{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateProductPlanStaff(all []*pb.ProductPlanStaff, p *commonpb.PaginationRequest) (int32, []*pb.ProductPlanStaff, *commonpb.PaginationResponse) {
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
