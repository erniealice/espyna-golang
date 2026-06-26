//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders)
//   - "ident"   → `ident` (backtick quoting)
//   - ILIKE     → LIKE (ci collation)
//   - active = true → active = 1
//   - LIMIT $2 OFFSET $3 → LIMIT ? OFFSET ?
//   - COUNT(*) OVER () stays (MySQL 8.0+ window function)
//
// CRITICAL: workspace_id isolation enforced on every raw-SQL query.
// Centavos (unit_price, total_price) are never scaled in SQL.
package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.RevenueLineItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql revenue_line_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLRevenueLineItemRepository(dbOps, tableName), nil
	})
}

// MySQLRevenueLineItemRepository implements revenue_line_item CRUD using MySQL 8.0+.
type MySQLRevenueLineItemRepository struct {
	revenuelineitempb.UnimplementedRevenueLineItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLRevenueLineItemRepository creates a new MySQL revenue line item repository.
func NewMySQLRevenueLineItemRepository(dbOps interfaces.DatabaseOperation, tableName string) revenuelineitempb.RevenueLineItemDomainServiceServer {
	if tableName == "" {
		tableName = "revenue_line_item"
	}
	return &MySQLRevenueLineItemRepository{
		dbOps:     dbOps,
		db:        getDB(dbOps),
		tableName: tableName,
	}
}

// CreateRevenueLineItem creates a new revenue line item.
func (r *MySQLRevenueLineItemRepository) CreateRevenueLineItem(ctx context.Context, req *revenuelineitempb.CreateRevenueLineItemRequest) (*revenuelineitempb.CreateRevenueLineItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue line item data is required")
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
		return nil, fmt.Errorf("failed to create revenue line item: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	lineItem := &revenuelineitempb.RevenueLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuelineitempb.CreateRevenueLineItemResponse{
		Data: []*revenuelineitempb.RevenueLineItem{lineItem},
	}, nil
}

// ReadRevenueLineItem retrieves a revenue line item by ID.
func (r *MySQLRevenueLineItemRepository) ReadRevenueLineItem(ctx context.Context, req *revenuelineitempb.ReadRevenueLineItemRequest) (*revenuelineitempb.ReadRevenueLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue line item ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue line item: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	lineItem := &revenuelineitempb.RevenueLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuelineitempb.ReadRevenueLineItemResponse{
		Data: []*revenuelineitempb.RevenueLineItem{lineItem},
	}, nil
}

// UpdateRevenueLineItem updates a revenue line item.
func (r *MySQLRevenueLineItemRepository) UpdateRevenueLineItem(ctx context.Context, req *revenuelineitempb.UpdateRevenueLineItemRequest) (*revenuelineitempb.UpdateRevenueLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue line item ID is required")
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
		return nil, fmt.Errorf("failed to update revenue line item: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	lineItem := &revenuelineitempb.RevenueLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuelineitempb.UpdateRevenueLineItemResponse{
		Data: []*revenuelineitempb.RevenueLineItem{lineItem},
	}, nil
}

// DeleteRevenueLineItem soft-deletes a revenue line item.
func (r *MySQLRevenueLineItemRepository) DeleteRevenueLineItem(ctx context.Context, req *revenuelineitempb.DeleteRevenueLineItemRequest) (*revenuelineitempb.DeleteRevenueLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue line item ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete revenue line item: %w", err)
	}

	return &revenuelineitempb.DeleteRevenueLineItemResponse{Success: true}, nil
}

// ListRevenueLineItems lists revenue line items with optional filters.
func (r *MySQLRevenueLineItemRepository) ListRevenueLineItems(ctx context.Context, req *revenuelineitempb.ListRevenueLineItemsRequest) (*revenuelineitempb.ListRevenueLineItemsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenue line items: %w", err)
	}

	var lineItems []*revenuelineitempb.RevenueLineItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal revenue_line_item row: %v", err)
			continue
		}
		lineItem := &revenuelineitempb.RevenueLineItem{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, lineItem); err != nil {
			log.Printf("WARN: protojson unmarshal revenue_line_item: %v", err)
			continue
		}
		lineItems = append(lineItems, lineItem)
	}

	return &revenuelineitempb.ListRevenueLineItemsResponse{Data: lineItems}, nil
}

