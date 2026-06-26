//go:build postgresql

package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenuepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// revenue_payment records a (partial) payment received against a revenue
// document. It mirrors loan_payment.go (a payment entity filtered server-side by
// its parent id) — ListRevenuePayments honors the FilterRequest revenue_id
// filter by passing it through to the generic List op, exactly like
// loan_payment / rate_band.
//
// Field-map override: the proto field `amount` (centavos, Rule #1) is persisted
// to the DB column `amount_centavos`. protojson marshals the field to the JSON
// key "amount"; before handing the row map to the generic CRUD op we rename that
// key to "amount_centavos", and on the way back we rename the DB column
// "amount_centavos" to "amount" so protojson can decode it. The amount_centavos
// column is added by the SEPARATE W4 migration; referencing it by name compiles
// fine today.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RevenuePayment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres revenue_payment repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresRevenuePaymentRepository(dbOps, tableName), nil
	})
}

// PostgresRevenuePaymentRepository implements revenue_payment CRUD operations using PostgreSQL
type PostgresRevenuePaymentRepository struct {
	revenuepaymentpb.UnimplementedRevenuePaymentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRevenuePaymentRepository creates a new PostgreSQL revenue_payment repository
func NewPostgresRevenuePaymentRepository(dbOps interfaces.DatabaseOperation, tableName string) revenuepaymentpb.RevenuePaymentDomainServiceServer {
	if tableName == "" {
		tableName = "revenue_payment"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresRevenuePaymentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// remapAmountToColumn renames the protojson key "amount" → DB column
// "amount_centavos" on the write path.
func remapAmountToColumn(data map[string]any) {
	if v, ok := data["amount"]; ok {
		data["amount_centavos"] = v
		delete(data, "amount")
	}
}

// remapColumnToAmount renames the DB column "amount_centavos" → proto field
// "amount" on the read path so protojson can decode it.
func remapColumnToAmount(result map[string]any) {
	if v, ok := result["amount_centavos"]; ok {
		result["amount"] = v
		delete(result, "amount_centavos")
	}
}

// CreateRevenuePayment creates a new revenue_payment record
func (r *PostgresRevenuePaymentRepository) CreateRevenuePayment(ctx context.Context, req *revenuepaymentpb.CreateRevenuePaymentRequest) (*revenuepaymentpb.CreateRevenuePaymentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue_payment data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	remapAmountToColumn(data)

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create revenue_payment: %w", err)
	}

	remapColumnToAmount(result)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenuePayment := &revenuepaymentpb.RevenuePayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenuePayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepaymentpb.CreateRevenuePaymentResponse{
		Success: true,
		Data:    []*revenuepaymentpb.RevenuePayment{revenuePayment},
	}, nil
}

// ReadRevenuePayment retrieves a revenue_payment record by ID
func (r *PostgresRevenuePaymentRepository) ReadRevenuePayment(ctx context.Context, req *revenuepaymentpb.ReadRevenuePaymentRequest) (*revenuepaymentpb.ReadRevenuePaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_payment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue_payment: %w", err)
	}

	remapColumnToAmount(result)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenuePayment := &revenuepaymentpb.RevenuePayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenuePayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepaymentpb.ReadRevenuePaymentResponse{
		Success: true,
		Data:    []*revenuepaymentpb.RevenuePayment{revenuePayment},
	}, nil
}

// UpdateRevenuePayment updates a revenue_payment record
func (r *PostgresRevenuePaymentRepository) UpdateRevenuePayment(ctx context.Context, req *revenuepaymentpb.UpdateRevenuePaymentRequest) (*revenuepaymentpb.UpdateRevenuePaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_payment ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	remapAmountToColumn(data)

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update revenue_payment: %w", err)
	}

	remapColumnToAmount(result)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenuePayment := &revenuepaymentpb.RevenuePayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenuePayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepaymentpb.UpdateRevenuePaymentResponse{
		Success: true,
		Data:    []*revenuepaymentpb.RevenuePayment{revenuePayment},
	}, nil
}

