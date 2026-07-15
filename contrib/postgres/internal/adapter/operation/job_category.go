//go:build postgresql

package operation

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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobCategory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobCategoryRepository(dbOps, tableName), nil
	})
}

// PostgresJobCategoryRepository implements job_category CRUD via PostgreSQL.
// PageData reads delegate to the workspace-aware dbOps decorator, which scopes
// every list/read by the request context's workspace_id (the table carries its
// own workspace_id column → the decorator's direct-column path applies).
type PostgresJobCategoryRepository struct {
	pb.UnimplementedJobCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresJobCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "job_category"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJobCategoryRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresJobCategoryRepository) CreateJobCategory(ctx context.Context, req *pb.CreateJobCategoryRequest) (*pb.CreateJobCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job category data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job category: %w", err)
	}
	item, err := jobCategoryFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateJobCategoryResponse{Data: []*pb.JobCategory{item}, Success: true}, nil
}

func (r *PostgresJobCategoryRepository) ReadJobCategory(ctx context.Context, req *pb.ReadJobCategoryRequest) (*pb.ReadJobCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job category ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job category: %w", err)
	}
	item, err := jobCategoryFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadJobCategoryResponse{Data: []*pb.JobCategory{item}, Success: true}, nil
}

func (r *PostgresJobCategoryRepository) UpdateJobCategory(ctx context.Context, req *pb.UpdateJobCategoryRequest) (*pb.UpdateJobCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job category ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job category: %w", err)
	}
	item, err := jobCategoryFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateJobCategoryResponse{Data: []*pb.JobCategory{item}, Success: true}, nil
}

func (r *PostgresJobCategoryRepository) DeleteJobCategory(ctx context.Context, req *pb.DeleteJobCategoryRequest) (*pb.DeleteJobCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job category ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job category: %w", err)
	}
	return &pb.DeleteJobCategoryResponse{Success: true}, nil
}

// ListJobCategories lists job_category records with optional filters + pagination.
//
// Pagination pass-through (mirrors ListTaskOutcomes, task_outcome.go): the
// caller's req.Pagination MUST be forwarded into ListParams. Dropping it forces
// the generic core List onto its default page at offset 0 on EVERY call, so a
// paging caller (the classes-list tab builder) would re-read the same first page
// and never advance. Callers that omit both Filters and Pagination keep the
// prior nil-params behavior byte-for-byte.
func (r *PostgresJobCategoryRepository) ListJobCategories(ctx context.Context, req *pb.ListJobCategoriesRequest) (*pb.ListJobCategoriesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil) {
		params = &interfaces.ListParams{Filters: req.Filters, Pagination: req.Pagination}
	}
	items, err := r.listWith(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.ListJobCategoriesResponse{Data: items, Success: true}, nil
}

func (r *PostgresJobCategoryRepository) GetJobCategoryListPageData(ctx context.Context, req *pb.GetJobCategoryListPageDataRequest) (*pb.GetJobCategoryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	var params *interfaces.ListParams
	if req.GetFilters() != nil {
		params = &interfaces.ListParams{Filters: req.GetFilters()}
	}
	all, err := r.listWith(ctx, params)
	if err != nil {
		return nil, err
	}
	_, items, pagination := paginateJobCategory(all, req.GetPagination())
	return &pb.GetJobCategoryListPageDataResponse{JobCategoryList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresJobCategoryRepository) GetJobCategoryItemPageData(ctx context.Context, req *pb.GetJobCategoryItemPageDataRequest) (*pb.GetJobCategoryItemPageDataResponse, error) {
	if req == nil || req.JobCategoryId == "" {
		return nil, fmt.Errorf("job category ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.JobCategoryId)
	if err != nil {
		return nil, fmt.Errorf("failed to read job category: %w", err)
	}
	item, err := jobCategoryFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetJobCategoryItemPageDataResponse{JobCategory: item, Success: true}, nil
}

func (r *PostgresJobCategoryRepository) listWith(ctx context.Context, params *interfaces.ListParams) ([]*pb.JobCategory, error) {
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job categories: %w", err)
	}
	var items []*pb.JobCategory
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.JobCategory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func jobCategoryFromResult(result any) (*pb.JobCategory, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.JobCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateJobCategory(all []*pb.JobCategory, p *commonpb.PaginationRequest) (int32, []*pb.JobCategory, *commonpb.PaginationResponse) {
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
