//go:build postgres

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "client_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres client_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresClientAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresClientAttributeRepository implements client attribute CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_client_attribute_active ON client_attribute(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_client_attribute_client_id ON client_attribute(client_id) - Filter by client
//   - CREATE INDEX idx_client_attribute_key ON client_attribute(key) - Search by attribute key
//   - CREATE INDEX idx_client_attribute_date_created ON client_attribute(date_created DESC) - Default sorting
type PostgresClientAttributeRepository struct {
	clientattributepb.UnimplementedClientAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresClientAttributeRepository creates a new PostgreSQL client attribute repository
func NewPostgresClientAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) clientattributepb.ClientAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "client_attribute" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresClientAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateClientAttribute creates a new client attribute using common PostgreSQL operations
func (r *PostgresClientAttributeRepository) CreateClientAttribute(ctx context.Context, req *clientattributepb.CreateClientAttributeRequest) (*clientattributepb.CreateClientAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client attribute data is required")
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
		return nil, fmt.Errorf("failed to create client attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientAttribute := &clientattributepb.ClientAttribute{}
	if err := protojson.Unmarshal(resultJSON, clientAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientattributepb.CreateClientAttributeResponse{
		Data: []*clientattributepb.ClientAttribute{clientAttribute},
	}, nil
}

// ReadClientAttribute retrieves a client attribute using common PostgreSQL operations
func (r *PostgresClientAttributeRepository) ReadClientAttribute(ctx context.Context, req *clientattributepb.ReadClientAttributeRequest) (*clientattributepb.ReadClientAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client attribute ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client attribute: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientAttribute := &clientattributepb.ClientAttribute{}
	if err := protojson.Unmarshal(resultJSON, clientAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientattributepb.ReadClientAttributeResponse{
		Data: []*clientattributepb.ClientAttribute{clientAttribute},
	}, nil
}

// UpdateClientAttribute updates a client attribute using common PostgreSQL operations
func (r *PostgresClientAttributeRepository) UpdateClientAttribute(ctx context.Context, req *clientattributepb.UpdateClientAttributeRequest) (*clientattributepb.UpdateClientAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client attribute ID is required")
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
		return nil, fmt.Errorf("failed to update client attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientAttribute := &clientattributepb.ClientAttribute{}
	if err := protojson.Unmarshal(resultJSON, clientAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientattributepb.UpdateClientAttributeResponse{
		Data: []*clientattributepb.ClientAttribute{clientAttribute},
	}, nil
}

// DeleteClientAttribute deletes a client attribute using common PostgreSQL operations
func (r *PostgresClientAttributeRepository) DeleteClientAttribute(ctx context.Context, req *clientattributepb.DeleteClientAttributeRequest) (*clientattributepb.DeleteClientAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete client attribute: %w", err)
	}

	return &clientattributepb.DeleteClientAttributeResponse{
		Success: true,
	}, nil
}

// ListClientAttributes lists client attributes using common PostgreSQL operations
func (r *PostgresClientAttributeRepository) ListClientAttributes(ctx context.Context, req *clientattributepb.ListClientAttributesRequest) (*clientattributepb.ListClientAttributesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list client attributes: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var clientAttributes []*clientattributepb.ClientAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		clientAttribute := &clientattributepb.ClientAttribute{}
		if err := protojson.Unmarshal(resultJSON, clientAttribute); err != nil {
			// Log error and continue with next item
			continue
		}
		clientAttributes = append(clientAttributes, clientAttribute)
	}

	return &clientattributepb.ListClientAttributesResponse{
		Data: clientAttributes,
	}, nil
}

// GetClientAttributeListPageData retrieves paginated client attribute list data with CTE
func (r *PostgresClientAttributeRepository) GetClientAttributeListPageData(ctx context.Context, req *clientattributepb.GetClientAttributeListPageDataRequest) (*clientattributepb.GetClientAttributeListPageDataResponse, error) {
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

	query := `WITH enriched AS (SELECT id, client_id, key, value, active, date_created, date_modified FROM client_attribute WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR key ILIKE $1 OR value ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var clientAttributes []*clientattributepb.ClientAttribute
	var totalCount int64
	for rows.Next() {
		var rawData map[string]interface{}
		var id, clientId, attributeKey, attributeValue string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &clientId, &attributeKey, &attributeValue, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total

		// Build data map and convert to protobuf
		rawData = map[string]interface{}{
			"id":       id,
			"clientId": clientId,
			"key":      attributeKey,
			"value":    attributeValue,
			"active":   active,
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
		clientAttribute := &clientattributepb.ClientAttribute{}
		if err := protojson.Unmarshal(dataJSON, clientAttribute); err == nil {
			clientAttributes = append(clientAttributes, clientAttribute)
		}
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &clientattributepb.GetClientAttributeListPageDataResponse{ClientAttributeList: clientAttributes, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetClientAttributeItemPageData retrieves client attribute item page data
func (r *PostgresClientAttributeRepository) GetClientAttributeItemPageData(ctx context.Context, req *clientattributepb.GetClientAttributeItemPageDataRequest) (*clientattributepb.GetClientAttributeItemPageDataResponse, error) {
	if req == nil || req.ClientAttributeId == "" {
		return nil, fmt.Errorf("client attribute ID required")
	}
	query := `SELECT id, client_id, key, value, active, date_created, date_modified FROM client_attribute WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.ClientAttributeId)
	var id, clientId, attributeKey, attributeValue string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &clientId, &attributeKey, &attributeValue, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("client attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Build data map and convert to protobuf
	rawData := map[string]interface{}{
		"id":       id,
		"clientId": clientId,
		"key":      attributeKey,
		"value":    attributeValue,
		"active":   active,
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
	clientAttribute := &clientattributepb.ClientAttribute{}
	if err := protojson.Unmarshal(dataJSON, clientAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &clientattributepb.GetClientAttributeItemPageDataResponse{ClientAttribute: clientAttribute, Success: true}, nil
}


// NewClientAttributeRepository creates a new PostgreSQL client_attribute repository (old-style constructor)
func NewClientAttributeRepository(db *sql.DB, tableName string) clientattributepb.ClientAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresClientAttributeRepository(dbOps, tableName)
}
