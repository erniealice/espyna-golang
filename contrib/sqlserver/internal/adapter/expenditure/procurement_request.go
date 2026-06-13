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

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProcurementRequest, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver procurement_request repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProcurementRequestRepository(db, dbOps, tableName), nil
	})
}

// SQLServerProcurementRequestRepository implements procurement request CRUD using SQL Server.
type SQLServerProcurementRequestRepository struct {
	procurementrequestpb.UnimplementedProcurementRequestDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerProcurementRequestRepository creates a new SQL Server procurement request repository.
func NewSQLServerProcurementRequestRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) procurementrequestpb.ProcurementRequestDomainServiceServer {
	if tableName == "" {
		tableName = "procurement_request"
	}
	return &SQLServerProcurementRequestRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerProcurementRequestRepository) CreateProcurementRequest(ctx context.Context, req *procurementrequestpb.CreateProcurementRequestRequest) (*procurementrequestpb.CreateProcurementRequestResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("procurement request data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create procurement_request: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	pr := &procurementrequestpb.ProcurementRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &procurementrequestpb.CreateProcurementRequestResponse{Success: true, Data: []*procurementrequestpb.ProcurementRequest{pr}}, nil
}

func (r *SQLServerProcurementRequestRepository) ReadProcurementRequest(ctx context.Context, req *procurementrequestpb.ReadProcurementRequestRequest) (*procurementrequestpb.ReadProcurementRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read procurement_request: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	pr := &procurementrequestpb.ProcurementRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &procurementrequestpb.ReadProcurementRequestResponse{Success: true, Data: []*procurementrequestpb.ProcurementRequest{pr}}, nil
}

func (r *SQLServerProcurementRequestRepository) UpdateProcurementRequest(ctx context.Context, req *procurementrequestpb.UpdateProcurementRequestRequest) (*procurementrequestpb.UpdateProcurementRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update procurement_request: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	pr := &procurementrequestpb.ProcurementRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &procurementrequestpb.UpdateProcurementRequestResponse{Success: true, Data: []*procurementrequestpb.ProcurementRequest{pr}}, nil
}

func (r *SQLServerProcurementRequestRepository) DeleteProcurementRequest(ctx context.Context, req *procurementrequestpb.DeleteProcurementRequestRequest) (*procurementrequestpb.DeleteProcurementRequestResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete procurement_request: %w", err)
	}
	return &procurementrequestpb.DeleteProcurementRequestResponse{Success: true}, nil
}

