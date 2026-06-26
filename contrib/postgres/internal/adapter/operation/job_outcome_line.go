//go:build postgresql

package operation

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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobOutcomeLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_outcome_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobOutcomeLineRepository(dbOps, tableName), nil
	})
}

// PostgresJobOutcomeLineRepository implements job_outcome_line CRUD via PostgreSQL.
type PostgresJobOutcomeLineRepository struct {
	pb.UnimplementedJobOutcomeLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresJobOutcomeLineRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobOutcomeLineDomainServiceServer {
	if tableName == "" {
		tableName = "job_outcome_line"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJobOutcomeLineRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresJobOutcomeLineRepository) CreateJobOutcomeLine(ctx context.Context, req *pb.CreateJobOutcomeLineRequest) (*pb.CreateJobOutcomeLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job outcome line data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job outcome line: %w", err)
	}
	item, err := jobOutcomeLineFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateJobOutcomeLineResponse{Data: []*pb.JobOutcomeLine{item}, Success: true}, nil
}

func (r *PostgresJobOutcomeLineRepository) ReadJobOutcomeLine(ctx context.Context, req *pb.ReadJobOutcomeLineRequest) (*pb.ReadJobOutcomeLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job outcome line: %w", err)
	}
	item, err := jobOutcomeLineFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadJobOutcomeLineResponse{Data: []*pb.JobOutcomeLine{item}, Success: true}, nil
}

func (r *PostgresJobOutcomeLineRepository) UpdateJobOutcomeLine(ctx context.Context, req *pb.UpdateJobOutcomeLineRequest) (*pb.UpdateJobOutcomeLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome line ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job outcome line: %w", err)
	}
	item, err := jobOutcomeLineFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateJobOutcomeLineResponse{Data: []*pb.JobOutcomeLine{item}, Success: true}, nil
}

func (r *PostgresJobOutcomeLineRepository) DeleteJobOutcomeLine(ctx context.Context, req *pb.DeleteJobOutcomeLineRequest) (*pb.DeleteJobOutcomeLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job outcome line: %w", err)
	}
	return &pb.DeleteJobOutcomeLineResponse{Success: true}, nil
}

func (r *PostgresJobOutcomeLineRepository) ListJobOutcomeLines(ctx context.Context, req *pb.ListJobOutcomeLinesRequest) (*pb.ListJobOutcomeLinesResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListJobOutcomeLinesResponse{Data: items, Success: true}, nil
}

func (r *PostgresJobOutcomeLineRepository) GetJobOutcomeLineListPageData(ctx context.Context, req *pb.GetJobOutcomeLineListPageDataRequest) (*pb.GetJobOutcomeLineListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateJobOutcomeLine(all, req.GetPagination())
	_ = page
	return &pb.GetJobOutcomeLineListPageDataResponse{JobOutcomeLineList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresJobOutcomeLineRepository) GetJobOutcomeLineItemPageData(ctx context.Context, req *pb.GetJobOutcomeLineItemPageDataRequest) (*pb.GetJobOutcomeLineItemPageDataResponse, error) {
	if req == nil || req.JobOutcomeLineId == "" {
		return nil, fmt.Errorf("job outcome line ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.JobOutcomeLineId)
	if err != nil {
		return nil, fmt.Errorf("failed to read job outcome line: %w", err)
	}
	item, err := jobOutcomeLineFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetJobOutcomeLineItemPageDataResponse{JobOutcomeLine: item, Success: true}, nil
}

func (r *PostgresJobOutcomeLineRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.JobOutcomeLine, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job outcome lines: %w", err)
	}
	var items []*pb.JobOutcomeLine
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.JobOutcomeLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func jobOutcomeLineFromResult(result any) (*pb.JobOutcomeLine, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.JobOutcomeLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateJobOutcomeLine(all []*pb.JobOutcomeLine, p *commonpb.PaginationRequest) (int32, []*pb.JobOutcomeLine, *commonpb.PaginationResponse) {
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
