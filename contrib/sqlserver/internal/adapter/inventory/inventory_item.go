//go:build sqlserver

package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	inventoryitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/inventory/inventory_item"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.InventoryItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver inventory_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerInventoryItemRepository(dbOps, tableName), nil
	})
}

// SQLServerInventoryItemRepository implements inventory_item CRUD operations using SQL Server.
type SQLServerInventoryItemRepository struct {
	inventoryitempb.UnimplementedInventoryItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerInventoryItemRepository creates a new SQL Server inventory item repository.
func NewSQLServerInventoryItemRepository(dbOps interfaces.DatabaseOperation, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	if tableName == "" {
		tableName = "inventory_item"
	}
	return &SQLServerInventoryItemRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// getExec extracts a DBExecutor from the dbOps wrapper.
func (r *SQLServerInventoryItemRepository) getExec(ctx context.Context) dbExecutor {
	return r.dbOps.(executorProvider).GetExecutor(ctx)
}

// CreateInventoryItem creates a new inventory item.
func (r *SQLServerInventoryItemRepository) CreateInventoryItem(ctx context.Context, req *inventoryitempb.CreateInventoryItemRequest) (*inventoryitempb.CreateInventoryItemResponse, error) {
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

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.CreateInventoryItemResponse{Data: []*inventoryitempb.InventoryItem{inventoryItem}}, nil
}

// ReadInventoryItem retrieves an inventory item.
func (r *SQLServerInventoryItemRepository) ReadInventoryItem(ctx context.Context, req *inventoryitempb.ReadInventoryItemRequest) (*inventoryitempb.ReadInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory item: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.ReadInventoryItemResponse{Data: []*inventoryitempb.InventoryItem{inventoryItem}}, nil
}

// UpdateInventoryItem updates an inventory item.
func (r *SQLServerInventoryItemRepository) UpdateInventoryItem(ctx context.Context, req *inventoryitempb.UpdateInventoryItemRequest) (*inventoryitempb.UpdateInventoryItemResponse, error) {
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

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	inventoryItem := &inventoryitempb.InventoryItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &inventoryitempb.UpdateInventoryItemResponse{Data: []*inventoryitempb.InventoryItem{inventoryItem}}, nil
}

// DeleteInventoryItem deletes an inventory item (soft delete).
func (r *SQLServerInventoryItemRepository) DeleteInventoryItem(ctx context.Context, req *inventoryitempb.DeleteInventoryItemRequest) (*inventoryitempb.DeleteInventoryItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete inventory item: %w", err)
	}

	return &inventoryitempb.DeleteInventoryItemResponse{Success: true}, nil
}

var inventoryItemSortableSQLCols = []string{
	"ii.id", "ii.active", "ii.name", "ii.product_id", "ii.location_id", "ii.sku",
	"ii.quantity_on_hand", "ii.quantity_reserved", "ii.quantity_available",
	"ii.reorder_level", "ii.unit_of_measure", "ii.date_created", "ii.date_modified",
	"p.name",
}

// inventoryItemSortAllowlist maps view-facing sort keys to SQL column names.
var inventoryItemSortAllowlist = map[string]string{
	"product_name":  "p.name",
	"quantity":      "ii.quantity_on_hand",
	"status":        "ii.active",
	"date_created":  "ii.date_created",
	"date_modified": "ii.date_modified",
	"sku":           "ii.sku",
}

// ListInventoryItems lists inventory items.
func (r *SQLServerInventoryItemRepository) ListInventoryItems(ctx context.Context, req *inventoryitempb.ListInventoryItemsRequest) (*inventoryitempb.ListInventoryItemsResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		inventoryItem := &inventoryitempb.InventoryItem{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, inventoryItem); err != nil {
			continue
		}
		inventoryItems = append(inventoryItems, inventoryItem)
	}

	return &inventoryitempb.ListInventoryItemsResponse{Data: inventoryItems}, nil
}

