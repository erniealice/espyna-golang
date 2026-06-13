//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// jobPhaseSortableSQLCols lists the SQL column names safe to sort by in
// GetJobPhaseListPageData. Routed through core.BuildOrderBy (A2 guard).
//
// SQL Server: columns are bracket-quoted automatically by BuildOrderBy.
var jobPhaseSortableSQLCols = []string{
	"jp.phase_order",
	"jp.date_created",
	"jp.date_modified",
	"jp.name",
	"jp.status",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobPhase, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_phase repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobPhaseRepository(dbOps, tableName), nil
	})
}

// SQLServerJobPhaseRepository implements job_phase CRUD operations using SQL Server.
type SQLServerJobPhaseRepository struct {
	pb.UnimplementedJobPhaseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerJobPhaseRepository creates a new SQL Server job_phase repository.
func NewSQLServerJobPhaseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobPhaseDomainServiceServer {
	if tableName == "" {
		tableName = "job_phase"
	}
	return &SQLServerJobPhaseRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateJobPhase creates a new job phase record.
func (r *SQLServerJobPhaseRepository) CreateJobPhase(ctx context.Context, req *pb.CreateJobPhaseRequest) (*pb.CreateJobPhaseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job phase data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job phase: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// ReadJobPhase retrieves a job phase record by ID.
func (r *SQLServerJobPhaseRepository) ReadJobPhase(ctx context.Context, req *pb.ReadJobPhaseRequest) (*pb.ReadJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job phase: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// UpdateJobPhase updates a job phase record.
func (r *SQLServerJobPhaseRepository) UpdateJobPhase(ctx context.Context, req *pb.UpdateJobPhaseRequest) (*pb.UpdateJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job phase: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// DeleteJobPhase soft-deletes a job phase record.
func (r *SQLServerJobPhaseRepository) DeleteJobPhase(ctx context.Context, req *pb.DeleteJobPhaseRequest) (*pb.DeleteJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job phase: %w", err)
	}

	return &pb.DeleteJobPhaseResponse{
		Success: true,
	}, nil
}

// ListJobPhases lists job phase records with optional filters.
func (r *SQLServerJobPhaseRepository) ListJobPhases(ctx context.Context, req *pb.ListJobPhasesRequest) (*pb.ListJobPhasesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job phases: %w", err)
	}

	var phases []*pb.JobPhase
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job_phase row: %v", err)
			continue
		}

		phase := &pb.JobPhase{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
			log.Printf("WARN: protojson unmarshal job_phase: %v", err)
			continue
		}
		phases = append(phases, phase)
	}

	return &pb.ListJobPhasesResponse{
		Success: true,
		Data:    phases,
	}, nil
}

// GetJobPhaseListPageData retrieves job phases with pagination, filtering, sorting, and search.
//
// SQL Server differences vs postgres gold standard:
//   - ILIKE → LIKE (SQL Server default CI collation is case-insensitive).
//   - $1/$2/$3 → @p1/@p2/@p3.
//   - active = true → active = 1.
//   - Pagination: LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER () retained — SQL Server 2017+ supports it.
//   - workspace_id filter added (multi-tenancy guardrail).
func (r *SQLServerJobPhaseRepository) GetJobPhaseListPageData(
	ctx context.Context,
	req *pb.GetJobPhaseListPageDataRequest,
) (*pb.GetJobPhaseListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job phase list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Sort — fail-closed against whitelist (A2 guard).
	// BuildOrderBy emits bracket-quoted identifiers for SQL Server.
	orderByClause, err := sqlserverCore.BuildOrderBy(jobPhaseSortableSQLCols, req.GetSort(), "jp.phase_order ASC")
	if err != nil {
		return nil, err
	}

	// Build WHERE clauses. @p1=workspace_id, @p2=search; pagination params start at @p3.
	whereSQL := "WHERE jp.workspace_id = @p1 AND jp.active = 1"
	queryArgs := []any{workspaceID}
	nextIdx := 2

	if searchPattern != "" {
		whereSQL += fmt.Sprintf(" AND (jp.name LIKE @p%d)", nextIdx)
		queryArgs = append(queryArgs, searchPattern)
		nextIdx++
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs = append(queryArgs, offset, limit)

	// CTE with COUNT(*) OVER () — identical pattern to the postgres gold standard.
	// SQL Server differences: [bracket] quoting (handled by BuildOrderBy), @pN placeholders,
	// active = 1, and OFFSET/FETCH pagination.
	whereForCTE := strings.Replace(whereSQL, "WHERE ", "WHERE ", 1)
	_ = whereForCTE

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				jp.id,
				jp.date_created,
				jp.date_modified,
				jp.active,
				jp.job_id,
				jp.name,
				jp.phase_order,
				jp.status
			FROM job_phase jp
			%s
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, orderByClause, offsetIdx, limitIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job phase list page data: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobPhase
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobID        string
			name         string
			phaseOrder   int32
			status       string
			total        int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobID,
			&name,
			&phaseOrder,
			&status,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job phase row: %w", err)
		}

		totalCount = total

		phase := &pb.JobPhase{
			Id:         id,
			Active:     active,
			JobId:      jobID,
			Name:       name,
			PhaseOrder: phaseOrder,
		}

		if v, ok := pb.PhaseStatus_value[status]; ok {
			phase.Status = pb.PhaseStatus(v)
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			phase.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			phase.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			phase.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			phase.DateModifiedString = &dmStr
		}

		phases = append(phases, phase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job phase rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobPhaseListPageDataResponse{
		JobPhaseList: phases,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetJobPhaseItemPageData retrieves a single job phase with enriched data.
//
// SQL Server differences vs postgres:
//   - $1/$2 → @p1/@p2.
//   - active = true → active = 1.
//   - workspace_id filter (multi-tenancy guardrail).
func (r *SQLServerJobPhaseRepository) GetJobPhaseItemPageData(
	ctx context.Context,
	req *pb.GetJobPhaseItemPageDataRequest,
) (*pb.GetJobPhaseItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job phase item page data request is required")
	}
	if req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		SELECT
			jp.id,
			jp.date_created,
			jp.date_modified,
			jp.active,
			jp.job_id,
			jp.name,
			jp.phase_order,
			jp.status
		FROM job_phase jp
		WHERE jp.id = @p1 AND jp.workspace_id = @p2 AND jp.active = 1
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.JobPhaseId, workspaceID)

	var (
		id           string
		dateCreated  time.Time
		dateModified time.Time
		active       bool
		jobID        string
		name         string
		phaseOrder   int32
		status       string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&jobID,
		&name,
		&phaseOrder,
		&status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job phase with ID '%s' not found", req.JobPhaseId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job phase item page data: %w", err)
	}

	phase := &pb.JobPhase{
		Id:         id,
		Active:     active,
		JobId:      jobID,
		Name:       name,
		PhaseOrder: phaseOrder,
	}

	if v, ok := pb.PhaseStatus_value[status]; ok {
		phase.Status = pb.PhaseStatus(v)
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		phase.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		phase.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		phase.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		phase.DateModifiedString = &dmStr
	}

	return &pb.GetJobPhaseItemPageDataResponse{
		JobPhase: phase,
		Success:  true,
	}, nil
}

// ListByJob retrieves all phases for a given job, ordered by phase_order.
//
// SQL Server differences vs postgres:
//   - $1 → @p1
//   - active = true → active = 1
func (r *SQLServerJobPhaseRepository) ListByJob(
	ctx context.Context,
	req *pb.ListJobPhasesByJobRequest,
) (*pb.ListJobPhasesByJobResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	query := `
		SELECT
			jp.id,
			jp.date_created,
			jp.date_modified,
			jp.active,
			jp.job_id,
			jp.name,
			jp.phase_order,
			jp.status
		FROM job_phase jp
		WHERE jp.job_id = @p1 AND jp.active = 1
		ORDER BY jp.phase_order ASC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to list job phases by job: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobPhase
	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobID        string
			name         string
			phaseOrder   int32
			status       string
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobID,
			&name,
			&phaseOrder,
			&status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job phase row: %w", err)
		}

		phase := &pb.JobPhase{
			Id:         id,
			Active:     active,
			JobId:      jobID,
			Name:       name,
			PhaseOrder: phaseOrder,
		}

		if v, ok := pb.PhaseStatus_value[status]; ok {
			phase.Status = pb.PhaseStatus(v)
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			phase.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			phase.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			phase.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			phase.DateModifiedString = &dmStr
		}

		phases = append(phases, phase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job phase rows: %w", err)
	}

	return &pb.ListJobPhasesByJobResponse{
		JobPhases: phases,
		Success:   true,
	}, nil
}
