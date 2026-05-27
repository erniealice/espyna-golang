//go:build mysql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Location, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql location repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLLocationRepository(dbOps, tableName), nil
	})
}

// MySQLLocationRepository implements location CRUD operations using MySQL 8.0+.
type MySQLLocationRepository struct {
	locationpb.UnimplementedLocationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLLocationRepository creates a new MySQL location repository.
func NewMySQLLocationRepository(dbOps interfaces.DatabaseOperation, tableName string) locationpb.LocationDomainServiceServer {
	if tableName == "" {
		tableName = "location"
	}
	return &MySQLLocationRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateLocation creates a new location using common MySQL operations.
func (r *MySQLLocationRepository) CreateLocation(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("location data is required")
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
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	location := &locationpb.Location{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationpb.CreateLocationResponse{
		Data: []*locationpb.Location{location},
	}, nil
}

// ReadLocation retrieves a location using common MySQL operations.
func (r *MySQLLocationRepository) ReadLocation(ctx context.Context, req *locationpb.ReadLocationRequest) (*locationpb.ReadLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read location: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	location := &locationpb.Location{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationpb.ReadLocationResponse{
		Data: []*locationpb.Location{location},
	}, nil
}

// UpdateLocation updates a location using common MySQL operations.
func (r *MySQLLocationRepository) UpdateLocation(ctx context.Context, req *locationpb.UpdateLocationRequest) (*locationpb.UpdateLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
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
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	location := &locationpb.Location{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationpb.UpdateLocationResponse{
		Data: []*locationpb.Location{location},
	}, nil
}

// DeleteLocation hard-deletes a location from the database.
func (r *MySQLLocationRepository) DeleteLocation(ctx context.Context, req *locationpb.DeleteLocationRequest) (*locationpb.DeleteLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete location: %w", err)
	}

	return &locationpb.DeleteLocationResponse{
		Success: true,
	}, nil
}

var locationSortableSQLCols = []string{
	"id", "active", "name", "address", "description", "timezone",
	"location_area_id", "workspace_id", "date_created", "date_modified",
}

var locationSortSpec = espynahttp.SortSpec{AllowedCols: locationSortableSQLCols}

// ListLocations lists locations using common MySQL operations.
func (r *MySQLLocationRepository) ListLocations(ctx context.Context, req *locationpb.ListLocationsRequest) (*locationpb.ListLocationsResponse, error) {
	if err := espynahttp.ValidateSortColumns(locationSortSpec, req.GetSort(), "location"); err != nil {
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
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}

	var locations []*locationpb.Location
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		location := &locationpb.Location{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, location); err != nil {
			continue
		}
		locations = append(locations, location)
	}

	return &locationpb.ListLocationsResponse{
		Data: locations,
	}, nil
}

// GetLocationListPageData retrieves locations with attributes using MySQL CTE.
//
// Dialect translation from postgres gold standard:
//   - $1/$2,... → ? (MySQL positional placeholders, args in same left-to-right order)
//   - jsonb_agg ... FILTER (WHERE la.id IS NOT NULL) → JSON_ARRAYAGG (inner WHERE already filters nulls)
//   - jsonb_build_object → JSON_OBJECT
//   - COALESCE(..., '[]'::jsonb) → COALESCE(..., JSON_ARRAY())
//   - $1::text IS NULL OR $1::text = ” → ? IS NULL OR ? = ” (two ? args — same workspaceID passed twice)
//   - ILIKE → LIKE; active = true → active = 1 (no active filter in this query)
//
// CRITICAL: workspace_id IS NULL check allows global/non-tenanted locations to appear.
func (r *MySQLLocationRepository) GetLocationListPageData(
	ctx context.Context,
	req *locationpb.GetLocationListPageDataRequest,
) (*locationpb.GetLocationListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
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

	sortField := "name"
	sortOrder := "ASC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Build filter/search WHERE clauses.
	// First arg is workspaceID (passed twice for the IS NULL / = '' check); filter builder starts at idx 2.
	searchFields := []string{"l.name", "l.address"}
	filterClauses, filterArgs, _ := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	// Dialect: $1::text IS NULL OR $1::text = '' OR l.workspace_id = $1
	// MySQL: (? IS NULL OR ? = '' OR l.workspace_id = ?) — three ? bindings for the same workspaceID.
	whereSQL := "WHERE (? IS NULL OR ? = '' OR l.workspace_id = ?)"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	queryArgs := []any{workspaceID, workspaceID, workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

	query := fmt.Sprintf(`
		WITH location_attributes_agg AS (
			SELECT
				la.location_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', la.id,
						'location_id', la.location_id,
						'attribute_id', la.attribute_id,
						'value', la.value
					)
				) AS attributes
			FROM location_attribute la
			GROUP BY la.location_id
		),
		enriched AS (
			SELECT
				l.id,
				l.name,
				l.address,
				l.active,
				l.date_created,
				l.date_modified,
				COALESCE(l.timezone, 'Asia/Manila') AS timezone,
				l.location_area_id,
				COALESCE(la2.name, '') AS location_area_name,
				COALESCE(laa.attributes, JSON_ARRAY()) AS location_attributes
			FROM location l
			LEFT JOIN location_attributes_agg laa ON l.id = laa.location_id
			LEFT JOIN location_area la2 ON l.location_area_id = la2.id
			%s
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT e.*, c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT ? OFFSET ?;
	`, whereSQL, sortField, sortOrder)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var locations []*locationpb.Location
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			name             string
			address          *string
			active           bool
			dateCreated      time.Time
			dateModified     time.Time
			timezone         string
			locationAreaID   *string
			locationAreaName string
			attributesJSON   []byte
			total            int64
		)

		err := rows.Scan(
			&id, &name, &address,
			&active, &dateCreated, &dateModified,
			&timezone, &locationAreaID, &locationAreaName, &attributesJSON, &total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}

		totalCount = total

		location := &locationpb.Location{
			Id:     id,
			Name:   name,
			Active: active,
		}

		if address != nil {
			location.Address = *address
		}
		location.Timezone = &timezone
		if locationAreaID != nil {
			location.LocationAreaId = locationAreaID
		}
		if locationAreaName != "" {
			location.Description = &locationAreaName
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			location.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			location.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			location.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			location.DateModifiedString = &dmStr
		}

		var attributes []*locationattributepb.LocationAttribute
		if len(attributesJSON) > 0 && string(attributesJSON) != "[]" {
			var attrMaps []map[string]interface{}
			if err := json.Unmarshal(attributesJSON, &attrMaps); err == nil {
				for _, attrMap := range attrMaps {
					attr := &locationattributepb.LocationAttribute{}
					if attrId, ok := attrMap["id"].(string); ok {
						attr.Id = attrId
					}
					if locationID, ok := attrMap["location_id"].(string); ok {
						attr.LocationId = locationID
					}
					if attrID, ok := attrMap["attribute_id"].(string); ok {
						attr.AttributeId = attrID
					}
					if value, ok := attrMap["value"].(string); ok {
						attr.Value = value
					}
					attributes = append(attributes, attr)
				}
			}
		}
		_ = attributes // Location proto does not expose LocationAttributes directly

		locations = append(locations, location)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	return &locationpb.GetLocationListPageDataResponse{
		LocationList: locations,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
		Success: true,
	}, nil
}

// GetLocationItemPageData retrieves a single location with attributes.
//
// Dialect translation:
//   - jsonb_agg ... FILTER (WHERE la.id IS NOT NULL) → JSON_ARRAYAGG (inner WHERE already filters)
//   - jsonb_build_object → JSON_OBJECT
//   - COALESCE(..., '[]'::jsonb) → COALESCE(..., JSON_ARRAY())
//   - $1/$2 → ? (positional); $2::text IS NULL → ? IS NULL OR ? = ” (two bindings)
func (r *MySQLLocationRepository) GetLocationItemPageData(
	ctx context.Context,
	req *locationpb.GetLocationItemPageDataRequest,
) (*locationpb.GetLocationItemPageDataResponse, error) {
	if req == nil || req.LocationId == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Dialect: $1 → ? (locationId); $2::text IS NULL OR l.workspace_id = $2 → two ? bindings for workspaceID.
	query := `
		WITH location_attributes_agg AS (
			SELECT
				la.location_id,
				JSON_ARRAYAGG(
					JSON_OBJECT(
						'id', la.id,
						'location_id', la.location_id,
						'attribute_id', la.attribute_id,
						'value', la.value
					)
				) AS attributes
			FROM location_attribute la
			WHERE la.location_id = ?
			GROUP BY la.location_id
		)
		SELECT
			l.id, l.name, l.address,
			l.active, l.date_created, l.date_modified,
			COALESCE(l.timezone, 'Asia/Manila') AS timezone,
			COALESCE(laa.attributes, JSON_ARRAY()) AS location_attributes
		FROM location l
		LEFT JOIN location_attributes_agg laa ON l.id = laa.location_id
		WHERE l.id = ?
		  AND (? IS NULL OR ? = '' OR l.workspace_id = ?)
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.LocationId, req.LocationId, workspaceID, workspaceID, workspaceID)

	var (
		id             string
		name           string
		address        *string
		active         bool
		dateCreated    time.Time
		dateModified   time.Time
		timezone       string
		attributesJSON []byte
	)

	err := row.Scan(
		&id, &name, &address,
		&active, &dateCreated, &dateModified,
		&timezone, &attributesJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("location not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	location := &locationpb.Location{
		Id:     id,
		Name:   name,
		Active: active,
	}

	if address != nil {
		location.Address = *address
	}
	location.Timezone = &timezone

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		location.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		location.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		location.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		location.DateModifiedString = &dmStr
	}

	var attributes []*locationattributepb.LocationAttribute
	if len(attributesJSON) > 0 && string(attributesJSON) != "[]" {
		var attrMaps []map[string]interface{}
		if err := json.Unmarshal(attributesJSON, &attrMaps); err == nil {
			for _, attrMap := range attrMaps {
				attr := &locationattributepb.LocationAttribute{}
				if attrId, ok := attrMap["id"].(string); ok {
					attr.Id = attrId
				}
				if locationID, ok := attrMap["location_id"].(string); ok {
					attr.LocationId = locationID
				}
				if key, ok := attrMap["key"].(string); ok {
					attr.AttributeId = key
				}
				if value, ok := attrMap["value"].(string); ok {
					attr.Value = value
				}
				attributes = append(attributes, attr)
			}
		}
	}
	_ = attributes

	return &locationpb.GetLocationItemPageDataResponse{
		Location: location,
		Success:  true,
	}, nil
}

// NewLocationRepository creates a new MySQL location repository (old-style constructor).
func NewLocationRepository(db *sql.DB, tableName string) locationpb.LocationDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLLocationRepository(dbOps, tableName)
}
