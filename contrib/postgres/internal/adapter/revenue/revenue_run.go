//go:build postgresql

// Package revenue — postgres adapter for the revenue_run domain service.
//
// The service shape is unusual: one gRPC service exposes CRUD over two tables
// (revenue_run + revenue_run_attempt). The provider only wires
// `entityid.RevenueRun`; attempt operations live as methods on the same
// repository because protojson defines them under the same
// `RevenueRunDomainServiceServer` interface.
//
// Adapter behaviour:
//   - protojson round-trip for proto<->row marshalling. Enums (RevenueRunStatus,
//     RevenueRunScopeKind, RevenueRunAttemptOutcome) are stored as their full
//     wire-name strings (e.g. "REVENUE_RUN_STATUS_PENDING") in the `text`
//     columns. Read-back via protojson restores the enum value exactly.
//   - epoch-millisecond proto fields (initiated_at, completed_at, attempted_at)
//     are converted to time.Time before the underlying dbOps.Create / .Update
//     so postgres timestamp columns accept them.
//   - workspace_id filtering is provided by WorkspaceAwareOperations; the
//     adapter never adds it manually.
package revenue

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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RevenueRun, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres revenue_run repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresRevenueRunRepository(db, dbOps, tableName), nil
	})
}

// PostgresRevenueRunRepository implements revenue_run + revenue_run_attempt
// operations using PostgreSQL.
type PostgresRevenueRunRepository struct {
	revenuerunpb.UnimplementedRevenueRunDomainServiceServer
	dbOps        interfaces.DatabaseOperation
	db           *sql.DB
	runTableName string
	attemptTable string
}

// NewPostgresRevenueRunRepository creates a new PostgreSQL revenue_run repository.
// Attempt rows live in `revenue_run_attempt`; both names default if blank.
func NewPostgresRevenueRunRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) revenuerunpb.RevenueRunDomainServiceServer {
	if tableName == "" {
		tableName = entityid.RevenueRun
	}
	return &PostgresRevenueRunRepository{
		dbOps:        dbOps,
		db:           db,
		runTableName: tableName,
		attemptTable: entityid.RevenueRunAttempt,
	}
}

func unmarshalRevenueRun(raw map[string]any) (*revenuerunpb.RevenueRun, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw revenue_run: %w", err)
	}
	r := &revenuerunpb.RevenueRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, r); err != nil {
		return nil, fmt.Errorf("unmarshal revenue_run proto: %w", err)
	}
	return r, nil
}

func unmarshalRevenueRunAttempt(raw map[string]any) (*revenuerunpb.RevenueRunAttempt, error) {
	js, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw revenue_run_attempt: %w", err)
	}
	a := &revenuerunpb.RevenueRunAttempt{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, a); err != nil {
		return nil, fmt.Errorf("unmarshal revenue_run_attempt proto: %w", err)
	}
	return a, nil
}

// CreateRevenueRun inserts a parent run row.
func (r *PostgresRevenueRunRepository) CreateRevenueRun(ctx context.Context, req *revenuerunpb.CreateRevenueRunRequest) (*revenuerunpb.CreateRevenueRunResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("revenue_run data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal revenue_run to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal revenue_run JSON to map: %w", err)
	}
	// timestamp columns: convert proto millis (camelCase, post-protojson) to time.Time
	convertMillisToTime(data, "initiatedAt", "initiated_at")
	convertMillisToTime(data, "completedAt", "completed_at")

	result, err := r.dbOps.Create(ctx, r.runTableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_run: %w", err)
	}
	postgresCore.ConvertMillisToDateStr(result, "as_of_date")
	run, err := unmarshalRevenueRun(result)
	if err != nil {
		return nil, err
	}
	return &revenuerunpb.CreateRevenueRunResponse{
		Success: true,
		Data:    []*revenuerunpb.RevenueRun{run},
	}, nil
}

// ReadRevenueRun fetches one run by ID.
func (r *PostgresRevenueRunRepository) ReadRevenueRun(ctx context.Context, req *revenuerunpb.ReadRevenueRunRequest) (*revenuerunpb.ReadRevenueRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_run ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.runTableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue_run: %w", err)
	}
	postgresCore.ConvertMillisToDateStr(result, "as_of_date")
	run, err := unmarshalRevenueRun(result)
	if err != nil {
		return nil, err
	}
	return &revenuerunpb.ReadRevenueRunResponse{
		Success: true,
		Data:    []*revenuerunpb.RevenueRun{run},
	}, nil
}

// UpdateRevenueRun updates the parent run row. Called by GenerateRevenueRun
// after all attempts complete to set final counts + status + completed_at.
func (r *PostgresRevenueRunRepository) UpdateRevenueRun(ctx context.Context, req *revenuerunpb.UpdateRevenueRunRequest) (*revenuerunpb.UpdateRevenueRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_run ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal revenue_run to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal revenue_run JSON to map: %w", err)
	}
	convertMillisToTime(data, "initiatedAt", "initiated_at")
	convertMillisToTime(data, "completedAt", "completed_at")

	result, err := r.dbOps.Update(ctx, r.runTableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update revenue_run: %w", err)
	}
	postgresCore.ConvertMillisToDateStr(result, "as_of_date")
	run, err := unmarshalRevenueRun(result)
	if err != nil {
		return nil, err
	}
	return &revenuerunpb.UpdateRevenueRunResponse{
		Success: true,
		Data:    []*revenuerunpb.RevenueRun{run},
	}, nil
}