// DeleteRevenuePayment deletes a revenue_payment record
func (r *PostgresRevenuePaymentRepository) DeleteRevenuePayment(ctx context.Context, req *revenuepaymentpb.DeleteRevenuePaymentRequest) (*revenuepaymentpb.DeleteRevenuePaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue_payment ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete revenue_payment: %w", err)
	}

	return &revenuepaymentpb.DeleteRevenuePaymentResponse{
		Success: true,
	}, nil
}

// ListRevenuePayments lists revenue_payment records with optional filters.
//
// The FilterRequest (req.Filters) carries the server-side revenue_id filter
// (design doc §4 / §5.3) and is passed through unchanged to the generic List op,
// mirroring loan_payment's parent-id filter.
func (r *PostgresRevenuePaymentRepository) ListRevenuePayments(ctx context.Context, req *revenuepaymentpb.ListRevenuePaymentsRequest) (*revenuepaymentpb.ListRevenuePaymentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenue_payments: %w", err)
	}

	var revenuePayments []*revenuepaymentpb.RevenuePayment
	for _, result := range listResult.Data {
		remapColumnToAmount(result)
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal revenue_payment row: %v", err)
			continue
		}

		revenuePayment := &revenuepaymentpb.RevenuePayment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenuePayment); err != nil {
			log.Printf("WARN: protojson unmarshal revenue_payment: %v", err)
			continue
		}
		revenuePayments = append(revenuePayments, revenuePayment)
	}

	return &revenuepaymentpb.ListRevenuePaymentsResponse{
		Success: true,
		Data:    revenuePayments,
	}, nil
}

// revenuePaymentSortableSQLCols is the fail-closed sort whitelist for
// GetRevenuePaymentListPageData (A2). Only columns projected by the CTE SELECT
// are included so ORDER BY can never reference an unprojected/injected
// identifier. amount_centavos is a centavo integer (Rule #1).
var revenuePaymentSortableSQLCols = []string{
	"id", "revenue_id", "collection_method_id", "amount_centavos",
	"currency", "reference_number", "collection_type", "status", "active",
	"payment_method", "received_by", "received_role", "notes",
	"payment_date", "date_created",
}

