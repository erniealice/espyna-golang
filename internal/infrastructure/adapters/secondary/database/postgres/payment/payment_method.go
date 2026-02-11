//go:build postgresql

package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "payment_method", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres payment_method repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPaymentMethodRepository(dbOps, tableName), nil
	})
}

// PostgresPaymentMethodRepository implements payment method CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_payment_method_active ON payment_method(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_payment_method_name ON payment_method(name) - Search on name field
//   - CREATE INDEX idx_payment_method_provider_name ON payment_method(provider_name) - Search on provider_name field
//   - CREATE INDEX idx_payment_method_name_trgm ON payment_method USING gin(name gin_trgm_ops) - Fuzzy search support
//   - CREATE INDEX idx_payment_method_date_created ON payment_method(date_created DESC) - Default sorting
type PostgresPaymentMethodRepository struct {
	paymentmethodpb.UnimplementedPaymentMethodDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresPaymentMethodRepository creates a new PostgreSQL payment method repository
func NewPostgresPaymentMethodRepository(dbOps interfaces.DatabaseOperation, tableName string) paymentmethodpb.PaymentMethodDomainServiceServer {
	if tableName == "" {
		tableName = "payment_method" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPaymentMethodRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePaymentMethod creates a new payment method using common PostgreSQL operations
func (r *PostgresPaymentMethodRepository) CreatePaymentMethod(ctx context.Context, req *paymentmethodpb.CreatePaymentMethodRequest) (*paymentmethodpb.CreatePaymentMethodResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment method data is required")
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
		return nil, fmt.Errorf("failed to create payment method: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentMethod := &paymentmethodpb.PaymentMethod{}
	if err := protojson.Unmarshal(resultJSON, paymentMethod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentmethodpb.CreatePaymentMethodResponse{
		Data: []*paymentmethodpb.PaymentMethod{paymentMethod},
	}, nil
}

// ReadPaymentMethod retrieves a payment method using common PostgreSQL operations
func (r *PostgresPaymentMethodRepository) ReadPaymentMethod(ctx context.Context, req *paymentmethodpb.ReadPaymentMethodRequest) (*paymentmethodpb.ReadPaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment method: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentMethod := &paymentmethodpb.PaymentMethod{}
	if err := protojson.Unmarshal(resultJSON, paymentMethod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentmethodpb.ReadPaymentMethodResponse{
		Data: []*paymentmethodpb.PaymentMethod{paymentMethod},
	}, nil
}

// UpdatePaymentMethod updates a payment method using common PostgreSQL operations
func (r *PostgresPaymentMethodRepository) UpdatePaymentMethod(ctx context.Context, req *paymentmethodpb.UpdatePaymentMethodRequest) (*paymentmethodpb.UpdatePaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required")
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
		return nil, fmt.Errorf("failed to update payment method: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	paymentMethod := &paymentmethodpb.PaymentMethod{}
	if err := protojson.Unmarshal(resultJSON, paymentMethod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentmethodpb.UpdatePaymentMethodResponse{
		Data: []*paymentmethodpb.PaymentMethod{paymentMethod},
	}, nil
}

// DeletePaymentMethod deletes a payment method using common PostgreSQL operations
func (r *PostgresPaymentMethodRepository) DeletePaymentMethod(ctx context.Context, req *paymentmethodpb.DeletePaymentMethodRequest) (*paymentmethodpb.DeletePaymentMethodResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment method: %w", err)
	}

	return &paymentmethodpb.DeletePaymentMethodResponse{
		Success: true,
	}, nil
}

// ListPaymentMethods lists payment methods using common PostgreSQL operations
func (r *PostgresPaymentMethodRepository) ListPaymentMethods(ctx context.Context, req *paymentmethodpb.ListPaymentMethodsRequest) (*paymentmethodpb.ListPaymentMethodsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var paymentMethods []*paymentmethodpb.PaymentMethod
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		paymentMethod := &paymentmethodpb.PaymentMethod{}
		if err := protojson.Unmarshal(resultJSON, paymentMethod); err != nil {
			// Log error and continue with next item
			continue
		}
		paymentMethods = append(paymentMethods, paymentMethod)
	}

	return &paymentmethodpb.ListPaymentMethodsResponse{
		Data: paymentMethods,
	}, nil
}

// GetPaymentMethodListPageData retrieves payment methods with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresPaymentMethodRepository) GetPaymentMethodListPageData(
	ctx context.Context,
	req *paymentmethodpb.GetPaymentMethodListPageDataRequest,
) (*paymentmethodpb.GetPaymentMethodListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment method list page data request is required")
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
		// Handle offset pagination
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Simple entity pattern with name/provider_name search
	query := `
		WITH enriched AS (
			SELECT
				pm.id,
				pm.name,
				pm.provider_name,
				pm.active,
				pm.date_created,
				pm.date_modified
			FROM payment_method pm
			WHERE pm.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   pm.name ILIKE $1 OR
				   pm.provider_name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query payment method list page data: %w", err)
	}
	defer rows.Close()

	var paymentMethods []*paymentmethodpb.PaymentMethod
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			name               string
			providerName       *string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)

		err := rows.Scan(
			&id,
			&name,
			&providerName,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment method row: %w", err)
		}

		totalCount = total

		paymentMethod := &paymentmethodpb.PaymentMethod{
			Id:     id,
			Name:   name,
			Active: active,
		}

		if providerName != nil {
			paymentMethod.ProviderName = providerName
		}

		// Handle nullable timestamp fields

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		paymentMethod.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		paymentMethod.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		paymentMethod.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		paymentMethod.DateModifiedString = &dmStr
	}

		paymentMethods = append(paymentMethods, paymentMethod)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payment method rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &paymentmethodpb.GetPaymentMethodListPageDataResponse{
		PaymentMethodList: paymentMethods,
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

// GetPaymentMethodItemPageData retrieves a single payment method with enhanced item page data
func (r *PostgresPaymentMethodRepository) GetPaymentMethodItemPageData(
	ctx context.Context,
	req *paymentmethodpb.GetPaymentMethodItemPageDataRequest,
) (*paymentmethodpb.GetPaymentMethodItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get payment method item page data request is required")
	}
	if req.PaymentMethodId == "" {
		return nil, fmt.Errorf("payment method ID is required")
	}

	// Simple query for single payment method item
	query := `
		SELECT
			pm.id,
			pm.name,
			pm.provider_name,
			pm.active,
			pm.date_created,
			pm.date_modified
		FROM payment_method pm
		WHERE pm.id = $1 AND pm.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.PaymentMethodId)

	var (
		id                 string
		name               string
		providerName       *string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&providerName,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment method with ID '%s' not found", req.PaymentMethodId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query payment method item page data: %w", err)
	}

	paymentMethod := &paymentmethodpb.PaymentMethod{
		Id:     id,
		Name:   name,
		Active: active,
	}

	if providerName != nil {
		paymentMethod.ProviderName = providerName
	}

	// Handle nullable timestamp fields

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		paymentMethod.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		paymentMethod.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		paymentMethod.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		paymentMethod.DateModifiedString = &dmStr
	}

	return &paymentmethodpb.GetPaymentMethodItemPageDataResponse{
		PaymentMethod: paymentMethod,
		Success:       true,
	}, nil
}

// parsePaymentMethodTimestamp converts string timestamp to Unix timestamp (milliseconds)
func parsePaymentMethodTimestamp(timestampStr string) (int64, error) {
	// Try parsing as RFC3339 format first (most common)
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// NewPaymentMethodRepository creates a new PostgreSQL payment_method repository (old-style constructor)
func NewPaymentMethodRepository(db *sql.DB, tableName string) paymentmethodpb.PaymentMethodDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPaymentMethodRepository(dbOps, tableName)
}
