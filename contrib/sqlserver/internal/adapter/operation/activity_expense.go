//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/activity_expense"
)

// SQLServerActivityExpenseRepository implements activity_expense CRUD operations using SQL Server.
// activity_expense uses activity_id as its PK (1:1 with job_activity).
//
// SQL Server dialect differences vs postgres gold standard:
//   - Placeholders: $1 → @p1
//   - Identifier quoting: "x" → [x]
//   - active = true → active = 1
//   - RETURNING → OUTPUT inserted.*
type SQLServerActivityExpenseRepository struct {
	pb.UnimplementedActivityExpenseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ActivityExpense, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver activity_expense repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerActivityExpenseRepository(dbOps, tableName), nil
	})
}

// NewSQLServerActivityExpenseRepository creates a new SQL Server activity expense repository.
func NewSQLServerActivityExpenseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ActivityExpenseDomainServiceServer {
	if tableName == "" {
		tableName = "activity_expense"
	}
	return &SQLServerActivityExpenseRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateActivityExpense creates a new activity expense record.
// activity_id is the PK (1:1 with job_activity).
func (r *SQLServerActivityExpenseRepository) CreateActivityExpense(ctx context.Context, req *pb.CreateActivityExpenseRequest) (*pb.CreateActivityExpenseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity expense data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// activity_expense uses activity_id as PK; map it to id for dbOps.
	if activityId, ok := data["activityId"]; ok {
		data["id"] = activityId
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity expense: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	expense := &pb.ActivityExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expense); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateActivityExpenseResponse{
		Data: []*pb.ActivityExpense{expense},
	}, nil
}

// ReadActivityExpense retrieves an activity expense by activity_id.
func (r *SQLServerActivityExpenseRepository) ReadActivityExpense(ctx context.Context, req *pb.ReadActivityExpenseRequest) (*pb.ReadActivityExpenseResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to read activity expense: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	expense := &pb.ActivityExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expense); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadActivityExpenseResponse{
		Data: []*pb.ActivityExpense{expense},
	}, nil
}

// UpdateActivityExpense updates an activity expense record.
func (r *SQLServerActivityExpenseRepository) UpdateActivityExpense(ctx context.Context, req *pb.UpdateActivityExpenseRequest) (*pb.UpdateActivityExpenseResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.ActivityId, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update activity expense: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	expense := &pb.ActivityExpense{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expense); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateActivityExpenseResponse{
		Data: []*pb.ActivityExpense{expense},
	}, nil
}

// DeleteActivityExpense soft-deletes an activity expense record.
func (r *SQLServerActivityExpenseRepository) DeleteActivityExpense(ctx context.Context, req *pb.DeleteActivityExpenseRequest) (*pb.DeleteActivityExpenseResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.ActivityId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete activity expense: %w", err)
	}

	return &pb.DeleteActivityExpenseResponse{
		Success: true,
	}, nil
}

// ListActivityExpenses lists activity expense records with optional filters.
func (r *SQLServerActivityExpenseRepository) ListActivityExpenses(ctx context.Context, req *pb.ListActivityExpensesRequest) (*pb.ListActivityExpensesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity expenses: %w", err)
	}

	var expenses []*pb.ActivityExpense
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		expense := &pb.ActivityExpense{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, expense); err != nil {
			continue
		}
		expenses = append(expenses, expense)
	}

	return &pb.ListActivityExpensesResponse{
		Data: expenses,
	}, nil
}

// GetActivityExpenseListPageData retrieves paginated activity expense list.
// TODO: Implement CTE-based paginated query with job_activity join.
func (r *SQLServerActivityExpenseRepository) GetActivityExpenseListPageData(ctx context.Context, req *pb.GetActivityExpenseListPageDataRequest) (*pb.GetActivityExpenseListPageDataResponse, error) {
	return nil, fmt.Errorf("GetActivityExpenseListPageData not yet implemented")
}

// GetActivityExpenseItemPageData retrieves a single activity expense with related data.
// TODO: Implement CTE-based single item query with job_activity join.
func (r *SQLServerActivityExpenseRepository) GetActivityExpenseItemPageData(ctx context.Context, req *pb.GetActivityExpenseItemPageDataRequest) (*pb.GetActivityExpenseItemPageDataResponse, error) {
	return nil, fmt.Errorf("GetActivityExpenseItemPageData not yet implemented")
}
