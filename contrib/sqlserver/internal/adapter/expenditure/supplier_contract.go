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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// supplierContractSortableSQLCols is the fail-closed sort whitelist (A2).
var supplierContractSortableSQLCols = []string{
	"name",
	"kind",
	"status",
	"currency",
	"date_time_start",
	"date_time_end",
	"committed_amount",
	"released_amount",
	"billed_amount",
	"remaining_amount",
	"reference_number",
	"supplier_name",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.SupplierContract, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver supplier_contract repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerSupplierContractRepository(db, dbOps, tableName), nil
	})
}

// SQLServerSupplierContractRepository implements supplier contract CRUD using SQL Server.
type SQLServerSupplierContractRepository struct {
	suppliercontractpb.UnimplementedSupplierContractDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerSupplierContractRepository creates a new SQL Server supplier contract repository.
func NewSQLServerSupplierContractRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) suppliercontractpb.SupplierContractDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_contract"
	}
	return &SQLServerSupplierContractRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerSupplierContractRepository) CreateSupplierContract(ctx context.Context, req *suppliercontractpb.CreateSupplierContractRequest) (*suppliercontractpb.CreateSupplierContractResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier contract data is required")
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
		return nil, fmt.Errorf("failed to create supplier_contract: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	sc := &suppliercontractpb.SupplierContract{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &suppliercontractpb.CreateSupplierContractResponse{Success: true, Data: []*suppliercontractpb.SupplierContract{sc}}, nil
}

func (r *SQLServerSupplierContractRepository) ReadSupplierContract(ctx context.Context, req *suppliercontractpb.ReadSupplierContractRequest) (*suppliercontractpb.ReadSupplierContractResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	sc := &suppliercontractpb.SupplierContract{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &suppliercontractpb.ReadSupplierContractResponse{Success: true, Data: []*suppliercontractpb.SupplierContract{sc}}, nil
}

func (r *SQLServerSupplierContractRepository) UpdateSupplierContract(ctx context.Context, req *suppliercontractpb.UpdateSupplierContractRequest) (*suppliercontractpb.UpdateSupplierContractResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract ID is required")
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
		return nil, fmt.Errorf("failed to update supplier_contract: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	sc := &suppliercontractpb.SupplierContract{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &suppliercontractpb.UpdateSupplierContractResponse{Success: true, Data: []*suppliercontractpb.SupplierContract{sc}}, nil
}

func (r *SQLServerSupplierContractRepository) DeleteSupplierContract(ctx context.Context, req *suppliercontractpb.DeleteSupplierContractRequest) (*suppliercontractpb.DeleteSupplierContractResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_contract: %w", err)
	}
	return &suppliercontractpb.DeleteSupplierContractResponse{Success: true}, nil
}

func (r *SQLServerSupplierContractRepository) ListSupplierContracts(ctx context.Context, req *suppliercontractpb.ListSupplierContractsRequest) (*suppliercontractpb.ListSupplierContractsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_contracts: %w", err)
	}
	var contracts []*suppliercontractpb.SupplierContract
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_contract row: %v", err)
			continue
		}
		sc := &suppliercontractpb.SupplierContract{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sc); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_contract: %v", err)
			continue
		}
		contracts = append(contracts, sc)
	}
	return &suppliercontractpb.ListSupplierContractsResponse{Success: true, Data: contracts}, nil
}

