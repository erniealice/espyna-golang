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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobOutcomeSummary, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_outcome_summary repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobOutcomeSummaryRepository(dbOps, tableName), nil
	})
}

// PostgresJobOutcomeSummaryRepository implements job_outcome_summary CRUD operations using PostgreSQL
type PostgresJobOutcomeSummaryRepository struct {
	pb.UnimplementedJobOutcomeSummaryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobOutcomeSummaryRepository creates a new PostgreSQL job_outcome_summary repository
func NewPostgresJobOutcomeSummaryRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobOutcomeSummaryDomainServiceServer {
	if tableName == "" {
		tableName = "job_outcome_summary"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobOutcomeSummaryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobOutcomeSummary creates a new job_outcome_summary record
func (r *PostgresJobOutcomeSummaryRepository) CreateJobOutcomeSummary(ctx context.Context, req *pb.CreateJobOutcomeSummaryRequest) (*pb.CreateJobOutcomeSummaryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job outcome summary data is required")
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
		return nil, fmt.Errorf("failed to create job outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.JobOutcomeSummary{summary},
	}, nil
}

// ReadJobOutcomeSummary retrieves a job_outcome_summary record by ID
func (r *PostgresJobOutcomeSummaryRepository) ReadJobOutcomeSummary(ctx context.Context, req *pb.ReadJobOutcomeSummaryRequest) (*pb.ReadJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.JobOutcomeSummary{summary},
	}, nil
}

// UpdateJobOutcomeSummary updates a job_outcome_summary record
func (r *PostgresJobOutcomeSummaryRepository) UpdateJobOutcomeSummary(ctx context.Context, req *pb.UpdateJobOutcomeSummaryRequest) (*pb.UpdateJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
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
		return nil, fmt.Errorf("failed to update job outcome summary: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	summary := &pb.JobOutcomeSummary{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobOutcomeSummaryResponse{
		Success: true,
		Data:    []*pb.JobOutcomeSummary{summary},
	}, nil
}

// DeleteJobOutcomeSummary deletes a job_outcome_summary record (soft delete)
func (r *PostgresJobOutcomeSummaryRepository) DeleteJobOutcomeSummary(ctx context.Context, req *pb.DeleteJobOutcomeSummaryRequest) (*pb.DeleteJobOutcomeSummaryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job outcome summary: %w", err)
	}

	return &pb.DeleteJobOutcomeSummaryResponse{
		Success: true,
	}, nil
}

// ListJobOutcomeSummarys lists job_outcome_summary records with optional filters
func (r *PostgresJobOutcomeSummaryRepository) ListJobOutcomeSummarys(ctx context.Context, req *pb.ListJobOutcomeSummarysRequest) (*pb.ListJobOutcomeSummarysResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job outcome summarys: %w", err)
	}

	var summaries []*pb.JobOutcomeSummary
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_outcome_summary row: %v", err)
			continue
		}

		summary := &pb.JobOutcomeSummary{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, summary); err != nil {
			log.Printf("WARN: protojson unmarshal job_outcome_summary: %v", err)
			continue
		}
		summaries = append(summaries, summary)
	}

	return &pb.ListJobOutcomeSummarysResponse{
		Success: true,
		Data:    summaries,
	}, nil
}