// GetRevenueLineItemListPageData retrieves revenue line items with pagination,
// filtering, sorting, and search using a counted CTE. Joins revenue and product
// tables for enriched display.
//
// Dialect: $1/$2/$3 → ?; ILIKE → LIKE; active = true → active = 1;
// COUNT(*) OVER() stays (MySQL 8.0+).
func (r *MySQLRevenueLineItemRepository) GetRevenueLineItemListPageData(
	ctx context.Context,
	req *revenuelineitempb.GetRevenueLineItemListPageDataRequest,
) (*revenuelineitempb.GetRevenueLineItemListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue line item list page data request is required")
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

	sortField := "rli.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Dialect: $1::text IS NULL OR ... ILIKE $1 →
	// (? = '' OR rli.description LIKE ? OR p.name LIKE ? OR rv.name LIKE ?)
	// Pass searchPattern four times: empty-check + three LIKE args.
	query := `
		WITH enriched AS (
			SELECT
				rli.id,
				rli.date_created,
				rli.date_modified,
				rli.active,
				rli.revenue_id,
				rli.product_id,
				rli.description,
				rli.quantity,
				rli.unit_price,
				rli.total_price,
				rli.notes,
				rli.line_item_type,
				rli.inventory_item_id,
				rli.inventory_serial_id,
				rli.product_price_plan_id,
				rli.price_product_id,
				COALESCE(rv.name, '') as revenue_name,
				COALESCE(p.name, '') as product_name
			FROM revenue_line_item rli
			LEFT JOIN revenue rv ON rli.revenue_id = rv.id AND rv.active = 1
			LEFT JOIN product p ON rli.product_id = p.id AND p.active = 1
			WHERE rli.active = 1
			  AND (? = '' OR rli.description LIKE ? OR p.name LIKE ? OR rv.name LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT ? OFFSET ?
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue line item list page data: %w", err)
	}
	defer rows.Close()

	var lineItems []*revenuelineitempb.RevenueLineItem
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			dateCreated        time.Time
			dateModified       time.Time
			active             bool
			revenueID          string
			productID          *string
			description        string
			quantity           float64
			unitPrice          int64
			totalPrice         int64
			notes              *string
			lineItemType       *string
			inventoryItemID    *string
			inventorySerialID  *string
			productPricePlanID *string
			priceProductID     *string
			revenueName        string
			productName        string
			total              int64
		)

		if err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&revenueID,
			&productID,
			&description,
			&quantity,
			&unitPrice,
			&totalPrice,
			&notes,
			&lineItemType,
			&inventoryItemID,
			&inventorySerialID,
			&productPricePlanID,
			&priceProductID,
			&revenueName,
			&productName,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan revenue line item row: %w", err)
		}

		totalCount = total

		lineItem := &revenuelineitempb.RevenueLineItem{
			Id:          id,
			Active:      active,
			RevenueId:   revenueID,
			ProductId:   productID,
			Description: description,
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  totalPrice,
			Notes:       notes,
		}

		if lineItemType != nil {
			lineItem.LineItemType = *lineItemType
		}
		if inventoryItemID != nil {
			lineItem.InventoryItemId = *inventoryItemID
		}
		if inventorySerialID != nil {
			lineItem.InventorySerialId = *inventorySerialID
		}
		// ProductPricePlanId removed from proto schema
		_ = productPricePlanID
		if priceProductID != nil {
			lineItem.PriceProductId = priceProductID
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			lineItem.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			lineItem.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			lineItem.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			lineItem.DateModifiedString = &dmStr
		}

		lineItems = append(lineItems, lineItem)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue line item rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenuelineitempb.GetRevenueLineItemListPageDataResponse{
		RevenueLineItemList: lineItems,
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

// GetRevenueLineItemItemPageData retrieves a single revenue line item with
// enriched data using CTE.
//
// Dialect: $1 → ?; active = true → active = 1.
func (r *MySQLRevenueLineItemRepository) GetRevenueLineItemItemPageData(
	ctx context.Context,
	req *revenuelineitempb.GetRevenueLineItemItemPageDataRequest,
) (*revenuelineitempb.GetRevenueLineItemItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue line item item page data request is required")
	}
	if req.RevenueLineItemId == "" {
		return nil, fmt.Errorf("revenue line item ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				rli.id,
				rli.date_created,
				rli.date_modified,
				rli.active,
				rli.revenue_id,
				rli.product_id,
				rli.description,
				rli.quantity,
				rli.unit_price,
				rli.total_price,
				rli.notes,
				rli.line_item_type,
				rli.inventory_item_id,
				rli.inventory_serial_id,
				rli.product_price_plan_id,
				rli.price_product_id,
				COALESCE(rv.name, '') as revenue_name,
				COALESCE(p.name, '') as product_name
			FROM revenue_line_item rli
			LEFT JOIN revenue rv ON rli.revenue_id = rv.id AND rv.active = 1
			LEFT JOIN product p ON rli.product_id = p.id AND p.active = 1
			WHERE rli.id = ? AND rli.active = 1
		)
		SELECT * FROM enriched LIMIT 1
	`

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	row := r.db.QueryRowContext(ctx, query, req.RevenueLineItemId)

	var (
		id                 string
		dateCreated        time.Time
		dateModified       time.Time
		active             bool
		revenueID          string
		productID          *string
		description        string
		quantity           float64
		unitPrice          int64
		totalPrice         int64
		notes              *string
		lineItemType       *string
		inventoryItemID    *string
		inventorySerialID  *string
		productPricePlanID *string
		priceProductID     *string
		revenueName        string
		productName        string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&revenueID,
		&productID,
		&description,
		&quantity,
		&unitPrice,
		&totalPrice,
		&notes,
		&lineItemType,
		&inventoryItemID,
		&inventorySerialID,
		&productPricePlanID,
		&priceProductID,
		&revenueName,
		&productName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue line item with ID '%s' not found", req.RevenueLineItemId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue line item item page data: %w", err)
	}

	lineItem := &revenuelineitempb.RevenueLineItem{
		Id:          id,
		Active:      active,
		RevenueId:   revenueID,
		ProductId:   productID,
		Description: description,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
		TotalPrice:  totalPrice,
		Notes:       notes,
	}

	if lineItemType != nil {
		lineItem.LineItemType = *lineItemType
	}
	if inventoryItemID != nil {
		lineItem.InventoryItemId = *inventoryItemID
	}
	if inventorySerialID != nil {
		lineItem.InventorySerialId = *inventorySerialID
	}
	// ProductPricePlanId removed from proto schema
	_ = productPricePlanID
	if priceProductID != nil {
		lineItem.PriceProductId = priceProductID
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		lineItem.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		lineItem.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		lineItem.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		lineItem.DateModifiedString = &dmStr
	}

	return &revenuelineitempb.GetRevenueLineItemItemPageDataResponse{
		RevenueLineItem: lineItem,
		Success:         true,
	}, nil
}

// NewRevenueLineItemRepository creates a new MySQL revenue line item repository (old-style constructor).
func NewRevenueLineItemRepository(db *sql.DB, tableName string) revenuelineitempb.RevenueLineItemDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLRevenueLineItemRepository(dbOps, tableName)
}
