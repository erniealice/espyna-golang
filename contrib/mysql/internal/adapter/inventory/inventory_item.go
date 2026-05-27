//go:build mysql

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.InventoryItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql inventory_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLInventoryItemRepository(dbOps, tableName), nil
	})
}

// MySQLInventoryItemRepository implements inventory_item CRUD operations using MySQL 8.0+.
type MySQLInventoryItemRepository struct {
	inventoryitempb.UnimplementedInventoryItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLInventoryItemRepository creates a new MySQL inventory item repository.
func NewMySQLInventoryItemRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_item"
	}

	var db *sql.DB
	if myOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = myOps.GetDB()
	}

	return &MySQLInventoryItemRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateInventoryItem creates a new inventory item.
func (r *MySQLInventoryItemRepository) CreateInventoryItem(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) (*inventoryitempb.CreateInventoryItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("inventory item data is required")
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
		return nil, fmt.Errorf("failed to create inventory item: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.CreateInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// ReadInventoryItem retrieves an inventory item.
func (r *MySQLInventoryItemRepository) ReadInventoryItem(ctx context.Context, req *inventoryitempb.ReadInventoryItemRequest) (*inventoryitempb.ReadInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory item: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.ReadInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// UpdateInventoryItem updates an inventory item.
func (r *MySQLInventoryItemRepository) UpdateInventoryItem(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) (*inventoryitempb.UpdateInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
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
		return nil, fmt.Errorf("failed to update inventory item: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.UpdateInventoryItemResponse{
		Data: []*inventoryitempb.InventoryItem{inventoryItem},
	}, nil
}

// DeleteInventoryItem deletes an inventory item (soft delete).
func (r *MySQLInventoryItemRepository) DeleteInventoryItem(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) (*inventoryitempb.DeleteInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inventory item: %w", err)
	}

	return &inventoryitempb.DeleteInventoryItemResponse{
		Success: true,
	}, nil
}

var inventoryItemSortableSQLCols = []string{
	"id", "active", "name", "product_id", "location_id", "sku",
	"quantity_on_hand", "quantity_reserved", "quantity_available",
	"reorder_level", "unit_of_measure", "date_created", "date_modified",
}

var inventoryItemSortSpec = espynahttp.SortSpec{AllowedCols: inventoryItemSortableSQLCols}

// ListInventoryItems lists inventory items.
func (r *MySQLInventoryItemRepository) ListInventoryItems(ctx context.Context, req *inventoryitempb.ListInventoryItemsRequest) (*inventoryitempb.ListInventoryItemsResponse, error) {
	if err := espynahttp.ValidateSortColumns(inventoryItemSortSpec, req.GetSort(), "inventory_item"); err != nil {
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
		return nil, fmt.Errorf("failed to list inventory items: %w", err)
	}

	var inventoryItems []*inventoryitempb.InventoryItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		inventoryItem := &inventoryitempb.InventoryItem{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
			continue
		}
		inventoryItems = append(inventoryItems, inventoryItem)
	}

	return &inventoryitempb.ListInventoryItemsResponse{
		Data: inventoryItems,
	}, nil
}

// GetInventoryItemListPageData retrieves inventory items with advanced filtering, sorting,
// searching, and pagination using CTE.
//
// Dialect translation from postgres gold standard:
//   - $N → ? (positional args re-sequenced for MySQL)
//   - BuildFilterWhere returns filter clauses; nextIdx used only for tracking, not embedded in SQL
//   - active = true → active = 1 (MySQL TINYINT(1))
//   - ILIKE → LIKE (ci collation handles case-insensitivity)
//   - COUNT(*) OVER() stays — MySQL 8.0+ window functions
//   - backtick quoting in ORDER BY via mysqlCore.BuildOrderBy
//
// CRITICAL: workspace_id isolation enforced by WorkspaceAwareOperations on CRUD path.
func (r *MySQLInventoryItemRepository) GetInventoryItemListPageData(
	ctx context.Context,
	req *inventoryitempb.GetInventoryItemListPageDataRequest,
) (*inventoryitempb.GetInventoryItemListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory item list page data request is required")
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

	// Sort allowlist (view-key → SQL column).
	sortAllowlist := map[string]string{
		"product_name":  "p.name",
		"quantity":      "ii.quantity_on_hand",
		"status":        "ii.active",
		"date_created":  "ii.date_created",
		"date_modified": "ii.date_modified",
		"sku":           "ii.sku",
	}
	sortCol := "ii.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		if col, ok := sortAllowlist[req.Sort.Fields[0].Field]; ok {
			sortCol = col
		}
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Build parameterized WHERE clauses via shared helper.
	// MySQL uses ? placeholders — nextIdx is tracked but not embedded in SQL.
	searchFields := []string{"p.name", "ii.sku"}
	filterClauses, filterArgs, nextIdx := mysqlCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)
	_ = nextIdx

	var whereStr string
	if len(filterClauses) > 0 {
		whereStr = " AND " + strings.Join(filterClauses, " AND ")
	}

	// Args: [...filterArgs, limit, offset]
	queryArgs := append(filterArgs, limit, offset) //nolint:gocritic

	// Dialect: active = true → active = 1; $N LIMIT/OFFSET → LIMIT ? OFFSET ?
	// COUNT(*) OVER() supported in MySQL 8.0+.
	query := fmt.Sprintf(`
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
				COALESCE(p.tracking_mode, '') as tracking_mode,
				COALESCE(p.name, '') as product_name,
				COUNT(*) OVER() AS total_count
			FROM inventory_item ii
			LEFT JOIN product p ON ii.product_id = p.id AND p.active = 1
			WHERE ii.active = 1%s
		)
		SELECT * FROM enriched
		ORDER BY %s %s
		LIMIT ? OFFSET ?`, whereStr, sortCol, sortOrder)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
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
			trackingMode      string
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
			&trackingMode,
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

		if productID != nil {
			inventoryItem.ProductId = productID
			inventoryItem.Product = &productpb.Product{
				Id:           *productID,
				Name:         productName,
				TrackingMode: trackingMode,
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

// GetInventoryItemItemPageData retrieves a single inventory item with enhanced item page data.
//
// Dialect: $1 → ?, active = true → active = 1.
func (r *MySQLInventoryItemRepository) GetInventoryItemItemPageData(
	ctx context.Context,
	req *inventoryitempb.GetInventoryItemItemPageDataRequest,
) (*inventoryitempb.GetInventoryItemItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory item item page data request is required")
	}
	if req.InventoryItemId == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	// Dialect: $1 → ?, active = true → active = 1
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
				COALESCE(p.tracking_mode, '') as tracking_mode,
				ii.product_variant_id,
				ii.notes,
				COALESCE(p.name, '') as product_name,
				COALESCE(p.price, 0) as product_price
			FROM inventory_item ii
			LEFT JOIN product p ON ii.product_id = p.id AND p.active = 1
			WHERE ii.id = ? AND ii.active = 1
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
		trackingMode      string
		productVariantID  *string
		notes             *string
		productName       string
		productPrice      sql.NullInt64
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
		&trackingMode,
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

	if productID != nil {
		inventoryItem.ProductId = productID
		inventoryItem.Product = &productpb.Product{
			Id:           *productID,
			Name:         productName,
			TrackingMode: trackingMode,
		}
		if productPrice.Valid {
			p := productPrice.Int64
			inventoryItem.Product.Price = &p
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

// NewInventoryItemRepository creates a new MySQL inventory item repository (old-style constructor).
func NewInventoryItemRepository(db *sql.DB, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLInventoryItemRepository(dbOps, tableName)
}
