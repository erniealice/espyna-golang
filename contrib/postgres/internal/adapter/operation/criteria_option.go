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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CriteriaOption, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres criteria_option repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCriteriaOptionRepository(dbOps, tableName), nil
	})
}

// PostgresCriteriaOptionRepository implements criteria_option CRUD operations using PostgreSQL
type PostgresCriteriaOptionRepository struct {
	pb.UnimplementedCriteriaOptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCriteriaOptionRepository creates a new PostgreSQL criteria_option repository
func NewPostgresCriteriaOptionRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.CriteriaOptionDomainServiceServer {
	if tableName == "" {
		tableName = "criteria_option"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCriteriaOptionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCriteriaOption creates a new criteria_option record
func (r *PostgresCriteriaOptionRepository) CreateCriteriaOption(ctx context.Context, req *pb.CreateCriteriaOptionRequest) (*pb.CreateCriteriaOptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("criteria option data is required")
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
		return nil, fmt.Errorf("failed to create criteria option: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	option := &pb.CriteriaOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, option); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateCriteriaOptionResponse{
		Success: true,
		Data:    []*pb.CriteriaOption{option},
	}, nil
}

// ReadCriteriaOption retrieves a criteria_option record by ID
func (r *PostgresCriteriaOptionRepository) ReadCriteriaOption(ctx context.Context, req *pb.ReadCriteriaOptionRequest) (*pb.ReadCriteriaOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria option ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read criteria option: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	option := &pb.CriteriaOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, option); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadCriteriaOptionResponse{
		Success: true,
		Data:    []*pb.CriteriaOption{option},
	}, nil
}

// UpdateCriteriaOption updates a criteria_option record
func (r *PostgresCriteriaOptionRepository) UpdateCriteriaOption(ctx context.Context, req *pb.UpdateCriteriaOptionRequest) (*pb.UpdateCriteriaOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria option ID is required")
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
		return nil, fmt.Errorf("failed to update criteria option: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	option := &pb.CriteriaOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, option); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateCriteriaOptionResponse{
		Success: true,
		Data:    []*pb.CriteriaOption{option},
	}, nil
}

// DeleteCriteriaOption deletes a criteria_option record (soft delete)
func (r *PostgresCriteriaOptionRepository) DeleteCriteriaOption(ctx context.Context, req *pb.DeleteCriteriaOptionRequest) (*pb.DeleteCriteriaOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria option ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete criteria option: %w", err)
	}

	return &pb.DeleteCriteriaOptionResponse{
		Success: true,
	}, nil
}

// ListCriteriaOptions lists criteria_option records with optional filters
func (r *PostgresCriteriaOptionRepository) ListCriteriaOptions(ctx context.Context, req *pb.ListCriteriaOptionsRequest) (*pb.ListCriteriaOptionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria options: %w", err)
	}

	var options []*pb.CriteriaOption
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal criteria_option row: %v", err)
			continue
		}

		option := &pb.CriteriaOption{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, option); err != nil {
			log.Printf("WARN: protojson unmarshal criteria_option: %v", err)
			continue
		}
		options = append(options, option)
	}

	return &pb.ListCriteriaOptionsResponse{
		Success: true,
		Data:    options,
	}, nil
}

var criteriaOptionSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "outcome_criteria_id",
	"option_label", "option_key", "display_order", "severity",
}

