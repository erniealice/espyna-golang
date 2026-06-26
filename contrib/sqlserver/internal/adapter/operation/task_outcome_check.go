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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TaskOutcomeCheck, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver task_outcome_check repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTaskOutcomeCheckRepository(dbOps, tableName), nil
	})
}

// SQLServerTaskOutcomeCheckRepository implements task_outcome_check CRUD operations using SQL Server.
type SQLServerTaskOutcomeCheckRepository struct {
	pb.UnimplementedTaskOutcomeCheckDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerTaskOutcomeCheckRepository creates a new SQL Server task_outcome_check repository.
func NewSQLServerTaskOutcomeCheckRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TaskOutcomeCheckDomainServiceServer {
	if tableName == "" {
		tableName = "task_outcome_check"
	}
	return &SQLServerTaskOutcomeCheckRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerTaskOutcomeCheckRepository) CreateTaskOutcomeCheck(ctx context.Context, req *pb.CreateTaskOutcomeCheckRequest) (*pb.CreateTaskOutcomeCheckResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("task_outcome_check data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_outcome_check: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	toc := &pb.TaskOutcomeCheck{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, toc) //nolint:errcheck
	return &pb.CreateTaskOutcomeCheckResponse{Success: true, Data: []*pb.TaskOutcomeCheck{toc}}, nil
}

func (r *SQLServerTaskOutcomeCheckRepository) ReadTaskOutcomeCheck(ctx context.Context, req *pb.ReadTaskOutcomeCheckRequest) (*pb.ReadTaskOutcomeCheckResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task_outcome_check ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read task_outcome_check: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	toc := &pb.TaskOutcomeCheck{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, toc) //nolint:errcheck
	return &pb.ReadTaskOutcomeCheckResponse{Success: true, Data: []*pb.TaskOutcomeCheck{toc}}, nil
}

func (r *SQLServerTaskOutcomeCheckRepository) UpdateTaskOutcomeCheck(ctx context.Context, req *pb.UpdateTaskOutcomeCheckRequest) (*pb.UpdateTaskOutcomeCheckResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task_outcome_check ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update task_outcome_check: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	toc := &pb.TaskOutcomeCheck{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, toc) //nolint:errcheck
	return &pb.UpdateTaskOutcomeCheckResponse{Success: true, Data: []*pb.TaskOutcomeCheck{toc}}, nil
}

func (r *SQLServerTaskOutcomeCheckRepository) DeleteTaskOutcomeCheck(ctx context.Context, req *pb.DeleteTaskOutcomeCheckRequest) (*pb.DeleteTaskOutcomeCheckResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task_outcome_check ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete task_outcome_check: %w", err)
	}
	return &pb.DeleteTaskOutcomeCheckResponse{Success: true}, nil
}

func (r *SQLServerTaskOutcomeCheckRepository) ListTaskOutcomeChecks(ctx context.Context, req *pb.ListTaskOutcomeChecksRequest) (*pb.ListTaskOutcomeChecksResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list task_outcome_checks: %w", err)
	}
	var items []*pb.TaskOutcomeCheck
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal task_outcome_check row: %v", err)
			continue
		}
		toc := &pb.TaskOutcomeCheck{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, toc); err != nil {
			log.Printf("WARN: protojson unmarshal task_outcome_check: %v", err)
			continue
		}
		items = append(items, toc)
	}
	return &pb.ListTaskOutcomeChecksResponse{Success: true, Data: items}, nil
}

func (r *SQLServerTaskOutcomeCheckRepository) GetTaskOutcomeCheckListPageData(ctx context.Context, req *pb.GetTaskOutcomeCheckListPageDataRequest) (*pb.GetTaskOutcomeCheckListPageDataResponse, error) {

	return nil, fmt.Errorf("GetTaskOutcomeCheckListPageData not yet implemented")
}

func (r *SQLServerTaskOutcomeCheckRepository) GetTaskOutcomeCheckItemPageData(ctx context.Context, req *pb.GetTaskOutcomeCheckItemPageDataRequest) (*pb.GetTaskOutcomeCheckItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetTaskOutcomeCheckItemPageData not yet implemented")
}
