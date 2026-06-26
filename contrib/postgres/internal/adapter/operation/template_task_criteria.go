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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TemplateTaskCriteria, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres template_task_criteria repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTemplateTaskCriteriaRepository(dbOps, tableName), nil
	})
}

// PostgresTemplateTaskCriteriaRepository implements template_task_criteria CRUD operations using PostgreSQL
type PostgresTemplateTaskCriteriaRepository struct {
	pb.UnimplementedTemplateTaskCriteriaDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTemplateTaskCriteriaRepository creates a new PostgreSQL template_task_criteria repository
func NewPostgresTemplateTaskCriteriaRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TemplateTaskCriteriaDomainServiceServer {
	if tableName == "" {
		tableName = "template_task_criteria"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresTemplateTaskCriteriaRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateTemplateTaskCriteria creates a new template_task_criteria record
func (r *PostgresTemplateTaskCriteriaRepository) CreateTemplateTaskCriteria(ctx context.Context, req *pb.CreateTemplateTaskCriteriaRequest) (*pb.CreateTemplateTaskCriteriaResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("template task criteria data is required")
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create template task criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ttc := &pb.TemplateTaskCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateTemplateTaskCriteriaResponse{
		Success: true,
		Data:    []*pb.TemplateTaskCriteria{ttc},
	}, nil
}

// ReadTemplateTaskCriteria retrieves a template_task_criteria record by ID
func (r *PostgresTemplateTaskCriteriaRepository) ReadTemplateTaskCriteria(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRequest) (*pb.ReadTemplateTaskCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template task criteria ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read template task criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ttc := &pb.TemplateTaskCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadTemplateTaskCriteriaResponse{
		Success: true,
		Data:    []*pb.TemplateTaskCriteria{ttc},
	}, nil
}

// UpdateTemplateTaskCriteria updates a template_task_criteria record
func (r *PostgresTemplateTaskCriteriaRepository) UpdateTemplateTaskCriteria(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRequest) (*pb.UpdateTemplateTaskCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template task criteria ID is required")
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update template task criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	ttc := &pb.TemplateTaskCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateTemplateTaskCriteriaResponse{
		Success: true,
		Data:    []*pb.TemplateTaskCriteria{ttc},
	}, nil
}

// DeleteTemplateTaskCriteria deletes a template_task_criteria record (soft delete)
func (r *PostgresTemplateTaskCriteriaRepository) DeleteTemplateTaskCriteria(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRequest) (*pb.DeleteTemplateTaskCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template task criteria ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete template task criteria: %w", err)
	}

	return &pb.DeleteTemplateTaskCriteriaResponse{
		Success: true,
	}, nil
}

// ListTemplateTaskCriterias lists template_task_criteria records with optional filters
func (r *PostgresTemplateTaskCriteriaRepository) ListTemplateTaskCriterias(ctx context.Context, req *pb.ListTemplateTaskCriteriasRequest) (*pb.ListTemplateTaskCriteriasResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list template task criterias: %w", err)
	}

	var ttcs []*pb.TemplateTaskCriteria
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal template_task_criteria row: %v", err)
			continue
		}

		ttc := &pb.TemplateTaskCriteria{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ttc); err != nil {
			log.Printf("WARN: protojson unmarshal template_task_criteria: %v", err)
			continue
		}
		ttcs = append(ttcs, ttc)
	}

	return &pb.ListTemplateTaskCriteriasResponse{
		Success: true,
		Data:    ttcs,
	}, nil
}

var templateTaskCriteriaSortableSQLCols = []string{
	"id", "date_created", "active", "job_template_task_id",
	"outcome_criteria_id", "sequence_order", "required_override",
}

