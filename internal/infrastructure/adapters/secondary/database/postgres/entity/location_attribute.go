//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"time"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "location_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres location_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresLocationAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresLocationAttributeRepository implements location attribute CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_location_attribute_active ON location_attribute(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_location_attribute_location_id ON location_attribute(location_id) - Filter by location
//   - CREATE INDEX idx_location_attribute_key ON location_attribute(key) - Search by attribute key
//   - CREATE INDEX idx_location_attribute_date_created ON location_attribute(date_created DESC) - Default sorting
type PostgresLocationAttributeRepository struct {
	locationattributepb.UnimplementedLocationAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresLocationAttributeRepository creates a new PostgreSQL location attribute repository
func NewPostgresLocationAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) locationattributepb.LocationAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "location_attribute" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresLocationAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateLocationAttribute creates a new location attribute using common PostgreSQL operations
func (r *PostgresLocationAttributeRepository) CreateLocationAttribute(ctx context.Context, req *locationattributepb.CreateLocationAttributeRequest) (*locationattributepb.CreateLocationAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("location attribute data is required")
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
		return nil, fmt.Errorf("failed to create location attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	locationAttribute := &locationattributepb.LocationAttribute{}
	if err := protojson.Unmarshal(resultJSON, locationAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationattributepb.CreateLocationAttributeResponse{
		Data: []*locationattributepb.LocationAttribute{locationAttribute},
	}, nil
}

// ReadLocationAttribute retrieves a location attribute using common PostgreSQL operations
func (r *PostgresLocationAttributeRepository) ReadLocationAttribute(ctx context.Context, req *locationattributepb.ReadLocationAttributeRequest) (*locationattributepb.ReadLocationAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read location attribute: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	locationAttribute := &locationattributepb.LocationAttribute{}
	if err := protojson.Unmarshal(resultJSON, locationAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationattributepb.ReadLocationAttributeResponse{
		Data: []*locationattributepb.LocationAttribute{locationAttribute},
	}, nil
}

// UpdateLocationAttribute updates a location attribute using common PostgreSQL operations
func (r *PostgresLocationAttributeRepository) UpdateLocationAttribute(ctx context.Context, req *locationattributepb.UpdateLocationAttributeRequest) (*locationattributepb.UpdateLocationAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required")
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
		return nil, fmt.Errorf("failed to update location attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	locationAttribute := &locationattributepb.LocationAttribute{}
	if err := protojson.Unmarshal(resultJSON, locationAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &locationattributepb.UpdateLocationAttributeResponse{
		Data: []*locationattributepb.LocationAttribute{locationAttribute},
	}, nil
}

// DeleteLocationAttribute deletes a location attribute using common PostgreSQL operations
func (r *PostgresLocationAttributeRepository) DeleteLocationAttribute(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) (*locationattributepb.DeleteLocationAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete location attribute: %w", err)
	}

	return &locationattributepb.DeleteLocationAttributeResponse{
		Success: true,
	}, nil
}

// ListLocationAttributes lists location attributes using common PostgreSQL operations
func (r *PostgresLocationAttributeRepository) ListLocationAttributes(ctx context.Context, req *locationattributepb.ListLocationAttributesRequest) (*locationattributepb.ListLocationAttributesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list location attributes: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var locationAttributes []*locationattributepb.LocationAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		locationAttribute := &locationattributepb.LocationAttribute{}
		if err := protojson.Unmarshal(resultJSON, locationAttribute); err != nil {
			// Log error and continue with next item
			continue
		}
		locationAttributes = append(locationAttributes, locationAttribute)
	}

	return &locationattributepb.ListLocationAttributesResponse{
		Data: locationAttributes,
	}, nil
}

// GetLocationAttributeListPageData retrieves paginated location attribute list data with CTE
func (r *PostgresLocationAttributeRepository) GetLocationAttributeListPageData(ctx context.Context, req *locationattributepb.GetLocationAttributeListPageDataRequest) (*locationattributepb.GetLocationAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}
	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}
	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `WITH enriched AS (SELECT id, location_id, key, value, active, date_created, date_modified FROM location_attribute WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR key ILIKE $1 OR value ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var locationAttributes []*locationattributepb.LocationAttribute
	var totalCount int64
	for rows.Next() {
		var rawData map[string]interface{}
		var id, locationId, attributeKey, attributeValue string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &locationId, &attributeKey, &attributeValue, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total

		// Build data map and convert to protobuf
		rawData = map[string]interface{}{
			"id":          id,
			"locationId":  locationId,
			"key":         attributeKey,
			"value":       attributeValue,
			"active":      active,
		}
		
		
		if !dateCreated.IsZero() {
		rawData["dateCreated"] = dateCreated.UnixMilli()
		rawData["dateCreatedString"] = dateCreated.Format(time.RFC3339)
	}
		if !dateModified.IsZero() {
		rawData["dateModified"] = dateModified.UnixMilli()
		rawData["dateModifiedString"] = dateModified.Format(time.RFC3339)
	}

		// Convert to protobuf
		dataJSON, _ := json.Marshal(rawData)
		locationAttribute := &locationattributepb.LocationAttribute{}
		if err := protojson.Unmarshal(dataJSON, locationAttribute); err == nil {
			locationAttributes = append(locationAttributes, locationAttribute)
		}
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &locationattributepb.GetLocationAttributeListPageDataResponse{LocationAttributeList: locationAttributes, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetLocationAttributeItemPageData retrieves location attribute item page data
func (r *PostgresLocationAttributeRepository) GetLocationAttributeItemPageData(ctx context.Context, req *locationattributepb.GetLocationAttributeItemPageDataRequest) (*locationattributepb.GetLocationAttributeItemPageDataResponse, error) {
	if req == nil || req.LocationAttributeId == "" {
		return nil, fmt.Errorf("location attribute ID required")
	}
	query := `SELECT id, location_id, key, value, active, date_created, date_modified FROM location_attribute WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.LocationAttributeId)
	var id, locationId, attributeKey, attributeValue string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &locationId, &attributeKey, &attributeValue, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("location attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Build data map and convert to protobuf
	rawData := map[string]interface{}{
		"id":          id,
		"locationId":  locationId,
		"key":         attributeKey,
		"value":       attributeValue,
		"active":      active,
	}
	
	
	if !dateCreated.IsZero() {
		rawData["dateCreated"] = dateCreated.UnixMilli()
		rawData["dateCreatedString"] = dateCreated.Format(time.RFC3339)
	}
	if !dateModified.IsZero() {
		rawData["dateModified"] = dateModified.UnixMilli()
		rawData["dateModifiedString"] = dateModified.Format(time.RFC3339)
	}

	// Convert to protobuf
	dataJSON, _ := json.Marshal(rawData)
	locationAttribute := &locationattributepb.LocationAttribute{}
	if err := protojson.Unmarshal(dataJSON, locationAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &locationattributepb.GetLocationAttributeItemPageDataResponse{LocationAttribute: locationAttribute, Success: true}, nil
}


// NewLocationAttributeRepository creates a new PostgreSQL location_attribute repository (old-style constructor)
func NewLocationAttributeRepository(db *sql.DB, tableName string) locationattributepb.LocationAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresLocationAttributeRepository(dbOps, tableName)
}
