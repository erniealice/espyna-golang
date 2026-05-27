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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// purchaseOrderSortableSQLCols is the fail-closed sort whitelist (A2).
var purchaseOrderSortableSQLCols = []string{
	"po_number",
	"po_type",
	"status",
	"order_date",
	"expected_delivery_date",
	"currency",
	"subtotal",
	"tax_amount",
	"total_amount",
	"reference_number",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PurchaseOrder, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver purchase_order repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPurchaseOrderRepository(db, dbOps, tableName), nil
	})
}

// SQLServerPurchaseOrderRepository implements purchase order CRUD using SQL Server.
type SQLServerPurchaseOrderRepository struct {
	purchaseorderpb.UnimplementedPurchaseOrderDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerPurchaseOrderRepository creates a new SQL Server purchase order repository.
func NewSQLServerPurchaseOrderRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) purchaseorderpb.PurchaseOrderDomainServiceServer {
	if tableName == "" {
		tableName = "purchase_order"
	}
	return &SQLServerPurchaseOrderRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerPurchaseOrderRepository) CreatePurchaseOrder(ctx context.Context, req *purchaseorderpb.CreatePurchaseOrderRequest) (*purchaseorderpb.CreatePurchaseOrderResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("purchase order data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "orderDate")
	convertMillisToTime(data, "expectedDeliveryDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase_order: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	po := &purchaseorderpb.PurchaseOrder{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &purchaseorderpb.CreatePurchaseOrderResponse{Success: true, Data: []*purchaseorderpb.PurchaseOrder{po}}, nil
}

func (r *SQLServerPurchaseOrderRepository) ReadPurchaseOrder(ctx context.Context, req *purchaseorderpb.ReadPurchaseOrderRequest) (*purchaseorderpb.ReadPurchaseOrderResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read purchase_order: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	po := &purchaseorderpb.PurchaseOrder{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &purchaseorderpb.ReadPurchaseOrderResponse{Success: true, Data: []*purchaseorderpb.PurchaseOrder{po}}, nil
}

func (r *SQLServerPurchaseOrderRepository) UpdatePurchaseOrder(ctx context.Context, req *purchaseorderpb.UpdatePurchaseOrderRequest) (*purchaseorderpb.UpdatePurchaseOrderResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "orderDate")
	convertMillisToTime(data, "expectedDeliveryDate")
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update purchase_order: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	po := &purchaseorderpb.PurchaseOrder{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &purchaseorderpb.UpdatePurchaseOrderResponse{Success: true, Data: []*purchaseorderpb.PurchaseOrder{po}}, nil
}

func (r *SQLServerPurchaseOrderRepository) DeletePurchaseOrder(ctx context.Context, req *purchaseorderpb.DeletePurchaseOrderRequest) (*purchaseorderpb.DeletePurchaseOrderResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete purchase_order: %w", err)
	}
	return &purchaseorderpb.DeletePurchaseOrderResponse{Success: true}, nil
}

func (r *SQLServerPurchaseOrderRepository) ListPurchaseOrders(ctx context.Context, req *purchaseorderpb.ListPurchaseOrdersRequest) (*purchaseorderpb.ListPurchaseOrdersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list purchase_orders: %w", err)
	}
	var pos []*purchaseorderpb.PurchaseOrder
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal purchase_order row: %v", err)
			continue
		}
		po := &purchaseorderpb.PurchaseOrder{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
			log.Printf("WARN: protojson unmarshal purchase_order: %v", err)
			continue
		}
		pos = append(pos, po)
	}
	return &purchaseorderpb.ListPurchaseOrdersResponse{Success: true, Data: pos}, nil
}

