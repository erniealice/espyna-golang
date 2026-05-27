//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders)
//   - active = true → active = 1
//   - mysqlCore.ConvertMillisToDateStr replaces postgresCore.ConvertMillisToDateStr
//   - convertMillisToTime calls use single-arg form (jsonKey only)
//
// Enum values (RevenueRunStatus, RevenueRunScopeKind, RevenueRunAttemptOutcome)
// are stored as wire-name strings in text columns; protojson round-trip restores
// them exactly. sourceKind int folding mirrors the postgres pattern.
package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.RevenueRun, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql revenue_run repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLRevenueRunRepository(db, dbOps, tableName), nil
	})
}

// MySQLRevenueRunRepository implements revenue_run + revenue_run_attempt
// operations using MySQL 8.0+.
type MySQLRevenueRunRepository struct {
	revenuerunpb.UnimplementedRevenueRunDomainServiceServer
	dbOps        interfaces.DatabaseOperation
	db           *sql.DB
	runTableName string
	attemptTable string
}

// NewMySQLRevenueRunRepository creates a new MySQL revenue_run repository.
func NewMySQLRevenueRunRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) revenuerunpb.RevenueRunDomainServiceServer {
	if tableName == "" {
		tableName = entityid.RevenueRun
	}
	return &MySQLRevenueRunRepository{
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
func (r *MySQLRevenueRunRepository) CreateRevenueRun(ctx context.Context, req *revenuerunpb.CreateRevenueRunRequest) (*revenuerunpb.CreateRevenueRunResponse, error) {
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
	convertMillisToTime(data, "initiatedAt")
	convertMillisToTime(data, "completedAt")

	result, err := r.dbOps.Create(ctx, r.runTableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_run: %w", err)
	}
	mysqlCore.ConvertMillisToDateStr(result, "as_of_date")
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
func (r *MySQLRevenueRunRepository) ReadRevenueRun(ctx context.Context, req *revenuerunpb.ReadRevenueRunRequest) (*revenuerunpb.ReadRevenueRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_run ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.runTableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue_run: %w", err)
	}
	mysqlCore.ConvertMillisToDateStr(result, "as_of_date")
	run, err := unmarshalRevenueRun(result)
	if err != nil {
		return nil, err
	}
	return &revenuerunpb.ReadRevenueRunResponse{
		Success: true,
		Data:    []*revenuerunpb.RevenueRun{run},
	}, nil
}

// UpdateRevenueRun updates the parent run row.
func (r *MySQLRevenueRunRepository) UpdateRevenueRun(ctx context.Context, req *revenuerunpb.UpdateRevenueRunRequest) (*revenuerunpb.UpdateRevenueRunResponse, error) {
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
	convertMillisToTime(data, "initiatedAt")
	convertMillisToTime(data, "completedAt")

	result, err := r.dbOps.Update(ctx, r.runTableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update revenue_run: %w", err)
	}
	mysqlCore.ConvertMillisToDateStr(result, "as_of_date")
	run, err := unmarshalRevenueRun(result)
	if err != nil {
		return nil, err
	}
	return &revenuerunpb.UpdateRevenueRunResponse{
		Success: true,
		Data:    []*revenuerunpb.RevenueRun{run},
	}, nil
}

// DeleteRevenueRun soft-deletes a run.
func (r *MySQLRevenueRunRepository) DeleteRevenueRun(ctx context.Context, req *revenuerunpb.DeleteRevenueRunRequest) (*revenuerunpb.DeleteRevenueRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_run ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.runTableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete revenue_run: %w", err)
	}
	return &revenuerunpb.DeleteRevenueRunResponse{Success: true}, nil
}

// ListRevenueRuns returns all runs visible in the current workspace.
func (r *MySQLRevenueRunRepository) ListRevenueRuns(ctx context.Context, req *revenuerunpb.ListRevenueRunsRequest) (*revenuerunpb.ListRevenueRunsResponse, error) {
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
		mysqlCore.ConvertMillisToDateStr(raw, "as_of_date")
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
// column. Mirrors the postgres pattern.
func foldRevenueRunAttemptEnumStringsToInt(data map[string]any) {
	if v, ok := data["sourceKind"].(string); ok {
		if num, ok := revenuerunpb.RevenueRunSourceKind_value[v]; ok {
			data["sourceKind"] = int32(num)
		}
	}
}

// CreateRevenueRunAttempt inserts one attempt row.
func (r *MySQLRevenueRunRepository) CreateRevenueRunAttempt(ctx context.Context, req *revenuerunpb.CreateRevenueRunAttemptRequest) (*revenuerunpb.CreateRevenueRunAttemptResponse, error) {
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
	convertMillisToTime(data, "attemptedAt")
	foldRevenueRunAttemptEnumStringsToInt(data)

	result, err := r.dbOps.Create(ctx, r.attemptTable, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_run_attempt: %w", err)
	}
	mysqlCore.ConvertMillisToDateStr(result, "period_start", "period_end")
	attempt, err := unmarshalRevenueRunAttempt(result)
	if err != nil {
		return nil, err
	}
	return &revenuerunpb.CreateRevenueRunAttemptResponse{
		Success: true,
		Data:    []*revenuerunpb.RevenueRunAttempt{attempt},
	}, nil
}

// ListRevenueRunAttempts returns every attempt for a given run, in attempt order.
func (r *MySQLRevenueRunRepository) ListRevenueRunAttempts(ctx context.Context, req *revenuerunpb.ListRevenueRunAttemptsRequest) (*revenuerunpb.ListRevenueRunAttemptsResponse, error) {
	if req == nil || req.RunId == "" {
		return nil, fmt.Errorf("revenue_run_attempt run_id is required")
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
		return nil, fmt.Errorf("failed to list revenue_run_attempts: %w", err)
	}
	attempts := make([]*revenuerunpb.RevenueRunAttempt, 0, len(listResult.Data))
	for _, raw := range listResult.Data {
		mysqlCore.ConvertMillisToDateStr(raw, "period_start", "period_end")
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

// Ensure unused import is suppressed.
var _ = sql.ErrNoRows
