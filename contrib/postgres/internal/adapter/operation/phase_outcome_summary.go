//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/phase_outcome_summary"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PhaseOutcomeSummary, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres phase_outcome_summary repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPhaseOutcomeSummaryRepository(dbOps, tableName), nil
	})
}

// PostgresPhaseOutcomeSummaryRepository implements phase_outcome_summary CRUD operations using PostgreSQL
type PostgresPhaseOutcomeSummaryRepository struct {
	pb.UnimplementedPhaseOutcomeSummaryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPhaseOutcomeSummaryRepository creates a new PostgreSQL phase_outcome_summary repository
func NewPostgresPhaseOutcomeSummaryRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.PhaseOutcomeSummaryDomainServiceServer {
	if tableName == "" {
		tableName = "phase_outcome_summary"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPhaseOutcomeSummaryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePhaseOutcomeSummary creates a new phase_outcome_summary record
func (r *PostgresPhaseOutcomeSummaryRepository) CreatePhaseOutcomeSummary(ctx context.Context, req *pb.CreatePhaseOutcomeSummaryRequest) (*pb.CreatePhaseOutcomeSummaryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("phase outcome summary data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create phase outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.PhaseOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreatePhaseOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.PhaseOutcomeSummary{summary},
	}, nil
}

// ReadPhaseOutcomeSummary retrieves a phase_outcome_summary record by ID
func (r *PostgresPhaseOutcomeSummaryRepository) ReadPhaseOutcomeSummary(ctx context.Context, req *pb.ReadPhaseOutcomeSummaryRequest) (*pb.ReadPhaseOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("phase outcome summary ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read phase outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.PhaseOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadPhaseOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.PhaseOutcomeSummary{summary},
	}, nil
}

// UpdatePhaseOutcomeSummary updates a phase_outcome_summary record
func (r *PostgresPhaseOutcomeSummaryRepository) UpdatePhaseOutcomeSummary(ctx context.Context, req *pb.UpdatePhaseOutcomeSummaryRequest) (*pb.UpdatePhaseOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("phase outcome summary ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update phase outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.PhaseOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdatePhaseOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.PhaseOutcomeSummary{summary},
	}, nil
}

// DeletePhaseOutcomeSummary deletes a phase_outcome_summary record (soft delete)
func (r *PostgresPhaseOutcomeSummaryRepository) DeletePhaseOutcomeSummary(ctx context.Context, req *pb.DeletePhaseOutcomeSummaryRequest) (*pb.DeletePhaseOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("phase outcome summary ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete phase outcome summary: %w", err)
	}

	return &pb.DeletePhaseOutcomeSummaryResponse{
		Success: true,
	}, nil
}

// ListPhaseOutcomeSummarys lists phase_outcome_summary records with optional filters
func (r *PostgresPhaseOutcomeSummaryRepository) ListPhaseOutcomeSummarys(ctx context.Context, req *pb.ListPhaseOutcomeSummarysRequest) (*pb.ListPhaseOutcomeSummarysResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list phase outcome summarys: %w", err)
	}

	var summaries []*pb.PhaseOutcomeSummary
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal phase_outcome_summary row: %v", err)
			continue
		}

		summary := &pb.PhaseOutcomeSummary{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
			log.Printf("WARN: protojson unmarshal phase_outcome_summary: %v", err)
			continue
		}
		summaries = append(summaries, summary)
	}

	return &pb.ListPhaseOutcomeSummarysResponse{
		Success: true,
		Data:    summaries,
	}, nil
}

