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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_threshold"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CriteriaThreshold, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres criteria_threshold repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCriteriaThresholdRepository(dbOps, tableName), nil
	})
}

// PostgresCriteriaThresholdRepository implements criteria_threshold CRUD operations using PostgreSQL
type PostgresCriteriaThresholdRepository struct {
	pb.UnimplementedCriteriaThresholdDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCriteriaThresholdRepository creates a new PostgreSQL criteria_threshold repository
func NewPostgresCriteriaThresholdRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.CriteriaThresholdDomainServiceServer {
	if tableName == "" {
		tableName = "criteria_threshold"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCriteriaThresholdRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCriteriaThreshold creates a new criteria_threshold record
func (r *PostgresCriteriaThresholdRepository) CreateCriteriaThreshold(ctx context.Context, req *pb.CreateCriteriaThresholdRequest) (*pb.CreateCriteriaThresholdResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("criteria threshold data is required")
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
		return nil, fmt.Errorf("failed to create criteria threshold: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.CriteriaThreshold{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateCriteriaThresholdResponse{
		Success: true,
		Data:    []*pb.CriteriaThreshold{item},
	}, nil
}

// ReadCriteriaThreshold retrieves a criteria_threshold record by ID
func (r *PostgresCriteriaThresholdRepository) ReadCriteriaThreshold(ctx context.Context, req *pb.ReadCriteriaThresholdRequest) (*pb.ReadCriteriaThresholdResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria threshold ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read criteria threshold: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.CriteriaThreshold{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadCriteriaThresholdResponse{
		Success: true,
		Data:    []*pb.CriteriaThreshold{item},
	}, nil
}

// UpdateCriteriaThreshold updates a criteria_threshold record
func (r *PostgresCriteriaThresholdRepository) UpdateCriteriaThreshold(ctx context.Context, req *pb.UpdateCriteriaThresholdRequest) (*pb.UpdateCriteriaThresholdResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria threshold ID is required")
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
		return nil, fmt.Errorf("failed to update criteria threshold: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	item := &pb.CriteriaThreshold{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateCriteriaThresholdResponse{
		Success: true,
		Data:    []*pb.CriteriaThreshold{item},
	}, nil
}

// DeleteCriteriaThreshold deletes a criteria_threshold record (soft delete)
func (r *PostgresCriteriaThresholdRepository) DeleteCriteriaThreshold(ctx context.Context, req *pb.DeleteCriteriaThresholdRequest) (*pb.DeleteCriteriaThresholdResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("criteria threshold ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete criteria threshold: %w", err)
	}

	return &pb.DeleteCriteriaThresholdResponse{
		Success: true,
	}, nil
}

// ListCriteriaThresholds lists criteria_threshold records with optional filters
func (r *PostgresCriteriaThresholdRepository) ListCriteriaThresholds(ctx context.Context, req *pb.ListCriteriaThresholdsRequest) (*pb.ListCriteriaThresholdsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria thresholds: %w", err)
	}

	var items []*pb.CriteriaThreshold
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal criteria_threshold row: %v", err)
			continue
		}

		item := &pb.CriteriaThreshold{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
			log.Printf("WARN: protojson unmarshal criteria_threshold: %v", err)
			continue
		}
		items = append(items, item)
	}

	return &pb.ListCriteriaThresholdsResponse{
		Success: true,
		Data:    items,
	}, nil
}

var criteriaThresholdSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "outcome_criteria_id",
	"threshold_role", "value",
}