func (r *SQLServerProcurementRequestRepository) ListProcurementRequests(ctx context.Context, req *procurementrequestpb.ListProcurementRequestsRequest) (*procurementrequestpb.ListProcurementRequestsResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// GetProcurementRequestListPageData retrieves procurement requests with pagination and supplier join.
// Dialect: @pN placeholders, LIKE, OFFSET/FETCH, active = 1, [bracket] identifiers.
func (r *SQLServerProcurementRequestRepository) GetProcurementRequestListPageData(ctx context.Context, req *procurementrequestpb.GetProcurementRequestListPageDataRequest) (*procurementrequestpb.GetProcurementRequestListPageDataResponse, error) {
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

	sortField := "[pr].[date_created]"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = "[pr].[date_created]"
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		WITH enriched AS (
			SELECT
				[pr].[id],
				[pr].[date_created],
				[pr].[date_modified],
				[pr].[active],
				[pr].[request_number],
				[pr].[status],
				[pr].[requester_user_id],
				[pr].[supplier_id],
				[pr].[currency],
				[pr].[estimated_total_amount],
				[pr].[needed_by_date],
				[pr].[justification],
				[pr].[approved_by],
				[pr].[rejection_reason],
				[pr].[purchase_order_id],
				[pr].[location_id],
				COALESCE([s].[name], '') AS [supplier_name],
				COUNT(*) OVER() AS [total]
			FROM [procurement_request] [pr]
			LEFT JOIN [supplier] [s] ON [pr].[supplier_id] = [s].[id] AND [s].[active] = 1
			WHERE [pr].[active] = 1
			  AND (@p1 IS NULL OR @p1 = '' OR [pr].[workspace_id] = @p1)
			  AND (@p2 IS NULL OR @p2 = '' OR
			       [pr].[request_number] LIKE @p2 OR
			       [pr].[justification] LIKE @p2 OR
			       [s].[name] LIKE @p2)
		)
		SELECT * FROM enriched
		ORDER BY ` + sortField + ` ` + sortOrder + `
		OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY
	`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, offset, limit)
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
		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active,
			&requestNumber, &status, &requesterUserID, &supplierID,
			&currency, &estimatedTotalAmount,
			&neededByDate, &justification,
			&approvedBy, &rejectionReason, &purchaseOrderID, &locationID,
			&supplierName, &total,
		); err != nil {
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

// GetProcurementRequestItemPageData retrieves a single procurement request with supplier join.
func (r *SQLServerProcurementRequestRepository) GetProcurementRequestItemPageData(ctx context.Context, req *procurementrequestpb.GetProcurementRequestItemPageDataRequest) (*procurementrequestpb.GetProcurementRequestItemPageDataResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}

	query := `
		SELECT TOP 1
			[pr].[id],
			[pr].[date_created],
			[pr].[date_modified],
			[pr].[active],
			[pr].[request_number],
			[pr].[status],
			[pr].[requester_user_id],
			[pr].[supplier_id],
			[pr].[currency],
			[pr].[estimated_total_amount],
			[pr].[needed_by_date],
			[pr].[justification],
			[pr].[approved_by],
			[pr].[rejection_reason],
			[pr].[purchase_order_id],
			[pr].[location_id],
			COALESCE([s].[name], '') AS [supplier_name]
		FROM [procurement_request] [pr]
		LEFT JOIN [supplier] [s] ON [pr].[supplier_id] = [s].[id] AND [s].[active] = 1
		WHERE [pr].[id] = @p1 AND [pr].[active] = 1
	`
	row := r.db.QueryRowContext(ctx, query, req.GetProcurementRequestId())

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
func (r *SQLServerProcurementRequestRepository) SubmitProcurementRequest(ctx context.Context, req *procurementrequestpb.SubmitProcurementRequestRequest) (*procurementrequestpb.SubmitProcurementRequestResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	newStatus := int32(procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_SUBMITTED)
	_, err := r.db.ExecContext(ctx,
		`UPDATE [procurement_request] SET [status] = @p1, [date_modified] = GETUTCDATE() WHERE [id] = @p2 AND [active] = 1`,
		newStatus, req.GetProcurementRequestId(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to submit procurement_request: %w", err)
	}
	return &procurementrequestpb.SubmitProcurementRequestResponse{Success: true}, nil
}

// ApproveProcurementRequest transitions a request to APPROVED.
func (r *SQLServerProcurementRequestRepository) ApproveProcurementRequest(ctx context.Context, req *procurementrequestpb.ApproveProcurementRequestRequest) (*procurementrequestpb.ApproveProcurementRequestResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	now := time.Now()
	approvedAt := now.UnixMilli()
	approvedAtStr := now.Format(time.RFC3339)
	newStatus := int32(procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_APPROVED)

	_, err := r.db.ExecContext(ctx,
		`UPDATE [procurement_request]
		 SET [status] = @p1, [approved_by] = @p2, [approved_at] = @p3, [approved_at_string] = @p4, [date_modified] = GETUTCDATE()
		 WHERE [id] = @p5 AND [active] = 1`,
		newStatus, req.ApprovedBy, approvedAt, approvedAtStr, req.GetProcurementRequestId(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to approve procurement_request: %w", err)
	}
	return &procurementrequestpb.ApproveProcurementRequestResponse{Success: true}, nil
}

// RejectProcurementRequest transitions a request to REJECTED.
func (r *SQLServerProcurementRequestRepository) RejectProcurementRequest(ctx context.Context, req *procurementrequestpb.RejectProcurementRequestRequest) (*procurementrequestpb.RejectProcurementRequestResponse, error) {
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, fmt.Errorf("procurement request ID is required")
	}
	newStatus := int32(procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_REJECTED)
	_, err := r.db.ExecContext(ctx,
		`UPDATE [procurement_request]
		 SET [status] = @p1, [rejection_reason] = @p2, [date_modified] = GETUTCDATE()
		 WHERE [id] = @p3 AND [active] = 1`,
		newStatus, req.GetRejectionReason(), req.GetProcurementRequestId(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reject procurement_request: %w", err)
	}
	return &procurementrequestpb.RejectProcurementRequestResponse{Success: true}, nil
}

// SpawnPurchaseOrder — TODO: translate postgres uuid_generate_v4()+transaction pattern for SQL Server.
// SQL Server uses NEWID() for UUID generation and does not support BulkInsertFromSelect helper.
func (r *SQLServerProcurementRequestRepository) SpawnPurchaseOrder(_ context.Context, _ *procurementrequestpb.SpawnPurchaseOrderRequest) (*procurementrequestpb.SpawnPurchaseOrderResponse, error) {
	return nil, fmt.Errorf("SpawnPurchaseOrder: TODO — translate INSERT … SELECT + NEWID() for SQL Server")
}
