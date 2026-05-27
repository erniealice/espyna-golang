//go:build sqlserver

// Package expenditure — SQL Server adapter for expense_recognition_run +
// expense_recognition_run_attempt (Plan A / 20260517-expense-run).
//
// Mirrors the postgres adapter: one repository struct serves both the parent
// run table and the attempt child table. Attempt methods are NOT on the proto
// domain-service interface; the run engine reaches them via the concrete type.
//
// Schema notes (same as postgres mirror):
//   - status/scope/source_kind/outcome are INTEGER columns; protojson emits
//     enum-name strings which the adapter folds to numeric wire values.
//   - initiated_at / completed_at / attempted_at are bigint (epoch millis);
//     no DATETIME2 conversion needed — leave as protojson-emitted strings.
//   - as_of_date / period_start / period_end / period_marker are text; no
//     conversion.
package expenditure

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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_run"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ExpenseRecognitionRun, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver expense_recognition_run repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerExpenseRecognitionRunRepository(dbOps, tableName), nil
	})
}

// SQLServerExpenseRecognitionRunRepository implements expense_recognition_run +
// expense_recognition_run_attempt operations using SQL Server.
type SQLServerExpenseRecognitionRunRepository struct {
	pb.UnimplementedExpenseRecognitionRunDomainServiceServer
	dbOps        interfaces.DatabaseOperation
	runTableName string
	attemptTable string
}

// NewSQLServerExpenseRecognitionRunRepository creates a new SQL Server repository.
// Attempt rows live in `expense_recognition_run_attempt`; both names default
// to their entityid constants when tableName is blank.
func NewSQLServerExpenseRecognitionRunRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ExpenseRecognitionRunDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ExpenseRecognitionRun
	}
	return &SQLServerExpenseRecognitionRunRepository{
		dbOps:        dbOps,
		runTableName: tableName,
		attemptTable: entityid.ExpenseRecognitionRunAttempt,
	}
}

// foldExpenseRunEnumStringsToInt collapses protojson enum-name strings on the
// run row to numeric wire values for the INTEGER-typed scope/status columns.
func foldExpenseRunEnumStringsToInt(data map[string]any) {
	if v, ok := data["scope"].(string); ok {
		if num, ok := pb.ExpenseRecognitionRunScope_value[v]; ok {
			data["scope"] = int32(num)
		}
	}
	if v, ok := data["status"].(string); ok {
		if num, ok := pb.ExpenseRecognitionRunStatus_value[v]; ok {
			data["status"] = int32(num)
		}
	}
}

// foldExpenseRunAttemptEnumStringsToInt collapses protojson enum-name strings
// on the attempt row to numeric wire values for source_kind / outcome columns.
func foldExpenseRunAttemptEnumStringsToInt(data map[string]any) {
	if v, ok := data["sourceKind"].(string); ok {
		if num, ok := pb.ExpenseRecognitionRunSourceKind_value[v]; ok {
			data["sourceKind"] = int32(num)
		}
	}
	if v, ok := data["outcome"].(string); ok {
		if num, ok := pb.ExpenseRecognitionRunAttemptOutcome_value[v]; ok {
			data["outcome"] = int32(num)
		}
	}
}

func unmarshalExpenseRecognitionRun(raw map[string]any) (*pb.ExpenseRecognitionRun, error) {
	js, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
	if err != nil {
		return nil, fmt.Errorf("marshal raw expense_recognition_run: %w", err)
	}
	r := &pb.ExpenseRecognitionRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, r); err != nil {
		return nil, fmt.Errorf("unmarshal expense_recognition_run proto: %w", err)
	}
	return r, nil
}

func unmarshalExpenseRecognitionRunAttempt(raw map[string]any) (*pb.ExpenseRecognitionRunAttempt, error) {
	js, err := json.Marshal(sqlserverCore.DenormalizeKeys(raw))
	if err != nil {
		return nil, fmt.Errorf("marshal raw expense_recognition_run_attempt: %w", err)
	}
	a := &pb.ExpenseRecognitionRunAttempt{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, a); err != nil {
		return nil, fmt.Errorf("unmarshal expense_recognition_run_attempt proto: %w", err)
	}
	return a, nil
}

// CreateExpenseRecognitionRun inserts a parent run row.
func (r *SQLServerExpenseRecognitionRunRepository) CreateExpenseRecognitionRun(ctx context.Context, req *pb.CreateExpenseRecognitionRunRequest) (*pb.CreateExpenseRecognitionRunResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("expense_recognition_run data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expense_recognition_run to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expense_recognition_run JSON to map: %w", err)
	}
	foldExpenseRunEnumStringsToInt(data)

	result, err := r.dbOps.Create(ctx, r.runTableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense_recognition_run: %w", err)
	}
	run, err := unmarshalExpenseRecognitionRun(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateExpenseRecognitionRunResponse{
		Success: true,
		Data:    []*pb.ExpenseRecognitionRun{run},
	}, nil
}

// ReadExpenseRecognitionRun fetches one run by ID.
func (r *SQLServerExpenseRecognitionRunRepository) ReadExpenseRecognitionRun(ctx context.Context, req *pb.ReadExpenseRecognitionRunRequest) (*pb.ReadExpenseRecognitionRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense_recognition_run ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.runTableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expense_recognition_run: %w", err)
	}
	run, err := unmarshalExpenseRecognitionRun(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadExpenseRecognitionRunResponse{
		Success: true,
		Data:    []*pb.ExpenseRecognitionRun{run},
	}, nil
}