// GetSupplierContractListPageData retrieves supplier contracts with supplier join, pagination, and search.
func (r *SQLServerSupplierContractRepository) GetSupplierContractListPageData(ctx context.Context, req *suppliercontractpb.GetSupplierContractListPageDataRequest) (*suppliercontractpb.GetSupplierContractListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier contract list page data request is required")
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

	orderBy, err := sqlserverCore.BuildOrderBy(supplierContractSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		WITH enriched AS (
			SELECT
				[sc].[id],
				[sc].[date_created],
				[sc].[date_modified],
				[sc].[active],
				[sc].[name],
				[sc].[kind],
				[sc].[status],
				[sc].[supplier_id],
				[sc].[currency],
				[sc].[date_time_start],
				[sc].[date_time_end],
				[sc].[committed_amount],
				[sc].[released_amount],
				[sc].[billed_amount],
				[sc].[remaining_amount],
				[sc].[reference_number],
				[sc].[location_id],
				COALESCE([s].[name], '') AS [supplier_name],
				COUNT(*) OVER() AS [total]
			FROM [supplier_contract] [sc]
			LEFT JOIN [supplier] [s] ON [sc].[supplier_id] = [s].[id] AND [s].[active] = 1
			WHERE [sc].[active] = 1
			  AND (@p1 IS NULL OR @p1 = '' OR [sc].[workspace_id] = @p1)
			  AND (@p2 IS NULL OR @p2 = '' OR
			       [sc].[name] LIKE @p2 OR
			       [sc].[reference_number] LIKE @p2 OR
			       [s].[name] LIKE @p2)
		)
		SELECT * FROM enriched
		` + orderBy + `
		OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY
	`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query supplier_contract list page data: %w", err)
	}
	defer rows.Close()

	var contracts []*suppliercontractpb.SupplierContract
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			name            string
			kind            int32
			status          int32
			supplierID      *string
			currency        string
			dateTimeStart   string
			dateTimeEnd     *string
			committedAmount *int64
			releasedAmount  *int64
			billedAmount    *int64
			remainingAmount *int64
			referenceNumber *string
			locationID      *string
			supplierName    string
			total           int64
		)
		if err := rows.Scan(
			&id, &dateCreated, &dateModified, &active, &name,
			&kind, &status, &supplierID, &currency,
			&dateTimeStart, &dateTimeEnd,
			&committedAmount, &releasedAmount, &billedAmount, &remainingAmount,
			&referenceNumber, &locationID,
			&supplierName, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan supplier_contract row: %w", err)
		}
		totalCount = total

		sc := &suppliercontractpb.SupplierContract{
			Id:              id,
			Active:          active,
			Name:            name,
			Kind:            suppliercontractpb.SupplierContractKind(kind),
			Status:          suppliercontractpb.SupplierContractStatus(status),
			Currency:        currency,
			DateTimeStart:   dateTimeStart,
			ReferenceNumber: referenceNumber,
			DateTimeEnd:     dateTimeEnd,
			CommittedAmount: committedAmount,
			ReleasedAmount:  releasedAmount,
			BilledAmount:    billedAmount,
			RemainingAmount: remainingAmount,
			LocationId:      locationID,
		}
		if supplierID != nil {
			sc.SupplierId = *supplierID
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			sc.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			sc.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			sc.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			sc.DateModifiedString = &dmStr
		}
		contracts = append(contracts, sc)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating supplier_contract rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &suppliercontractpb.GetSupplierContractListPageDataResponse{
		SupplierContractList: contracts,
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

// GetSupplierContractItemPageData retrieves a single supplier contract with supplier join.
func (r *SQLServerSupplierContractRepository) GetSupplierContractItemPageData(ctx context.Context, req *suppliercontractpb.GetSupplierContractItemPageDataRequest) (*suppliercontractpb.GetSupplierContractItemPageDataResponse, error) {
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, fmt.Errorf("supplier contract ID is required")
	}

	query := `
		SELECT TOP 1
			[sc].[id],
			[sc].[date_created],
			[sc].[date_modified],
			[sc].[active],
			[sc].[name],
			[sc].[kind],
			[sc].[status],
			[sc].[supplier_id],
			[sc].[currency],
			[sc].[date_time_start],
			[sc].[date_time_end],
			[sc].[committed_amount],
			[sc].[released_amount],
			[sc].[billed_amount],
			[sc].[remaining_amount],
			[sc].[reference_number],
			[sc].[location_id],
			[sc].[approved_by],
			[sc].[rejection_reason],
			[sc].[requested_by],
			[sc].[notes],
			COALESCE([s].[name], '') AS [supplier_name]
		FROM [supplier_contract] [sc]
		LEFT JOIN [supplier] [s] ON [sc].[supplier_id] = [s].[id] AND [s].[active] = 1
		WHERE [sc].[id] = @p1 AND [sc].[active] = 1
	`
	row := r.db.QueryRowContext(ctx, query, req.GetSupplierContractId())

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		name            string
		kind            int32
		status          int32
		supplierID      *string
		currency        string
		dateTimeStart   string
		dateTimeEnd     *string
		committedAmount *int64
		releasedAmount  *int64
		billedAmount    *int64
		remainingAmount *int64
		referenceNumber *string
		locationID      *string
		approvedBy      *string
		rejectionReason *string
		requestedBy     *string
		notes           *string
		supplierName    string
	)
	err := row.Scan(
		&id, &dateCreated, &dateModified, &active, &name,
		&kind, &status, &supplierID, &currency,
		&dateTimeStart, &dateTimeEnd,
		&committedAmount, &releasedAmount, &billedAmount, &remainingAmount,
		&referenceNumber, &locationID,
		&approvedBy, &rejectionReason, &requestedBy, &notes,
		&supplierName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("supplier_contract with ID '%s' not found", req.GetSupplierContractId())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query supplier_contract item page data: %w", err)
	}

	sc := &suppliercontractpb.SupplierContract{
		Id:              id,
		Active:          active,
		Name:            name,
		Kind:            suppliercontractpb.SupplierContractKind(kind),
		Status:          suppliercontractpb.SupplierContractStatus(status),
		Currency:        currency,
		DateTimeStart:   dateTimeStart,
		ReferenceNumber: referenceNumber,
		DateTimeEnd:     dateTimeEnd,
		CommittedAmount: committedAmount,
		ReleasedAmount:  releasedAmount,
		BilledAmount:    billedAmount,
		RemainingAmount: remainingAmount,
		LocationId:      locationID,
		ApprovedBy:      approvedBy,
		RejectionReason: rejectionReason,
		RequestedBy:     requestedBy,
		Notes:           notes,
	}
	if supplierID != nil {
		sc.SupplierId = *supplierID
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		sc.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		sc.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		sc.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		sc.DateModifiedString = &dmStr
	}

	return &suppliercontractpb.GetSupplierContractItemPageDataResponse{SupplierContract: sc, Success: true}, nil
}

// ApproveSupplierContract transitions the contract to APPROVED.
func (r *SQLServerSupplierContractRepository) ApproveSupplierContract(ctx context.Context, req *suppliercontractpb.ApproveSupplierContractRequest) (*suppliercontractpb.ApproveSupplierContractResponse, error) {
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, fmt.Errorf("supplier contract ID is required")
	}
	now := time.Now()
	approvedAt := now.UnixMilli()
	approvedAtStr := now.Format(time.RFC3339)
	newStatus := int32(suppliercontractpb.SupplierContractStatus_SUPPLIER_CONTRACT_STATUS_APPROVED)

	_, err := r.db.ExecContext(ctx,
		`UPDATE [supplier_contract]
		 SET [status] = @p1, [approved_by] = @p2, [approved_at] = @p3, [approved_at_string] = @p4, [date_modified] = GETUTCDATE()
		 WHERE [id] = @p5 AND [active] = 1`,
		newStatus, req.ApprovedBy, approvedAt, approvedAtStr, req.GetSupplierContractId(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to approve supplier_contract: %w", err)
	}
	return &suppliercontractpb.ApproveSupplierContractResponse{Success: true}, nil
}

// TerminateSupplierContract transitions the contract to TERMINATED.
func (r *SQLServerSupplierContractRepository) TerminateSupplierContract(ctx context.Context, req *suppliercontractpb.TerminateSupplierContractRequest) (*suppliercontractpb.TerminateSupplierContractResponse, error) {
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, fmt.Errorf("supplier contract ID is required")
	}
	newStatus := int32(suppliercontractpb.SupplierContractStatus_SUPPLIER_CONTRACT_STATUS_TERMINATED)
	_, err := r.db.ExecContext(ctx,
		`UPDATE [supplier_contract]
		 SET [status] = @p1, [rejection_reason] = @p2, [date_modified] = GETUTCDATE()
		 WHERE [id] = @p3 AND [active] = 1`,
		newStatus, req.GetReason(), req.GetSupplierContractId(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to terminate supplier_contract: %w", err)
	}
	return &suppliercontractpb.TerminateSupplierContractResponse{Success: true}, nil
}

// GenerateUpcomingExpenditures — P5 stub; recurrence engine deferred.
func (r *SQLServerSupplierContractRepository) GenerateUpcomingExpenditures(_ context.Context, _ *suppliercontractpb.GenerateUpcomingExpendituresRequest) (*suppliercontractpb.GenerateUpcomingExpendituresResponse, error) {
	return nil, fmt.Errorf("GenerateUpcomingExpenditures: recurrence engine deferred to P5")
}

// updateBalanceFields performs a locked balance update on a supplier_contract row.
func (r *SQLServerSupplierContractRepository) updateBalanceFields(ctx context.Context, contractID string, releasedDelta, billedDelta int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var (
		committedAmount int64
		releasedAmount  int64
		billedAmount    int64
	)
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE([committed_amount], 0), COALESCE([released_amount], 0), COALESCE([billed_amount], 0)
		 FROM [supplier_contract] WITH (UPDLOCK) WHERE [id] = @p1 AND [active] = 1`,
		contractID,
	).Scan(&committedAmount, &releasedAmount, &billedAmount)
	if err == sql.ErrNoRows {
		return fmt.Errorf("supplier_contract '%s' not found for balance update", contractID)
	}
	if err != nil {
		return fmt.Errorf("failed to lock supplier_contract for balance update: %w", err)
	}

	newReleased := releasedAmount + releasedDelta
	newBilled := billedAmount + billedDelta
	newRemaining := committedAmount - newBilled

	_, err = tx.ExecContext(ctx,
		`UPDATE [supplier_contract]
		 SET [released_amount] = @p1, [billed_amount] = @p2, [remaining_amount] = @p3, [date_modified] = GETUTCDATE()
		 WHERE [id] = @p4`,
		newReleased, newBilled, newRemaining, contractID,
	)
	if err != nil {
		return fmt.Errorf("failed to update balance fields on supplier_contract: %w", err)
	}
	return tx.Commit()
}

// RegisterRelease increments released_amount by the given centavo amount.
func (r *SQLServerSupplierContractRepository) RegisterRelease(ctx context.Context, contractID string, releasedCentavos int64) error {
	return r.updateBalanceFields(ctx, contractID, releasedCentavos, 0)
}

// RegisterBilling increments billed_amount and recomputes remaining_amount.
func (r *SQLServerSupplierContractRepository) RegisterBilling(ctx context.Context, contractID string, billedCentavos int64) error {
	return r.updateBalanceFields(ctx, contractID, 0, billedCentavos)
}

// RegisterCredit applies a negative billed delta (rebates, credits).
func (r *SQLServerSupplierContractRepository) RegisterCredit(ctx context.Context, contractID string, creditCentavos int64) error {
	return r.updateBalanceFields(ctx, contractID, 0, -creditCentavos)
}
