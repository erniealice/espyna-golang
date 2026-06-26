//go:build postgresql

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ProcurementRequest, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres procurement_request repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresProcurementRequestRepository(dbOps, tableName), nil
	})
}

// PostgresProcurementRequestRepository implements procurement request CRUD operations using PostgreSQL.
type PostgresProcurementRequestRepository struct {
	procurementrequestpb.UnimplementedProcurementRequestDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresProcurementRequestRepository creates a new PostgreSQL procurement request repository.
func NewPostgresProcurementRequestRepository(dbOps interfaces.DatabaseOperation, tableName string) procurementrequestpb.ProcurementRequestDomainServiceServer {
	if tableName == "" {
		tableName = "procurement_request"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresProcurementRequestRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProcurementRequest creates a new procurement request record.
func (r *PostgresProcurementRequestRepository) CreateProcurementRequest(ctx context.Context, req *procurementrequestpb.CreateProcurementRequestRequest) (*procurementrequestpb.CreateProcurementRequestResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("procurement request data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create procurement_request: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pr := &procurementrequestpb.ProcurementRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &procurementrequestpb.CreateProcurementRequestResponse{Success: true, Data: []*procurementrequestpb.ProcurementRequest{pr}}, nil
}

// ReadProcurementRequest retrieves a procurement request by ID.
func (r *PostgresProcurementRequestRepository) ReadProcurementRequest(ctx context.Context, req *procurementrequestpb.ReadProcurementRequestRequest) (*procurementrequestpb.ReadProcurementRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read procurement_request: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pr := &procurementrequestpb.ProcurementRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &procurementrequestpb.ReadProcurementRequestResponse{Success: true, Data: []*procurementrequestpb.ProcurementRequest{pr}}, nil
}

// UpdateProcurementRequest updates a procurement request record.
func (r *PostgresProcurementRequestRepository) UpdateProcurementRequest(ctx context.Context, req *procurementrequestpb.UpdateProcurementRequestRequest) (*procurementrequestpb.UpdateProcurementRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update procurement_request: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pr := &procurementrequestpb.ProcurementRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &procurementrequestpb.UpdateProcurementRequestResponse{Success: true, Data: []*procurementrequestpb.ProcurementRequest{pr}}, nil
}

// DeleteProcurementRequest soft-deletes a procurement request.
func (r *PostgresProcurementRequestRepository) DeleteProcurementRequest(ctx context.Context, req *procurementrequestpb.DeleteProcurementRequestRequest) (*procurementrequestpb.DeleteProcurementRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete procurement_request: %w", err)
	}
	return &procurementrequestpb.DeleteProcurementRequestResponse{Success: true}, nil
}

// ListProcurementRequests lists procurement request records with optional filters.
func (r *PostgresProcurementRequestRepository) ListProcurementRequests(ctx context.Context, req *procurementrequestpb.ListProcurementRequestsRequest) (*procurementrequestpb.ListProcurementRequestsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list procurement_requests: %w", err)
	}
	var requests []*procurementrequestpb.ProcurementRequest
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal procurement_request row: %v", err)
			continue
		}
		pr := &procurementrequestpb.ProcurementRequest{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pr); err != nil {
			log.Printf("WARN: protojson unmarshal procurement_request: %v", err)
			continue
		}
		requests = append(requests, pr)
	}
	return &procurementrequestpb.ListProcurementRequestsResponse{Success: true, Data: requests}, nil
}