// GetCriteriaThresholdListPageData retrieves criteria_thresholds with pagination, filtering, sorting, and search
func (r *PostgresCriteriaThresholdRepository) GetCriteriaThresholdListPageData(
	ctx context.Context,
	req *pb.GetCriteriaThresholdListPageDataRequest,
) (*pb.GetCriteriaThresholdListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get criteria threshold list page data request is required")
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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). Route the
	// caller-supplied sort column through core.BuildOrderBy so an unknown column
	// errors instead of being interpolated verbatim into ORDER BY. Default order
	// preserves the existing threshold_role ASC behavior.
	orderByClause, err := postgresCore.BuildOrderBy(criteriaThresholdSortableSQLCols, req.GetSort(), "threshold_role ASC")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				ct.id,
				ct.date_created,
				ct.date_modified,
				ct.active,
				ct.outcome_criteria_id,
				ct.threshold_role,
				ct.value
			FROM criteria_threshold ct
			WHERE ct.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       ct.outcome_criteria_id::text ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s
		LIMIT $2 OFFSET $3;
	`, orderByClause)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query criteria threshold list page data: %w", err)
	}
	defer rows.Close()

	var items []*pb.CriteriaThreshold
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			outcomeCriteriaID string
			thresholdRole     int32
			value             float64
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&outcomeCriteriaID,
			&thresholdRole,
			&value,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan criteria threshold row: %w", err)
		}

		totalCount = total

		item := &pb.CriteriaThreshold{
			Id:                id,
			Active:            active,
			OutcomeCriteriaId: outcomeCriteriaID,
			ThresholdRole:     enums.ThresholdRole(thresholdRole),
			Value:             value,
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
		return nil, fmt.Errorf("error iterating criteria threshold rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetCriteriaThresholdListPageDataResponse{
		CriteriaThresholdList: items,
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

// GetCriteriaThresholdItemPageData retrieves a single criteria_threshold with enriched data
func (r *PostgresCriteriaThresholdRepository) GetCriteriaThresholdItemPageData(
	ctx context.Context,
	req *pb.GetCriteriaThresholdItemPageDataRequest,
) (*pb.GetCriteriaThresholdItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get criteria threshold item page data request is required")
	}
	if req.CriteriaThresholdId == "" {
		return nil, fmt.Errorf("criteria threshold ID is required")
	}

	query := `
		SELECT
			ct.id,
			ct.date_created,
			ct.date_modified,
			ct.active,
			ct.outcome_criteria_id,
			ct.threshold_role,
			ct.value
		FROM criteria_threshold ct
		WHERE ct.id = $1 AND ct.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.CriteriaThresholdId)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		outcomeCriteriaID string
		thresholdRole     int32
		value             float64
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&outcomeCriteriaID,
		&thresholdRole,
		&value,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("criteria threshold with ID '%s' not found", req.CriteriaThresholdId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query criteria threshold item page data: %w", err)
	}

	item := &pb.CriteriaThreshold{
		Id:                id,
		Active:            active,
		OutcomeCriteriaId: outcomeCriteriaID,
		ThresholdRole:     enums.ThresholdRole(thresholdRole),
		Value:             value,
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

	return &pb.GetCriteriaThresholdItemPageDataResponse{
		CriteriaThreshold: item,
		Success:           true,
	}, nil
}

// ListByCriteria retrieves all criteria_thresholds for a given outcome_criteria
func (r *PostgresCriteriaThresholdRepository) ListByCriteria(
	ctx context.Context,
	req *pb.ListCriteriaThresholdsByCriteriaRequest,
) (*pb.ListCriteriaThresholdsByCriteriaResponse, error) {
	if req == nil || req.OutcomeCriteriaId == "" {
		return nil, fmt.Errorf("outcome criteria ID is required")
	}

	query := `
		SELECT
			ct.id,
			ct.date_created,
			ct.date_modified,
			ct.active,
			ct.outcome_criteria_id,
			ct.threshold_role,
			ct.value
		FROM criteria_threshold ct
		WHERE ct.outcome_criteria_id = $1 AND ct.active = true
		ORDER BY ct.threshold_role ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.OutcomeCriteriaId)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria thresholds by criteria: %w", err)
	}
	defer rows.Close()

	var items []*pb.CriteriaThreshold
	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			outcomeCriteriaID string
			thresholdRole     int32
			value             float64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&outcomeCriteriaID,
			&thresholdRole,
			&value,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan criteria threshold row: %w", err)
		}

		item := &pb.CriteriaThreshold{
			Id:                id,
			Active:            active,
			OutcomeCriteriaId: outcomeCriteriaID,
			ThresholdRole:     enums.ThresholdRole(thresholdRole),
			Value:             value,
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
		return nil, fmt.Errorf("error iterating criteria threshold rows: %w", err)
	}

	return &pb.ListCriteriaThresholdsByCriteriaResponse{
		CriteriaThresholds: items,
		Success:            true,
	}, nil
}

// NewCriteriaThresholdRepository creates a new PostgreSQL criteria_threshold repository (old-style constructor)
func NewCriteriaThresholdRepository(db *sql.DB, tableName string) pb.CriteriaThresholdDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCriteriaThresholdRepository(dbOps, tableName)
}
