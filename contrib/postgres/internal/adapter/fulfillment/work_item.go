//go:build postgresql

package fulfillment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Fulfillment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres fulfillment repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresFulfillmentRepository(dbOps, tableName), nil
	})
}

// PostgresFulfillmentRepository implements fulfillment CRUD operations using PostgreSQL
type PostgresFulfillmentRepository struct {
	pb.UnimplementedFulfillmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresFulfillmentRepository creates a new PostgreSQL fulfillment repository
func NewPostgresFulfillmentRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.FulfillmentDomainServiceServer {
	if tableName == "" {
		tableName = "fulfillment"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresFulfillmentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateFulfillment creates a new fulfillment record
func (r *PostgresFulfillmentRepository) CreateFulfillment(ctx context.Context, req *pb.CreateFulfillmentRequest) (*pb.CreateFulfillmentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("fulfillment data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateFulfillmentResponse{
		Data: fulfillment,
	}, nil
}

// GetFulfillment retrieves a fulfillment record by ID
func (r *PostgresFulfillmentRepository) GetFulfillment(ctx context.Context, req *pb.GetFulfillmentRequest) (*pb.GetFulfillmentResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.GetFulfillmentResponse{
		Data: fulfillment,
	}, nil
}

// UpdateFulfillment updates a fulfillment record
func (r *PostgresFulfillmentRepository) UpdateFulfillment(ctx context.Context, req *pb.UpdateFulfillmentRequest) (*pb.UpdateFulfillmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateFulfillmentResponse{
		Data: fulfillment,
	}, nil
}

// DeleteFulfillment soft-deletes a fulfillment record (SET active=false)
func (r *PostgresFulfillmentRepository) DeleteFulfillment(ctx context.Context, req *pb.DeleteFulfillmentRequest) (*pb.DeleteFulfillmentResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	query := `UPDATE fulfillment SET active = false, date_modified = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete fulfillment: %w", err)
	}

	return &pb.DeleteFulfillmentResponse{
		Success: true,
	}, nil
}

var fulfillmentSortableSQLCols = []string{
	"id", "active", "workspace_id", "revenue_id", "supplier_id",
	"delivery_mode", "status", "provider_status", "provider_reference",
	"delivery_cost", "currency", "expenditure_id", "scheduled_at", "delivered_at",
	"date_created", "date_modified",
}

var fulfillmentSortSpec = espynahttp.SortSpec{AllowedCols: fulfillmentSortableSQLCols}

// ListFulfillments lists fulfillment records with optional filters
func (r *PostgresFulfillmentRepository) ListFulfillments(ctx context.Context, req *pb.ListFulfillmentsRequest) (*pb.ListFulfillmentsResponse, error) {
	if err := espynahttp.ValidateSortColumns(fulfillmentSortSpec, req.GetSort(), "fulfillment"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filter
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillments: %w", err)
	}

	var fulfillments []*pb.Fulfillment
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal fulfillment row: %v", err)
			continue
		}

		fulfillment := &pb.Fulfillment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
			log.Printf("WARN: protojson unmarshal fulfillment: %v", err)
			continue
		}
		fulfillments = append(fulfillments, fulfillment)
	}

	return &pb.ListFulfillmentsResponse{
		Data: fulfillments,
	}, nil
}

// GetFulfillmentListPageData retrieves fulfillments with pagination, filtering, sorting, and search.
// It joins supplier name, counts line items, and counts status events via CTE.
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresFulfillmentRepository) GetFulfillmentListPageData(
	ctx context.Context,
	req *pb.GetFulfillmentListPageDataRequest,
) (*pb.GetFulfillmentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get fulfillment list page data request is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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

	sortField := "f.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		} else {
			sortOrder = "ASC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				f.id,
				f.date_created,
				f.date_modified,
				f.active,
				f.workspace_id,
				f.revenue_id,
				f.supplier_id,
				f.delivery_mode,
				f.status,
				f.provider_status,
				f.provider_reference,
				f.delivery_cost,
				f.currency,
				f.expenditure_id,
				f.scheduled_at,
				f.delivered_at,
				COALESCE(s.name, '') AS supplier_name,
				COUNT(DISTINCT fi.id) AS item_count,
				COUNT(DISTINCT fse.id) AS status_event_count
			FROM fulfillment f
			LEFT JOIN supplier s ON s.id = f.supplier_id AND s.active = true
			LEFT JOIN fulfillment_item fi ON fi.fulfillment_id = f.id
			LEFT JOIN fulfillment_status_event fse ON fse.fulfillment_id = f.id
			WHERE f.active = true
			  AND f.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
			       f.status ILIKE $2 OR f.provider_reference ILIKE $2)
			GROUP BY f.id, s.name
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.id,
			e.date_created,
			e.date_modified,
			e.active,
			e.workspace_id,
			e.revenue_id,
			e.supplier_id,
			e.delivery_mode,
			e.status,
			e.provider_status,
			e.provider_reference,
			e.delivery_cost,
			e.currency,
			e.expenditure_id,
			e.scheduled_at,
			e.delivered_at,
			e.supplier_name,
			e.item_count,
			e.status_event_count,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $3 OFFSET $4;
	`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillment list page data: %w", err)
	}
	defer rows.Close()

	var resultRows []*pb.FulfillmentListRow
	var totalCount int64

	for rows.Next() {
		var (
			id                string
			dateCreated       time.Time
			dateModified      time.Time
			active            bool
			workspaceID       string
			revenueID         string
			supplierID        sql.NullString
			deliveryMode string
			status            string
			providerStatus    string
			providerReference string
			deliveryCost      int64
			currency          string
			expenditureID     sql.NullString
			scheduledAt       sql.NullTime
			deliveredAt       sql.NullTime
			supplierName      string
			itemCount         int32
			statusEventCount  int32
			total             int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&workspaceID,
			&revenueID,
			&supplierID,
			&deliveryMode,
			&status,
			&providerStatus,
			&providerReference,
			&deliveryCost,
			&currency,
			&expenditureID,
			&scheduledAt,
			&deliveredAt,
			&supplierName,
			&itemCount,
			&statusEventCount,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fulfillment list row: %w", err)
		}

		totalCount = total

		f := &pb.Fulfillment{
			Id:                id,
			Active:            active,
			WorkspaceId:       workspaceID,
			RevenueId:         revenueID,
			DeliveryMode: deliveryMode,
			Status:            status,
			ProviderStatus:    providerStatus,
			ProviderReference: providerReference,
			DeliveryCost:      deliveryCost,
			Currency:          currency,
		}

		if supplierID.Valid {
			f.SupplierId = &supplierID.String
		}
		if expenditureID.Valid {
			f.ExpenditureId = &expenditureID.String
		}
		if scheduledAt.Valid {
			// stored as timestamptz, surfaced as proto Timestamp
			ts := scheduledAt.Time.UnixMilli()
			_ = ts // proto Timestamp — leave to use case layer for now
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			f.DateCreated = &ts
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			f.DateModified = &ts
		}

		row := &pb.FulfillmentListRow{
			Fulfillment:      f,
			SupplierName:     supplierName,
			ItemCount:        itemCount,
			StatusEventCount: statusEventCount,
		}
		resultRows = append(resultRows, row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetFulfillmentListPageDataResponse{
		Rows: resultRows,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetFulfillmentItemPageData retrieves a single fulfillment with its items, status events, and returns.
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresFulfillmentRepository) GetFulfillmentItemPageData(
	ctx context.Context,
	req *pb.GetFulfillmentItemPageDataRequest,
) (*pb.GetFulfillmentItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get fulfillment item page data request is required")
	}
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Fetch main fulfillment record with supplier name and revenue reference
	query := `
		SELECT
			f.id,
			f.date_created,
			f.date_modified,
			f.active,
			f.workspace_id,
			f.revenue_id,
			f.supplier_id,
			f.delivery_mode,
			f.status,
			f.provider_status,
			f.provider_reference,
			f.delivery_cost,
			f.currency,
			f.expenditure_id,
			COALESCE(s.name, '') AS supplier_name,
			COALESCE(CAST(f.revenue_id AS text), '') AS revenue_reference
		FROM fulfillment f
		LEFT JOIN supplier s ON s.id = f.supplier_id AND s.active = true
		WHERE f.id = $1 AND f.workspace_id = $2 AND f.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.Id, workspaceID)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		scannedWorkspaceID string
		revenueID         string
		supplierID        sql.NullString
		deliveryMode string
		status            string
		providerStatus    string
		providerReference string
		deliveryCost      int64
		currency          string
		expenditureID     sql.NullString
		supplierName      string
		revenueReference  string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&scannedWorkspaceID,
		&revenueID,
		&supplierID,
		&deliveryMode,
		&status,
		&providerStatus,
		&providerReference,
		&deliveryCost,
		&currency,
		&expenditureID,
		&supplierName,
		&revenueReference,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fulfillment with ID '%s' not found", req.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillment item page data: %w", err)
	}

	f := &pb.Fulfillment{
		Id:           id,
		Active:       active,
		WorkspaceId:  scannedWorkspaceID,
		RevenueId:    revenueID,
		DeliveryMode: deliveryMode,
		Status:       status,
		ProviderStatus:    providerStatus,
		ProviderReference: providerReference,
		DeliveryCost:      deliveryCost,
		Currency:          currency,
	}
	if supplierID.Valid {
		f.SupplierId = &supplierID.String
	}
	if expenditureID.Valid {
		f.ExpenditureId = &expenditureID.String
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		f.DateCreated = &ts
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		f.DateModified = &ts
	}

	// Fetch line items
	itemsQuery := `
		SELECT id, fulfillment_id, revenue_line_item_id, product_id, delivery_mode,
		       source_type, source_id, quantity_ordered, quantity_delivered, status, notes
		FROM fulfillment_item
		WHERE fulfillment_id = $1
		ORDER BY id ASC
	`
	itemRows, err := r.db.QueryContext(ctx, itemsQuery, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillment items: %w", err)
	}
	defer itemRows.Close()

	var items []*pb.FulfillmentItem
	for itemRows.Next() {
		var (
			itemID              string
			itemFulfillmentID   string
			revenueLineItemID   string
			productID           string
			itemMethod          string
			sourceType          sql.NullString
			sourceID            sql.NullString
			quantityOrdered     float64
			quantityDelivered   float64
			itemStatus          string
			notes               string
		)
		if err := itemRows.Scan(
			&itemID, &itemFulfillmentID, &revenueLineItemID, &productID, &itemMethod,
			&sourceType, &sourceID, &quantityOrdered, &quantityDelivered, &itemStatus, &notes,
		); err != nil {
			log.Printf("WARN: scan fulfillment_item row: %v", err)
			continue
		}
		item := &pb.FulfillmentItem{
			Id:                itemID,
			FulfillmentId:     itemFulfillmentID,
			RevenueLineItemId: revenueLineItemID,
			ProductId:         productID,
			DeliveryMode: itemMethod,
			QuantityOrdered:   quantityOrdered,
			QuantityDelivered: quantityDelivered,
			Status:            itemStatus,
			Notes:             notes,
		}
		if sourceType.Valid {
			item.SourceType = &sourceType.String
		}
		if sourceID.Valid {
			item.SourceId = &sourceID.String
		}
		items = append(items, item)
	}
	if err = itemRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_item rows: %w", err)
	}

	// Fetch status events
	eventsQuery := `
		SELECT id, fulfillment_id, from_status, to_status, provider_status, provider_reference,
		       triggered_by_id, reason, occurred_at
		FROM fulfillment_status_event
		WHERE fulfillment_id = $1
		ORDER BY occurred_at DESC
	`
	eventRows, err := r.db.QueryContext(ctx, eventsQuery, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillment status events: %w", err)
	}
	defer eventRows.Close()

	var events []*pb.FulfillmentStatusEvent
	for eventRows.Next() {
		var (
			eventID           int64
			eventFulfillID    string
			fromStatus        sql.NullString
			toStatus          string
			evtProviderStatus string
			evtProviderRef    string
			triggeredByID     sql.NullString
			reason            string
			occurredAt        sql.NullTime
		)
		if err := eventRows.Scan(
			&eventID, &eventFulfillID, &fromStatus, &toStatus, &evtProviderStatus, &evtProviderRef,
			&triggeredByID, &reason, &occurredAt,
		); err != nil {
			log.Printf("WARN: scan fulfillment_status_event row: %v", err)
			continue
		}
		evt := &pb.FulfillmentStatusEvent{
			Id:                eventID,
			FulfillmentId:     eventFulfillID,
			ToStatus:          toStatus,
			ProviderStatus:    evtProviderStatus,
			ProviderReference: evtProviderRef,
			Reason:            reason,
		}
		if fromStatus.Valid {
			evt.FromStatus = &fromStatus.String
		}
		if triggeredByID.Valid {
			evt.TriggeredById = &triggeredByID.String
		}
		events = append(events, evt)
	}
	if err = eventRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_status_event rows: %w", err)
	}

	// Fetch returns
	returnsQuery := `
		SELECT id, fulfillment_id, reason, status, refund_amount, currency,
		       processed_by_id, notes, active, date_created
		FROM fulfillment_return
		WHERE fulfillment_id = $1 AND active = true
		ORDER BY date_created DESC
	`
	returnRows, err := r.db.QueryContext(ctx, returnsQuery, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillment returns: %w", err)
	}
	defer returnRows.Close()

	var returns []*pb.FulfillmentReturn
	for returnRows.Next() {
		var (
			retID           string
			retFulfillID    string
			retReason       string
			retStatus       string
			refundAmount    sql.NullFloat64
			retCurrency     string
			processedByID   sql.NullString
			retNotes        string
			retActive       bool
			retDateCreated  time.Time
		)
		if err := returnRows.Scan(
			&retID, &retFulfillID, &retReason, &retStatus, &refundAmount, &retCurrency,
			&processedByID, &retNotes, &retActive, &retDateCreated,
		); err != nil {
			log.Printf("WARN: scan fulfillment_return row: %v", err)
			continue
		}
		ret := &pb.FulfillmentReturn{
			Id:            retID,
			FulfillmentId: retFulfillID,
			Reason:        retReason,
			Status:        retStatus,
			Currency:      retCurrency,
			Notes:         retNotes,
			Active:        retActive,
		}
		if refundAmount.Valid {
			refundAmtInt := int64(refundAmount.Float64)
			ret.RefundAmount = &refundAmtInt
		}
		if processedByID.Valid {
			ret.ProcessedById = &processedByID.String
		}
		if !retDateCreated.IsZero() {
			ts := retDateCreated.UnixMilli()
			ret.DateCreated = &ts
		}
		returns = append(returns, ret)
	}
	if err = returnRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_return rows: %w", err)
	}

	return &pb.GetFulfillmentItemPageDataResponse{
		Fulfillment:      f,
		Items:            items,
		StatusEvents:     events,
		Returns:          returns,
		SupplierName:     supplierName,
		RevenueReference: revenueReference,
	}, nil
}

// TransitionStatus atomically updates the fulfillment status and appends a status event.
// For DELIVERED status, delivered_at is also set.
func (r *PostgresFulfillmentRepository) TransitionStatus(
	ctx context.Context,
	req *pb.TransitionStatusRequest,
) (*pb.TransitionStatusResponse, error) {
	if req.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}
	if req.Event == "" {
		return nil, fmt.Errorf("transition event is required")
	}

	// Resolve to_status from event
	toStatus := resolveToStatus(req.Event)
	if toStatus == "" {
		return nil, fmt.Errorf("unknown transition event: %s", req.Event)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Read current status for from_status
	var fromStatus sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT status FROM fulfillment WHERE id = $1 AND active = true FOR UPDATE`, req.FulfillmentId).Scan(&fromStatus)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fulfillment with ID '%s' not found", req.FulfillmentId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read fulfillment for update: %w", err)
	}

	// Update fulfillment status
	updateQuery := `UPDATE fulfillment SET status = $1, date_modified = NOW()`
	args := []any{toStatus}

	if toStatus == "DELIVERED" {
		updateQuery += `, delivered_at = NOW()`
	}
	updateQuery += ` WHERE id = $2 AND active = true`
	args = append(args, req.FulfillmentId)

	_, err = tx.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update fulfillment status: %w", err)
	}

	// Insert status event
	var triggeredByID *string
	if req.TriggeredById != "" {
		triggeredByID = &req.TriggeredById
	}
	var fromStatusStr *string
	if fromStatus.Valid && fromStatus.String != "" {
		fromStatusStr = &fromStatus.String
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO fulfillment_status_event
			(fulfillment_id, from_status, to_status, provider_status, provider_reference, triggered_by_id, reason, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		req.FulfillmentId,
		fromStatusStr,
		toStatus,
		req.ProviderStatus,
		req.ProviderReference,
		triggeredByID,
		req.Reason,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert fulfillment status event: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit fulfillment status transition: %w", err)
	}

	// Read back updated record
	result, err := r.dbOps.Read(ctx, r.tableName, req.FulfillmentId)
	if err != nil {
		return nil, fmt.Errorf("failed to read updated fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.TransitionStatusResponse{
		Data: fulfillment,
	}, nil
}

// ListStatusEvents lists status events for a fulfillment, ordered by occurred_at DESC.
func (r *PostgresFulfillmentRepository) ListStatusEvents(
	ctx context.Context,
	req *pb.ListStatusEventsRequest,
) (*pb.ListStatusEventsResponse, error) {
	if req.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	query := `
		SELECT id, fulfillment_id, from_status, to_status, provider_status, provider_reference,
		       triggered_by_id, reason, occurred_at
		FROM fulfillment_status_event
		WHERE fulfillment_id = $1
		ORDER BY occurred_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, req.FulfillmentId)
	if err != nil {
		return nil, fmt.Errorf("failed to list fulfillment status events: %w", err)
	}
	defer rows.Close()

	var events []*pb.FulfillmentStatusEvent
	for rows.Next() {
		var (
			eventID           int64
			fulfillmentID     string
			fromStatus        sql.NullString
			toStatus          string
			providerStatus    string
			providerReference string
			triggeredByID     sql.NullString
			reason            string
			occurredAt        sql.NullTime
		)
		if err := rows.Scan(
			&eventID, &fulfillmentID, &fromStatus, &toStatus, &providerStatus, &providerReference,
			&triggeredByID, &reason, &occurredAt,
		); err != nil {
			log.Printf("WARN: scan fulfillment_status_event row: %v", err)
			continue
		}
		evt := &pb.FulfillmentStatusEvent{
			Id:                eventID,
			FulfillmentId:     fulfillmentID,
			ToStatus:          toStatus,
			ProviderStatus:    providerStatus,
			ProviderReference: providerReference,
			Reason:            reason,
		}
		if fromStatus.Valid {
			evt.FromStatus = &fromStatus.String
		}
		if triggeredByID.Valid {
			evt.TriggeredById = &triggeredByID.String
		}
		events = append(events, evt)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fulfillment_status_event rows: %w", err)
	}

	return &pb.ListStatusEventsResponse{
		Events: events,
	}, nil
}

// resolveToStatus maps a transition event name to a canonical status string.
func resolveToStatus(event string) string {
	switch event {
	case "mark_ready":
		return "READY"
	case "dispatch":
		return "IN_TRANSIT"
	case "deliver":
		return "DELIVERED"
	case "partial_deliver":
		return "PARTIALLY_DELIVERED"
	case "fail":
		return "FAILED"
	case "cancel":
		return "CANCELLED"
	case "reset":
		return "PENDING"
	default:
		return ""
	}
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}