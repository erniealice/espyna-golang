//go:build postgresql

package inventory_item

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
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "inventory_item", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres inventory_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresInventoryItemRepository(dbOps, tableName), nil
	})
}

// PostgresInventoryItemRepository implements inventory_item CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_inventory_item_active ON inventory_item(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_inventory_item_product_id ON inventory_item(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_inventory_item_location_id ON inventory_item(location_id) - FK lookup on location_id
//   - CREATE INDEX idx_inventory_item_name ON inventory_item(name) - Search on name field
//   - CREATE INDEX idx_inventory_item_sku ON inventory_item(sku) - Search on sku field
//   - CREATE INDEX idx_inventory_item_date_created ON inventory_item(date_created DESC) - Default sorting
type PostgresInventoryItemRepository struct {
	inventoryitempb.UnimplementedInventoryItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresInventoryItemRepository creates a new PostgreSQL inventory item repository
func NewPostgresInventoryItemRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_item" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresInventoryItemRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventoryItem creates a new inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) CreateInventoryItem(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) (*inventoryitempb.CreateInventoryItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory item data is required")
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
		return nil, fmt.Errorf("failed to create inventory item: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := protojson.Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.CreateInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// ReadInventoryItem retrieves an inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) ReadInventoryItem(ctx context.Context, req *inventoryitempb.ReadInventoryItemRequest) (*inventoryitempb.ReadInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory item: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := protojson.Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.ReadInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// UpdateInventoryItem updates an inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) UpdateInventoryItem(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) (*inventoryitempb.UpdateInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
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
		return nil, fmt.Errorf("failed to update inventory item: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := protojson.Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.UpdateInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// DeleteInventoryItem deletes an inventory item using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) DeleteInventoryItem(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) (*inventoryitempb.DeleteInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory item: %w", err)
	}

	return &inventoryitempb.DeleteInventoryItemResponse{
		Success: true,
	}, nil
}

// ListInventoryItems lists inventory items using common PostgreSQL operations
func (r *PostgresInventoryItemRepository) ListInventoryItems(ctx context.Context, req *inventoryitempb.ListInventoryItemsRequest) (*inventoryitempb.ListInventoryItemsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory items: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var inventoryItems []*inventoryitempb.InventoryItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		inventoryItem := &inventoryitempb.InventoryItem{}
		if err := protojson.Unmarshal(resultJSON, inventoryItem); err != nil {
			// Log error and continue with next item
			continue
		}
		inventoryItems = append(inventoryItems, inventoryItem)
	}

	return &inventoryitempb.ListInventoryItemsResponse{
		Data: inventoryItems,
	}, nil
}

