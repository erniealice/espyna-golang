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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/reporting_checkpoint"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ReportingCheckpoint, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres reporting_checkpoint repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresReportingCheckpointRepository(dbOps, tableName), nil
	})
}

// PostgresReportingCheckpointRepository implements reporting_checkpoint CRUD via PostgreSQL.
type PostgresReportingCheckpointRepository struct {
	pb.UnimplementedReportingCheckpointDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresReportingCheckpointRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ReportingCheckpointDomainServiceServer {
	if tableName == "" {
		tableName = "reporting_checkpoint"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresReportingCheckpointRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresReportingCheckpointRepository) CreateReportingCheckpoint(ctx context.Context, req *pb.CreateReportingCheckpointRequest) (*pb.CreateReportingCheckpointResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("reporting checkpoint data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create reporting checkpoint: %w", err)
	}
	item, err := reportingCheckpointFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateReportingCheckpointResponse{Data: []*pb.ReportingCheckpoint{item}, Success: true}, nil
}

func (r *PostgresReportingCheckpointRepository) ReadReportingCheckpoint(ctx context.Context, req *pb.ReadReportingCheckpointRequest) (*pb.ReadReportingCheckpointResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("reporting checkpoint ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read reporting checkpoint: %w", err)
	}
	item, err := reportingCheckpointFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadReportingCheckpointResponse{Data: []*pb.ReportingCheckpoint{item}, Success: true}, nil
}

func (r *PostgresReportingCheckpointRepository) UpdateReportingCheckpoint(ctx context.Context, req *pb.UpdateReportingCheckpointRequest) (*pb.UpdateReportingCheckpointResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("reporting checkpoint ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update reporting checkpoint: %w", err)
	}
	item, err := reportingCheckpointFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateReportingCheckpointResponse{Data: []*pb.ReportingCheckpoint{item}, Success: true}, nil
}

func (r *PostgresReportingCheckpointRepository) DeleteReportingCheckpoint(ctx context.Context, req *pb.DeleteReportingCheckpointRequest) (*pb.DeleteReportingCheckpointResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("reporting checkpoint ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete reporting checkpoint: %w", err)
	}
	return &pb.DeleteReportingCheckpointResponse{Success: true}, nil
}

func (r *PostgresReportingCheckpointRepository) ListReportingCheckpoints(ctx context.Context, req *pb.ListReportingCheckpointsRequest) (*pb.ListReportingCheckpointsResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListReportingCheckpointsResponse{Data: items, Success: true}, nil
}

func (r *PostgresReportingCheckpointRepository) GetReportingCheckpointListPageData(ctx context.Context, req *pb.GetReportingCheckpointListPageDataRequest) (*pb.GetReportingCheckpointListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateReportingCheckpoint(all, req.GetPagination())
	_ = page
	return &pb.GetReportingCheckpointListPageDataResponse{ReportingCheckpointList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresReportingCheckpointRepository) GetReportingCheckpointItemPageData(ctx context.Context, req *pb.GetReportingCheckpointItemPageDataRequest) (*pb.GetReportingCheckpointItemPageDataResponse, error) {
	if req == nil || req.ReportingCheckpointId == "" {
		return nil, fmt.Errorf("reporting checkpoint ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ReportingCheckpointId)
	if err != nil {
		return nil, fmt.Errorf("failed to read reporting checkpoint: %w", err)
	}
	item, err := reportingCheckpointFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetReportingCheckpointItemPageDataResponse{ReportingCheckpoint: item, Success: true}, nil
}

func (r *PostgresReportingCheckpointRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ReportingCheckpoint, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list reporting checkpoints: %w", err)
	}
	var items []*pb.ReportingCheckpoint
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ReportingCheckpoint{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func reportingCheckpointFromResult(result any) (*pb.ReportingCheckpoint, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ReportingCheckpoint{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateReportingCheckpoint(all []*pb.ReportingCheckpoint, p *commonpb.PaginationRequest) (int32, []*pb.ReportingCheckpoint, *commonpb.PaginationResponse) {
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
