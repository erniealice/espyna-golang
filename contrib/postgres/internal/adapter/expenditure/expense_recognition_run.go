//go:build postgresql

// Package expenditure — postgres adapter for the expense_recognition_run
// domain service (Plan A / 20260517-expense-run).
//
// expense_recognition_run is the buying-side mirror of revenue_run. One gRPC
// service exposes CRUD over the parent table, and Attempt rows live as
// methods on the same repository so the recurrence engine can persist
// per-selection outcomes through a single repository handle. Attempt CRUD is
// NOT part of the proto domain-service interface (the interface only declares
// parent-level Create/Read/Update/Delete/List) — those methods are reachable
// via concrete-type calls from the run engine, mirroring the revenue_run
// repository pattern.
//
// Schema notes (per migration 20260517160000_expense_run_tables.sql, which
// differs from revenue_run):
//   - status/scope/source_kind/outcome are INTEGER columns. protojson
//     emits them as enum-name strings; the adapter folds them to numeric
//     wire values before passing the map to dbOps.{Create,Update}.
//   - initiated_at / completed_at / attempted_at are `bigint` columns
//     storing epoch millis directly (NOT postgres timestamp). The adapter
//     therefore leaves them as the protojson-emitted millis string and
//     does NOT call convertMillisToTime — the underlying postgres driver
//     accepts the string-encoded int64 via protojson round-trip path
//     normalisation in WorkspaceAwareOperations.
//   - as_of_date / period_start / period_end / period_marker are `text`
//     (not date) columns; no conversion needed.
package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_run"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ExpenseRecognitionRun, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres expense_recognition_run repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresExpenseRecognitionRunRepository(db, dbOps, tableName), nil
	})
}

// PostgresExpenseRecognitionRunRepository implements expense_recognition_run +
// expense_recognition_run_attempt operations using PostgreSQL.
type PostgresExpenseRecognitionRunRepository struct {
	pb.UnimplementedExpenseRecognitionRunDomainServiceServer
	dbOps        interfaces.DatabaseOperation
	db           *sql.DB
	runTableName string
	attemptTable string
}

// NewPostgresExpenseRecognitionRunRepository creates a new PostgreSQL repository.
// Attempt rows live in `expense_recognition_run_attempt`; both names default
// to their entityid constants when tableName is blank.
func NewPostgresExpenseRecognitionRunRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) pb.ExpenseRecognitionRunDomainServiceServer {
	if tableName == "" {
		tableName = entityid.ExpenseRecognitionRun
	}
	return &PostgresExpenseRecognitionRunRepository{
		dbOps:        dbOps,
		db:           db,
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

func unmarshalExpenseRecognitionRun(raw map[string]any) (*pb.ExpenseRecognitionRun, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw expense_recognition_run: %w", err)
	}
	r := &pb.ExpenseRecognitionRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, r); err != nil {
		return nil, fmt.Errorf("unmarshal expense_recognition_run proto: %w", err)
	}
	return r, nil
}

// CreateExpenseRecognitionRun inserts a parent run row.
func (r *PostgresExpenseRecognitionRunRepository) CreateExpenseRecognitionRun(ctx context.Context, req *pb.CreateExpenseRecognitionRunRequest) (*pb.CreateExpenseRecognitionRunResponse, error) {
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
func (r *PostgresExpenseRecognitionRunRepository) ReadExpenseRecognitionRun(ctx context.Context, req *pb.ReadExpenseRecognitionRunRequest) (*pb.ReadExpenseRecognitionRunResponse, error) {
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

// UpdateExpenseRecognitionRun updates the parent run row. Called by the
// GenerateExpenseRun engine after all attempts complete to set final counts
// + status + completed_at.
func (r *PostgresExpenseRecognitionRunRepository) UpdateExpenseRecognitionRun(ctx context.Context, req *pb.UpdateExpenseRecognitionRunRequest) (*pb.UpdateExpenseRecognitionRunResponse, error) {
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

// DeleteExpenseRecognitionRun soft-deletes a run. Attempt rows are not
// cascaded by the adapter (the migration sets ON DELETE CASCADE only for
// hard deletes, which this adapter never performs).
func (r *PostgresExpenseRecognitionRunRepository) DeleteExpenseRecognitionRun(ctx context.Context, req *pb.DeleteExpenseRecognitionRunRequest) (*pb.DeleteExpenseRecognitionRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expense_recognition_run ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.runTableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expense_recognition_run: %w", err)
	}
	return &pb.DeleteExpenseRecognitionRunResponse{Success: true}, nil
}

// ListExpenseRecognitionRuns returns all runs visible in the current workspace.
func (r *PostgresExpenseRecognitionRunRepository) ListExpenseRecognitionRuns(ctx context.Context, req *pb.ListExpenseRecognitionRunsRequest) (*pb.ListExpenseRecognitionRunsResponse, error) {
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
	resp := &pb.ListExpenseRecognitionRunsResponse{
		Success: true,
		Data:    runs,
	}
	return resp, nil
}

// Attempt-row methods live in expense_recognition_run_attempt.go to keep
// each file focused on a single physical table; they share the same
// PostgresExpenseRecognitionRunRepository receiver.