// GetTemplateTaskCriteriaListPageData retrieves template task criterias with pagination
func (r *PostgresTemplateTaskCriteriaRepository) GetTemplateTaskCriteriaListPageData(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaListPageDataRequest,
) (*pb.GetTemplateTaskCriteriaListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get template task criteria list page data request is required")
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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The ORDER BY
	// runs against the outer `enriched e` projection (unprefixed cols), so the
	// whitelist + fallback are unprefixed. Default preserves sequence_order ASC.
	orderByClause, err := postgresCore.BuildOrderBy(templateTaskCriteriaSortableSQLCols, req.GetSort(), "sequence_order ASC")
	if err != nil {
		return nil, err
	}

	query := `
		WITH enriched AS (
			SELECT
				ttc.id,
				ttc.date_created,
				ttc.active,
				ttc.job_template_task_id,
				ttc.outcome_criteria_id,
				ttc.sequence_order,
				ttc.required_override
			FROM template_task_criteria ttc
			WHERE ttc.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       ttc.outcome_criteria_id::text ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query template task criteria list page data: %w", err)
	}
	defer rows.Close()

	var ttcs []*pb.TemplateTaskCriteria
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			active            bool
			jobTemplateTaskID string
			outcomeCriteriaID string
			sequenceOrder     int32
			requiredOverride  sql.NullBool
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&active,
			&jobTemplateTaskID,
			&outcomeCriteriaID,
			&sequenceOrder,
			&requiredOverride,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template task criteria row: %w", err)
		}

		totalCount = total

		ttc := &pb.TemplateTaskCriteria{
			Id:                id,
			Active:            active,
			JobTemplateTaskId: jobTemplateTaskID,
			OutcomeCriteriaId: outcomeCriteriaID,
			SequenceOrder:     sequenceOrder,
			RequiredOverride:  nilBoolPtr(requiredOverride),
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			ttc.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			ttc.DateCreatedString = &dcStr
		}

		ttcs = append(ttcs, ttc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating template task criteria rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetTemplateTaskCriteriaListPageDataResponse{
		TemplateTaskCriteriaList: ttcs,
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

// GetTemplateTaskCriteriaItemPageData retrieves a single template task criteria with enriched data
func (r *PostgresTemplateTaskCriteriaRepository) GetTemplateTaskCriteriaItemPageData(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaItemPageDataRequest,
) (*pb.GetTemplateTaskCriteriaItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get template task criteria item page data request is required")
	}
	if req.TemplateTaskCriteriaId == "" {
		return nil, fmt.Errorf("template task criteria ID is required")
	}

	query := `
		SELECT
			ttc.id,
			ttc.date_created,
			ttc.active,
			ttc.job_template_task_id,
			ttc.outcome_criteria_id,
			ttc.sequence_order,
			ttc.required_override
		FROM template_task_criteria ttc
		WHERE ttc.id = $1 AND ttc.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.TemplateTaskCriteriaId)

	var (
		id                string
		dateCreated       time.Time
		active            bool
		jobTemplateTaskID string
		outcomeCriteriaID string
		sequenceOrder     int32
		requiredOverride  sql.NullBool
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&active,
		&jobTemplateTaskID,
		&outcomeCriteriaID,
		&sequenceOrder,
		&requiredOverride,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template task criteria with ID '%s' not found", req.TemplateTaskCriteriaId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query template task criteria item page data: %w", err)
	}

	ttc := &pb.TemplateTaskCriteria{
		Id:                id,
		Active:            active,
		JobTemplateTaskId: jobTemplateTaskID,
		OutcomeCriteriaId: outcomeCriteriaID,
		SequenceOrder:     sequenceOrder,
		RequiredOverride:  nilBoolPtr(requiredOverride),
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		ttc.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		ttc.DateCreatedString = &dcStr
	}

	return &pb.GetTemplateTaskCriteriaItemPageDataResponse{
		TemplateTaskCriteria: ttc,
		Success:              true,
	}, nil
}

// ListByTemplateTask retrieves all template task criterias for a given job template task, ordered by sequence_order ASC
func (r *PostgresTemplateTaskCriteriaRepository) ListByTemplateTask(
	ctx context.Context,
	req *pb.ListTemplateTaskCriteriasByTemplateTaskRequest,
) (*pb.ListTemplateTaskCriteriasByTemplateTaskResponse, error) {
	if req == nil || req.JobTemplateTaskId == "" {
		return nil, fmt.Errorf("job template task ID is required")
	}

	query := `
		SELECT
			ttc.id,
			ttc.date_created,
			ttc.active,
			ttc.job_template_task_id,
			ttc.outcome_criteria_id,
			ttc.sequence_order,
			ttc.required_override
		FROM template_task_criteria ttc
		WHERE ttc.job_template_task_id = $1 AND ttc.active = true
		ORDER BY ttc.sequence_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.JobTemplateTaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to list template task criterias by template task: %w", err)
	}
	defer rows.Close()

	ttcs, err := scanTemplateTaskCriteriaRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.ListTemplateTaskCriteriasByTemplateTaskResponse{
		TemplateTaskCriterias: ttcs,
		Success:               true,
	}, nil
}

// ListByCriteria retrieves all template task criterias for a given outcome criteria
func (r *PostgresTemplateTaskCriteriaRepository) ListByCriteria(
	ctx context.Context,
	req *pb.ListTemplateTaskCriteriasByCriteriaRequest,
) (*pb.ListTemplateTaskCriteriasByCriteriaResponse, error) {
	if req == nil || req.OutcomeCriteriaId == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	query := `
		SELECT
			ttc.id,
			ttc.date_created,
			ttc.active,
			ttc.job_template_task_id,
			ttc.outcome_criteria_id,
			ttc.sequence_order,
			ttc.required_override
		FROM template_task_criteria ttc
		WHERE ttc.outcome_criteria_id = $1 AND ttc.active = true
		ORDER BY ttc.sequence_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.OutcomeCriteriaId)
	if err != nil {
		return nil, fmt.Errorf("failed to list template task criterias by criteria: %w", err)
	}
	defer rows.Close()

	ttcs, err := scanTemplateTaskCriteriaRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.ListTemplateTaskCriteriasByCriteriaResponse{
		TemplateTaskCriterias: ttcs,
		Success:               true,
	}, nil
}

// scanTemplateTaskCriteriaRows scans multiple rows into TemplateTaskCriteria protos
func scanTemplateTaskCriteriaRows(rows *sql.Rows) ([]*pb.TemplateTaskCriteria, error) {
	var ttcs []*pb.TemplateTaskCriteria
	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			active            bool
			jobTemplateTaskID string
			outcomeCriteriaID string
			sequenceOrder     int32
			requiredOverride  sql.NullBool
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&active,
			&jobTemplateTaskID,
			&outcomeCriteriaID,
			&sequenceOrder,
			&requiredOverride,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template task criteria row: %w", err)
		}

		ttc := &pb.TemplateTaskCriteria{
			Id:                id,
			Active:            active,
			JobTemplateTaskId: jobTemplateTaskID,
			OutcomeCriteriaId: outcomeCriteriaID,
			SequenceOrder:     sequenceOrder,
			RequiredOverride:  nilBoolPtr(requiredOverride),
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			ttc.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			ttc.DateCreatedString = &dcStr
		}

		ttcs = append(ttcs, ttc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating template task criteria rows: %w", err)
	}

	return ttcs, nil
}

func nilBoolPtr(nb sql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}
