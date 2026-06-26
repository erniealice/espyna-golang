//go:build sqlserver

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// purchaseOrderLineItemSortableSQLCols is the fail-closed sort whitelist (A2).
var purchaseOrderLineItemSortableSQLCols = []string{
	"purchase_order_id",
	"product_id",
	"description",
	"line_type",
	"quantity_ordered",
	"quantity_received",
	"quantity_billed",
	"unit_price",
	"total_price",
	"line_number",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PurchaseOrderLineItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver purchase_order_line_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPurchaseOrderLineItemRepository(db, dbOps, tableName), nil
	})
}

// SQLServerPurchaseOrderLineItemRepository implements purchase order line item CRUD using SQL Server.
type SQLServerPurchaseOrderLineItemRepository struct {
	purchaseorderlineitempb.UnimplementedPurchaseOrderLineItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerPurchaseOrderLineItemRepository creates a new SQL Server purchase order line item repository.
func NewSQLServerPurchaseOrderLineItemRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer {
	if tableName == "" {
		tableName = "purchase_order_line_item"
	}
	return &SQLServerPurchaseOrderLineItemRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerPurchaseOrderLineItemRepository) CreatePurchaseOrderLineItem(ctx context.Context, req *purchaseorderlineitempb.CreatePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.CreatePurchaseOrderLineItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("purchase order line item data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "requiredByDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase_order_line_item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	poli := &purchaseorderlineitempb.PurchaseOrderLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, poli); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &purchaseorderlineitempb.CreatePurchaseOrderLineItemResponse{Success: true, Data: []*purchaseorderlineitempb.PurchaseOrderLineItem{poli}}, nil
}

func (r *SQLServerPurchaseOrderLineItemRepository) ReadPurchaseOrderLineItem(ctx context.Context, req *purchaseorderlineitempb.ReadPurchaseOrderLineItemRequest) (*purchaseorderlineitempb.ReadPurchaseOrderLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order line item ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read purchase_order_line_item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	poli := &purchaseorderlineitempb.PurchaseOrderLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, poli); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &purchaseorderlineitempb.ReadPurchaseOrderLineItemResponse{Success: true, Data: []*purchaseorderlineitempb.PurchaseOrderLineItem{poli}}, nil
}

func (r *SQLServerPurchaseOrderLineItemRepository) UpdatePurchaseOrderLineItem(ctx context.Context, req *purchaseorderlineitempb.UpdatePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.UpdatePurchaseOrderLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order line item ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "requiredByDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update purchase_order_line_item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	poli := &purchaseorderlineitempb.PurchaseOrderLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, poli); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &purchaseorderlineitempb.UpdatePurchaseOrderLineItemResponse{Success: true, Data: []*purchaseorderlineitempb.PurchaseOrderLineItem{poli}}, nil
}

func (r *SQLServerPurchaseOrderLineItemRepository) DeletePurchaseOrderLineItem(ctx context.Context, req *purchaseorderlineitempb.DeletePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.DeletePurchaseOrderLineItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order line item ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete purchase_order_line_item: %w", err)
	}
	return &purchaseorderlineitempb.DeletePurchaseOrderLineItemResponse{Success: true}, nil
}

func (r *SQLServerPurchaseOrderLineItemRepository) ListPurchaseOrderLineItems(ctx context.Context, req *purchaseorderlineitempb.ListPurchaseOrderLineItemsRequest) (*purchaseorderlineitempb.ListPurchaseOrderLineItemsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list purchase_order_line_items: %w", err)
	}
	var polis []*purchaseorderlineitempb.PurchaseOrderLineItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal purchase_order_line_item row: %v", err)
			continue
		}
		poli := &purchaseorderlineitempb.PurchaseOrderLineItem{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, poli); err != nil {
			log.Printf("WARN: protojson unmarshal purchase_order_line_item: %v", err)
			continue
		}
		polis = append(polis, poli)
	}
	return &purchaseorderlineitempb.ListPurchaseOrderLineItemsResponse{Success: true, Data: polis}, nil
}