// GetPhaseOutcomeSummaryListPageData retrieves phase outcome summaries with pagination
func (r *PostgresPhaseOutcomeSummaryRepository) GetPhaseOutcomeSummaryListPageData(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryListPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get phase outcome summary list page data request is required")
	}

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

	sortField := "pos.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	posColumns := `
		pos.id, pos.job_phase_id, pos.job_id, pos.summary_type,
		pos.phase_determination, pos.scoring_method, pos.summary_score,
		pos.total_criteria_count, pos.pass_count, pos.fail_count,
		pos.conditional_count, pos.deferred_count, pos.na_count,
		pos.narrative, pos.issued_by, pos.issued_date,
		pos.supersedes_id, pos.active, pos.date_created, pos.date_modified
	`

	query := `
		WITH enriched AS (
			SELECT ` + posColumns + `
			FROM phase_outcome_summary pos
			WHERE pos.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       pos.narrative ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*, c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query phase outcome summary list page data: %w", err)
	}
	defer rows.Close()

	var summaries []*pb.PhaseOutcomeSummary
	var totalCount int64

	for rows.Next() {
		summary, cnt, err := scanPhaseOutcomeSummaryRowWithTotal(rows)
		if err != nil {
			return nil, err
		}
		totalCount = cnt
		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating phase outcome summary rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetPhaseOutcomeSummaryListPageDataResponse{
		PhaseOutcomeSummaryList: summaries,
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

// GetPhaseOutcomeSummaryItemPageData retrieves a single phase outcome summary
func (r *PostgresPhaseOutcomeSummaryRepository) GetPhaseOutcomeSummaryItemPageData(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryItemPageDataRequest,
) (*pb.GetPhaseOutcomeSummaryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get phase outcome summary item page data request is required")
	}
	if req.PhaseOutcomeSummaryId == "" {
		return nil, fmt.Errorf("phase outcome summary ID is required")
	}

	query := `
		SELECT
			pos.id, pos.job_phase_id, pos.job_id, pos.summary_type,
			pos.phase_determination, pos.scoring_method, pos.summary_score,
			pos.total_criteria_count, pos.pass_count, pos.fail_count,
			pos.conditional_count, pos.deferred_count, pos.na_count,
			pos.narrative, pos.issued_by, pos.issued_date,
			pos.supersedes_id, pos.active, pos.date_created, pos.date_modified
		FROM phase_outcome_summary pos
		WHERE pos.id = $1 AND pos.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.PhaseOutcomeSummaryId)
	summary, err := scanPhaseOutcomeSummarySingleRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("phase outcome summary with ID '%s' not found", req.PhaseOutcomeSummaryId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query phase outcome summary item page data: %w", err)
	}

	return &pb.GetPhaseOutcomeSummaryItemPageDataResponse{
		PhaseOutcomeSummary: summary,
		Success:             true,
	}, nil
}

// GetByJobPhase retrieves the latest phase outcome summary for a given job phase
func (r *PostgresPhaseOutcomeSummaryRepository) GetByJobPhase(
	ctx context.Context,
	req *pb.GetPhaseOutcomeSummaryByJobPhaseRequest,
) (*pb.GetPhaseOutcomeSummaryByJobPhaseResponse, error) {
	if req == nil || req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	query := `
		SELECT
			pos.id, pos.job_phase_id, pos.job_id, pos.summary_type,
			pos.phase_determination, pos.scoring_method, pos.summary_score,
			pos.total_criteria_count, pos.pass_count, pos.fail_count,
			pos.conditional_count, pos.deferred_count, pos.na_count,
			pos.narrative, pos.issued_by, pos.issued_date,
			pos.supersedes_id, pos.active, pos.date_created, pos.date_modified
		FROM phase_outcome_summary pos
		WHERE pos.job_phase_id = $1 AND pos.active = true
		ORDER BY pos.date_created DESC
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, req.JobPhaseId)
	summary, err := scanPhaseOutcomeSummarySingleRow(row)
	if err == sql.ErrNoRows {
		return &pb.GetPhaseOutcomeSummaryByJobPhaseResponse{
			Success: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get phase outcome summary by job phase: %w", err)
	}

	return &pb.GetPhaseOutcomeSummaryByJobPhaseResponse{
		PhaseOutcomeSummary: summary,
		Success:             true,
	}, nil
}

// ListByJob retrieves all phase outcome summaries for a given job
func (r *PostgresPhaseOutcomeSummaryRepository) ListByJob(
	ctx context.Context,
	req *pb.ListPhaseOutcomeSummarysByJobRequest,
) (*pb.ListPhaseOutcomeSummarysByJobResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	query := `
		SELECT
			pos.id, pos.job_phase_id, pos.job_id, pos.summary_type,
			pos.phase_determination, pos.scoring_method, pos.summary_score,
			pos.total_criteria_count, pos.pass_count, pos.fail_count,
			pos.conditional_count, pos.deferred_count, pos.na_count,
			pos.narrative, pos.issued_by, pos.issued_date,
			pos.supersedes_id, pos.active, pos.date_created, pos.date_modified
		FROM phase_outcome_summary pos
		WHERE pos.job_id = $1 AND pos.active = true
		ORDER BY pos.date_created DESC
	`

	rows, err := r.db.QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to list phase outcome summaries by job: %w", err)
	}
	defer rows.Close()

	summaries, err := scanPhaseOutcomeSummaryRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.ListPhaseOutcomeSummarysByJobResponse{
		PhaseOutcomeSummarys: summaries,
		Success:              true,
	}, nil
}

