//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobOutcomeSummary, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_outcome_summary repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobOutcomeSummaryRepository(dbOps, tableName), nil
	})
}

// SQLServerJobOutcomeSummaryRepository implements job_outcome_summary CRUD using SQL Server.
type SQLServerJobOutcomeSummaryRepository struct {
	pb.UnimplementedJobOutcomeSummaryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerJobOutcomeSummaryRepository creates a new SQL Server job_outcome_summary repository.
func NewSQLServerJobOutcomeSummaryRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobOutcomeSummaryDomainServiceServer {
	if tableName == "" {
		tableName = "job_outcome_summary"
	}
	return &SQLServerJobOutcomeSummaryRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerJobOutcomeSummaryRepository) CreateJobOutcomeSummary(ctx context.Context, req *pb.CreateJobOutcomeSummaryRequest) (*pb.CreateJobOutcomeSummaryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job outcome summary data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job outcome summary: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	s := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.CreateJobOutcomeSummaryResponse{Success: true, Data: []*pb.JobOutcomeSummary{s}}, nil
}

func (r *SQLServerJobOutcomeSummaryRepository) ReadJobOutcomeSummary(ctx context.Context, req *pb.ReadJobOutcomeSummaryRequest) (*pb.ReadJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job outcome summary: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	s := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.ReadJobOutcomeSummaryResponse{Success: true, Data: []*pb.JobOutcomeSummary{s}}, nil
}

func (r *SQLServerJobOutcomeSummaryRepository) UpdateJobOutcomeSummary(ctx context.Context, req *pb.UpdateJobOutcomeSummaryRequest) (*pb.UpdateJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job outcome summary: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	s := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pb.UpdateJobOutcomeSummaryResponse{Success: true, Data: []*pb.JobOutcomeSummary{s}}, nil
}

func (r *SQLServerJobOutcomeSummaryRepository) DeleteJobOutcomeSummary(ctx context.Context, req *pb.DeleteJobOutcomeSummaryRequest) (*pb.DeleteJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job outcome summary: %w", err)
	}
	return &pb.DeleteJobOutcomeSummaryResponse{Success: true}, nil
}

func (r *SQLServerJobOutcomeSummaryRepository) ListJobOutcomeSummarys(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) (*pb.ListJobOutcomeSummarysResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job outcome summaries: %w", err)
	}
	var items []*pb.JobOutcomeSummary
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job_outcome_summary row: %v", err)
			continue
		}
		s := &pb.JobOutcomeSummary{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, s); err != nil {
			log.Printf("WARN: protojson unmarshal job_outcome_summary: %v", err)
			continue
		}
		items = append(items, s)
	}
	return &pb.ListJobOutcomeSummarysResponse{Success: true, Data: items}, nil
}

func (r *SQLServerJobOutcomeSummaryRepository) GetJobOutcomeSummaryListPageData(ctx context.Context, req *pb.GetJobOutcomeSummaryListPageDataRequest) (*pb.GetJobOutcomeSummaryListPageDataResponse, error) {
	// TODO: Implement CTE-based paginated query.
	return nil, fmt.Errorf("GetJobOutcomeSummaryListPageData not yet implemented")
}

func (r *SQLServerJobOutcomeSummaryRepository) GetJobOutcomeSummaryItemPageData(ctx context.Context, req *pb.GetJobOutcomeSummaryItemPageDataRequest) (*pb.GetJobOutcomeSummaryItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJobOutcomeSummaryItemPageData not yet implemented")
}
