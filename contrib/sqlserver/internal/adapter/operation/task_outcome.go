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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.TaskOutcome, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver task_outcome repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerTaskOutcomeRepository(dbOps, tableName), nil
	})
}

// SQLServerTaskOutcomeRepository implements task_outcome CRUD operations using SQL Server.
type SQLServerTaskOutcomeRepository struct {
	pb.UnimplementedTaskOutcomeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerTaskOutcomeRepository creates a new SQL Server task_outcome repository.
func NewSQLServerTaskOutcomeRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TaskOutcomeDomainServiceServer {
	if tableName == "" {
		tableName = "task_outcome"
	}
	return &SQLServerTaskOutcomeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerTaskOutcomeRepository) CreateTaskOutcome(ctx context.Context, req *pb.CreateTaskOutcomeRequest) (*pb.CreateTaskOutcomeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("task_outcome data is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create task_outcome: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	to := &pb.TaskOutcome{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, to) //nolint:errcheck
	return &pb.CreateTaskOutcomeResponse{Success: true, Data: []*pb.TaskOutcome{to}}, nil
}

func (r *SQLServerTaskOutcomeRepository) ReadTaskOutcome(ctx context.Context, req *pb.ReadTaskOutcomeRequest) (*pb.ReadTaskOutcomeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task_outcome ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read task_outcome: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	to := &pb.TaskOutcome{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, to) //nolint:errcheck
	return &pb.ReadTaskOutcomeResponse{Success: true, Data: []*pb.TaskOutcome{to}}, nil
}

func (r *SQLServerTaskOutcomeRepository) UpdateTaskOutcome(ctx context.Context, req *pb.UpdateTaskOutcomeRequest) (*pb.UpdateTaskOutcomeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task_outcome ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update task_outcome: %w", err)
	}
	resultJSON, _ := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	to := &pb.TaskOutcome{}
	(protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, to) //nolint:errcheck
	return &pb.UpdateTaskOutcomeResponse{Success: true, Data: []*pb.TaskOutcome{to}}, nil
}

func (r *SQLServerTaskOutcomeRepository) DeleteTaskOutcome(ctx context.Context, req *pb.DeleteTaskOutcomeRequest) (*pb.DeleteTaskOutcomeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task_outcome ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete task_outcome: %w", err)
	}
	return &pb.DeleteTaskOutcomeResponse{Success: true}, nil
}

func (r *SQLServerTaskOutcomeRepository) ListTaskOutcomes(ctx context.Context, req *pb.ListTaskOutcomesRequest) (*pb.ListTaskOutcomesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list task_outcomes: %w", err)
	}
	var items []*pb.TaskOutcome
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal task_outcome row: %v", err)
			continue
		}
		to := &pb.TaskOutcome{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, to); err != nil {
			log.Printf("WARN: protojson unmarshal task_outcome: %v", err)
			continue
		}
		items = append(items, to)
	}
	return &pb.ListTaskOutcomesResponse{Success: true, Data: items}, nil
}

func (r *SQLServerTaskOutcomeRepository) GetTaskOutcomeListPageData(ctx context.Context, req *pb.GetTaskOutcomeListPageDataRequest) (*pb.GetTaskOutcomeListPageDataResponse, error) {
	// TODO: Implement CTE-based paginated query.
	return nil, fmt.Errorf("GetTaskOutcomeListPageData not yet implemented")
}

func (r *SQLServerTaskOutcomeRepository) GetTaskOutcomeItemPageData(ctx context.Context, req *pb.GetTaskOutcomeItemPageDataRequest) (*pb.GetTaskOutcomeItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetTaskOutcomeItemPageData not yet implemented")
}
