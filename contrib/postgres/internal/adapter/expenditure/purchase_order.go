package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PurchaseOrder, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres purchase_order repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPurchaseOrderRepository(dbOps, tableName), nil
	})
}

// PostgresPurchaseOrderRepository implements purchase order CRUD operations using PostgreSQL
type PostgresPurchaseOrderRepository struct {
	purchaseorderpb.UnimplementedPurchaseOrderDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPurchaseOrderRepository creates a new PostgreSQL purchase order repository
func NewPostgresPurchaseOrderRepository(dbOps interfaces.DatabaseOperation, tableName string) purchaseorderpb.PurchaseOrderDomainServiceServer {
	if tableName == "" {
		tableName = "purchase_order"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPurchaseOrderRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePurchaseOrder creates a new purchase order record
func (r *PostgresPurchaseOrderRepository) CreatePurchaseOrder(ctx context.Context, req *purchaseorderpb.CreatePurchaseOrderRequest) (*purchaseorderpb.CreatePurchaseOrderResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("purchase order data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "orderDate", "order_date")
	convertMillisToTime(data, "expectedDeliveryDate", "expected_delivery_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase order: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	po := &purchaseorderpb.PurchaseOrder{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &purchaseorderpb.CreatePurchaseOrderResponse{
		Success: true,
		Data:    []*purchaseorderpb.PurchaseOrder{po},
	}, nil
}

// ReadPurchaseOrder retrieves a purchase order record by ID
func (r *PostgresPurchaseOrderRepository) ReadPurchaseOrder(ctx context.Context, req *purchaseorderpb.ReadPurchaseOrderRequest) (*purchaseorderpb.ReadPurchaseOrderResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read purchase order: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	po := &purchaseorderpb.PurchaseOrder{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &purchaseorderpb.ReadPurchaseOrderResponse{
		Success: true,
		Data:    []*purchaseorderpb.PurchaseOrder{po},
	}, nil
}

// UpdatePurchaseOrder updates a purchase order record
func (r *PostgresPurchaseOrderRepository) UpdatePurchaseOrder(ctx context.Context, req *purchaseorderpb.UpdatePurchaseOrderRequest) (*purchaseorderpb.UpdatePurchaseOrderResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "orderDate", "order_date")
	convertMillisToTime(data, "expectedDeliveryDate", "expected_delivery_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update purchase order: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	po := &purchaseorderpb.PurchaseOrder{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, po); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &purchaseorderpb.UpdatePurchaseOrderResponse{
		Success: true,
		Data:    []*purchaseorderpb.PurchaseOrder{po},
	}, nil
}

// DeletePurchaseOrder deletes a purchase order record (soft delete)
func (r *PostgresPurchaseOrderRepository) DeletePurchaseOrder(ctx context.Context, req *purchaseorderpb.DeletePurchaseOrderRequest) (*purchaseorderpb.DeletePurchaseOrderResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete purchase order: %w", err)
	}

	return &purchaseorderpb.DeletePurchaseOrderResponse{
		Success: true,
	}, nil
}

// ListPurchaseOrders lists purchase order records with optional filters
func (r *PostgresPurchaseOrderRepository) ListPurchaseOrders(ctx context.Context, req *purchaseorderpb.ListPurchaseOrdersRequest) (*purchaseorderpb.ListPurchaseOrdersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list purchase orders: %w", err)
	}

	var pos []*purchaseorderpb.PurchaseOrder
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
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

	return &purchaseorderpb.ListPurchaseOrdersResponse{
		Success: true,
		Data:    pos,
	}, nil
}

// GetPurchaseOrderListPageData retrieves purchase orders with pagination, filtering, sorting, and search using CTE
func (r *PostgresPurchaseOrderRepository) GetPurchaseOrderListPageData(
	ctx context.Context,
	req *purchaseorderpb.GetPurchaseOrderListPageDataRequest,
) (*purchaseorderpb.GetPurchaseOrderListPageDataResponse, error) {
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

	sortField := "po.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				po.id,
				po.po_number,
				po.po_type,
				po.status,
				po.supplier_id,
				po.location_id,
				po.order_date,
				po.expected_delivery_date,
				po.currency,
				po.subtotal,
				po.tax_amount,
				po.total_amount,
				po.payment_terms,
				po.shipping_terms,
				po.approved_by,
				po.notes,
				po.reference_number,
				po.active,
				po.date_created,
				po.date_modified,
				po.parent_po_id,
				po.payment_term_id
			FROM %s po
			WHERE po.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       po.po_number ILIKE $1 OR
			       po.reference_number ILIKE $1 OR
			       po.status ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT $2 OFFSET $3;
	`, r.tableName, sortField, sortOrder)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase order list page data: %w", err)
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

		err := rows.Scan(
			&id,
			&poNumber,
			&poType,
			&status,
			&supplierID,
			&locationID,
			&orderDate,
			&expectedDeliveryDate,
			&currency,
			&subtotal,
			&taxAmount,
			&totalAmount,
			&paymentTerms,
			&shippingTerms,
			&approvedBy,
			&notes,
			&referenceNumber,
			&active,
			&dateCreated,
			&dateModified,
			&parentPoID,
			&paymentTermID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase order row: %w", err)
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
		}

		if locationID != nil {
			po.LocationId = locationID
		}
		if paymentTerms != nil {
			po.PaymentTerms = paymentTerms
		}
		if shippingTerms != nil {
			po.ShippingTerms = shippingTerms
		}
		if approvedBy != nil {
			po.ApprovedBy = approvedBy
		}
		if notes != nil {
			po.Notes = notes
		}
		if referenceNumber != nil {
			po.ReferenceNumber = referenceNumber
		}
		if parentPoID != nil {
			po.ParentPoId = parentPoID
		}
		if paymentTermID != nil {
			po.PaymentTermId = paymentTermID
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
		return nil, fmt.Errorf("error iterating purchase order rows: %w", err)
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

// GetPurchaseOrderItemPageData retrieves a single purchase order with enriched data
func (r *PostgresPurchaseOrderRepository) GetPurchaseOrderItemPageData(
	ctx context.Context,
	req *purchaseorderpb.GetPurchaseOrderItemPageDataRequest,
) (*purchaseorderpb.GetPurchaseOrderItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get purchase order item page data request is required")
	}
	if req.PurchaseOrderId == "" {
		return nil, fmt.Errorf("purchase order ID is required")
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				po.id,
				po.po_number,
				po.po_type,
				po.status,
				po.supplier_id,
				po.location_id,
				po.order_date,
				po.expected_delivery_date,
				po.currency,
				po.subtotal,
				po.tax_amount,
				po.total_amount,
				po.payment_terms,
				po.shipping_terms,
				po.approved_by,
				po.notes,
				po.reference_number,
				po.active,
				po.date_created,
				po.date_modified,
				po.parent_po_id,
				po.payment_term_id
			FROM %s po
			WHERE po.id = $1 AND po.active = true
		)
		SELECT * FROM enriched LIMIT 1;
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
		&id,
		&poNumber,
		&poType,
		&status,
		&supplierID,
		&locationID,
		&orderDate,
		&expectedDeliveryDate,
		&currency,
		&subtotal,
		&taxAmount,
		&totalAmount,
		&paymentTerms,
		&shippingTerms,
		&approvedBy,
		&notes,
		&referenceNumber,
		&active,
		&dateCreated,
		&dateModified,
		&parentPoID,
		&paymentTermID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("purchase order with ID '%s' not found", req.PurchaseOrderId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query purchase order item page data: %w", err)
	}

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
	}

	if locationID != nil {
		po.LocationId = locationID
	}
	if paymentTerms != nil {
		po.PaymentTerms = paymentTerms
	}
	if shippingTerms != nil {
		po.ShippingTerms = shippingTerms
	}
	if approvedBy != nil {
		po.ApprovedBy = approvedBy
	}
	if notes != nil {
		po.Notes = notes
	}
	if referenceNumber != nil {
		po.ReferenceNumber = referenceNumber
	}
	if parentPoID != nil {
		po.ParentPoId = parentPoID
	}
	if paymentTermID != nil {
		po.PaymentTermId = paymentTermID
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

	return &purchaseorderpb.GetPurchaseOrderItemPageDataResponse{
		PurchaseOrder: po,
		Success:       true,
	}, nil
}

// NewPurchaseOrderRepository creates a new PostgreSQL purchase order repository (old-style constructor)
func NewPurchaseOrderRepository(db *sql.DB, tableName string) purchaseorderpb.PurchaseOrderDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPurchaseOrderRepository(dbOps, tableName)
}
