//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PaymentTerm, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres payment_term repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPaymentTermRepository(dbOps, tableName), nil
	})
}

// PostgresPaymentTermRepository implements payment term CRUD operations using PostgreSQL
type PostgresPaymentTermRepository struct {
	paymenttermpb.UnimplementedPaymentTermDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresPaymentTermRepository creates a new PostgreSQL payment term repository
func NewPostgresPaymentTermRepository(dbOps interfaces.DatabaseOperation, tableName string) paymenttermpb.PaymentTermDomainServiceServer {
	if tableName == "" {
		tableName = "payment_term" // default fallback
	}

	return &PostgresPaymentTermRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreatePaymentTerm creates a new payment term using common PostgreSQL operations
func (r *PostgresPaymentTermRepository) CreatePaymentTerm(ctx context.Context, req *paymenttermpb.CreatePaymentTermRequest) (*paymenttermpb.CreatePaymentTermResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment term data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment term: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentTerm := &paymenttermpb.PaymentTerm{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, paymentTerm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymenttermpb.CreatePaymentTermResponse{
		Data: []*paymenttermpb.PaymentTerm{paymentTerm},
	}, nil
}

// ReadPaymentTerm retrieves a payment term by ID
func (r *PostgresPaymentTermRepository) ReadPaymentTerm(ctx context.Context, req *paymenttermpb.ReadPaymentTermRequest) (*paymenttermpb.ReadPaymentTermResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment term ID is required")
	}

	query := `
		SELECT
			id,
			date_created,
			date_modified,
			active,
			name,
			code,
			type,
			net_days,
			discount_days,
			discount_percent_bps,
			entity_scope,
			is_default,
			description,
			display_order
		FROM ` + r.tableName + `
		WHERE id = $1 AND active = true
		  AND ($2::text = '' OR workspace_id = $2::text)
	`

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.Data.Id, wsID)

	var (
		id                 string
		dateCreated        *time.Time
		dateModified       *time.Time
		active             bool
		name               string
		code               string
		ptType             string
		netDays            int32
		discountDays       *int32
		discountPercentBps *int32
		entityScope        string
		isDefault          bool
		description        *string
		displayOrder       *int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&code,
		&ptType,
		&netDays,
		&discountDays,
		&discountPercentBps,
		&entityScope,
		&isDefault,
		&description,
		&displayOrder,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment term with ID '%s' not found", req.Data.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read payment term: %w", err)
	}

	paymentTerm := buildPaymentTermFromScan(
		id, dateCreated, dateModified, active,
		name, code, ptType, netDays,
		discountDays, discountPercentBps, entityScope, isDefault,
		description, displayOrder,
	)

	return &paymenttermpb.ReadPaymentTermResponse{
		Data:    []*paymenttermpb.PaymentTerm{paymentTerm},
		Success: true,
	}, nil
}

// UpdatePaymentTerm updates a payment term using common PostgreSQL operations
func (r *PostgresPaymentTermRepository) UpdatePaymentTerm(ctx context.Context, req *paymenttermpb.UpdatePaymentTermRequest) (*paymenttermpb.UpdatePaymentTermResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment term ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment term: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentTerm := &paymenttermpb.PaymentTerm{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, paymentTerm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymenttermpb.UpdatePaymentTermResponse{
		Data: []*paymenttermpb.PaymentTerm{paymentTerm},
	}, nil
}

// DeletePaymentTerm deletes a payment term using common PostgreSQL operations (soft delete)
func (r *PostgresPaymentTermRepository) DeletePaymentTerm(ctx context.Context, req *paymenttermpb.DeletePaymentTermRequest) (*paymenttermpb.DeletePaymentTermResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment term ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment term: %w", err)
	}

	return &paymenttermpb.DeletePaymentTermResponse{
		Success: true,
	}, nil
}

// ListPaymentTerms lists payment terms using common PostgreSQL operations
func (r *PostgresPaymentTermRepository) ListPaymentTerms(ctx context.Context, req *paymenttermpb.ListPaymentTermsRequest) (*paymenttermpb.ListPaymentTermsResponse, error) {
	// Pass through filters from the request
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}

	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment terms: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var paymentTerms []*paymenttermpb.PaymentTerm
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		paymentTerm := &paymenttermpb.PaymentTerm{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, paymentTerm); err != nil {
			continue
		}
		paymentTerms = append(paymentTerms, paymentTerm)
	}

	return &paymenttermpb.ListPaymentTermsResponse{
		Data: paymentTerms,
	}, nil
}

var paymentTermSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "name", "code", "type",
	"net_days", "discount_days", "discount_percent_bps", "entity_scope",
	"is_default", "description", "display_order",
}