// GetInventoryItemListPageData retrieves inventory items with advanced filtering, sorting, searching, and pagination.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1.
//   - Pagination: LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER () is retained — SQL Server 2017+ supports it.
//   - ILIKE → LIKE.
func (r *SQLServerInventoryItemRepository) GetInventoryItemListPageData(
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

	// A2 sort guard — map view key to SQL column, then whitelist-check via BuildOrderBy.
	sortCol := "ii.date_created"
	var sortReq *commonpb.SortRequest
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		if col, ok := inventoryItemSortAllowlist[req.Sort.Fields[0].Field]; ok {
			sortCol = col
		} else if req.Sort.Fields[0].Field != "" {
			sortCol = req.Sort.Fields[0].Field
		}
		sortReq = &commonpb.SortRequest{
			Fields: []*commonpb.SortField{
				{Field: sortCol, Direction: req.Sort.Fields[0].Direction},
			},
		}
	}

	orderByClause, err := sqlserverCore.BuildOrderBy(inventoryItemSortableSQLCols, sortReq, "ii.date_created DESC")
	if err != nil {
		return nil, err
	}

	// Build filter/search clauses; start params at @p1 (no workspace_id predicate here — inventory_item has no workspace_id FK directly).
	searchFields := []string{"p.name", "ii.sku"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 1)

	whereStr := "WHERE ii.active = 1"
	if len(filterClauses) > 0 {
		whereStr += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := append(filterArgs, offset, limit) //nolint:gocritic

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
				COALESCE(p.tracking_mode, '') AS tracking_mode,
				COALESCE(p.name, '') AS product_name,
				COUNT(*) OVER() AS total_count
			FROM inventory_item ii
			LEFT JOIN product p ON ii.product_id = p.id AND p.active = 1
			%s
		)
		SELECT * FROM enriched
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereStr, orderByClause, offsetIdx, limitIdx)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
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

		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active, &name,
			&productID, &locationID, &sku,
			&quantityOnHand, &quantityReserved, &quantityAvailable,
			&reorderLevel, &unitOfMeasure, &trackingMode, &productName, &total,
		); err != nil {
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
// SQL Server differences:
//   - SELECT TOP 1 * FROM enriched instead of LIMIT 1.
//   - @p1 placeholder.
//   - active = 1.
//   - p.active = 1 on join condition.
func (r *SQLServerInventoryItemRepository) GetInventoryItemItemPageData(
	ctx context.Context,
	req *inventoryitempb.GetInventoryItemItemPageDataRequest,
) (*inventoryitempb.GetInventoryItemItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get inventory item item page data request is required")
	}
	if req.InventoryItemId == "" {
		return nil, fmt.Errorf("inventory item ID is required")
	}

	const query = `
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
				COALESCE(p.tracking_mode, '') AS tracking_mode,
				ii.product_variant_id,
				ii.notes,
				COALESCE(p.name, '') AS product_name,
				COALESCE(p.price, 0) AS product_price
			FROM inventory_item ii
			LEFT JOIN product p ON ii.product_id = p.id AND p.active = 1
			WHERE ii.id = @p1 AND ii.active = 1
		)
		SELECT TOP 1 * FROM enriched;
	`

	exec := r.getExec(ctx)
	row := exec.QueryRowContext(ctx, query, req.InventoryItemId)

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
		&id, &dateCreated, &dateModified, &active, &name,
		&productID, &locationID, &sku,
		&quantityOnHand, &quantityReserved, &quantityAvailable,
		&reorderLevel, &unitOfMeasure, &trackingMode,
		&productVariantID, &notes, &productName, &productPrice,
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

// NewInventoryItemRepository creates a new SQL Server inventory item repository (old-style constructor).
func NewInventoryItemRepository(db *sql.DB, tableName string) inventoryitempb.InventoryItemDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerInventoryItemRepository(dbOps, tableName)
}
