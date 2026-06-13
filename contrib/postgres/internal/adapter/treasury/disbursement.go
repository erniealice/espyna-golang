//go:build postgresql

package treasury

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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// disbursementSortableSQLCols lists the SQL column names that are safe to sort
// by in GetDisbursementListPageData. The query uses direct ORDER BY interpolation
// so this guard is critical — an unrecognised column is a potential SQL-injection
// vector and must be rejected loudly before query execution.
var disbursementSortableSQLCols = []string{
	"d.date_created",
	"d.date_modified",
	"d.name",
	"d.amount",
	"d.status",
	"d.payment_date",
	"d.reference_number",
}

// disbursementViewToSQLColMap translates view-facing sort column keys to the SQL
// column names used in the query. Columns absent from the map pass through unchanged.
var disbursementViewToSQLColMap = map[string]string{
	"date_created":     "d.date_created",
	"date_modified":    "d.date_modified",
	"name":             "d.name",
	"amount":           "d.amount",
	"status":           "d.status",
	"payment_date":     "d.payment_date",
	"reference_number": "d.reference_number",
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TreasuryDisbursement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres disbursement repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresDisbursementRepository(dbOps, tableName), nil
	})
}

// PostgresDisbursementRepository implements disbursement CRUD operations using PostgreSQL
type PostgresDisbursementRepository struct {
	disbursementpb.UnimplementedDisbursementDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresDisbursementRepository creates a new PostgreSQL disbursement repository
func NewPostgresDisbursementRepository(dbOps interfaces.DatabaseOperation, tableName string) disbursementpb.DisbursementDomainServiceServer {
	if tableName == "" {
		tableName = "treasury_disbursement"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresDisbursementRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateDisbursement creates a new disbursement record
func (r *PostgresDisbursementRepository) CreateDisbursement(ctx context.Context, req *disbursementpb.CreateDisbursementRequest) (*disbursementpb.CreateDisbursementResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("disbursement data is required")
	}

	// BURN_DOWN guard moved to the use case layer (Phase 1.C-iv of
	// 20260518-hexagonal-strict-adherence). See
	// internal/application/usecases/domain/treasury/disbursement/validate_advance_kind.go.

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create disbursement: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementpb.CreateDisbursementResponse{
		Success: true,
		Data:    []*disbursementpb.Disbursement{disbursement},
	}, nil
}

// ReadDisbursement retrieves a disbursement record by ID
func (r *PostgresDisbursementRepository) ReadDisbursement(ctx context.Context, req *disbursementpb.ReadDisbursementRequest) (*disbursementpb.ReadDisbursementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read disbursement: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementpb.ReadDisbursementResponse{
		Success: true,
		Data:    []*disbursementpb.Disbursement{disbursement},
	}, nil
}

// UpdateDisbursement updates a disbursement record
func (r *PostgresDisbursementRepository) UpdateDisbursement(ctx context.Context, req *disbursementpb.UpdateDisbursementRequest) (*disbursementpb.UpdateDisbursementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement ID is required")
	}

	// BURN_DOWN guard moved to the use case layer (Phase 1.C-iv of
	// 20260518-hexagonal-strict-adherence). See
	// internal/application/usecases/domain/treasury/disbursement/validate_advance_kind.go.

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update disbursement: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementpb.UpdateDisbursementResponse{
		Success: true,
		Data:    []*disbursementpb.Disbursement{disbursement},
	}, nil
}

// DeleteDisbursement deletes a disbursement record (soft delete)
func (r *PostgresDisbursementRepository) DeleteDisbursement(ctx context.Context, req *disbursementpb.DeleteDisbursementRequest) (*disbursementpb.DeleteDisbursementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete disbursement: %w", err)
	}

	return &disbursementpb.DeleteDisbursementResponse{
		Success: true,
	}, nil
}

// ListDisbursements lists disbursement records with optional filters
func (r *PostgresDisbursementRepository) ListDisbursements(ctx context.Context, req *disbursementpb.ListDisbursementsRequest) (*disbursementpb.ListDisbursementsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list disbursements: %w", err)
	}

	var disbursements []*disbursementpb.Disbursement
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(result, "payment_date")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal disbursement row: %v", err)
			continue
		}

		disbursement := &disbursementpb.Disbursement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
			log.Printf("WARN: protojson unmarshal disbursement: %v", err)
			continue
		}
		disbursements = append(disbursements, disbursement)
	}

	return &disbursementpb.ListDisbursementsResponse{
		Success: true,
		Data:    disbursements,
	}, nil
}

