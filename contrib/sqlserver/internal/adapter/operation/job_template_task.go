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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobTemplateTask, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_template_task repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobTemplateTaskRepository(dbOps, tableName), nil
	})
}

// SQLServerJobTemplateTaskRepository implements job_template_task CRUD operations using SQL Server.
type SQLServerJobTemplateTaskRepository struct {
	pb.UnimplementedJobTemplateTaskDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerJobTemplateTaskRepository creates a new SQL Server job_template_task repository.
func NewSQLServerJobTemplateTaskRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplateTaskDomainServiceServer {
	if tableName == "" {
		tableName = "job_template_task"
	}
	return &SQLServerJobTemplateTaskRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerJobTemplateTaskRepository) CreateJobTemplateTask(ctx context.Context, req *pb.CreateJobTemplateTaskRequest) (*pb.CreateJobTemplateTaskResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job_template_task data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template_task: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	t := &pb.JobTemplateTask{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t) //nolint:errcheck
	return &pb.CreateJobTemplateTaskResponse{Success: true, Data: []*pb.JobTemplateTask{t}}, nil
}

func (r *SQLServerJobTemplateTaskRepository) ReadJobTemplateTask(ctx context.Context, req *pb.ReadJobTemplateTaskRequest) (*pb.ReadJobTemplateTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_task ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job_template_task: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	t := &pb.JobTemplateTask{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t) //nolint:errcheck
	return &pb.ReadJobTemplateTaskResponse{Success: true, Data: []*pb.JobTemplateTask{t}}, nil
}

func (r *SQLServerJobTemplateTaskRepository) UpdateJobTemplateTask(ctx context.Context, req *pb.UpdateJobTemplateTaskRequest) (*pb.UpdateJobTemplateTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_task ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job_template_task: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	t := &pb.JobTemplateTask{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t) //nolint:errcheck
	return &pb.UpdateJobTemplateTaskResponse{Success: true, Data: []*pb.JobTemplateTask{t}}, nil
}

func (r *SQLServerJobTemplateTaskRepository) DeleteJobTemplateTask(ctx context.Context, req *pb.DeleteJobTemplateTaskRequest) (*pb.DeleteJobTemplateTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_task ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete job_template_task: %w", err)
	}
	return &pb.DeleteJobTemplateTaskResponse{Success: true}, nil
}

func (r *SQLServerJobTemplateTaskRepository) ListJobTemplateTasks(ctx context.Context, req *pb.ListJobTemplateTasksRequest) (*pb.ListJobTemplateTasksResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job_template_tasks: %w", err)
	}
	var items []*pb.JobTemplateTask
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job_template_task row: %v", err)
			continue
		}
		t := &pb.JobTemplateTask{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, t); err != nil {
			log.Printf("WARN: protojson unmarshal job_template_task: %v", err)
			continue
		}
		items = append(items, t)
	}
	return &pb.ListJobTemplateTasksResponse{Success: true, Data: items}, nil
}

func (r *SQLServerJobTemplateTaskRepository) GetJobTemplateTaskListPageData(ctx context.Context, req *pb.GetJobTemplateTaskListPageDataRequest) (*pb.GetJobTemplateTaskListPageDataResponse, error) {

	return nil, fmt.Errorf("GetJobTemplateTaskListPageData not yet implemented")
}

func (r *SQLServerJobTemplateTaskRepository) GetJobTemplateTaskItemPageData(ctx context.Context, req *pb.GetJobTemplateTaskItemPageDataRequest) (*pb.GetJobTemplateTaskItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetJobTemplateTaskItemPageData not yet implemented")
}
