//go:build postgresql

package revenue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"
	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/consumer"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// revenueSortableSQLCols lists the SQL column names that are safe to sort by in
// GetRevenueListPageData. The query uses direct ORDER BY interpolation, so any
// unrecognised value is a potential SQL-injection vector and must be rejected
// loudly before query execution.
var revenueSortableSQLCols = []string{
	"reference_number",
	"total_amount",
	"status",
	"date_created",
	"date_modified",
	"client_name",
	"revenue_date_string",
}

// revenueViewToSQLColMap translates view-facing sort column keys to SQL column
// names. Columns absent from the map pass through unchanged.
var revenueViewToSQLColMap = map[string]string{}

// periodMarkerUniqueIndex is the partial unique index added by migration
// 20260428100000_revenue_period_marker_unique. When the concurrent-Generate
// race loses, postgres returns a unique_violation referencing this constraint.
// The CreateRevenue path translates that into the same `period_already_invoiced`
// shape the read-time idempotency check uses, so the drawer's blocking banner
// renders identically regardless of which side caught the conflict.
const periodMarkerUniqueIndex = "idx_revenue_subscription_period_unique"

// advancePeriodMarkerUniqueIndex is the mirror partial unique index added by
// migration 20260517180000_revenue_advance_period_marker_unique for the
// advance-amortization path (advance_collection_id, period_marker). Concurrent
// `AmortizeAdvanceCollection` runs that race past the read-before-write
// idempotency check land here; we translate the violation to the same
// `ErrPeriodAlreadyInvoiced` sentinel so the use case can map the outcome to
// SKIPPED + ConflictingRevenueId.
const advancePeriodMarkerUniqueIndex = "idx_revenue_advance_period_unique"

// ErrPeriodAlreadyInvoiced is the sentinel the recognize use case looks for to
// surface the user-facing "period already invoiced" banner.
var ErrPeriodAlreadyInvoiced = errors.New("period_already_invoiced")

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Revenue, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres revenue repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresRevenueRepository(dbOps, tableName), nil
	})
}

