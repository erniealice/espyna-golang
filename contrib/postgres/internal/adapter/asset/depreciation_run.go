//go:build postgresql

package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.DepreciationRun, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres depreciation_run repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresDepreciationRunRepository(dbOps, tableName), nil
	})
}

var depreciationRunSortableSQLCols = []string{
	"id", "workspace_id", "scope_kind", "scope_id", "as_of_date",
	"initiator_id", "initiated_at", "completed_at", "status",
	"created_count", "skipped_count", "errored_count",
	"active", "created_at", "updated_at",
}

var depreciationRunSortSpec = espynahttp.SortSpec{AllowedCols: depreciationRunSortableSQLCols}

// PostgresDepreciationRunRepository implements depreciation_run CRUD + list
// operations using PostgreSQL. The GenerateDepreciationRun and
// ListDepreciationCandidates RPCs are handled at the use-case layer, not here;
// those methods return Unimplemented from the embedded base.
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_depreciation_run_workspace_id ON depreciation_run(workspace_id)
//   - CREATE INDEX idx_depreciation_run_status ON depreciation_run(status)
//   - CREATE INDEX idx_depreciation_run_as_of_date ON depreciation_run(as_of_date)
type PostgresDepreciationRunRepository struct {
	deprunpb.UnimplementedDepreciationRunDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresDepreciationRunRepository creates a new PostgreSQL depreciation_run repository.
func NewPostgresDepreciationRunRepository(dbOps interfaces.DatabaseOperation, tableName string) deprunpb.DepreciationRunDomainServiceServer {
	if tableName == "" {
		tableName = "depreciation_run"
	}
	return &PostgresDepreciationRunRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateDepreciationRun inserts a new depreciation_run row.
func (r *PostgresDepreciationRunRepository) CreateDepreciationRun(ctx context.Context, req *deprunpb.CreateDepreciationRunRequest) (*deprunpb.CreateDepreciationRunResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("depreciation_run data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_run protobuf: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_run JSON: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create depreciation_run: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_run result: %w", err)
	}

	run := &deprunpb.DepreciationRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_run result: %w", err)
	}

	return &deprunpb.CreateDepreciationRunResponse{
		Data:    []*deprunpb.DepreciationRun{run},
		Success: true,
	}, nil
}

// ReadDepreciationRun retrieves a single depreciation_run row by ID.
func (r *PostgresDepreciationRunRepository) ReadDepreciationRun(ctx context.Context, req *deprunpb.ReadDepreciationRunRequest) (*deprunpb.ReadDepreciationRunResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("depreciation_run ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read depreciation_run: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("depreciation_run with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_run result: %w", err)
	}

	run := &deprunpb.DepreciationRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_run result: %w", err)
	}

	return &deprunpb.ReadDepreciationRunResponse{
		Data:    []*deprunpb.DepreciationRun{run},
		Success: true,
	}, nil
}

// UpdateDepreciationRun patches a depreciation_run row (e.g., status + counts after generation).
func (r *PostgresDepreciationRunRepository) UpdateDepreciationRun(ctx context.Context, req *deprunpb.UpdateDepreciationRunRequest) (*deprunpb.UpdateDepreciationRunResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("depreciation_run ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_run protobuf: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_run JSON: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update depreciation_run: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal depreciation_run result: %w", err)
	}

	run := &deprunpb.DepreciationRun{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal depreciation_run result: %w", err)
	}

	return &deprunpb.UpdateDepreciationRunResponse{
		Data:    []*deprunpb.DepreciationRun{run},
		Success: true,
	}, nil
}

// DeleteDepreciationRun soft-deletes a depreciation_run row.
func (r *PostgresDepreciationRunRepository) DeleteDepreciationRun(ctx context.Context, req *deprunpb.DeleteDepreciationRunRequest) (*deprunpb.DeleteDepreciationRunResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("depreciation_run ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete depreciation_run: %w", err)
	}

	return &deprunpb.DeleteDepreciationRunResponse{
		Success: true,
	}, nil
}

// ListDepreciationRuns lists depreciation_run rows.
func (r *PostgresDepreciationRunRepository) ListDepreciationRuns(ctx context.Context, req *deprunpb.ListDepreciationRunsRequest) (*deprunpb.ListDepreciationRunsResponse, error) {
	if err := espynahttp.ValidateSortColumns(depreciationRunSortSpec, req.GetSort(), "depreciation_run"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list depreciation_runs: %w", err)
	}

	var runs []*deprunpb.DepreciationRun
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		run := &deprunpb.DepreciationRun{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, run); err != nil {
			continue
		}
		runs = append(runs, run)
	}

	return &deprunpb.ListDepreciationRunsResponse{
		Data:    runs,
		Success: true,
	}, nil
}

// NewDepreciationRunRepository creates a new PostgreSQL depreciation_run repository (old-style constructor).
func NewDepreciationRunRepository(db *sql.DB, tableName string) deprunpb.DepreciationRunDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresDepreciationRunRepository(dbOps, tableName)
}