// DeleteRevenueRun soft-deletes a run. Attempt rows are not cascaded by the
// adapter; the migration sets `ON DELETE CASCADE` on the FK only for hard
// deletes (which this adapter does not perform).
func (r *PostgresRevenueRunRepository) DeleteRevenueRun(ctx context.Context, req *revenuerunpb.DeleteRevenueRunRequest) (*revenuerunpb.DeleteRevenueRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_run ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.runTableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete revenue_run: %w", err)
	}
	return &revenuerunpb.DeleteRevenueRunResponse{Success: true}, nil
}

// ListRevenueRuns returns all runs visible in the current workspace, filtered
// by the supplied search/filter/sort/pagination. The view layer applies
// status filtering client-side; this adapter passes the proto filters through
// unchanged.
func (r *PostgresRevenueRunRepository) ListRevenueRuns(ctx context.Context, req *revenuerunpb.ListRevenueRunsRequest) (*revenuerunpb.ListRevenueRunsResponse, error) {
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
		return nil, fmt.Errorf("failed to list revenue_runs: %w", err)
	}
	runs := make([]*revenuerunpb.RevenueRun, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(raw, "as_of_date")
		run, err := unmarshalRevenueRun(raw)
		if err != nil {
			log.Printf("WARN: unmarshal revenue_run row: %v", err)
			continue
		}
		runs = append(runs, run)
	}
	resp := &revenuerunpb.ListRevenueRunsResponse{
		Success: true,
		Data:    runs,
	}
	if listResult.Pagination != nil {
		resp.Pagination = listResult.Pagination
	}
	return resp, nil
}

// foldRevenueRunAttemptEnumStringsToInt collapses protojson enum-name strings
// on the attempt row to numeric wire values for the INTEGER-typed source_kind
// column. The legacy `outcome` column is TEXT and so is left alone.
//
// Mirrors the expense_recognition_run_attempt pattern (Plan A Phase 5).
func foldRevenueRunAttemptEnumStringsToInt(data map[string]any) {
	if v, ok := data["sourceKind"].(string); ok {
		if num, ok := revenuerunpb.RevenueRunSourceKind_value[v]; ok {
			data["sourceKind"] = int32(num)
		}
	}
}

// CreateRevenueRunAttempt inserts one attempt row. Called from inside the
// per-selection loop of GenerateRevenueRun. Errors are tolerated by the caller
// (it logs and continues); this method itself returns the row on success.
func (r *PostgresRevenueRunRepository) CreateRevenueRunAttempt(ctx context.Context, req *revenuerunpb.CreateRevenueRunAttemptRequest) (*revenuerunpb.CreateRevenueRunAttemptResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("revenue_run_attempt data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal revenue_run_attempt to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal revenue_run_attempt JSON to map: %w", err)
	}
	convertMillisToTime(data, "attemptedAt", "attempted_at")
	foldRevenueRunAttemptEnumStringsToInt(data)

	result, err := r.dbOps.Create(ctx, r.attemptTable, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_run_attempt: %w", err)
	}
	postgresCore.ConvertMillisToDateStr(result, "period_start", "period_end")
	attempt, err := unmarshalRevenueRunAttempt(result)
	if err != nil {
		return nil, err
	}
	return &revenuerunpb.CreateRevenueRunAttemptResponse{
		Success: true,
		Data:    []*revenuerunpb.RevenueRunAttempt{attempt},
	}, nil
}

// ListRevenueRunAttempts returns every attempt for a given run, in attempt
// order. RunId on the request is required and is the only filter callers
// currently use; any user-supplied Filters are merged in alongside it.
func (r *PostgresRevenueRunRepository) ListRevenueRunAttempts(ctx context.Context, req *revenuerunpb.ListRevenueRunAttemptsRequest) (*revenuerunpb.ListRevenueRunAttemptsResponse, error) {
	if req == nil || req.RunId == "" {
		return nil, fmt.Errorf("revenue_run_attempt run_id is required")
	}

	// Build a StringFilter on run_id and merge with any caller-supplied filters.
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

	// Default sort: attempted_at ASC (chronological per-run insert order).
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
		return nil, fmt.Errorf("failed to list revenue_run_attempts: %w", err)
	}
	attempts := make([]*revenuerunpb.RevenueRunAttempt, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(raw, "period_start", "period_end")
		a, err := unmarshalRevenueRunAttempt(raw)
		if err != nil {
			log.Printf("WARN: unmarshal revenue_run_attempt row: %v", err)
			continue
		}
		attempts = append(attempts, a)
	}
	return &revenuerunpb.ListRevenueRunAttemptsResponse{
		Success: true,
		Data:    attempts,
	}, nil
}
