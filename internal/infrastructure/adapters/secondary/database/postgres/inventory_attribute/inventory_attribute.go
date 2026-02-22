//go:build postgresql

package inventory_attribute

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventoryattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "inventory_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInventoryAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresInventoryAttributeRepository implements inventory_attribute CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_attribute_inventory_item_id ON inventory_attribute(inventory_item_id) - FK lookup
//   - CREATE INDEX idx_inventory_attribute_attribute_id ON inventory_attribute(attribute_id) - FK lookup
//   - CREATE INDEX idx_inventory_attribute_active ON inventory_attribute(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_attribute_date_created ON inventory_attribute(date_created DESC) - Default sorting
type PostgresInventoryAttributeRepository struct {
	inventoryattributepb.UnimplementedInventoryAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresInventoryAttributeRepository creates a new PostgreSQL inventory attribute repository
func NewPostgresInventoryAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryattributepb.InventoryAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_attribute" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresInventoryAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventoryAttribute creates a new inventory attribute using common PostgreSQL operations
func (r *PostgresInventoryAttributeRepository) CreateInventoryAttribute(ctx context.Context, req *inventoryattributepb.CreateInventoryAttributeRequest) (*inventoryattributepb.CreateInventoryAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory attribute data is required")
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
		return nil, fmt.Errorf("failed to create inventory attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryAttribute := &inventoryattributepb.InventoryAttribute{}
	if err := protojson.Unmarshal(resultJSON, inventoryAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryattributepb.CreateInventoryAttributeResponse{
		Data: []*inventoryattributepb.InventoryAttribute{inventoryAttribute},
	}, nil
}

// ReadInventoryAttribute retrieves an inventory attribute using common PostgreSQL operations
func (r *PostgresInventoryAttributeRepository) ReadInventoryAttribute(ctx context.Context, req *inventoryattributepb.ReadInventoryAttributeRequest) (*inventoryattributepb.ReadInventoryAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory attribute ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory attribute: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryAttribute := &inventoryattributepb.InventoryAttribute{}
	if err := protojson.Unmarshal(resultJSON, inventoryAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryattributepb.ReadInventoryAttributeResponse{
		Data: []*inventoryattributepb.InventoryAttribute{inventoryAttribute},
	}, nil
}

// UpdateInventoryAttribute updates an inventory attribute using common PostgreSQL operations
func (r *PostgresInventoryAttributeRepository) UpdateInventoryAttribute(ctx context.Context, req *inventoryattributepb.UpdateInventoryAttributeRequest) (*inventoryattributepb.UpdateInventoryAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory attribute ID is required")
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
		return nil, fmt.Errorf("failed to update inventory attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryAttribute := &inventoryattributepb.InventoryAttribute{}
	if err := protojson.Unmarshal(resultJSON, inventoryAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryattributepb.UpdateInventoryAttributeResponse{
		Data: []*inventoryattributepb.InventoryAttribute{inventoryAttribute},
	}, nil
}

// DeleteInventoryAttribute deletes an inventory attribute using common PostgreSQL operations
func (r *PostgresInventoryAttributeRepository) DeleteInventoryAttribute(ctx context.Context, req *inventoryattributepb.DeleteInventoryAttributeRequest) (*inventoryattributepb.DeleteInventoryAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory attribute: %w", err)
	}

	return &inventoryattributepb.DeleteInventoryAttributeResponse{
		Success: true,
	}, nil
}

// ListInventoryAttributes lists inventory attributes using common PostgreSQL operations
func (r *PostgresInventoryAttributeRepository) ListInventoryAttributes(ctx context.Context, req *inventoryattributepb.ListInventoryAttributesRequest) (*inventoryattributepb.ListInventoryAttributesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory attributes: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var inventoryAttributes []*inventoryattributepb.InventoryAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		inventoryAttribute := &inventoryattributepb.InventoryAttribute{}
		if err := protojson.Unmarshal(resultJSON, inventoryAttribute); err != nil {
			// Log error and continue with next item
			continue
		}
		inventoryAttributes = append(inventoryAttributes, inventoryAttribute)
	}

	return &inventoryattributepb.ListInventoryAttributesResponse{
		Data: inventoryAttributes,
	}, nil
}

// GetInventoryAttributeListPageData retrieves inventory attributes with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the inventory_item table to include the parent item name
// Supports search on attribute value and inventory item name
func (r *PostgresInventoryAttributeRepository) GetInventoryAttributeListPageData(
	ctx context.Context,
	req *inventoryattributepb.GetInventoryAttributeListPageDataRequest,
) (*inventoryattributepb.GetInventoryAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory attribute list page data request is required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values
	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		// Handle offset pagination
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "ia.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with inventory_item join
	query := `
		WITH enriched AS (
			SELECT
				ia.id,
				ia.date_created,
				ia.date_modified,
				ia.active,
				ia.inventory_item_id,
				ia.attribute_id,
				ia.value,
				COALESCE(ii.name, '') as inventory_item_name
			FROM inventory_attribute ia
			LEFT JOIN inventory_item ii ON ia.inventory_item_id = ii.id AND ii.active = true
			WHERE ia.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       ia.value ILIKE $1 OR
			       ii.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query inventory attribute list page data: %w", err)
	}
	defer rows.Close()

	var inventoryAttributes []*inventoryattributepb.InventoryAttribute
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			inventoryItemID   string
			attributeID       string
			value             string
			inventoryItemName string
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&inventoryItemID,
			&attributeID,
			&value,
			&inventoryItemName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory attribute row: %w", err)
		}

		totalCount = total

		inventoryAttribute := &inventoryattributepb.InventoryAttribute{
			Id:              id,
			Active:          active,
			InventoryItemId: inventoryItemID,
			AttributeId:     attributeID,
			Value:           value,
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			inventoryAttribute.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			inventoryAttribute.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			inventoryAttribute.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			inventoryAttribute.DateModifiedString = &dmStr
		}

		// Note: inventoryItemName is available from the join but not directly mapped
		// to the InventoryAttribute protobuf. Could be populated via the
		// InventoryItem field if needed for frontend display.

		inventoryAttributes = append(inventoryAttributes, inventoryAttribute)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory attribute rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &inventoryattributepb.GetInventoryAttributeListPageDataResponse{
		InventoryAttributeList: inventoryAttributes,
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

// GetInventoryAttributeItemPageData retrieves a single inventory attribute with enhanced item page data using CTE
// This method joins with the inventory_item table for the parent item reference
func (r *PostgresInventoryAttributeRepository) GetInventoryAttributeItemPageData(
	ctx context.Context,
	req *inventoryattributepb.GetInventoryAttributeItemPageDataRequest,
) (*inventoryattributepb.GetInventoryAttributeItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory attribute item page data request is required")
	}
	if req.InventoryAttributeId == "" {
		return nil, fmt.Errorf("inventory attribute ID is required")
	}

	// CTE Query - Single round-trip with inventory_item join
	query := `
		WITH enriched AS (
			SELECT
				ia.id,
				ia.date_created,
				ia.date_modified,
				ia.active,
				ia.inventory_item_id,
				ia.attribute_id,
				ia.value,
				COALESCE(ii.name, '') as inventory_item_name,
				COALESCE(ii.sku, '') as inventory_item_sku
			FROM inventory_attribute ia
			LEFT JOIN inventory_item ii ON ia.inventory_item_id = ii.id AND ii.active = true
			WHERE ia.id = $1 AND ia.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.InventoryAttributeId)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		inventoryItemID   string
		attributeID       string
		value             string
		inventoryItemName string
		inventoryItemSku  string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&inventoryItemID,
		&attributeID,
		&value,
		&inventoryItemName,
		&inventoryItemSku,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory attribute with ID '%s' not found", req.InventoryAttributeId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory attribute item page data: %w", err)
	}

	inventoryAttribute := &inventoryattributepb.InventoryAttribute{
		Id:              id,
		Active:          active,
		InventoryItemId: inventoryItemID,
		AttributeId:     attributeID,
		Value:           value,
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		inventoryAttribute.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		inventoryAttribute.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		inventoryAttribute.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		inventoryAttribute.DateModifiedString = &dmStr
	}

	// Note: inventoryItemName and inventoryItemSku are available from the join
	// but not directly mapped to the InventoryAttribute protobuf. These could be
	// returned via the InventoryItem field or processed separately.

	return &inventoryattributepb.GetInventoryAttributeItemPageDataResponse{
		InventoryAttribute: inventoryAttribute,
		Success:            true,
	}, nil
}

// NewInventoryAttributeRepository creates a new PostgreSQL inventory attribute repository (old-style constructor)
func NewInventoryAttributeRepository(db *sql.DB, tableName string) inventoryattributepb.InventoryAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresInventoryAttributeRepository(dbOps, tableName)
}