// GetInventoryItemListPageData retrieves inventory items with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the product table to include the parent product name
func (r *PostgresInventoryItemRepository) GetInventoryItemListPageData(
	ctx context.Context,
	req *inventoryitempb.GetInventoryItemListPageDataRequest,
) (*inventoryitempb.GetInventoryItemListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory item list page data request is required")
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
	sortField := "ii.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with product join for parent product name
	query := `
		WITH enriched AS (
			SELECT
				ii.id,
				ii.date_created,
				ii.date_modified,
				ii.active,
				ii.name,
				ii.product_id,
				ii.location_id,
				ii.sku,
				ii.quantity_on_hand,
				ii.quantity_reserved,
				ii.quantity_available,
				ii.reorder_level,
				ii.unit_of_measure,
				COALESCE(p.item_type, '') as item_type,
				COALESCE(p.name, '') as product_name
			FROM inventory_item ii
			LEFT JOIN product p ON ii.product_id = p.id AND p.active = true
			WHERE ii.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       ii.name ILIKE $1 OR
			       ii.sku ILIKE $1 OR
			       p.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query inventory item list page data: %w", err)
	}
	defer rows.Close()

	var inventoryItems []*inventoryitempb.InventoryItem
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			name              string
			productID         *string
			locationID        *string
			sku               *string
			quantityOnHand    float64
			quantityReserved  float64
			quantityAvailable float64
			reorderLevel      *float64
			unitOfMeasure     string
			itemType          string
			productName       string
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&productID,
			&locationID,
			&sku,
			&quantityOnHand,
			&quantityReserved,
			&quantityAvailable,
			&reorderLevel,
			&unitOfMeasure,
			&itemType,
			&productName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory item row: %w", err)
		}

		totalCount = total

		inventoryItem := &inventoryitempb.InventoryItem{
			Id:                id,
			Active:            active,
			Name:              name,
			QuantityOnHand:    quantityOnHand,
			QuantityReserved:  quantityReserved,
			QuantityAvailable: quantityAvailable,
			UnitOfMeasure:     unitOfMeasure,
		}

		// Handle nullable fields
		if productID != nil {
			inventoryItem.ProductId = productID
			inventoryItem.Product = &productpb.Product{
				Id:       *productID,
				Name:     productName,
				ItemType: itemType,
			}
		}
		if locationID != nil {
			inventoryItem.LocationId = locationID
		}
		if sku != nil {
			inventoryItem.Sku = sku
		}
		if reorderLevel != nil {
			inventoryItem.ReorderLevel = reorderLevel
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			inventoryItem.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			inventoryItem.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			inventoryItem.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			inventoryItem.DateModifiedString = &dmStr
		}

		inventoryItems = append(inventoryItems, inventoryItem)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory item rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &inventoryitempb.GetInventoryItemListPageDataResponse{
		InventoryItemList: inventoryItems,
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

// GetInventoryItemItemPageData retrieves a single inventory item with enhanced item page data using CTE
// This method joins with the product table for the parent product reference
func (r *PostgresInventoryItemRepository) GetInventoryItemItemPageData(
	ctx context.Context,
	req *inventoryitempb.GetInventoryItemItemPageDataRequest,
) (*inventoryitempb.GetInventoryItemItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory item item page data request is required")
	}
	if req.InventoryItemId == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	// CTE Query - Single round-trip with product join
	query := `
		WITH enriched AS (
			SELECT
				ii.id,
				ii.date_created,
				ii.date_modified,
				ii.active,
				ii.name,
				ii.product_id,
				ii.location_id,
				ii.sku,
				ii.quantity_on_hand,
				ii.quantity_reserved,
				ii.quantity_available,
				ii.reorder_level,
				ii.unit_of_measure,
				COALESCE(p.item_type, '') as item_type,
				ii.product_variant_id,
				ii.notes,
				COALESCE(p.name, '') as product_name,
				COALESCE(p.price, 0) as product_price
			FROM inventory_item ii
			LEFT JOIN product p ON ii.product_id = p.id AND p.active = true
			WHERE ii.id = $1 AND ii.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.InventoryItemId)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		name              string
		productID         *string
		locationID        *string
		sku               *string
		quantityOnHand    float64
		quantityReserved  float64
		quantityAvailable float64
		reorderLevel      *float64
		unitOfMeasure     string
		itemType          string
		productVariantID  *string
		notes             *string
		productName       string
		productPrice      float64
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&productID,
		&locationID,
		&sku,
		&quantityOnHand,
		&quantityReserved,
		&quantityAvailable,
		&reorderLevel,
		&unitOfMeasure,
		&itemType,
		&productVariantID,
		&notes,
		&productName,
		&productPrice,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory item with ID '%s' not found", req.InventoryItemId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory item item page data: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{
		Id:                id,
		Active:            active,
		Name:              name,
		QuantityOnHand:    quantityOnHand,
		QuantityReserved:  quantityReserved,
		QuantityAvailable: quantityAvailable,
		UnitOfMeasure:     unitOfMeasure,
	}

	// Handle nullable fields
	if productID != nil {
		inventoryItem.ProductId = productID
		inventoryItem.Product = &productpb.Product{
			Id:       *productID,
			Name:     productName,
			Price:    productPrice,
			ItemType: itemType,
		}
	}
	if locationID != nil {
		inventoryItem.LocationId = locationID
	}
	if sku != nil {
		inventoryItem.Sku = sku
	}
	if reorderLevel != nil {
		inventoryItem.ReorderLevel = reorderLevel
	}
	if productVariantID != nil {
		inventoryItem.ProductVariantId = productVariantID
	}
	if notes != nil {
		inventoryItem.Notes = notes
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		inventoryItem.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		inventoryItem.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		inventoryItem.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		inventoryItem.DateModifiedString = &dmStr
	}

	return &inventoryitempb.GetInventoryItemItemPageDataResponse{
		InventoryItem: inventoryItem,
		Success:       true,
	}, nil
}

// NewInventoryItemRepository creates a new PostgreSQL inventory item repository (old-style constructor)
func NewInventoryItemRepository(db *sql.DB, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresInventoryItemRepository(dbOps, tableName)
}