// GetCriteriaOptionListPageData retrieves criteria options with pagination, filtering, sorting, and search
func (r *PostgresCriteriaOptionRepository) GetCriteriaOptionListPageData(
	ctx context.Context,
	req *pb.GetCriteriaOptionListPageDataRequest,
) (*pb.GetCriteriaOptionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get criteria option list page data request is required")
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
	// whitelist + fallback are unprefixed. Default preserves display_order ASC.
	orderByClause, err := postgresCore.BuildOrderBy(criteriaOptionSortableSQLCols, req.GetSort(), "display_order ASC")
	if err != nil {
		return nil, err
	}

	query := `
		WITH enriched AS (
			SELECT
				co.id,
				co.date_created,
				co.date_modified,
				co.active,
				co.outcome_criteria_id,
				co.option_label,
				co.option_key,
				co.display_order,
				co.severity
			FROM criteria_option co
			WHERE co.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       co.option_label ILIKE $1)
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
		return nil, fmt.Errorf("failed to query criteria option list page data: %w", err)
	}
	defer rows.Close()

	var options []*pb.CriteriaOption
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			outcomeCriteriaID string
			optionLabel       string
			optionKey         string
			displayOrder      int32
			severity          *int32
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&outcomeCriteriaID,
			&optionLabel,
			&optionKey,
			&displayOrder,
			&severity,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan criteria option row: %w", err)
		}

		totalCount = total

		option := &pb.CriteriaOption{
			Id:                id,
			Active:            active,
			OutcomeCriteriaId: outcomeCriteriaID,
			OptionLabel:       optionLabel,
			OptionKey:         optionKey,
			DisplayOrder:      displayOrder,
		}

		if severity != nil {
			option.Severity = severity
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			option.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			option.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			option.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			option.DateModifiedString = &dmStr
		}

		options = append(options, option)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating criteria option rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetCriteriaOptionListPageDataResponse{
		CriteriaOptionList: options,
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

// GetCriteriaOptionItemPageData retrieves a single criteria option with enriched data
func (r *PostgresCriteriaOptionRepository) GetCriteriaOptionItemPageData(
	ctx context.Context,
	req *pb.GetCriteriaOptionItemPageDataRequest,
) (*pb.GetCriteriaOptionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get criteria option item page data request is required")
	}
	if req.CriteriaOptionId == "" {
		return nil, fmt.Errorf("criteria option ID is required")
	}

	query := `
		SELECT
			co.id,
			co.date_created,
			co.date_modified,
			co.active,
			co.outcome_criteria_id,
			co.option_label,
			co.option_key,
			co.display_order,
			co.severity
		FROM criteria_option co
		WHERE co.id = $1 AND co.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.CriteriaOptionId)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		outcomeCriteriaID string
		optionLabel       string
		optionKey         string
		displayOrder      int32
		severity          *int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&outcomeCriteriaID,
		&optionLabel,
		&optionKey,
		&displayOrder,
		&severity,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("criteria option with ID '%s' not found", req.CriteriaOptionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query criteria option item page data: %w", err)
	}

	option := &pb.CriteriaOption{
		Id:                id,
		Active:            active,
		OutcomeCriteriaId: outcomeCriteriaID,
		OptionLabel:       optionLabel,
		OptionKey:         optionKey,
		DisplayOrder:      displayOrder,
	}

	if severity != nil {
		option.Severity = severity
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		option.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		option.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		option.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		option.DateModifiedString = &dmStr
	}

	return &pb.GetCriteriaOptionItemPageDataResponse{
		CriteriaOption: option,
		Success:        true,
	}, nil
}

// ListByCriteria retrieves all options for a given outcome criteria, ordered by display_order ASC
func (r *PostgresCriteriaOptionRepository) ListByCriteria(
	ctx context.Context,
	req *pb.ListCriteriaOptionsByCriteriaRequest,
) (*pb.ListCriteriaOptionsByCriteriaResponse, error) {
	if req == nil || req.OutcomeCriteriaId == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	query := `
		SELECT
			co.id,
			co.date_created,
			co.date_modified,
			co.active,
			co.outcome_criteria_id,
			co.option_label,
			co.option_key,
			co.display_order,
			co.severity
		FROM criteria_option co
		WHERE co.outcome_criteria_id = $1 AND co.active = true
		ORDER BY co.display_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.OutcomeCriteriaId)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria options by criteria: %w", err)
	}
	defer rows.Close()

	var options []*pb.CriteriaOption
	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			outcomeCriteriaID string
			optionLabel       string
			optionKey         string
			displayOrder      int32
			severity          *int32
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&outcomeCriteriaID,
			&optionLabel,
			&optionKey,
			&displayOrder,
			&severity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan criteria option row: %w", err)
		}

		option := &pb.CriteriaOption{
			Id:                id,
			Active:            active,
			OutcomeCriteriaId: outcomeCriteriaID,
			OptionLabel:       optionLabel,
			OptionKey:         optionKey,
			DisplayOrder:      displayOrder,
		}

		if severity != nil {
			option.Severity = severity
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			option.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			option.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			option.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			option.DateModifiedString = &dmStr
		}

		options = append(options, option)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating criteria option rows: %w", err)
	}

	return &pb.ListCriteriaOptionsByCriteriaResponse{
		CriteriaOptions: options,
		Success:         true,
	}, nil
}
