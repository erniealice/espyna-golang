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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.OutcomeCriteria, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres outcome_criteria repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresOutcomeCriteriaRepository(dbOps, tableName), nil
	})
}

// PostgresOutcomeCriteriaRepository implements outcome_criteria CRUD operations using PostgreSQL
type PostgresOutcomeCriteriaRepository struct {
	pb.UnimplementedOutcomeCriteriaDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresOutcomeCriteriaRepository creates a new PostgreSQL outcome_criteria repository
func NewPostgresOutcomeCriteriaRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.OutcomeCriteriaDomainServiceServer {
	if tableName == "" {
		tableName = "outcome_criteria"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresOutcomeCriteriaRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateOutcomeCriteria creates a new outcome_criteria record
func (r *PostgresOutcomeCriteriaRepository) CreateOutcomeCriteria(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest) (*pb.CreateOutcomeCriteriaResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("outcome criteria data is required")
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
		return nil, fmt.Errorf("failed to create outcome criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.OutcomeCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateOutcomeCriteriaResponse{
		Success: true,
		Data:    []*pb.OutcomeCriteria{item},
	}, nil
}

// ReadOutcomeCriteria retrieves an outcome_criteria record by ID
func (r *PostgresOutcomeCriteriaRepository) ReadOutcomeCriteria(ctx context.Context, req *pb.ReadOutcomeCriteriaRequest) (*pb.ReadOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read outcome criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.OutcomeCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadOutcomeCriteriaResponse{
		Success: true,
		Data:    []*pb.OutcomeCriteria{item},
	}, nil
}

// UpdateOutcomeCriteria updates an outcome_criteria record
func (r *PostgresOutcomeCriteriaRepository) UpdateOutcomeCriteria(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest) (*pb.UpdateOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
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
		return nil, fmt.Errorf("failed to update outcome criteria: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.OutcomeCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateOutcomeCriteriaResponse{
		Success: true,
		Data:    []*pb.OutcomeCriteria{item},
	}, nil
}

// DeleteOutcomeCriteria deletes an outcome_criteria record (soft delete)
func (r *PostgresOutcomeCriteriaRepository) DeleteOutcomeCriteria(ctx context.Context, req *pb.DeleteOutcomeCriteriaRequest) (*pb.DeleteOutcomeCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete outcome criteria: %w", err)
	}

	return &pb.DeleteOutcomeCriteriaResponse{
		Success: true,
	}, nil
}

// ListOutcomeCriterias lists outcome_criteria records with optional filters
func (r *PostgresOutcomeCriteriaRepository) ListOutcomeCriterias(ctx context.Context, req *pb.ListOutcomeCriteriasRequest) (*pb.ListOutcomeCriteriasResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list outcome criterias: %w", err)
	}

	var items []*pb.OutcomeCriteria
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal outcome_criteria row: %v", err)
			continue
		}

		item := &pb.OutcomeCriteria{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
			log.Printf("WARN: protojson unmarshal outcome_criteria: %v", err)
			continue
		}
		items = append(items, item)
	}

	return &pb.ListOutcomeCriteriasResponse{
		Success: true,
		Data:    items,
	}, nil
}

// GetOutcomeCriteriaListPageData retrieves outcome_criterias with pagination, filtering, sorting, and search
func (r *PostgresOutcomeCriteriaRepository) GetOutcomeCriteriaListPageData(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaListPageDataRequest,
) (*pb.GetOutcomeCriteriaListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get outcome criteria list page data request is required")
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

	sortField := "oc.name"
	sortOrder := "ASC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				oc.id,
				oc.date_created,
				oc.date_modified,
				oc.active,
				oc.criteria_group_id,
				oc.version,
				oc.version_status,
				oc.scope,
				oc.name,
				oc.criteria_type
			FROM outcome_criteria oc
			WHERE oc.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       oc.name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query outcome criteria list page data: %w", err)
	}
	defer rows.Close()

	var items []*pb.OutcomeCriteria
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			criteriaGroupID string
			version         int32
			versionStatus   int32
			scope           int32
			name            string
			criteriaType    int32
			total           int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&criteriaGroupID,
			&version,
			&versionStatus,
			&scope,
			&name,
			&criteriaType,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outcome criteria row: %w", err)
		}

		totalCount = total

		item := &pb.OutcomeCriteria{
			Id:              id,
			Active:          active,
			CriteriaGroupId: criteriaGroupID,
			Version:         version,
			Name:            name,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			item.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			item.DateModifiedString = &dmStr
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outcome criteria rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetOutcomeCriteriaListPageDataResponse{
		OutcomeCriteriaList: items,
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

// GetOutcomeCriteriaItemPageData retrieves a single outcome_criteria with enriched data
func (r *PostgresOutcomeCriteriaRepository) GetOutcomeCriteriaItemPageData(
	ctx context.Context,
	req *pb.GetOutcomeCriteriaItemPageDataRequest,
) (*pb.GetOutcomeCriteriaItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get outcome criteria item page data request is required")
	}
	if req.OutcomeCriteriaId == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM outcome_criteria oc
		WHERE oc.id = $1 AND oc.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.OutcomeCriteriaId)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		criteriaGroupID string
		version         int32
		versionStatus   int32
		scope           int32
		name            string
		criteriaType    int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&criteriaGroupID,
		&version,
		&versionStatus,
		&scope,
		&name,
		&criteriaType,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("outcome criteria with ID '%s' not found", req.OutcomeCriteriaId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query outcome criteria item page data: %w", err)
	}

	item := &pb.OutcomeCriteria{
		Id:              id,
		Active:          active,
		CriteriaGroupId: criteriaGroupID,
		Version:         version,
		Name:            name,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		item.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		item.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		item.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		item.DateModifiedString = &dmStr
	}

	return &pb.GetOutcomeCriteriaItemPageDataResponse{
		OutcomeCriteria: item,
		Success:         true,
	}, nil
}