// GetDisbursementListPageData retrieves disbursements with pagination, filtering, sorting, and search using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresDisbursementRepository) GetDisbursementListPageData(
	ctx context.Context,
	req *disbursementpb.GetDisbursementListPageDataRequest,
) (*disbursementpb.GetDisbursementListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement list page data request is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := identity.Must(ctx).WorkspaceID

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

	// Translate view-facing column key to SQL column name via ColMap.
	sortColKey := "d.date_created"
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
	}
	if mapped, ok := disbursementViewToSQLColMap[sortColKey]; ok {
		sortColKey = mapped
	}

	// A2 sort guard: reject any column not in the whitelist via core.BuildOrderBy.
	// Builds "ORDER BY <col> <DIR>" with the column safely double-quoted.
	// Pass a synthesised *commonpb.SortRequest so the helper picks up direction.
	sortFragment, err := postgresCore.BuildOrderBy(
		disbursementSortableSQLCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: sortColKey, Direction: func() commonpb.SortDirection {
			if req.Sort != nil && len(req.Sort.Fields) > 0 {
				return req.Sort.Fields[0].Direction
			}
			return commonpb.SortDirection_DESC
		}()}}},
		"d.date_created DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for disbursement: %w", err)
	}

	// 20260517 advance-cash-events: extend the CTE with all advance_* schedule
	// columns + supplier_id (buying-side mirror of collection.go).
	query := `
		WITH enriched AS (
			SELECT
				d.id,
				d.date_created,
				d.date_modified,
				d.active,
				d.name,
				d.subscription_id,
				d.amount,
				d.status,
				d.expenditure_id,
				d.disbursement_type,
				d.disbursement_method_id,
				d.currency,
				d.reference_number,
				d.payment_date,
				d.approved_by,
				d.advance_kind,
				d.advance_status,
				d.advance_start_date,
				d.advance_end_date,
				d.advance_period_count,
				d.advance_period_unit,
				d.advance_total_amount,
				d.advance_remaining_amount,
				d.advance_recognized_amount,
				d.advance_balance_account_id,
				d.advance_target_account_id,
				d.advance_expiry_date,
				d.advance_proration_policy,
				d.supplier_id
			FROM treasury_disbursement d
			WHERE d.active = true
			  AND d.workspace_id = $1
			  AND ($2::text IS NULL OR $2::text = '' OR
			       d.name ILIKE $2 OR
			       d.reference_number ILIKE $2 OR
			       d.status ILIKE $2 OR
			       d.disbursement_type ILIKE $2)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + sortFragment + `
		LIMIT $3 OFFSET $4;
	`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query disbursement list page data: %w", err)
	}
	defer rows.Close()

	var disbursements []*disbursementpb.Disbursement
	var totalCount int64

	for rows.Next() {
		var (
			id                      string
			dateCreated             time.Time
			dateModified            time.Time
			active                  bool
			name                    string
			subscriptionID          *string
			amount                  int64
			status                  *string
			expenditureID           *string
			disbursementType        *string
			disbursementMethodID    *string
			currency                *string
			referenceNumber         *string
			paymentDate             *time.Time
			approvedBy              *string
			advanceKind             sql.NullInt32
			advanceStatus           sql.NullInt32
			advanceStartDate        *string
			advanceEndDate          *string
			advancePeriodCount      sql.NullInt32
			advancePeriodUnit       *string
			advanceTotalAmount      sql.NullInt64
			advanceRemainingAmount  sql.NullInt64
			advanceRecognizedAmount sql.NullInt64
			advanceBalanceAccountID *string
			advanceTargetAccountID  *string
			advanceExpiryDate       *string
			advanceProrationPolicy  sql.NullInt32
			supplierID              *string
			total                   int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&subscriptionID,
			&amount,
			&status,
			&expenditureID,
			&disbursementType,
			&disbursementMethodID,
			&currency,
			&referenceNumber,
			&paymentDate,
			&approvedBy,
			&advanceKind,
			&advanceStatus,
			&advanceStartDate,
			&advanceEndDate,
			&advancePeriodCount,
			&advancePeriodUnit,
			&advanceTotalAmount,
			&advanceRemainingAmount,
			&advanceRecognizedAmount,
			&advanceBalanceAccountID,
			&advanceTargetAccountID,
			&advanceExpiryDate,
			&advanceProrationPolicy,
			&supplierID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan disbursement row: %w", err)
		}

		totalCount = total

		disbursement := &disbursementpb.Disbursement{
			Id:     id,
			Active: active,
			Name:   name,
			Amount: amount,
		}

		if subscriptionID != nil {
			disbursement.SubscriptionId = *subscriptionID
		}
		if status != nil {
			disbursement.Status = *status
		}
		if expenditureID != nil {
			disbursement.ExpenditureId = *expenditureID
		}
		if disbursementType != nil {
			disbursement.DisbursementType = *disbursementType
		}
		if disbursementMethodID != nil {
			disbursement.DisbursementMethodId = *disbursementMethodID
		}
		if currency != nil {
			disbursement.Currency = *currency
		}
		if referenceNumber != nil {
			disbursement.ReferenceNumber = *referenceNumber
		}
		if approvedBy != nil {
			disbursement.ApprovedBy = *approvedBy
		}
		if paymentDate != nil && !paymentDate.IsZero() {
			disbursement.PaymentDate = paymentDate.Format("2006-01-02")
		}
		assignAdvanceFieldsDisbursement(disbursement,
			advanceKind, advanceStatus, advanceStartDate, advanceEndDate,
			advancePeriodCount, advancePeriodUnit,
			advanceTotalAmount, advanceRemainingAmount, advanceRecognizedAmount,
			advanceBalanceAccountID, advanceTargetAccountID, advanceExpiryDate,
			advanceProrationPolicy, supplierID,
		)

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			disbursement.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			disbursement.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			disbursement.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			disbursement.DateModifiedString = &dmStr
		}

		disbursements = append(disbursements, disbursement)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating disbursement rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &disbursementpb.GetDisbursementListPageDataResponse{
		DisbursementList: disbursements,
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

// GetDisbursementItemPageData retrieves a single disbursement with enriched data using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresDisbursementRepository) GetDisbursementItemPageData(
	ctx context.Context,
	req *disbursementpb.GetDisbursementItemPageDataRequest,
) (*disbursementpb.GetDisbursementItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement item page data request is required")
	}
	if req.DisbursementId == "" {
		return nil, fmt.Errorf("disbursement ID is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := identity.Must(ctx).WorkspaceID

	// 20260517 advance-cash-events: extend the CTE with all advance_* schedule
	// columns + supplier_id (mirrors GetDisbursementListPageData).
	query := `
		WITH enriched AS (
			SELECT
				d.id,
				d.date_created,
				d.date_modified,
				d.active,
				d.name,
				d.subscription_id,
				d.amount,
				d.status,
				d.expenditure_id,
				d.disbursement_type,
				d.disbursement_method_id,
				d.currency,
				d.reference_number,
				d.payment_date,
				d.approved_by,
				d.advance_kind,
				d.advance_status,
				d.advance_start_date,
				d.advance_end_date,
				d.advance_period_count,
				d.advance_period_unit,
				d.advance_total_amount,
				d.advance_remaining_amount,
				d.advance_recognized_amount,
				d.advance_balance_account_id,
				d.advance_target_account_id,
				d.advance_expiry_date,
				d.advance_proration_policy,
				d.supplier_id
			FROM treasury_disbursement d
			WHERE d.id = $1 AND d.workspace_id = $2 AND d.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.DisbursementId, workspaceID)

	var (
		id                      string
		dateCreated             time.Time
		dateModified            time.Time
		active                  bool
		name                    string
		subscriptionID          *string
		amount                  int64
		status                  *string
		expenditureID           *string
		disbursementType        *string
		disbursementMethodID    *string
		currency                *string
		referenceNumber         *string
		paymentDate             *time.Time
		approvedBy              *string
		advanceKind             sql.NullInt32
		advanceStatus           sql.NullInt32
		advanceStartDate        *string
		advanceEndDate          *string
		advancePeriodCount      sql.NullInt32
		advancePeriodUnit       *string
		advanceTotalAmount      sql.NullInt64
		advanceRemainingAmount  sql.NullInt64
		advanceRecognizedAmount sql.NullInt64
		advanceBalanceAccountID *string
		advanceTargetAccountID  *string
		advanceExpiryDate       *string
		advanceProrationPolicy  sql.NullInt32
		supplierID              *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&subscriptionID,
		&amount,
		&status,
		&expenditureID,
		&disbursementType,
		&disbursementMethodID,
		&currency,
		&referenceNumber,
		&paymentDate,
		&approvedBy,
		&advanceKind,
		&advanceStatus,
		&advanceStartDate,
		&advanceEndDate,
		&advancePeriodCount,
		&advancePeriodUnit,
		&advanceTotalAmount,
		&advanceRemainingAmount,
		&advanceRecognizedAmount,
		&advanceBalanceAccountID,
		&advanceTargetAccountID,
		&advanceExpiryDate,
		&advanceProrationPolicy,
		&supplierID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("disbursement with ID '%s' not found", req.DisbursementId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query disbursement item page data: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{
		Id:     id,
		Active: active,
		Name:   name,
		Amount: amount,
	}

	if subscriptionID != nil {
		disbursement.SubscriptionId = *subscriptionID
	}
	if status != nil {
		disbursement.Status = *status
	}
	if expenditureID != nil {
		disbursement.ExpenditureId = *expenditureID
	}
	if disbursementType != nil {
		disbursement.DisbursementType = *disbursementType
	}
	if disbursementMethodID != nil {
		disbursement.DisbursementMethodId = *disbursementMethodID
	}
	if currency != nil {
		disbursement.Currency = *currency
	}
	if referenceNumber != nil {
		disbursement.ReferenceNumber = *referenceNumber
	}
	if approvedBy != nil {
		disbursement.ApprovedBy = *approvedBy
	}
	if paymentDate != nil && !paymentDate.IsZero() {
		disbursement.PaymentDate = paymentDate.Format("2006-01-02")
	}
	assignAdvanceFieldsDisbursement(disbursement,
		advanceKind, advanceStatus, advanceStartDate, advanceEndDate,
		advancePeriodCount, advancePeriodUnit,
		advanceTotalAmount, advanceRemainingAmount, advanceRecognizedAmount,
		advanceBalanceAccountID, advanceTargetAccountID, advanceExpiryDate,
		advanceProrationPolicy, supplierID,
	)

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		disbursement.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		disbursement.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		disbursement.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		disbursement.DateModifiedString = &dmStr
	}

	return &disbursementpb.GetDisbursementItemPageDataResponse{
		Disbursement: disbursement,
		Success:      true,
	}, nil
}

