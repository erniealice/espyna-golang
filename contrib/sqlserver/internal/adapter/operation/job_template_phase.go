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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobTemplatePhase, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_template_phase repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobTemplatePhaseRepository(dbOps, tableName), nil
	})
}

// SQLServerJobTemplatePhaseRepository implements job_template_phase CRUD operations using SQL Server.
type SQLServerJobTemplatePhaseRepository struct {
	pb.UnimplementedJobTemplatePhaseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerJobTemplatePhaseRepository creates a new SQL Server job_template_phase repository.
func NewSQLServerJobTemplatePhaseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplatePhaseDomainServiceServer {
	if tableName == "" {
		tableName = "job_template_phase"
	}
	return &SQLServerJobTemplatePhaseRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerJobTemplatePhaseRepository) CreateJobTemplatePhase(ctx context.Context, req *pb.CreateJobTemplatePhaseRequest) (*pb.CreateJobTemplatePhaseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job_template_phase data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template_phase: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	p := &pb.JobTemplatePhase{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, p) //nolint:errcheck
	return &pb.CreateJobTemplatePhaseResponse{Success: true, Data: []*pb.JobTemplatePhase{p}}, nil
}

func (r *SQLServerJobTemplatePhaseRepository) ReadJobTemplatePhase(ctx context.Context, req *pb.ReadJobTemplatePhaseRequest) (*pb.ReadJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_phase ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job_template_phase: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	p := &pb.JobTemplatePhase{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, p) //nolint:errcheck
	return &pb.ReadJobTemplatePhaseResponse{Success: true, Data: []*pb.JobTemplatePhase{p}}, nil
}

func (r *SQLServerJobTemplatePhaseRepository) UpdateJobTemplatePhase(ctx context.Context, req *pb.UpdateJobTemplatePhaseRequest) (*pb.UpdateJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_phase ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job_template_phase: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	p := &pb.JobTemplatePhase{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, p) //nolint:errcheck
	return &pb.UpdateJobTemplatePhaseResponse{Success: true, Data: []*pb.JobTemplatePhase{p}}, nil
}

func (r *SQLServerJobTemplatePhaseRepository) DeleteJobTemplatePhase(ctx context.Context, req *pb.DeleteJobTemplatePhaseRequest) (*pb.DeleteJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_phase ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job_template_phase: %w", err)
	}
	return &pb.DeleteJobTemplatePhaseResponse{Success: true}, nil
}

func (r *SQLServerJobTemplatePhaseRepository) ListJobTemplatePhases(ctx context.Context, req *pb.ListJobTemplatePhasesRequest) (*pb.ListJobTemplatePhasesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job_template_phases: %w", err)
	}
	var items []*pb.JobTemplatePhase
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job_template_phase row: %v", err)
			continue
		}
		p := &pb.JobTemplatePhase{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, p); err != nil {
			log.Printf("WARN: protojson unmarshal job_template_phase: %v", err)
			continue
		}
		items = append(items, p)
	}
	return &pb.ListJobTemplatePhasesResponse{Success: true, Data: items}, nil
}

func (r *SQLServerJobTemplatePhaseRepository) GetJobTemplatePhaseListPageData(ctx context.Context, req *pb.GetJobTemplatePhaseListPageDataRequest) (*pb.GetJobTemplatePhaseListPageDataResponse, error) {

	return nil, fmt.Errorf("GetJobTemplatePhaseListPageData not yet implemented")
}

func (r *SQLServerJobTemplatePhaseRepository) GetJobTemplatePhaseItemPageData(ctx context.Context, req *pb.GetJobTemplatePhaseItemPageDataRequest) (*pb.GetJobTemplatePhaseItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJobTemplatePhaseItemPageData not yet implemented")
}