func scanPOSFields(scanFn func(dest ...any) error) (
	id string, jobPhaseID string, jobID string,
	summaryType string, phaseDetermination string, scoringMethod string,
	summaryScore sql.NullFloat64, totalCriteriaCount int32,
	passCount int32, failCount int32, conditionalCount int32,
	deferredCount int32, naCount int32, narrative sql.NullString,
	issuedBy string, issuedDate sql.NullInt64,
	supersedesId sql.NullString, active bool,
	dateCreated sql.NullInt64, dateModified sql.NullInt64, err error,
) {
	err = scanFn(
		&id, &jobPhaseID, &jobID, &summaryType,
		&phaseDetermination, &scoringMethod, &summaryScore,
		&totalCriteriaCount, &passCount, &failCount,
		&conditionalCount, &deferredCount, &naCount,
		&narrative, &issuedBy, &issuedDate,
		&supersedesId, &active, &dateCreated, &dateModified,
	)
	return
}

func scanPhaseOutcomeSummaryRows(rows *sql.Rows) ([]*pb.PhaseOutcomeSummary, error) {
	var summaries []*pb.PhaseOutcomeSummary
	for rows.Next() {
		id, jobPhaseID, jobID, summaryType, phaseDetermination, scoringMethod,
			summaryScore, totalCriteriaCount, passCount, failCount, conditionalCount,
			deferredCount, naCount, narrative, issuedBy, issuedDate,
			supersedesId, active, dateCreated, dateModified, err := scanPOSFields(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan phase outcome summary row: %w", err)
		}
		summaries = append(summaries, buildPhaseOutcomeSummary(
			id, jobPhaseID, jobID, summaryType, phaseDetermination, scoringMethod,
			summaryScore, totalCriteriaCount, passCount, failCount, conditionalCount,
			deferredCount, naCount, narrative, issuedBy, issuedDate,
			supersedesId, active, dateCreated, dateModified))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating phase outcome summary rows: %w", err)
	}
	return summaries, nil
}