// assignAdvanceFieldsDisbursement folds the optional advance_* schedule columns
// scanned from a treasury_disbursement row into the Disbursement proto.
// Centralised so GetDisbursementListPageData and GetDisbursementItemPageData
// stay in lock-step.
func assignAdvanceFieldsDisbursement(
	out *disbursementpb.Disbursement,
	advanceKind sql.NullInt32,
	advanceStatus sql.NullInt32,
	advanceStartDate *string,
	advanceEndDate *string,
	advancePeriodCount sql.NullInt32,
	advancePeriodUnit *string,
	advanceTotalAmount sql.NullInt64,
	advanceRemainingAmount sql.NullInt64,
	advanceRecognizedAmount sql.NullInt64,
	advanceBalanceAccountID *string,
	advanceTargetAccountID *string,
	advanceExpiryDate *string,
	advanceProrationPolicy sql.NullInt32,
	supplierID *string,
) {
	if advanceKind.Valid {
		k := advancekindpb.AdvanceKind(advanceKind.Int32)
		out.AdvanceKind = &k
	}
	if advanceStatus.Valid {
		s := advancekindpb.AdvanceStatus(advanceStatus.Int32)
		out.AdvanceStatus = &s
	}
	if advanceStartDate != nil {
		out.AdvanceStartDate = advanceStartDate
	}
	if advanceEndDate != nil {
		out.AdvanceEndDate = advanceEndDate
	}
	if advancePeriodCount.Valid {
		pc := advancePeriodCount.Int32
		out.AdvancePeriodCount = &pc
	}
	if advancePeriodUnit != nil {
		out.AdvancePeriodUnit = advancePeriodUnit
	}
	if advanceTotalAmount.Valid {
		v := advanceTotalAmount.Int64
		out.AdvanceTotalAmount = &v
	}
	if advanceRemainingAmount.Valid {
		v := advanceRemainingAmount.Int64
		out.AdvanceRemainingAmount = &v
	}
	if advanceRecognizedAmount.Valid {
		v := advanceRecognizedAmount.Int64
		out.AdvanceRecognizedAmount = &v
	}
	if advanceBalanceAccountID != nil {
		out.AdvanceBalanceAccountId = advanceBalanceAccountID
	}
	if advanceTargetAccountID != nil {
		out.AdvanceTargetAccountId = advanceTargetAccountID
	}
	if advanceExpiryDate != nil {
		out.AdvanceExpiryDate = advanceExpiryDate
	}
	if advanceProrationPolicy.Valid {
		p := advancekindpb.AdvanceProrationPolicy(advanceProrationPolicy.Int32)
		out.AdvanceProrationPolicy = &p
	}
	if supplierID != nil {
		out.SupplierId = supplierID
	}
}

// NewDisbursementRepository creates a new PostgreSQL disbursement repository (old-style constructor)
func NewDisbursementRepository(db *sql.DB, tableName string) disbursementpb.DisbursementDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresDisbursementRepository(dbOps, tableName)
}