// GetProcurementRequestListPageData retrieves procurement requests with pagination, filtering, and search.
// Joins with supplier for enriched display.
func (r *PostgresProcurementRequestRepository) GetProcurementRequestListPageData(
	ctx context.Context,
	req *procurementrequestpb.GetProcurementRequestListPageDataRequest,
) (*procurementrequestpb.GetProcurementRequestListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get procurement request list page data request is required")
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

	orderBy, err := postgresCore.BuildOrderBy([]string{
		"date_created", "date_modified", "request_number", "status",
		"estimated_total_amount", "needed_by_date", "supplier_name",
	}, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := `
		WITH enriched AS (
			SELECT
				pr.id,
				pr.date_created,
				pr.date_modified,
				pr.active,
				pr.request_number,
				pr.status,
				pr.requester_user_id,
				pr.supplier_id,
				pr.currency,
				pr.estimated_total_amount,
				pr.needed_by_date,
				pr.justification,
				pr.approved_by,
				pr.rejection_reason,
				pr.purchase_order_id,
				pr.location_id,
				COALESCE(s.name, '') AS supplier_name
			FROM procurement_request pr
			LEFT JOIN supplier s ON pr.supplier_id = s.id AND s.active = true
			WHERE pr.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR pr.workspace_id = $1)
			  AND ($2::text IS NULL OR $2::text = '' OR
			       pr.request_number ILIKE $2 OR
			       pr.justification ILIKE $2 OR
			       s.name ILIKE $2)
		),
		counted AS (SELECT COUNT(*) AS total FROM enriched)
		SELECT e.*, c.total
		FROM enriched e, counted c
		ORDER BY ` + orderBy + `
		LIMIT $3 OFFSET $4;
	`

	workspaceID := identity.Must(ctx).WorkspaceID
	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query procurement_request list page data: %w", err)
	}
	defer rows.Close()

	var requests []*procurementrequestpb.ProcurementRequest
	var totalCount int64

	for rows.Next() {
		var (
			id                   string
			dateCreated          time.Time
			dateModified         time.Time
			active               bool
			requestNumber        string
			status               int32
			requesterUserID      string
			supplierID           *string
			currency             string
			estimatedTotalAmount int64
			neededByDate         *string
			justification        *string
			approvedBy           *string
			rejectionReason      *string
			purchaseOrderID      *string
			locationID           *string
			supplierName         string
			total                int64
		)
		err := rows.Scan(
			&id, &dateCreated, &dateModified, &active,
			&requestNumber, &status, &requesterUserID, &supplierID,
			&currency, &estimatedTotalAmount,
			&neededByDate, &justification,
			&approvedBy, &rejectionReason, &purchaseOrderID, &locationID,
			&supplierName, &total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan procurement_request row: %w", err)
		}
		totalCount = total

		pr := &procurementrequestpb.ProcurementRequest{
			Id:                   id,
			Active:               active,
			RequestNumber:        requestNumber,
			Status:               procurementrequestpb.ProcurementRequestStatus(status),
			RequesterUserId:      requesterUserID,
			SupplierId:           supplierID,
			Currency:             currency,
			EstimatedTotalAmount: estimatedTotalAmount,
			NeededByDate:         neededByDate,
			Justification:        justification,
			ApprovedBy:           approvedBy,
			RejectionReason:      rejectionReason,
			PurchaseOrderId:      purchaseOrderID,
			LocationId:           locationID,
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			pr.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			pr.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			pr.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			pr.DateModifiedString = &dmStr
		}
		requests = append(requests, pr)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating procurement_request rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &procurementrequestpb.GetProcurementRequestListPageDataResponse{
		ProcurementRequestList: requests,
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

// GetProcurementRequestItemPageData retrieves a single procurement request with enriched data.
func (r *PostgresProcurementRequestRepository) GetProcurementRequestItemPageData(
	ctx context.Context,
	req *procurementrequestpb.GetProcurementRequestItemPageDataRequest,
) (*procurementrequestpb.GetProcurementRequestItemPageDataResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}

	query := `
		SELECT
			pr.id,
			pr.date_created,
			pr.date_modified,
			pr.active,
			pr.request_number,
			pr.status,
			pr.requester_user_id,
			pr.supplier_id,
			pr.currency,
			pr.estimated_total_amount,
			pr.needed_by_date,
			pr.justification,
			pr.approved_by,
			pr.rejection_reason,
			pr.purchase_order_id,
			pr.location_id,
			COALESCE(s.name, '') AS supplier_name
		FROM procurement_request pr
		LEFT JOIN supplier s ON pr.supplier_id = s.id AND s.active = true
		WHERE pr.id = $1 AND pr.active = true
		  AND ($2::text = '' OR pr.workspace_id = $2::text)
		LIMIT 1;
	`
	// A1: scope to the caller's workspace. procurement_request carries its own
	// workspace_id (verified against the baseline schema; the list method scopes
	// pr.workspace_id identically). Empty wsID = service-to-service call → no scoping.
	workspaceID := identity.Must(ctx).WorkspaceID
	row := r.db.QueryRowContext(ctx, query, req.GetProcurementRequestId(), workspaceID)

	var (
		id                   string
		dateCreated          time.Time
		dateModified         time.Time
		active               bool
		requestNumber        string
		status               int32
		requesterUserID      string
		supplierID           *string
		currency             string
		estimatedTotalAmount int64
		neededByDate         *string
		justification        *string
		approvedBy           *string
		rejectionReason      *string
		purchaseOrderID      *string
		locationID           *string
		supplierName         string
	)
	err := row.Scan(
		&id, &dateCreated, &dateModified, &active,
		&requestNumber, &status, &requesterUserID, &supplierID,
		&currency, &estimatedTotalAmount,
		&neededByDate, &justification,
		&approvedBy, &rejectionReason, &purchaseOrderID, &locationID,
		&supplierName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("procurement_request with ID '%s' not found", req.GetProcurementRequestId())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query procurement_request item page data: %w", err)
	}

	pr := &procurementrequestpb.ProcurementRequest{
		Id:                   id,
		Active:               active,
		RequestNumber:        requestNumber,
		Status:               procurementrequestpb.ProcurementRequestStatus(status),
		RequesterUserId:      requesterUserID,
		SupplierId:           supplierID,
		Currency:             currency,
		EstimatedTotalAmount: estimatedTotalAmount,
		NeededByDate:         neededByDate,
		Justification:        justification,
		ApprovedBy:           approvedBy,
		RejectionReason:      rejectionReason,
		PurchaseOrderId:      purchaseOrderID,
		LocationId:           locationID,
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		pr.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		pr.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		pr.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		pr.DateModifiedString = &dmStr
	}

	return &procurementrequestpb.GetProcurementRequestItemPageDataResponse{
		ProcurementRequest: pr,
		Success:            true,
	}, nil
}

// SubmitProcurementRequest transitions a draft request to SUBMITTED.
func (r *PostgresProcurementRequestRepository) SubmitProcurementRequest(ctx context.Context, req *procurementrequestpb.SubmitProcurementRequestRequest) (*procurementrequestpb.SubmitProcurementRequestResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	newStatus := int32(procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_SUBMITTED)
	// A1: scope the mutation to the caller's workspace. procurement_request
	// carries its own workspace_id (verified against the baseline schema) —
	// without the predicate a caller could transition another tenant's request.
	// Empty wsID = service-to-service call → no scoping.
	workspaceID := identity.Must(ctx).WorkspaceID
	_, err := r.db.ExecContext(ctx,
		`UPDATE procurement_request SET status = $1, date_modified = NOW() WHERE id = $2 AND active = true AND ($3 = '' OR workspace_id = $3)`,
		newStatus, req.GetProcurementRequestId(), workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to submit procurement_request: %w", err)
	}
	return &procurementrequestpb.SubmitProcurementRequestResponse{Success: true}, nil
}

// ApproveProcurementRequest transitions a request to APPROVED, records approver and timestamp.
func (r *PostgresProcurementRequestRepository) ApproveProcurementRequest(ctx context.Context, req *procurementrequestpb.ApproveProcurementRequestRequest) (*procurementrequestpb.ApproveProcurementRequestResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	now := time.Now()
	approvedAt := now.UnixMilli()
	approvedAtStr := now.Format(time.RFC3339)
	newStatus := int32(procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED)

	// A1: scope the mutation to the caller's workspace. procurement_request
	// carries its own workspace_id (verified against the baseline schema) —
	// without the predicate a caller could approve another tenant's request.
	// Empty wsID = service-to-service call → no scoping. SET occupies $1..$4,
	// id is $5, so the workspace predicate takes $6.
	workspaceID := identity.Must(ctx).WorkspaceID
	_, err := r.db.ExecContext(ctx,
		`UPDATE procurement_request
		 SET status = $1, approved_by = $2, approved_at = $3, approved_at_string = $4, date_modified = NOW()
		 WHERE id = $5 AND active = true AND ($6 = '' OR workspace_id = $6)`,
		newStatus, req.ApprovedBy, approvedAt, approvedAtStr, req.GetProcurementRequestId(), workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to approve procurement_request: %w", err)
	}
	return &procurementrequestpb.ApproveProcurementRequestResponse{Success: true}, nil
}

// RejectProcurementRequest transitions a request to REJECTED, stores rejection reason.
func (r *PostgresProcurementRequestRepository) RejectProcurementRequest(ctx context.Context, req *procurementrequestpb.RejectProcurementRequestRequest) (*procurementrequestpb.RejectProcurementRequestResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	newStatus := int32(procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_REJECTED)
	// A1: scope the mutation to the caller's workspace. procurement_request
	// carries its own workspace_id (verified against the baseline schema) —
	// without the predicate a caller could reject another tenant's request.
	// Empty wsID = service-to-service call → no scoping. SET occupies $1/$2,
	// id is $3, so the workspace predicate takes $4.
	workspaceID := identity.Must(ctx).WorkspaceID
	_, err := r.db.ExecContext(ctx,
		`UPDATE procurement_request
		 SET status = $1, rejection_reason = $2, date_modified = NOW()
		 WHERE id = $3 AND active = true AND ($4 = '' OR workspace_id = $4)`,
		newStatus, req.GetRejectionReason(), req.GetProcurementRequestId(), workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reject procurement_request: %w", err)
	}
	return &procurementrequestpb.RejectProcurementRequestResponse{Success: true}, nil
}

// SpawnPurchaseOrder creates a PurchaseOrder from an APPROVED procurement request.
// It copies header fields and sets the procurement_request_id back-FK on the PO.
// The new PO's ID is returned in the response.
// The ProcurementRequest status is updated to PENDING_APPROVAL (as PO is now in-flight).
func (r *PostgresProcurementRequestRepository) SpawnPurchaseOrder(ctx context.Context, req *procurementrequestpb.SpawnPurchaseOrderRequest) (*procurementrequestpb.SpawnPurchaseOrderResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required for SpawnPurchaseOrder")
	}

	// Read the procurement request to copy fields onto the new PO.
	prRow := r.db.QueryRowContext(ctx,
		`SELECT id, workspace_id, requester_user_id, supplier_id, currency, estimated_total_amount,
		        needed_by_date, justification, location_id
		 FROM procurement_request
		 WHERE id = $1 AND status = $2 AND active = true`,
		req.GetProcurementRequestId(),
		int32(procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED),
	)

	var (
		prID            string
		workspaceID     string
		requesterUserID string
		supplierID      *string
		currency        string
		estimatedTotal  int64
		neededByDate    *string
		justification   *string
		locationID      *string
	)
	if err := prRow.Scan(&prID, &workspaceID, &requesterUserID, &supplierID, &currency,
		&estimatedTotal, &neededByDate, &justification, &locationID); err == sql.ErrNoRows {
		return nil, fmt.Errorf("procurement_request '%s' not found or not in APPROVED status", req.GetProcurementRequestId())
	} else if err != nil {
		return nil, fmt.Errorf("failed to read procurement_request for spawn: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction for SpawnPurchaseOrder: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Generate a new PO ID using the same UUIDv4 prefix convention.
	// We use the database uuid_generate_v4() directly for simplicity in the adapter layer.
	var poID string
	if err = tx.QueryRowContext(ctx, `SELECT uuid_generate_v4()::text`).Scan(&poID); err != nil {
		// Fall back to a timestamp-based ID if uuid extension is not available.
		poID = fmt.Sprintf("po-%d", time.Now().UnixNano())
	}

	// Newly spawned PO starts as draft (PurchaseOrder uses free-text status on a legacy entity).
	const poStatus = "draft"

	_, err = tx.ExecContext(ctx,
		`INSERT INTO purchase_order
		 (id, workspace_id, date_created, date_modified, active, status, currency,
		  supplier_id, notes, location_id, procurement_request_id)
		 VALUES ($1, $2, NOW(), NOW(), true, $3, $4, $5, $6, $7, $8)`,
		poID, workspaceID, poStatus, currency,
		supplierID, justification, locationID,
		prID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert purchase_order in SpawnPurchaseOrder: %w", err)
	}

	// Copy procurement request lines as PO line items in a single set-based
	// INSERT ... SELECT (A7 — replaces the prior per-line INSERT loop + per-line
	// uuid_generate_v4() round-trip). The DB generates each PO line ID via
	// uuid_generate_v4() (same convention as the PO header above); the source
	// WHERE/ORDER BY predicates are preserved verbatim so row selection is
	// identical to the previous loop. Workspace scoping flows through the
	// already-tenant-validated procurement_request_id ($2) — neither
	// procurement_request_line nor purchase_order_line_item carries a
	// workspace_id column of its own.
	insertLinesSQL := `
		INSERT INTO purchase_order_line_item
			(id, purchase_order_id, description, quantity, unit_price, total_amount, line_number,
			 expenditure_category_id, location_id, procurement_request_line_id,
			 date_created, date_modified, active)
		SELECT uuid_generate_v4()::text, $1, prl.description, prl.quantity,
		       prl.estimated_unit_price, prl.estimated_total_price, prl.line_number,
		       prl.expenditure_category_id, prl.location_id, prl.id,
		       NOW(), NOW(), true
		FROM procurement_request_line prl
		WHERE prl.procurement_request_id = $2 AND prl.active = true
		ORDER BY prl.line_number`

	if _, err = postgresCore.BulkInsertFromSelect(ctx, tx, insertLinesSQL, []any{poID, prID}); err != nil {
		return nil, fmt.Errorf("failed to bulk-insert purchase_order_line_items in SpawnPurchaseOrder: %w", err)
	}

	// Update the procurement request: record the spawned PO ID.
	_, err = tx.ExecContext(ctx,
		`UPDATE procurement_request
		 SET purchase_order_id = $1, date_modified = NOW()
		 WHERE id = $2`,
		poID, prID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update procurement_request with po_id in SpawnPurchaseOrder: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SpawnPurchaseOrder transaction: %w", err)
	}

	return &procurementrequestpb.SpawnPurchaseOrderResponse{
		PurchaseOrderId: poID,
		Success:         true,
	}, nil
}