// ListByGroup retrieves all outcome_criteria for a given criteria group, ordered by version DESC
func (r *PostgresOutcomeCriteriaRepository) ListByGroup(
	ctx context.Context,
	req *pb.ListOutcomeCriteriasByGroupRequest,
) (*pb.ListOutcomeCriteriasByGroupResponse, error) {
	if req == nil || req.CriteriaGroupId == "" {
		return nil, fmt.Errorf("criteria group ID is required")
	}

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM outcome_criteria oc
		WHERE oc.criteria_group_id = $1 AND oc.active = true
		ORDER BY oc.version DESC
	`

	rows, err := r.db.QueryContext(ctx, query, req.CriteriaGroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to list outcome criterias by group: %w", err)
	}
	defer rows.Close()

	var items []*pb.OutcomeCriteria
	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			criteriaGroupID string
			version         int32
			versionStatus   int32
			scope           int32
			name            string
			criteriaType    int32
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&criteriaGroupID,
			&version,
			&versionStatus,
			&scope,
			&name,
			&criteriaType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outcome criteria row: %w", err)
		}

		item := &pb.OutcomeCriteria{
			Id:              id,
			Active:          active,
			CriteriaGroupId: criteriaGroupID,
			Version:         version,
			Name:            name,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			item.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			item.DateModifiedString = &dmStr
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outcome criteria rows: %w", err)
	}

	return &pb.ListOutcomeCriteriasByGroupResponse{
		OutcomeCriterias: items,
		Success:          true,
	}, nil
}

// GetCurrentPublished retrieves the current published outcome_criteria for a criteria group
func (r *PostgresOutcomeCriteriaRepository) GetCurrentPublished(
	ctx context.Context,
	req *pb.GetCurrentPublishedOutcomeCriteriaRequest,
) (*pb.GetCurrentPublishedOutcomeCriteriaResponse, error) {
	if req == nil || req.CriteriaGroupId == "" {
		return nil, fmt.Errorf("criteria group ID is required")
	}

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM outcome_criteria oc
		WHERE oc.criteria_group_id = $1 AND oc.version_status = 2 AND oc.active = true
		ORDER BY oc.version DESC
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, req.CriteriaGroupId)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		criteriaGroupID string
		version         int32
		versionStatus   int32
		scope           int32
		name            string
		criteriaType    int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&criteriaGroupID,
		&version,
		&versionStatus,
		&scope,
		&name,
		&criteriaType,
	)
	if err == sql.ErrNoRows {
		return &pb.GetCurrentPublishedOutcomeCriteriaResponse{
			Success: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query current published outcome criteria: %w", err)
	}

	item := &pb.OutcomeCriteria{
		Id:              id,
		Active:          active,
		CriteriaGroupId: criteriaGroupID,
		Version:         version,
		Name:            name,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		item.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		item.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		item.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		item.DateModifiedString = &dmStr
	}

	return &pb.GetCurrentPublishedOutcomeCriteriaResponse{
		OutcomeCriteria: item,
		Success:         true,
	}, nil
}

// ListByScope retrieves outcome_criterias filtered by scope, industry code, and workspace
func (r *PostgresOutcomeCriteriaRepository) ListByScope(
	ctx context.Context,
	req *pb.ListOutcomeCriteriasByScopeRequest,
) (*pb.ListOutcomeCriteriasByScopeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("list outcome criterias by scope request is required")
	}

	industryCode := ""
	if req.IndustryCode != nil {
		industryCode = *req.IndustryCode
	}
	workspaceID := ""
	if req.WorkspaceId != nil {
		workspaceID = *req.WorkspaceId
	}

	query := `
		SELECT
			oc.id,
			oc.date_created,
			oc.date_modified,
			oc.active,
			oc.criteria_group_id,
			oc.version,
			oc.version_status,
			oc.scope,
			oc.name,
			oc.criteria_type
		FROM outcome_criteria oc
		WHERE oc.scope = $1
		  AND ($2 = '' OR oc.industry_code = $2)
		  AND ($3 = '' OR oc.workspace_id = $3)
		  AND oc.active = true
		ORDER BY oc.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, int32(req.Scope), industryCode, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list outcome criterias by scope: %w", err)
	}
	defer rows.Close()

	var items []*pb.OutcomeCriteria
	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			criteriaGroupID string
			version         int32
			versionStatus   int32
			scope           int32
			name            string
			criteriaType    int32
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&criteriaGroupID,
			&version,
			&versionStatus,
			&scope,
			&name,
			&criteriaType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outcome criteria row: %w", err)
		}

		item := &pb.OutcomeCriteria{
			Id:              id,
			Active:          active,
			CriteriaGroupId: criteriaGroupID,
			Version:         version,
			Name:            name,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			item.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			item.DateModifiedString = &dmStr
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outcome criteria rows: %w", err)
	}

	return &pb.ListOutcomeCriteriasByScopeResponse{
		OutcomeCriterias: items,
		Success:          true,
	}, nil
}

// NewOutcomeCriteriaRepository creates a new PostgreSQL outcome_criteria repository (old-style constructor)
func NewOutcomeCriteriaRepository(db *sql.DB, tableName string) pb.OutcomeCriteriaDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresOutcomeCriteriaRepository(dbOps, tableName)
}