// PostgresRevenueRepository implements revenue CRUD operations using PostgreSQL
type PostgresRevenueRepository struct {
	revenuepb.UnimplementedRevenueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRevenueRepository creates a new PostgreSQL revenue repository
func NewPostgresRevenueRepository(dbOps interfaces.DatabaseOperation, tableName string) revenuepb.RevenueDomainServiceServer {
	if tableName == "" {
		tableName = "revenue"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresRevenueRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRevenue creates a new revenue record
func (r *PostgresRevenueRepository) CreateRevenue(ctx context.Context, req *revenuepb.CreateRevenueRequest) (*revenuepb.CreateRevenueResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "revenueDate", "revenue_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	convertMillisToTime(data, "dueDate", "due_date")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		// Concurrent-Generate race: the partial unique index on
		// (subscription_id, period_marker) caught the second writer. Surface
		// as the same period_already_invoiced sentinel the read-time
		// idempotency check uses so the drawer can render one banner.
		if isPeriodMarkerUniqueViolation(err) {
			return nil, fmt.Errorf("%w: %v", ErrPeriodAlreadyInvoiced, err)
		}
		return nil, fmt.Errorf("failed to create revenue: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "revenue_date", "due_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.CreateRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// isPeriodMarkerUniqueViolation returns true when err comes from either of
// the partial unique indexes that protect Revenue period_marker idempotency:
//   - idx_revenue_subscription_period_unique (subscription path)
//   - idx_revenue_advance_period_unique      (advance-amortization path)
//
// lib/pq surfaces the constraint name on pq.Error.Constraint; a substring
// match in the message is the fallback when the error is wrapped through
// dbOps.Create.
func isPeriodMarkerUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if pqErr.Code.Name() == "unique_violation" &&
			(pqErr.Constraint == periodMarkerUniqueIndex ||
				pqErr.Constraint == advancePeriodMarkerUniqueIndex) {
			return true
		}
	}
	msg := err.Error()
	return strings.Contains(msg, periodMarkerUniqueIndex) ||
		strings.Contains(msg, advancePeriodMarkerUniqueIndex)
}

// ReadRevenue retrieves a revenue record by ID
func (r *PostgresRevenueRepository) ReadRevenue(ctx context.Context, req *revenuepb.ReadRevenueRequest) (*revenuepb.ReadRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "revenue_date", "due_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.ReadRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// UpdateRevenue updates a revenue record
func (r *PostgresRevenueRepository) UpdateRevenue(ctx context.Context, req *revenuepb.UpdateRevenueRequest) (*revenuepb.UpdateRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "revenueDate", "revenue_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	convertMillisToTime(data, "dueDate", "due_date")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update revenue: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "revenue_date", "due_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	revenue := &revenuepb.Revenue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuepb.UpdateRevenueResponse{
		Success: true,
		Data:    []*revenuepb.Revenue{revenue},
	}, nil
}

// DeleteRevenue deletes a revenue record (soft delete)
func (r *PostgresRevenueRepository) DeleteRevenue(ctx context.Context, req *revenuepb.DeleteRevenueRequest) (*revenuepb.DeleteRevenueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete revenue: %w", err)
	}

	return &revenuepb.DeleteRevenueResponse{
		Success: true,
	}, nil
}

// ListRevenues lists revenue records with optional filters
func (r *PostgresRevenueRepository) ListRevenues(ctx context.Context, req *revenuepb.ListRevenuesRequest) (*revenuepb.ListRevenuesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenues: %w", err)
	}

	var revenues []*revenuepb.Revenue
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(result, "revenue_date", "due_date")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal revenue row: %v", err)
			continue
		}

		revenue := &revenuepb.Revenue{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, revenue); err != nil {
			log.Printf("WARN: protojson unmarshal revenue: %v", err)
			continue
		}
		revenues = append(revenues, revenue)
	}

	return &revenuepb.ListRevenuesResponse{
		Success: true,
		Data:    revenues,
	}, nil
}

