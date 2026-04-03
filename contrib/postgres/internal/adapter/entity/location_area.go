package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.LocationArea, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres location_area repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresLocationAreaRepository(dbOps, tableName), nil
	})
}

// PostgresLocationAreaRepository implements location area CRUD operations using PostgreSQL
type PostgresLocationAreaRepository struct {
	locationareapb.UnimplementedLocationAreaDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresLocationAreaRepository creates a new PostgreSQL location area repository
func NewPostgresLocationAreaRepository(dbOps interfaces.DatabaseOperation, tableName string) locationareapb.LocationAreaDomainServiceServer {
	if tableName == "" {
		tableName = "location_area" // default fallback
	}

	return &PostgresLocationAreaRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLocationArea creates a new location area using common PostgreSQL operations
func (r *PostgresLocationAreaRepository) CreateLocationArea(ctx context.Context, req *locationareapb.CreateLocationAreaRequest) (*locationareapb.CreateLocationAreaResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("location area data is required")
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
		return nil, fmt.Errorf("failed to create location area: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	locationArea := &locationareapb.LocationArea{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, locationArea); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationareapb.CreateLocationAreaResponse{
		Data: []*locationareapb.LocationArea{locationArea},
	}, nil
}

// ReadLocationArea retrieves a location area by ID
func (r *PostgresLocationAreaRepository) ReadLocationArea(ctx context.Context, req *locationareapb.ReadLocationAreaRequest) (*locationareapb.ReadLocationAreaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location area ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read location area: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	locationArea := &locationareapb.LocationArea{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, locationArea); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationareapb.ReadLocationAreaResponse{
		Data: []*locationareapb.LocationArea{locationArea},
	}, nil
}

// UpdateLocationArea updates a location area using common PostgreSQL operations
func (r *PostgresLocationAreaRepository) UpdateLocationArea(ctx context.Context, req *locationareapb.UpdateLocationAreaRequest) (*locationareapb.UpdateLocationAreaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location area ID is required")
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
		return nil, fmt.Errorf("failed to update location area: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	locationArea := &locationareapb.LocationArea{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, locationArea); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationareapb.UpdateLocationAreaResponse{
		Data: []*locationareapb.LocationArea{locationArea},
	}, nil
}

// DeleteLocationArea soft-deletes a location area using common PostgreSQL operations
func (r *PostgresLocationAreaRepository) DeleteLocationArea(ctx context.Context, req *locationareapb.DeleteLocationAreaRequest) (*locationareapb.DeleteLocationAreaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location area ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete location area: %w", err)
	}

	return &locationareapb.DeleteLocationAreaResponse{
		Success: true,
	}, nil
}

// ListLocationAreas lists location areas using common PostgreSQL operations
func (r *PostgresLocationAreaRepository) ListLocationAreas(ctx context.Context, req *locationareapb.ListLocationAreasRequest) (*locationareapb.ListLocationAreasResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list location areas: %w", err)
	}

	var locationAreas []*locationareapb.LocationArea
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		locationArea := &locationareapb.LocationArea{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, locationArea); err != nil {
			continue
		}
		locationAreas = append(locationAreas, locationArea)
	}

	return &locationareapb.ListLocationAreasResponse{
		Data: locationAreas,
	}, nil
}

// GetLocationAreaListPageData retrieves location areas with filtering, sorting, searching, and pagination
func (r *PostgresLocationAreaRepository) GetLocationAreaListPageData(
	ctx context.Context,
	req *locationareapb.GetLocationAreaListPageDataRequest,
) (*locationareapb.GetLocationAreaListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get location area list page data request is required")
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

	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				id,
				name,
				description,
				active,
				date_created,
				date_modified
			FROM ` + r.tableName + `
			WHERE active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   name ILIKE $1 OR
				   description ILIKE $1)
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

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query location area list page data: %w", err)
	}
	defer rows.Close()

	var locationAreas []*locationareapb.LocationArea
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			name         string
			description  string
			active       bool
			dateCreated  *time.Time
			dateModified *time.Time
			total        int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location area row: %w", err)
		}

		totalCount = total

		locationArea := &locationareapb.LocationArea{
			Id:          id,
			Name:        name,
			Description: description,
			Active:      active,
		}

		if dateCreated != nil && !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			locationArea.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			locationArea.DateCreatedString = &dcStr
		}
		if dateModified != nil && !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			locationArea.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			locationArea.DateModifiedString = &dmStr
		}

		locationAreas = append(locationAreas, locationArea)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating location area rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &locationareapb.GetLocationAreaListPageDataResponse{
		LocationAreaList: locationAreas,
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

// GetLocationAreaItemPageData retrieves a single location area by ID
func (r *PostgresLocationAreaRepository) GetLocationAreaItemPageData(
	ctx context.Context,
	req *locationareapb.GetLocationAreaItemPageDataRequest,
) (*locationareapb.GetLocationAreaItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get location area item page data request is required")
	}
	if req.LocationAreaId == "" {
		return nil, fmt.Errorf("location area ID is required")
	}

	query := `
		SELECT
			id,
			name,
			description,
			active,
			date_created,
			date_modified
		FROM ` + r.tableName + `
		WHERE id = $1
		LIMIT 1;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.LocationAreaId)

	var (
		id           string
		name         string
		description  string
		active       bool
		dateCreated  *time.Time
		dateModified *time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("location area with ID '%s' not found", req.LocationAreaId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query location area item page data: %w", err)
	}

	locationArea := &locationareapb.LocationArea{
		Id:          id,
		Name:        name,
		Description: description,
		Active:      active,
	}

	if dateCreated != nil && !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		locationArea.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		locationArea.DateCreatedString = &dcStr
	}
	if dateModified != nil && !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		locationArea.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		locationArea.DateModifiedString = &dmStr
	}

	return &locationareapb.GetLocationAreaItemPageDataResponse{
		LocationArea: locationArea,
		Success:      true,
	}, nil
}

// NewLocationAreaRepository creates a new PostgreSQL location area repository (old-style constructor)
func NewLocationAreaRepository(db *sql.DB, tableName string) locationareapb.LocationAreaDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresLocationAreaRepository(dbOps, tableName)
}