// UpdateExpenseRecognitionRun updates the parent run row.
func (r *SQLServerExpenseRecognitionRunRepository) UpdateExpenseRecognitionRun(ctx context.Context, req *pb.UpdateExpenseRecognitionRunRequest) (*pb.UpdateExpenseRecognitionRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense_recognition_run ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expense_recognition_run to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expense_recognition_run JSON to map: %w", err)
	}
	foldExpenseRunEnumStringsToInt(data)

	result, err := r.dbOps.Update(ctx, r.runTableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expense_recognition_run: %w", err)
	}
	run, err := unmarshalExpenseRecognitionRun(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateExpenseRecognitionRunResponse{
		Success: true,
		Data:    []*pb.ExpenseRecognitionRun{run},
	}, nil
}

// DeleteExpenseRecognitionRun soft-deletes a run.
func (r *SQLServerExpenseRecognitionRunRepository) DeleteExpenseRecognitionRun(ctx context.Context, req *pb.DeleteExpenseRecognitionRunRequest) (*pb.DeleteExpenseRecognitionRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense_recognition_run ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.runTableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expense_recognition_run: %w", err)
	}
	return &pb.DeleteExpenseRecognitionRunResponse{Success: true}, nil
}

// ListExpenseRecognitionRuns returns all runs visible in the current workspace.
func (r *SQLServerExpenseRecognitionRunRepository) ListExpenseRecognitionRuns(ctx context.Context, req *pb.ListExpenseRecognitionRunsRequest) (*pb.ListExpenseRecognitionRunsResponse, error) {
	var params *interfaces.ListParams
	if req != nil {
		params = &interfaces.ListParams{
			Search:     req.Search,
			Filters:    req.Filters,
			Sort:       req.Sort,
			Pagination: req.Pagination,
		}
	}
	listResult, err := r.dbOps.List(ctx, r.runTableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expense_recognition_runs: %w", err)
	}
	runs := make([]*pb.ExpenseRecognitionRun, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		run, err := unmarshalExpenseRecognitionRun(raw)
		if err != nil {
			log.Printf("WARN: unmarshal expense_recognition_run row: %v", err)
			continue
		}
		runs = append(runs, run)
	}
	return &pb.ListExpenseRecognitionRunsResponse{
		Success: true,
		Data:    runs,
	}, nil
}

// --- Attempt methods (not on proto interface; called via concrete type) ---

// CreateExpenseRecognitionRunAttempt inserts one attempt row.
func (r *SQLServerExpenseRecognitionRunRepository) CreateExpenseRecognitionRunAttempt(ctx context.Context, req *pb.CreateExpenseRecognitionRunAttemptRequest) (*pb.CreateExpenseRecognitionRunAttemptResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("expense_recognition_run_attempt data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expense_recognition_run_attempt to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expense_recognition_run_attempt JSON to map: %w", err)
	}
	foldExpenseRunAttemptEnumStringsToInt(data)

	result, err := r.dbOps.Create(ctx, r.attemptTable, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense_recognition_run_attempt: %w", err)
	}
	attempt, err := unmarshalExpenseRecognitionRunAttempt(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateExpenseRecognitionRunAttemptResponse{
		Success: true,
		Data:    []*pb.ExpenseRecognitionRunAttempt{attempt},
	}, nil
}

// ListExpenseRecognitionRunAttempts returns every attempt for a given run,
// in attempt order. RunId is required.
func (r *SQLServerExpenseRecognitionRunRepository) ListExpenseRecognitionRunAttempts(ctx context.Context, req *pb.ListExpenseRecognitionRunAttemptsRequest) (*pb.ListExpenseRecognitionRunAttemptsResponse, error) {
	if req == nil || req.RunId == "" {
		return nil, fmt.Errorf("expense_recognition_run_attempt run_id is required")
	}

	runFilter := &commonpb.TypedFilter{
		Field: "run_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    req.RunId,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{runFilter}}
	if req.Filters != nil {
		filters.Filters = append(filters.Filters, req.Filters.Filters...)
	}

	sort := req.Sort
	if sort == nil || len(sort.Fields) == 0 {
		sort = &commonpb.SortRequest{
			Fields: []*commonpb.SortField{{
				Field:     "attempted_at",
				Direction: commonpb.SortDirection_ASC,
			}},
		}
	}

	params := &interfaces.ListParams{Filters: filters, Sort: sort}
	listResult, err := r.dbOps.List(ctx, r.attemptTable, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expense_recognition_run_attempts: %w", err)
	}
	attempts := make([]*pb.ExpenseRecognitionRunAttempt, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		a, err := unmarshalExpenseRecognitionRunAttempt(raw)
		if err != nil {
			log.Printf("WARN: unmarshal expense_recognition_run_attempt row: %v", err)
			continue
		}
		attempts = append(attempts, a)
	}
	return &pb.ListExpenseRecognitionRunAttemptsResponse{
		Success: true,
		Data:    attempts,
	}, nil
}
