//go:build sqlserver

package fulfillment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Fulfillment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver fulfillment repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerFulfillmentRepository(dbOps, tableName), nil
	})
}

// SQLServerFulfillmentRepository implements fulfillment CRUD operations using SQL Server.
type SQLServerFulfillmentRepository struct {
	pb.UnimplementedFulfillmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerFulfillmentRepository creates a new SQL Server fulfillment repository.
func NewSQLServerFulfillmentRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.FulfillmentDomainServiceServer {
	if tableName == "" {
		tableName = "fulfillment"
	}
	return &SQLServerFulfillmentRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// getExec extracts a DBExecutor from the dbOps wrapper.
func (r *SQLServerFulfillmentRepository) getExec(ctx context.Context) dbExecutor {
	return r.dbOps.(executorProvider).GetExecutor(ctx)
}

// CreateFulfillment creates a new fulfillment record.
func (r *SQLServerFulfillmentRepository) CreateFulfillment(ctx context.Context, req *pb.CreateFulfillmentRequest) (*pb.CreateFulfillmentResponse, error) {
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateFulfillmentResponse{Data: fulfillment}, nil
}

// GetFulfillment retrieves a fulfillment record by ID.
func (r *SQLServerFulfillmentRepository) GetFulfillment(ctx context.Context, req *pb.GetFulfillmentRequest) (*pb.GetFulfillmentResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.GetFulfillmentResponse{Data: fulfillment}, nil
}

// UpdateFulfillment updates a fulfillment record.
func (r *SQLServerFulfillmentRepository) UpdateFulfillment(ctx context.Context, req *pb.UpdateFulfillmentRequest) (*pb.UpdateFulfillmentResponse, error) {
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update fulfillment: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	fulfillment := &pb.Fulfillment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, fulfillment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateFulfillmentResponse{Data: fulfillment}, nil
}

// DeleteFulfillment soft-deletes a fulfillment record.
// SQL Server: active = 0; GETUTCDATE() instead of NOW().
func (r *SQLServerFulfillmentRepository) DeleteFulfillment(ctx context.Context, req *pb.DeleteFulfillmentRequest) (*pb.DeleteFulfillmentResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	const query = `UPDATE fulfillment SET active = 0, date_modified = GETUTCDATE() WHERE id = @p1`
	exec := r.getExec(ctx)
	if _, err := exec.ExecContext(ctx, query, req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete fulfillment: %w", err)
	}

	return &pb.DeleteFulfillmentResponse{Success: true}, nil
}

var fulfillmentSortableSQLCols = []string{
	"f.id", "f.active", "f.workspace_id", "f.revenue_id", "f.supplier_id",
	"f.delivery_mode", "f.status", "f.provider_status", "f.provider_reference",
	"f.delivery_cost", "f.currency", "f.expenditure_id", "f.scheduled_at",
	"f.delivered_at", "f.date_created", "f.date_modified",
}

// ListFulfillments lists fulfillment records with optional filters.
func (r *SQLServerFulfillmentRepository) ListFulfillments(ctx context.Context, req *pb.ListFulfillmentsRequest) (*pb.ListFulfillmentsResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

	return &pb.ListFulfillmentsResponse{Data: fulfillments}, nil
}

// GetFulfillmentListPageData retrieves fulfillments with pagination, filtering, sorting, and search.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders.
//   - active = true → active = 1.
//   - ILIKE → LIKE (default CI collation).
//   - Pagination: ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - COUNT(*) OVER() retained — SQL Server 2017+ supports it.
//   - s.active = true → s.active = 1.
func (r *SQLServerFulfillmentRepository) GetFulfillmentListPageData(
	ctx context.Context,
	req *pb.GetFulfillmentListPageDataRequest,
) (*pb.GetFulfillmentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get fulfillment list page data request is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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

	orderByClause, err := sqlserverCore.BuildOrderBy(fulfillmentSortableSQLCols, req.GetSort(), "f.date_created DESC")
	if err != nil {
		return nil, err
	}

	// @p1 = workspaceID. Filter/search start at @p2.
	searchFields := []string{"f.status", "f.provider_reference"}
	filterClauses, filterArgs, nextIdx := sqlserverCore.BuildFilterWhere(req.Filter, req.Search, searchFields, 2)

	whereSQL := "WHERE f.active = 1 AND f.workspace_id = @p1"
	if len(filterClauses) > 0 {
		whereSQL += " AND " + strings.Join(filterClauses, " AND ")
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs := append([]any{workspaceID}, filterArgs...)
	queryArgs = append(queryArgs, offset, limit)

	query := fmt.Sprintf(`
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
				COUNT(DISTINCT fse.id) AS status_event_count,
				COUNT(*) OVER() AS total_count
			FROM fulfillment f
			LEFT JOIN supplier s ON s.id = f.supplier_id AND s.active = 1
			LEFT JOIN fulfillment_item fi ON fi.fulfillment_id = f.id
			LEFT JOIN fulfillment_status_event fse ON fse.fulfillment_id = f.id
			%s
			GROUP BY
				f.id, f.date_created, f.date_modified, f.active, f.workspace_id,
				f.revenue_id, f.supplier_id, f.delivery_mode, f.status,
				f.provider_status, f.provider_reference, f.delivery_cost, f.currency,
				f.expenditure_id, f.scheduled_at, f.delivered_at, s.name
		)
		SELECT
			id, date_created, date_modified, active, workspace_id,
			revenue_id, supplier_id, delivery_mode, status, provider_status,
			provider_reference, delivery_cost, currency, expenditure_id,
			scheduled_at, delivered_at, supplier_name, item_count,
			status_event_count, total_count
		FROM enriched
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, orderByClause, offsetIdx, limitIdx)

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
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
			wsID              string
			revenueID         string
			supplierID        sql.NullString
			deliveryMode      string
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

		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active, &wsID,
			&revenueID, &supplierID, &deliveryMode, &status,
			&providerStatus, &providerReference, &deliveryCost, &currency,
			&expenditureID, &scheduledAt, &deliveredAt,
			&supplierName, &itemCount, &statusEventCount, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan fulfillment list row: %w", err)
		}

		totalCount = total

		f := &pb.Fulfillment{
			Id:                id,
			Active:            active,
			WorkspaceId:       wsID,
			RevenueId:         revenueID,
			DeliveryMode:      deliveryMode,
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
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			f.DateCreated = &ts
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			f.DateModified = &ts
		}

		resultRows = append(resultRows, &pb.FulfillmentListRow{
			Fulfillment:      f,
			SupplierName:     supplierName,
			ItemCount:        itemCount,
			StatusEventCount: statusEventCount,
		})
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
// CRITICAL: Always filters by workspace_id for multi-tenancy.
// SQL Server differences: @pN, active = 1, s.active = 1.
func (r *SQLServerFulfillmentRepository) GetFulfillmentItemPageData(
	ctx context.Context,
	req *pb.GetFulfillmentItemPageDataRequest,
) (*pb.GetFulfillmentItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get fulfillment item page data request is required")
	}
	if req.Id == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	const query = `
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
			COALESCE(CAST(f.revenue_id AS nvarchar(max)), '') AS revenue_reference
		FROM fulfillment f
		LEFT JOIN supplier s ON s.id = f.supplier_id AND s.active = 1
		WHERE f.id = @p1 AND f.workspace_id = @p2 AND f.active = 1
	`

	exec := r.getExec(ctx)
	row := exec.QueryRowContext(ctx, query, req.Id, workspaceID)

	var (
		id                string
		dateCreated       time.Time
		dateModified      time.Time
		active            bool
		wsID              string
		revenueID         string
		supplierID        sql.NullString
		deliveryMode      string
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
		&id, &dateCreated, &dateModified, &active, &wsID,
		&revenueID, &supplierID, &deliveryMode, &status,
		&providerStatus, &providerReference, &deliveryCost, &currency,
		&expenditureID, &supplierName, &revenueReference,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fulfillment with ID '%s' not found", req.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query fulfillment item page data: %w", err)
	}

	f := &pb.Fulfillment{
		Id:                id,
		Active:            active,
		WorkspaceId:       wsID,
		RevenueId:         revenueID,
		DeliveryMode:      deliveryMode,
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
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		f.DateCreated = &ts
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		f.DateModified = &ts
	}

	return &pb.GetFulfillmentItemPageDataResponse{
		Fulfillment:      f,
		SupplierName:     supplierName,
		RevenueReference: revenueReference,
	}, nil
}

// TransitionStatus transitions fulfillment to a new status.
// SQL Server: OUTPUT inserted.id instead of RETURNING.
func (r *SQLServerFulfillmentRepository) TransitionStatus(
	ctx context.Context,
	req *pb.TransitionStatusRequest,
) (*pb.TransitionStatusResponse, error) {
	if req == nil || req.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	const query = `
		UPDATE fulfillment
		SET status = @p1, date_modified = GETUTCDATE()
		OUTPUT inserted.id
		WHERE id = @p2 AND active = 1
	`

	exec := r.getExec(ctx)
	var id string
	if err := exec.QueryRowContext(ctx, query, req.Event, req.FulfillmentId).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("fulfillment not found: %s", req.FulfillmentId)
		}
		return nil, fmt.Errorf("failed to transition fulfillment status: %w", err)
	}

	return &pb.TransitionStatusResponse{Data: &pb.Fulfillment{Id: id}}, nil
}

// ListStatusEvents lists status events for a fulfillment.
func (r *SQLServerFulfillmentRepository) ListStatusEvents(
	ctx context.Context,
	req *pb.ListStatusEventsRequest,
) (*pb.ListStatusEventsResponse, error) {
	if req == nil || req.FulfillmentId == "" {
		return nil, fmt.Errorf("fulfillment ID is required")
	}

	const query = `
		SELECT id, fulfillment_id, from_status, to_status, reason, occurred_at
		FROM fulfillment_status_event
		WHERE fulfillment_id = @p1
		ORDER BY occurred_at ASC
	`

	exec := r.getExec(ctx)
	rows, err := exec.QueryContext(ctx, query, req.FulfillmentId)
	if err != nil {
		return nil, fmt.Errorf("failed to list status events: %w", err)
	}
	defer rows.Close()

	var events []*pb.FulfillmentStatusEvent
	for rows.Next() {
		var (
			id            int64
			fulfillmentID string
			fromStatus    sql.NullString
			toStatus      string
			reason        string
			occurredAt    time.Time
		)
		if err := rows.Scan(&id, &fulfillmentID, &fromStatus, &toStatus, &reason, &occurredAt); err != nil {
			return nil, fmt.Errorf("failed to scan status event row: %w", err)
		}
		evt := &pb.FulfillmentStatusEvent{
			Id:            id,
			FulfillmentId: fulfillmentID,
			ToStatus:      toStatus,
			Reason:        reason,
		}
		if fromStatus.Valid {
			evt.FromStatus = &fromStatus.String
		}
		events = append(events, evt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status event rows: %w", err)
	}

	return &pb.ListStatusEventsResponse{Events: events}, nil
}

// NewFulfillmentRepository creates a new SQL Server fulfillment repository (old-style constructor).
func NewFulfillmentRepository(db *sql.DB, tableName string) pb.FulfillmentDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerFulfillmentRepository(dbOps, tableName)
}