// GetJobOutcomeSummaryListPageData retrieves job outcome summaries with pagination
func (r *PostgresJobOutcomeSummaryRepository) GetJobOutcomeSummaryListPageData(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryListPageDataRequest,
) (*pb.GetJobOutcomeSummaryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job outcome summary list page data request is required")
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

	sortField := "jos.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	josColumns := `
		jos.id, jos.job_id, jos.summary_type, jos.overall_determination,
		jos.scoring_method, jos.summary_score, jos.total_criteria_count,
		jos.pass_count, jos.fail_count, jos.conditional_count,
		jos.deferred_count, jos.na_count, jos.narrative,
		jos.issued_by, jos.issued_date, jos.valid_until_date,
		jos.supersedes_id, jos.attachment_ids, jos.active,
		jos.date_created, jos.date_modified
	`

	query := `
		WITH enriched AS (
			SELECT ` + josColumns + `
			FROM job_outcome_summary jos
			WHERE jos.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jos.narrative ILIKE $1)
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
		return nil, fmt.Errorf("failed to query job outcome summary list page data: %w", err)
	}
	defer rows.Close()

	var summaries []*pb.JobOutcomeSummary
	var totalCount int64

	for rows.Next() {
		summary, cnt, err := scanJobOutcomeSummaryRowWithTotal(rows)
		if err != nil {
			return nil, err
		}
		totalCount = cnt
		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job outcome summary rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobOutcomeSummaryListPageDataResponse{
		JobOutcomeSummaryList: summaries,
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

// GetJobOutcomeSummaryItemPageData retrieves a single job outcome summary with enriched data
func (r *PostgresJobOutcomeSummaryRepository) GetJobOutcomeSummaryItemPageData(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryItemPageDataRequest,
) (*pb.GetJobOutcomeSummaryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job outcome summary item page data request is required")
	}
	if req.JobOutcomeSummaryId == "" {
		return nil, fmt.Errorf("job outcome summary ID is required")
	}

	query := `
		SELECT
			jos.id, jos.job_id, jos.summary_type, jos.overall_determination,
			jos.scoring_method, jos.summary_score, jos.total_criteria_count,
			jos.pass_count, jos.fail_count, jos.conditional_count,
			jos.deferred_count, jos.na_count, jos.narrative,
			jos.issued_by, jos.issued_date, jos.valid_until_date,
			jos.supersedes_id, jos.attachment_ids, jos.active,
			jos.date_created, jos.date_modified
		FROM job_outcome_summary jos
		WHERE jos.id = $1 AND jos.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.JobOutcomeSummaryId)

	summary, err := scanJobOutcomeSummarySingleRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job outcome summary with ID '%s' not found", req.JobOutcomeSummaryId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job outcome summary item page data: %w", err)
	}

	return &pb.GetJobOutcomeSummaryItemPageDataResponse{
		JobOutcomeSummary: summary,
		Success:           true,
	}, nil
}

// GetByJob retrieves the latest job outcome summary for a given job
func (r *PostgresJobOutcomeSummaryRepository) GetByJob(
	ctx context.Context,
	req *pb.GetJobOutcomeSummaryByJobRequest,
) (*pb.GetJobOutcomeSummaryByJobResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	query := `
		SELECT
			jos.id, jos.job_id, jos.summary_type, jos.overall_determination,
			jos.scoring_method, jos.summary_score, jos.total_criteria_count,
			jos.pass_count, jos.fail_count, jos.conditional_count,
			jos.deferred_count, jos.na_count, jos.narrative,
			jos.issued_by, jos.issued_date, jos.valid_until_date,
			jos.supersedes_id, jos.attachment_ids, jos.active,
			jos.date_created, jos.date_modified
		FROM job_outcome_summary jos
		WHERE jos.job_id = $1 AND jos.active = true
		ORDER BY jos.date_created DESC
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, req.JobId)

	summary, err := scanJobOutcomeSummarySingleRow(row)
	if err == sql.ErrNoRows {
		return &pb.GetJobOutcomeSummaryByJobResponse{
			Success: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job outcome summary by job: %w", err)
	}

	return &pb.GetJobOutcomeSummaryByJobResponse{
		JobOutcomeSummary: summary,
		Success:           true,
	}, nil
}

// scanJobOutcomeSummaryRowWithTotal scans a row with a total count column
func scanJobOutcomeSummaryRowWithTotal(rows *sql.Rows) (*pb.JobOutcomeSummary, int64, error) {
	var (
		id                   string
		jobID                string
		summaryType          string
		overallDetermination string
		scoringMethod        string
		summaryScore         sql.NullFloat64
		totalCriteriaCount   int32
		passCount            int32
		failCount            int32
		conditionalCount     int32
		deferredCount        int32
		naCount              int32
		narrative            sql.NullString
		issuedBy             string
		issuedDate           sql.NullInt64
		validUntilDate       sql.NullString
		supersedesId         sql.NullString
		attachmentIdsStr     string
		active               bool
		dateCreated          sql.NullInt64
		dateModified         sql.NullInt64
		total                int64
	)

	err := rows.Scan(
		&id, &jobID, &summaryType, &overallDetermination,
		&scoringMethod, &summaryScore, &totalCriteriaCount,
		&passCount, &failCount, &conditionalCount,
		&deferredCount, &naCount, &narrative,
		&issuedBy, &issuedDate, &validUntilDate,
		&supersedesId, &attachmentIdsStr, &active,
		&dateCreated, &dateModified, &total,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to scan job outcome summary row: %w", err)
	}

	summary := buildJobOutcomeSummary(id, jobID, summaryType, overallDetermination,
		scoringMethod, summaryScore, totalCriteriaCount, passCount, failCount,
		conditionalCount, deferredCount, naCount, narrative, issuedBy, issuedDate,
		validUntilDate, supersedesId, attachmentIdsStr, active, dateCreated, dateModified)
	return summary, total, nil
}

// scanJobOutcomeSummarySingleRow scans a single sql.Row into a JobOutcomeSummary proto
func scanJobOutcomeSummarySingleRow(row *sql.Row) (*pb.JobOutcomeSummary, error) {
	var (
		id                   string
		jobID                string
		summaryType          string
		overallDetermination string
		scoringMethod        string
		summaryScore         sql.NullFloat64
		totalCriteriaCount   int32
		passCount            int32
		failCount            int32
		conditionalCount     int32
		deferredCount        int32
		naCount              int32
		narrative            sql.NullString
		issuedBy             string
		issuedDate           sql.NullInt64
		validUntilDate       sql.NullString
		supersedesId         sql.NullString
		attachmentIdsStr     string
		active               bool
		dateCreated          sql.NullInt64
		dateModified         sql.NullInt64
	)

	err := row.Scan(
		&id, &jobID, &summaryType, &overallDetermination,
		&scoringMethod, &summaryScore, &totalCriteriaCount,
		&passCount, &failCount, &conditionalCount,
		&deferredCount, &naCount, &narrative,
		&issuedBy, &issuedDate, &validUntilDate,
		&supersedesId, &attachmentIdsStr, &active,
		&dateCreated, &dateModified,
	)
	if err != nil {
		return nil, err
	}

	return buildJobOutcomeSummary(id, jobID, summaryType, overallDetermination,
		scoringMethod, summaryScore, totalCriteriaCount, passCount, failCount,
		conditionalCount, deferredCount, naCount, narrative, issuedBy, issuedDate,
		validUntilDate, supersedesId, attachmentIdsStr, active, dateCreated, dateModified), nil
}

func buildJobOutcomeSummary(
	id string, jobID string, summaryType string, overallDetermination string,
	scoringMethod string, summaryScore sql.NullFloat64, totalCriteriaCount int32,
	passCount int32, failCount int32, conditionalCount int32,
	deferredCount int32, naCount int32, narrative sql.NullString,
	issuedBy string, issuedDate sql.NullInt64, validUntilDate sql.NullString,
	supersedesId sql.NullString, attachmentIdsStr string, active bool,
	dateCreated sql.NullInt64, dateModified sql.NullInt64,
) *pb.JobOutcomeSummary {
	summary := &pb.JobOutcomeSummary{
		Id:                   id,
		Active:               active,
		JobId:                jobID,
		SummaryType:          enumspb.SummaryType(enumspb.SummaryType_value[summaryType]),
		OverallDetermination: enumspb.OverallDetermination(enumspb.OverallDetermination_value[overallDetermination]),
		ScoringMethod:        enumspb.ScoringMethod(enumspb.ScoringMethod_value[scoringMethod]),
		TotalCriteriaCount:   totalCriteriaCount,
		PassCount:            passCount,
		FailCount:            failCount,
		ConditionalCount:     conditionalCount,
		DeferredCount:        deferredCount,
		NaCount:              naCount,
		IssuedBy:             issuedBy,
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
	if validUntilDate.Valid {
		summary.ValidUntilDate = &validUntilDate.String
	}
	if supersedesId.Valid {
		summary.SupersedesId = &supersedesId.String
	}
	if attachmentIdsStr != "" {
		var ids []string
		if err := json.Unmarshal([]byte(attachmentIdsStr), &ids); err == nil {
			summary.AttachmentIds = ids
		}
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