func scanPhaseOutcomeSummaryRowWithTotal(rows *sql.Rows) (*pb.PhaseOutcomeSummary, int64, error) {
	var (
		id                 string
		jobPhaseID         string
		jobID              string
		summaryType        string
		phaseDetermination string
		scoringMethod      string
		summaryScore       sql.NullFloat64
		totalCriteriaCount int32
		passCount          int32
		failCount          int32
		conditionalCount   int32
		deferredCount      int32
		naCount            int32
		narrative          sql.NullString
		issuedBy           string
		issuedDate         sql.NullInt64
		supersedesId       sql.NullString
		active             bool
		dateCreated        sql.NullInt64
		dateModified       sql.NullInt64
		total              int64
	)

	err := rows.Scan(
		&id, &jobPhaseID, &jobID, &summaryType,
		&phaseDetermination, &scoringMethod, &summaryScore,
		&totalCriteriaCount, &passCount, &failCount,
		&conditionalCount, &deferredCount, &naCount,
		&narrative, &issuedBy, &issuedDate,
		&supersedesId, &active, &dateCreated, &dateModified, &total,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to scan phase outcome summary row: %w", err)
	}

	return buildPhaseOutcomeSummary(
		id, jobPhaseID, jobID, summaryType, phaseDetermination, scoringMethod,
		summaryScore, totalCriteriaCount, passCount, failCount, conditionalCount,
		deferredCount, naCount, narrative, issuedBy, issuedDate,
		supersedesId, active, dateCreated, dateModified), total, nil
}

func scanPhaseOutcomeSummarySingleRow(row *sql.Row) (*pb.PhaseOutcomeSummary, error) {
	id, jobPhaseID, jobID, summaryType, phaseDetermination, scoringMethod,
		summaryScore, totalCriteriaCount, passCount, failCount, conditionalCount,
		deferredCount, naCount, narrative, issuedBy, issuedDate,
		supersedesId, active, dateCreated, dateModified, err := scanPOSFields(row.Scan)
	if err != nil {
		return nil, err
	}
	return buildPhaseOutcomeSummary(
		id, jobPhaseID, jobID, summaryType, phaseDetermination, scoringMethod,
		summaryScore, totalCriteriaCount, passCount, failCount, conditionalCount,
		deferredCount, naCount, narrative, issuedBy, issuedDate,
		supersedesId, active, dateCreated, dateModified), nil
}

func buildPhaseOutcomeSummary(
	id string, jobPhaseID string, jobID string,
	summaryType string, phaseDetermination string, scoringMethod string,
	summaryScore sql.NullFloat64, totalCriteriaCount int32,
	passCount int32, failCount int32, conditionalCount int32,
	deferredCount int32, naCount int32, narrative sql.NullString,
	issuedBy string, issuedDate sql.NullInt64,
	supersedesId sql.NullString, active bool,
	dateCreated sql.NullInt64, dateModified sql.NullInt64,
) *pb.PhaseOutcomeSummary {
	summary := &pb.PhaseOutcomeSummary{
		Id:                 id,
		Active:             active,
		JobPhaseId:         jobPhaseID,
		JobId:              jobID,
		SummaryType:        enumspb.SummaryType(enumspb.SummaryType_value[summaryType]),
		PhaseDetermination: enumspb.OverallDetermination(enumspb.OverallDetermination_value[phaseDetermination]),
		ScoringMethod:      enumspb.ScoringMethod(enumspb.ScoringMethod_value[scoringMethod]),
		TotalCriteriaCount: totalCriteriaCount,
		PassCount:          passCount,
		FailCount:          failCount,
		ConditionalCount:   conditionalCount,
		DeferredCount:      deferredCount,
		NaCount:            naCount,
		IssuedBy:           issuedBy,
	}

	if summaryScore.Valid {
		summary.SummaryScore = &summaryScore.Float64
	}
	if narrative.Valid {
		summary.Narrative = &narrative.String
	}
	if issuedDate.Valid {
		summary.IssuedDate = &issuedDate.Int64
	}
	if supersedesId.Valid {
		summary.SupersedesId = &supersedesId.String
	}
	if dateCreated.Valid {
		summary.DateCreated = &dateCreated.Int64
		dcStr := time.UnixMilli(dateCreated.Int64).Format(time.RFC3339)
		summary.DateCreatedString = &dcStr
	}
	if dateModified.Valid {
		summary.DateModified = &dateModified.Int64
		dmStr := time.UnixMilli(dateModified.Int64).Format(time.RFC3339)
		summary.DateModifiedString = &dmStr
	}

	return summary
}