// GetPurchaseOrderLineItemListPageData retrieves purchase order line items with pagination.
func (r *SQLServerPurchaseOrderLineItemRepository) GetPurchaseOrderLineItemListPageData(ctx context.Context, req *purchaseorderlineitempb.GetPurchaseOrderLineItemListPageDataRequest) (*purchaseorderlineitempb.GetPurchaseOrderLineItemListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get purchase order line item list page data request is required")
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

	orderBy, err := sqlserverCore.BuildOrderBy(purchaseOrderLineItemSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				[poli].[id],
				[poli].[purchase_order_id],
				[poli].[product_id],
				[poli].[description],
				[poli].[line_type],
				[poli].[quantity_ordered],
				[poli].[quantity_received],
				[poli].[quantity_billed],
				[poli].[unit_price],
				[poli].[total_price],
				[poli].[location_id],
				[poli].[inventory_item_id],
				[poli].[required_by_date],
				[poli].[notes],
				[poli].[line_number],
				[poli].[active],
				[poli].[date_created],
				[poli].[date_modified],
				COUNT(*) OVER() AS [total]
			FROM [%s] [poli]
			WHERE [poli].[active] = 1
			  AND (@p1 IS NULL OR @p1 = '' OR [poli].[description] LIKE @p1)
		)
		SELECT * FROM enriched
		`+orderBy+`
		OFFSET @p2 ROWS FETCH NEXT @p3 ROWS ONLY
	`, r.tableName)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase_order_line_item list page data: %w", err)
	}
	defer rows.Close()

	var polis []*purchaseorderlineitempb.PurchaseOrderLineItem
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			purchaseOrderID string
			productID       *string
			description     string
			lineType        string
			qtyOrdered      float64
			qtyReceived     float64
			qtyBilled       float64
			unitPrice       int64
			totalPrice      int64
			locationID      *string
			inventoryItemID *string
			requiredByDate  *time.Time
			notes           *string
			lineNumber      int32
			active          bool
			dateCreated     time.Time
			dateModified    time.Time
			total           int64
		)
		if err := rows.Scan(
			&id, &purchaseOrderID, &productID, &description, &lineType,
			&qtyOrdered, &qtyReceived, &qtyBilled, &unitPrice, &totalPrice,
			&locationID, &inventoryItemID, &requiredByDate, &notes, &lineNumber,
			&active, &dateCreated, &dateModified, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan purchase_order_line_item row: %w", err)
		}
		totalCount = total

		poli := &purchaseorderlineitempb.PurchaseOrderLineItem{
			Id:               id,
			PurchaseOrderId:  purchaseOrderID,
			Description:      description,
			LineType:         lineType,
			QuantityOrdered:  qtyOrdered,
			QuantityReceived: qtyReceived,
			QuantityBilled:   qtyBilled,
			UnitPrice:        unitPrice,
			TotalPrice:       totalPrice,
			LineNumber:       lineNumber,
			Active:           active,
			ProductId:        productID,
			LocationId:       locationID,
			InventoryItemId:  inventoryItemID,
			Notes:            notes,
		}
		if requiredByDate != nil && !requiredByDate.IsZero() {
			ts := requiredByDate.UnixMilli()
			poli.RequiredByDate = &ts
			rbdStr := requiredByDate.Format("2006-01-02")
			poli.RequiredByDateString = &rbdStr
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			poli.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			poli.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			poli.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			poli.DateModifiedString = &dmStr
		}
		polis = append(polis, poli)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating purchase_order_line_item rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &purchaseorderlineitempb.GetPurchaseOrderLineItemListPageDataResponse{
		PurchaseOrderLineItemList: polis,
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

// GetPurchaseOrderLineItemItemPageData retrieves a single purchase order line item.
func (r *SQLServerPurchaseOrderLineItemRepository) GetPurchaseOrderLineItemItemPageData(ctx context.Context, req *purchaseorderlineitempb.GetPurchaseOrderLineItemItemPageDataRequest) (*purchaseorderlineitempb.GetPurchaseOrderLineItemItemPageDataResponse, error) {
	if req == nil || req.PurchaseOrderLineItemId == "" {
		return nil, fmt.Errorf("purchase order line item ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PurchaseOrderLineItemId)
	if err != nil {
		return nil, fmt.Errorf("failed to read purchase_order_line_item '%s': %w", req.PurchaseOrderLineItemId, err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	poli := &purchaseorderlineitempb.PurchaseOrderLineItem{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, poli); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &purchaseorderlineitempb.GetPurchaseOrderLineItemItemPageDataResponse{PurchaseOrderLineItem: poli, Success: true}, nil
}
