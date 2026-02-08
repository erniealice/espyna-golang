//go:build postgres

package entity

import (
	"time"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "location", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres location repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresLocationRepository(dbOps, tableName), nil
	})
}

// PostgresLocationRepository implements location CRUD operations using PostgreSQL
type PostgresLocationRepository struct {
	locationpb.UnimplementedLocationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresLocationRepository creates a new PostgreSQL location repository
func NewPostgresLocationRepository(dbOps interfaces.DatabaseOperation, tableName string) locationpb.LocationDomainServiceServer {
	if tableName == "" {
		tableName = "location" // default fallback
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresLocationRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateLocation creates a new location using common PostgreSQL operations
func (r *PostgresLocationRepository) CreateLocation(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("location data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	location := &locationpb.Location{}
	if err := protojson.Unmarshal(resultJSON, location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationpb.CreateLocationResponse{
		Data: []*locationpb.Location{location},
	}, nil
}

// ReadLocation retrieves a location using common PostgreSQL operations
func (r *PostgresLocationRepository) ReadLocation(ctx context.Context, req *locationpb.ReadLocationRequest) (*locationpb.ReadLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read location: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	location := &locationpb.Location{}
	if err := protojson.Unmarshal(resultJSON, location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationpb.ReadLocationResponse{
		Data: []*locationpb.Location{location},
	}, nil
}

// UpdateLocation updates a location using common PostgreSQL operations
func (r *PostgresLocationRepository) UpdateLocation(ctx context.Context, req *locationpb.UpdateLocationRequest) (*locationpb.UpdateLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	location := &locationpb.Location{}
	if err := protojson.Unmarshal(resultJSON, location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationpb.UpdateLocationResponse{
		Data: []*locationpb.Location{location},
	}, nil
}

// DeleteLocation deletes a location using common PostgreSQL operations
func (r *PostgresLocationRepository) DeleteLocation(ctx context.Context, req *locationpb.DeleteLocationRequest) (*locationpb.DeleteLocationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete location: %w", err)
	}

	return &locationpb.DeleteLocationResponse{
		Success: true,
	}, nil
}

// ListLocations lists locations using common PostgreSQL operations
func (r *PostgresLocationRepository) ListLocations(ctx context.Context, req *locationpb.ListLocationsRequest) (*locationpb.ListLocationsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var locations []*locationpb.Location
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		location := &locationpb.Location{}
		if err := protojson.Unmarshal(resultJSON, location); err != nil {
			// Log error and continue with next item
			continue
		}
		locations = append(locations, location)
	}

	return &locationpb.ListLocationsResponse{
		Data: locations,
	}, nil
}

// GetLocationListPageData retrieves locations with attributes using CTE
func (r *PostgresLocationRepository) GetLocationListPageData(
	ctx context.Context,
	req *locationpb.GetLocationListPageDataRequest,
) (*locationpb.GetLocationListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
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

	sortField := "name"
	sortOrder := "ASC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	query := `
		WITH location_attributes_agg AS (
			SELECT
				la.location_id,
				jsonb_agg(
					jsonb_build_object(
						'id', la.id,
						'location_id', la.location_id,
						'key', la.key,
						'value', la.value,
						'active', la.active
					) ORDER BY la.key
				) FILTER (WHERE la.id IS NOT NULL) as attributes
			FROM location_attribute la
			WHERE la.active = true
			GROUP BY la.location_id
		),
		enriched AS (
			SELECT
				l.id,
				l.name,
				l.address,
				l.city,
				l.state,
				l.country,
				l.postal_code,
				l.active,
				l.date_created,
				l.date_modified
				COALESCE(laa.attributes, '[]'::jsonb) as location_attributes
			FROM location l
			LEFT JOIN location_attributes_agg laa ON l.id = laa.location_id
			WHERE l.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   l.name ILIKE $1 OR
				   l.address ILIKE $1 OR
				   l.city ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT e.*, c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var locations []*locationpb.Location
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			name               string
			address            *string
			city               *string
			state              *string
			country            *string
			postalCode         *string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			attributesJSON     []byte
			total              int64
		)

		err := rows.Scan(
			&id, &name, &address, &city, &state, &country, &postalCode,
			&active, &dateCreated, &dateModified,
			&attributesJSON, &total,
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
		// Note: Removed city, state, country, postalCode assignments as they don't exist in the protobuf schema
		// The Location protobuf only has: id, name, address, description, timestamps, and active fields

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
					if id, ok := attrMap["id"].(string); ok {
						attr.Id = id
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
					// Note: Removed active field assignment as LocationAttribute doesn't have an Active field in protobuf
					attributes = append(attributes, attr)
				}
			}
		}
		// Note: Removed LocationAttributes assignment as Location doesn't have this field in protobuf
		// Attributes are handled separately via LocationAttribute entities

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

// GetLocationItemPageData retrieves a single location with attributes
func (r *PostgresLocationRepository) GetLocationItemPageData(
	ctx context.Context,
	req *locationpb.GetLocationItemPageDataRequest,
) (*locationpb.GetLocationItemPageDataResponse, error) {
	if req == nil || req.LocationId == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	query := `
		WITH location_attributes_agg AS (
			SELECT
				la.location_id,
				jsonb_agg(
					jsonb_build_object(
						'id', la.id,
						'location_id', la.location_id,
						'key', la.key,
						'value', la.value,
						'active', la.active
					) ORDER BY la.key
				) FILTER (WHERE la.id IS NOT NULL) as attributes
			FROM location_attribute la
			WHERE la.active = true AND la.location_id = $1
			GROUP BY la.location_id
		)
		SELECT
			l.id, l.name, l.address, l.city, l.state, l.country, l.postal_code,
			l.active, l.date_created, l.date_modified
			COALESCE(laa.attributes, '[]'::jsonb) as location_attributes
		FROM location l
		LEFT JOIN location_attributes_agg laa ON l.id = laa.location_id
		WHERE l.id = $1 AND l.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.LocationId)

	var (
		id                 string
		name               string
		address            *string
		city               *string
		state              *string
		country            *string
		postalCode         *string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
		attributesJSON     []byte
	)

	err := row.Scan(
		&id, &name, &address, &city, &state, &country, &postalCode,
		&active, &dateCreated, &dateModified,
		&attributesJSON,
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
	// Note: Removed city, state, country, postalCode assignments as they don't exist in the Location protobuf schema
	// The Location protobuf only has: id, name, address, description, timestamps, and active fields

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
				if id, ok := attrMap["id"].(string); ok {
					attr.Id = id
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
				// Note: Removed active field assignment as LocationAttribute doesn't have an Active field in protobuf
				attributes = append(attributes, attr)
			}
		}
	}
	// Note: Removed LocationAttributes assignment as Location doesn't have this field in protobuf

	return &locationpb.GetLocationItemPageDataResponse{
		Location: location,
		Success:  true,
	}, nil
}

// Note: parseLocationTimestamp function removed - now using common operations.ParseTimestamp

// NewLocationRepository creates a new PostgreSQL location repository (old-style constructor)
func NewLocationRepository(db *sql.DB, tableName string) locationpb.LocationDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresLocationRepository(dbOps, tableName)
}