// GetRevenuePaymentListPageData retrieves revenue_payments with pagination,
// filtering, sorting, and search using a CTE. The revenue_id filter (parent id)
// is applied server-side from req.Filters via a single equality predicate,
// mirroring loan_payment's parent-id scoping.
func (r *PostgresRevenuePaymentRepository) GetRevenuePaymentListPageData(
	ctx context.Context,
	req *revenuepaymentpb.GetRevenuePaymentListPageDataRequest,
) (*revenuepaymentpb.GetRevenuePaymentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue_payment list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Server-side parent-id filter: revenue_id pulled from the FilterRequest.
	revenueIDFilter := filterValue(req.GetFilters(), "revenue_id")

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

	orderByClause, err := postgresCore.BuildOrderBy(revenuePaymentSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := `
		WITH enriched AS (
			SELECT
				rp.id,
				rp.revenue_id,
				rp.collection_method_id,
				rp.amount_centavos,
				rp.currency,
				rp.reference_number,
				rp.collection_type,
				rp.status,
				rp.active,
				rp.payment_method,
				rp.received_by,
				rp.received_role,
				rp.notes,
				rp.payment_date,
				rp.date_created
			FROM revenue_payment rp
			WHERE ($4::text IS NULL OR $4::text = '' OR rp.revenue_id = $4)
			  AND ($1::text IS NULL OR $1::text = '' OR
			       rp.reference_number ILIKE $1 OR
			       rp.payment_method ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, revenueIDFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue_payment list page data: %w", err)
	}
	defer rows.Close()

	var revenuePayments []*revenuepaymentpb.RevenuePayment
	var totalCount int64

	for rows.Next() {
		rp, total, scanErr := scanRevenuePaymentRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan revenue_payment row: %w", scanErr)
		}
		totalCount = total
		revenuePayments = append(revenuePayments, rp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue_payment rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenuepaymentpb.GetRevenuePaymentListPageDataResponse{
		RevenuePaymentList: revenuePayments,
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

// GetRevenuePaymentItemPageData retrieves a single revenue_payment with enriched data using a CTE
func (r *PostgresRevenuePaymentRepository) GetRevenuePaymentItemPageData(
	ctx context.Context,
	req *revenuepaymentpb.GetRevenuePaymentItemPageDataRequest,
) (*revenuepaymentpb.GetRevenuePaymentItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue_payment item page data request is required")
	}
	if req.RevenuePaymentId == "" {
		return nil, fmt.Errorf("revenue_payment ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				rp.id,
				rp.revenue_id,
				rp.collection_method_id,
				rp.amount_centavos,
				rp.currency,
				rp.reference_number,
				rp.collection_type,
				rp.status,
				rp.active,
				rp.payment_method,
				rp.received_by,
				rp.received_role,
				rp.notes,
				rp.payment_date,
				rp.date_created
			FROM revenue_payment rp
			WHERE rp.id = $1
		)
		SELECT *, 0::bigint AS total FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.RevenuePaymentId)

	rp, _, err := scanRevenuePaymentRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue_payment with ID '%s' not found", req.RevenuePaymentId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue_payment item page data: %w", err)
	}

	return &revenuepaymentpb.GetRevenuePaymentItemPageDataResponse{
		RevenuePayment: rp,
		Success:        true,
	}, nil
}

// scanRevenuePaymentRow scans a fully-projected revenue_payment enriched row
// (plus the trailing total) into a proto message. The amount_centavos column is
// scanned directly and assigned to the proto Amount field (Rule #1, centavos).
func scanRevenuePaymentRow(scan func(...any) error) (*revenuepaymentpb.RevenuePayment, int64, error) {
	var (
		id                 string
		revenueID          string
		collectionMethodID *string
		amountCentavos     int64
		currency           string
		referenceNumber    *string
		collectionType     *string
		status             *string
		active             bool
		paymentMethod      *string
		receivedBy         *string
		receivedRole       *string
		notes              *string
		paymentDate        *string
		dateCreated        time.Time
		total              int64
	)

	err := scan(
		&id,
		&revenueID,
		&collectionMethodID,
		&amountCentavos,
		&currency,
		&referenceNumber,
		&collectionType,
		&status,
		&active,
		&paymentMethod,
		&receivedBy,
		&receivedRole,
		&notes,
		&paymentDate,
		&dateCreated,
		&total,
	)
	if err != nil {
		return nil, 0, err
	}

	rp := &revenuepaymentpb.RevenuePayment{
		Id:                 id,
		RevenueId:          revenueID,
		CollectionMethodId: collectionMethodID,
		Amount:             amountCentavos,
		Currency:           currency,
		ReferenceNumber:    referenceNumber,
		CollectionType:     collectionType,
		Status:             status,
		Active:             active,
		PaymentMethod:      paymentMethod,
		ReceivedBy:         receivedBy,
		ReceivedRole:       receivedRole,
		Notes:              notes,
		PaymentDate:        paymentDate,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		rp.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		rp.DateCreatedString = &dcStr
	}

	return rp, total, nil
}

// filterValue extracts the first string-filter value bound to fieldName in a
// FilterRequest, or "" when absent. Used to pull the revenue_id parent-id filter
// out of the page-data request's FilterRequest. revenue_id is an equality match
// carried as a StringFilter.
func filterValue(filters *commonpb.FilterRequest, fieldName string) string {
	if filters == nil {
		return ""
	}
	for _, f := range filters.GetFilters() {
		if f.GetField() == fieldName {
			return f.GetStringFilter().GetValue()
		}
	}
	return ""
}

// NewRevenuePaymentRepository creates a new PostgreSQL revenue_payment repository (old-style constructor)
func NewRevenuePaymentRepository(db *sql.DB, tableName string) revenuepaymentpb.RevenuePaymentDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresRevenuePaymentRepository(dbOps, tableName)
}