// GetPaymentTermListPageData retrieves payment terms with filtering, sorting, searching, and pagination using CTE
func (r *PostgresPaymentTermRepository) GetPaymentTermListPageData(
	ctx context.Context,
	req *paymenttermpb.GetPaymentTermListPageDataRequest,
) (*paymenttermpb.GetPaymentTermListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment term list page data request is required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). Route the
	// caller-supplied sort column through core.BuildOrderBy so an unknown column
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(paymentTermSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	// Workspace isolation: GetPaymentTermListPageData uses raw SQL and bypasses
	// the WorkspaceAwareOperations decorator, so we extract the workspace_id from
	// context and filter explicitly. Empty workspace_id (service-to-service call)
	// disables the filter — same convention as the decorator.
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	// CTE Query - flat table, no JOINs needed
	// entity_scope filter: show only client-scoped and shared (both) payment terms
	query := `
		WITH enriched AS (
			SELECT
				id,
				date_created,
				date_modified,
				active,
				name,
				code,
				type,
				net_days,
				discount_days,
				discount_percent_bps,
				entity_scope,
				is_default,
				description,
				display_order
			FROM ` + r.tableName + `
			WHERE entity_scope IN ('client', 'both')
			  AND ($4::text = '' OR workspace_id = $4::text)
			  AND ($1::text IS NULL OR $1::text = '' OR
				   name ILIKE $1 OR
				   code ILIKE $1 OR
				   description ILIKE $1)
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

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to query payment term list page data: %w", err)
	}
	defer rows.Close()

	var paymentTerms []*paymenttermpb.PaymentTerm
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			dateCreated        *time.Time
			dateModified       *time.Time
			active             bool
			name               string
			code               string
			ptType             string
			netDays            int32
			discountDays       *int32
			discountPercentBps *int32
			entityScope        string
			isDefault          bool
			description        *string
			displayOrder       *int32
			total              int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&code,
			&ptType,
			&netDays,
			&discountDays,
			&discountPercentBps,
			&entityScope,
			&isDefault,
			&description,
			&displayOrder,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment term row: %w", err)
		}

		totalCount = total

		paymentTerm := buildPaymentTermFromScan(
			id, dateCreated, dateModified, active,
			name, code, ptType, netDays,
			discountDays, discountPercentBps, entityScope, isDefault,
			description, displayOrder,
		)

		paymentTerms = append(paymentTerms, paymentTerm)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payment term rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &paymenttermpb.GetPaymentTermListPageDataResponse{
		PaymentTermList: paymentTerms,
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

// GetPaymentTermItemPageData retrieves a single payment term by ID
func (r *PostgresPaymentTermRepository) GetPaymentTermItemPageData(
	ctx context.Context,
	req *paymenttermpb.GetPaymentTermItemPageDataRequest,
) (*paymenttermpb.GetPaymentTermItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment term item page data request is required")
	}
	if req.PaymentTermId == "" {
		return nil, fmt.Errorf("payment term ID is required")
	}

	// CTE Query - single round-trip
	query := `
		WITH enriched AS (
			SELECT
				id,
				date_created,
				date_modified,
				active,
				name,
				code,
				type,
				net_days,
				discount_days,
				discount_percent_bps,
				entity_scope,
				is_default,
				description,
				display_order
			FROM ` + r.tableName + `
			WHERE id = $1
			  AND ($2::text = '' OR workspace_id = $2::text)
		)
		SELECT * FROM enriched LIMIT 1;
	`

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.PaymentTermId, wsID)

	var (
		id                 string
		dateCreated        *time.Time
		dateModified       *time.Time
		active             bool
		name               string
		code               string
		ptType             string
		netDays            int32
		discountDays       *int32
		discountPercentBps *int32
		entityScope        string
		isDefault          bool
		description        *string
		displayOrder       *int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&code,
		&ptType,
		&netDays,
		&discountDays,
		&discountPercentBps,
		&entityScope,
		&isDefault,
		&description,
		&displayOrder,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment term with ID '%s' not found", req.PaymentTermId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query payment term item page data: %w", err)
	}

	paymentTerm := buildPaymentTermFromScan(
		id, dateCreated, dateModified, active,
		name, code, ptType, netDays,
		discountDays, discountPercentBps, entityScope, isDefault,
		description, displayOrder,
	)

	return &paymenttermpb.GetPaymentTermItemPageDataResponse{
		PaymentTerm: paymentTerm,
		Success:     true,
	}, nil
}

// buildPaymentTermFromScan constructs a PaymentTerm protobuf from scanned SQL fields
func buildPaymentTermFromScan(
	id string, dateCreated *time.Time, dateModified *time.Time, active bool,
	name string, code string, ptType string, netDays int32,
	discountDays *int32, discountPercentBps *int32, entityScope string, isDefault bool,
	description *string, displayOrder *int32,
) *paymenttermpb.PaymentTerm {
	paymentTerm := &paymenttermpb.PaymentTerm{
		Id:          id,
		Active:      active,
		Name:        name,
		Code:        code,
		Type:        ptType,
		NetDays:     netDays,
		EntityScope: entityScope,
		IsDefault:   isDefault,
	}

	paymentTerm.DiscountDays = discountDays
	paymentTerm.DiscountPercentBps = discountPercentBps
	paymentTerm.Description = description
	paymentTerm.DisplayOrder = displayOrder

	// Parse timestamps
	if dateCreated != nil && !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		paymentTerm.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		paymentTerm.DateCreatedString = &dcStr
	}
	if dateModified != nil && !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		paymentTerm.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		paymentTerm.DateModifiedString = &dmStr
	}

	return paymentTerm
}

// NewPaymentTermRepository creates a new PostgreSQL payment term repository (old-style constructor)
func NewPaymentTermRepository(db *sql.DB, tableName string) paymenttermpb.PaymentTermDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPaymentTermRepository(dbOps, tableName)
}