// GetRevenueListPageData retrieves revenues with pagination, filtering, sorting, and search using CTE
// Joins with client and location tables for enriched display
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresRevenueRepository) GetRevenueListPageData(
	ctx context.Context,
	req *revenuepb.GetRevenueListPageDataRequest,
) (*revenuepb.GetRevenueListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue list page data request is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// Default pagination values
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

	// Sort with allowlist validation.
	sortCol := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortCol = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// Translate view-facing column key to SQL column name via ColMap.
	if mapped, ok := revenueViewToSQLColMap[sortCol]; ok {
		sortCol = mapped
	}

	// Loud-failure guard: reject any sort column not in the allowlist. This query
	// uses direct ORDER BY interpolation, so an unrecognised value is a potential
	// SQL-injection vector and must be rejected loudly before query execution.
	if sortCol != "" && !slices.Contains(revenueSortableSQLCols, sortCol) {
		return nil, fmt.Errorf("unknown sort column %q for entity %q (allowed: %v)", sortCol, "revenue", revenueSortableSQLCols)
	}

	// Build parameterized WHERE clauses via shared helper ($1 is reserved for workspace_id, start at $2)
	searchFields := []string{"rv.reference_number", "c.name"}
	filterClauses, filterArgs, nextIdx := postgresCore.BuildFilterWhere(req.Filters, req.Search, searchFields, 2)

	var whereStr string
	if len(filterClauses) > 0 {
		whereStr = " AND " + strings.Join(filterClauses, " AND ")
	}

	// Parameterized LIMIT/OFFSET come after filter args
	limitIdx := nextIdx
	offsetIdx := nextIdx + 1
	// workspace_id is $1; filter args follow; then limit/offset
	queryArgs := []any{workspaceID}
	queryArgs = append(queryArgs, filterArgs...)
	queryArgs = append(queryArgs, limit, offset)

	// 20260517 advance-cash-events: expose `advance_collection_id` so the list
	// row can flag advance-amortization revenue without a second round-trip.
	query := `
		WITH enriched AS (
			SELECT
				rv.id,
				rv.date_created,
				rv.date_modified,
				rv.active,
				rv.name,
				rv.client_id,
				rv.revenue_date_string,
				rv.total_amount,
				rv.currency,
				rv.status,
				rv.reference_number,
				rv.notes,
				rv.revenue_category_id,
				rv.location_id,
				rv.payment_term_id,
				rv.due_date_string,
				rv.subscription_id,
				rv.advance_collection_id,
				COALESCE(c.name, '') as client_name,
				COALESCE(l.name, '') as location_name,
				COALESCE(pt.name, '') as payment_term_name,
				EXISTS(SELECT 1 FROM treasury_collection tc WHERE tc.revenue_id = rv.id) as has_collection,
				COUNT(*) OVER() AS total_count
			FROM ` + r.tableName + ` rv
			LEFT JOIN client c ON rv.client_id = c.id AND c.active = true
			LEFT JOIN location l ON rv.location_id = l.id AND l.active = true
			LEFT JOIN payment_term pt ON rv.payment_term_id = pt.id
			WHERE rv.active = true AND rv.workspace_id = $1` + whereStr + `
		)
		SELECT * FROM enriched
		ORDER BY ` + sortCol + ` ` + sortOrder + fmt.Sprintf(`
		LIMIT $%d OFFSET $%d`, limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue list page data: %w", err)
	}
	defer rows.Close()

	var revenues []*revenuepb.Revenue
	var totalCount int64

	for rows.Next() {
		var (
			id                  string
			dateCreated         time.Time
			dateModified        time.Time
			active              bool
			name                string
			clientID            *string
			revenueDateString   *string
			totalAmount         int64
			currency            *string
			status              *string
			referenceNumber     *string
			notes               *string
			revenueCategoryID   *string
			locationID          *string
			paymentTermID       *string
			dueDateString       *string
			subscriptionID      *string
			advanceCollectionID *string
			clientName          string
			locationName        string
			paymentTermName     string
			hasCollection       bool
			total               int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&clientID,
			&revenueDateString,
			&totalAmount,
			&currency,
			&status,
			&referenceNumber,
			&notes,
			&revenueCategoryID,
			&locationID,
			&paymentTermID,
			&dueDateString,
			&subscriptionID,
			&advanceCollectionID,
			&clientName,
			&locationName,
			&paymentTermName,
			&hasCollection,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revenue row: %w", err)
		}

		totalCount = total

		revenue := &revenuepb.Revenue{
			Id:                id,
			Active:            active,
			Name:              name,
			TotalAmount:       totalAmount,
			ReferenceNumber:   referenceNumber,
			Notes:             notes,
			RevenueCategoryId: revenueCategoryID,
		}

		if clientID != nil {
			revenue.ClientId = *clientID
		}
		if locationID != nil {
			revenue.LocationId = *locationID
		}
		if currency != nil {
			revenue.Currency = *currency
		}
		if status != nil {
			revenue.Status = *status
		}
		if revenueDateString != nil {
			revenue.RevenueDate = revenueDateString
		}
		if paymentTermID != nil {
			revenue.PaymentTermId = paymentTermID
			if paymentTermName != "" {
				revenue.PaymentTerm = &paymenttermpb.PaymentTerm{
					Id:   *paymentTermID,
					Name: paymentTermName,
				}
			}
		}
		if dueDateString != nil {
			revenue.DueDate = dueDateString
		}
		if subscriptionID != nil {
			revenue.SubscriptionId = subscriptionID
		}
		if advanceCollectionID != nil {
			revenue.AdvanceCollectionId = advanceCollectionID
		}
		if hasCollection {
			hc := "has_collection"
			revenue.FulfillmentStatus = &hc
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			revenue.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			revenue.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			revenue.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			revenue.DateModifiedString = &dmStr
		}

		revenues = append(revenues, revenue)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenuepb.GetRevenueListPageDataResponse{
		RevenueList: revenues,
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

// GetRevenueItemPageData retrieves a single revenue with enriched data using CTE
// CRITICAL: Always filters by workspace_id for multi-tenancy
func (r *PostgresRevenueRepository) GetRevenueItemPageData(
	ctx context.Context,
	req *revenuepb.GetRevenueItemPageDataRequest,
) (*revenuepb.GetRevenueItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue item page data request is required")
	}
	if req.RevenueId == "" {
		return nil, fmt.Errorf("revenue ID is required")
	}

	// Extract workspace_id from context (REQUIRED for multi-tenancy)
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

	// 20260517 advance-cash-events: expose `advance_collection_id` so the
	// detail page can render the back-edge to the TreasuryCollection that
	// amortized into this revenue row.
	query := `
		WITH enriched AS (
			SELECT
				rv.id,
				rv.date_created,
				rv.date_modified,
				rv.active,
				rv.name,
				rv.client_id,
				rv.revenue_date_string,
				rv.total_amount,
				rv.currency,
				rv.status,
				rv.reference_number,
				rv.notes,
				rv.revenue_category_id,
				rv.location_id,
				rv.payment_term_id,
				rv.due_date_string,
				rv.subscription_id,
				rv.advance_collection_id,
				COALESCE(c.name, '') as client_name,
				COALESCE(l.name, '') as location_name
			FROM ` + r.tableName + ` rv
			LEFT JOIN client c ON rv.client_id = c.id AND c.active = true
			LEFT JOIN location l ON rv.location_id = l.id AND l.active = true
			WHERE rv.id = $1 AND rv.workspace_id = $2 AND rv.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.RevenueId, workspaceID)

	var (
		id                  string
		dateCreated         time.Time
		dateModified        time.Time
		active              bool
		name                string
		clientID            *string
		revenueDateString   *string
		totalAmount         int64
		currency            *string
		status              *string
		referenceNumber     *string
		notes               *string
		revenueCategoryID   *string
		locationID          *string
		paymentTermID       *string
		dueDateString       *string
		subscriptionID      *string
		advanceCollectionID *string
		clientName          string
		locationName        string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&clientID,
		&revenueDateString,
		&totalAmount,
		&currency,
		&status,
		&referenceNumber,
		&notes,
		&revenueCategoryID,
		&locationID,
		&paymentTermID,
		&dueDateString,
		&subscriptionID,
		&advanceCollectionID,
		&clientName,
		&locationName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue with ID '%s' not found", req.RevenueId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue item page data: %w", err)
	}

	revenue := &revenuepb.Revenue{
		Id:                id,
		Active:            active,
		Name:              name,
		TotalAmount:       totalAmount,
		ReferenceNumber:   referenceNumber,
		Notes:             notes,
		RevenueCategoryId: revenueCategoryID,
	}

	if clientID != nil {
		revenue.ClientId = *clientID
	}
	if locationID != nil {
		revenue.LocationId = *locationID
	}
	if currency != nil {
		revenue.Currency = *currency
	}
	if status != nil {
		revenue.Status = *status
	}
	if revenueDateString != nil {
		revenue.RevenueDate = revenueDateString
	}
	if paymentTermID != nil {
		revenue.PaymentTermId = paymentTermID
	}
	if dueDateString != nil {
		revenue.DueDate = dueDateString
	}
	if subscriptionID != nil {
		revenue.SubscriptionId = subscriptionID
	}
	if advanceCollectionID != nil {
		revenue.AdvanceCollectionId = advanceCollectionID
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		revenue.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		revenue.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		revenue.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		revenue.DateModifiedString = &dmStr
	}

	return &revenuepb.GetRevenueItemPageDataResponse{
		Revenue: revenue,
		Success: true,
	}, nil
}

// NewRevenueRepository creates a new PostgreSQL revenue repository (old-style constructor)
func NewRevenueRepository(db *sql.DB, tableName string) revenuepb.RevenueDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresRevenueRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey, _ string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string
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