// GetPurchaseOrderListPageData retrieves purchase orders with pagination and search.
func (r *SQLServerPurchaseOrderRepository) GetPurchaseOrderListPageData(ctx context.Context, req *purchaseorderpb.GetPurchaseOrderListPageDataRequest) (*purchaseorderpb.GetPurchaseOrderListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get purchase order list page data request is required")
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

	orderBy, err := sqlserverCore.BuildOrderBy(purchaseOrderSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				[po].[id],
				[po].[po_number],
				[po].[po_type],
				[po].[status],
				[po].[supplier_id],
				[po].[location_id],
				[po].[order_date],
				[po].[expected_delivery_date],
				[po].[currency],
				[po].[subtotal],
				[po].[tax_amount],
				[po].[total_amount],
				[po].[payment_terms],
				[po].[shipping_terms],
				[po].[approved_by],
				[po].[notes],
				[po].[reference_number],
				[po].[active],
				[po].[date_created],
				[po].[date_modified],
				[po].[parent_po_id],
				[po].[payment_term_id],
				COUNT(*) OVER() AS [total]
			FROM [%s] [po]
			WHERE [po].[active] = 1
			  AND (@p1 IS NULL OR @p1 = '' OR
			       [po].[po_number] LIKE @p1 OR
			       [po].[reference_number] LIKE @p1 OR
			       [po].[status] LIKE @p1)
		)
		SELECT * FROM enriched
		`+orderBy+`
		OFFSET @p2 ROWS FETCH NEXT @p3 ROWS ONLY
	`, r.tableName)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase_order list page data: %w", err)
	}
	defer rows.Close()

	var pos []*purchaseorderpb.PurchaseOrder
	var totalCount int64

	for rows.Next() {
		var (
			id                   string
			poNumber             string
			poType               string
			status               string
			supplierID           string
			locationID           *string
			orderDate            time.Time
			expectedDeliveryDate *time.Time
			currency             string
			subtotal             int64
			taxAmount            int64
			totalAmount          int64
			paymentTerms         *string
			shippingTerms        *string
			approvedBy           *string
			notes                *string
			referenceNumber      *string
			active               bool
			dateCreated          time.Time
			dateModified         time.Time
			parentPoID           *string
			paymentTermID        *string
			total                int64
		)
		if err := rows.Scan(
			&id, &poNumber, &poType, &status, &supplierID, &locationID,
			&orderDate, &expectedDeliveryDate, &currency, &subtotal, &taxAmount, &totalAmount,
			&paymentTerms, &shippingTerms, &approvedBy, &notes, &referenceNumber,
			&active, &dateCreated, &dateModified, &parentPoID, &paymentTermID, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan purchase_order row: %w", err)
		}
		totalCount = total

		po := &purchaseorderpb.PurchaseOrder{
			Id:          id,
			PoNumber:    poNumber,
			PoType:      poType,
			Status:      status,
			SupplierId:  supplierID,
			Currency:    currency,
			Subtotal:    subtotal,
			TaxAmount:   taxAmount,
			TotalAmount: totalAmount,
			Active:      active,
			LocationId:  locationID, PaymentTerms: paymentTerms, ShippingTerms: shippingTerms,
			ApprovedBy: approvedBy, Notes: notes, ReferenceNumber: referenceNumber,
			ParentPoId: parentPoID, PaymentTermId: paymentTermID,
		}
		if !orderDate.IsZero() {
			ts := orderDate.UnixMilli()
			po.OrderDate = ts
			odStr := orderDate.Format("2006-01-02")
			po.OrderDateString = &odStr
		}
		if expectedDeliveryDate != nil && !expectedDeliveryDate.IsZero() {
			ts := expectedDeliveryDate.UnixMilli()
			po.ExpectedDeliveryDate = &ts
			eddStr := expectedDeliveryDate.Format("2006-01-02")
			po.ExpectedDeliveryDateString = &eddStr
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			po.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			po.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			po.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			po.DateModifiedString = &dmStr
		}
		pos = append(pos, po)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating purchase_order rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &purchaseorderpb.GetPurchaseOrderListPageDataResponse{
		PurchaseOrderList: pos,
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

// GetPurchaseOrderItemPageData retrieves a single purchase order.
func (r *SQLServerPurchaseOrderRepository) GetPurchaseOrderItemPageData(ctx context.Context, req *purchaseorderpb.GetPurchaseOrderItemPageDataRequest) (*purchaseorderpb.GetPurchaseOrderItemPageDataResponse, error) {
	if req == nil || req.PurchaseOrderId == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				[po].[id],
				[po].[po_number],
				[po].[po_type],
				[po].[status],
				[po].[supplier_id],
				[po].[location_id],
				[po].[order_date],
				[po].[expected_delivery_date],
				[po].[currency],
				[po].[subtotal],
				[po].[tax_amount],
				[po].[total_amount],
				[po].[payment_terms],
				[po].[shipping_terms],
				[po].[approved_by],
				[po].[notes],
				[po].[reference_number],
				[po].[active],
				[po].[date_created],
				[po].[date_modified],
				[po].[parent_po_id],
				[po].[payment_term_id]
			FROM [%s] [po]
			WHERE [po].[id] = @p1 AND [po].[active] = 1
		)
		SELECT TOP 1 * FROM enriched
	`, r.tableName)

	row := r.db.QueryRowContext(ctx, query, req.PurchaseOrderId)

	var (
		id                   string
		poNumber             string
		poType               string
		status               string
		supplierID           string
		locationID           *string
		orderDate            time.Time
		expectedDeliveryDate *time.Time
		currency             string
		subtotal             int64
		taxAmount            int64
		totalAmount          int64
		paymentTerms         *string
		shippingTerms        *string
		approvedBy           *string
		notes                *string
		referenceNumber      *string
		active               bool
		dateCreated          time.Time
		dateModified         time.Time
		parentPoID           *string
		paymentTermID        *string
	)
	err := row.Scan(
		&id, &poNumber, &poType, &status, &supplierID, &locationID,
		&orderDate, &expectedDeliveryDate, &currency, &subtotal, &taxAmount, &totalAmount,
		&paymentTerms, &shippingTerms, &approvedBy, &notes, &referenceNumber,
		&active, &dateCreated, &dateModified, &parentPoID, &paymentTermID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("purchase_order with ID '%s' not found", req.PurchaseOrderId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase_order item page data: %w", err)
	}

	po := &purchaseorderpb.PurchaseOrder{
		Id:              id,
		PoNumber:        poNumber,
		PoType:          poType,
		Status:          status,
		SupplierId:      supplierID,
		Currency:        currency,
		Subtotal:        subtotal,
		TaxAmount:       taxAmount,
		TotalAmount:     totalAmount,
		Active:          active,
		LocationId:      locationID,
		PaymentTerms:    paymentTerms,
		ShippingTerms:   shippingTerms,
		ApprovedBy:      approvedBy,
		Notes:           notes,
		ReferenceNumber: referenceNumber,
		ParentPoId:      parentPoID,
		PaymentTermId:   paymentTermID,
	}
	if !orderDate.IsZero() {
		ts := orderDate.UnixMilli()
		po.OrderDate = ts
		odStr := orderDate.Format("2006-01-02")
		po.OrderDateString = &odStr
	}
	if expectedDeliveryDate != nil && !expectedDeliveryDate.IsZero() {
		ts := expectedDeliveryDate.UnixMilli()
		po.ExpectedDeliveryDate = &ts
		eddStr := expectedDeliveryDate.Format("2006-01-02")
		po.ExpectedDeliveryDateString = &eddStr
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		po.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		po.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		po.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		po.DateModifiedString = &dmStr
	}

	return &purchaseorderpb.GetPurchaseOrderItemPageDataResponse{PurchaseOrder: po, Success: true}, nil
}
