//go:build postgres

package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "payment", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres payment repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPaymentRepository(dbOps, tableName), nil
	})
}

// PostgresPaymentRepository implements payment CRUD operations using PostgreSQL
type PostgresPaymentRepository struct {
	paymentpb.UnimplementedPaymentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewPostgresPaymentRepository creates a new PostgreSQL payment repository
func NewPostgresPaymentRepository(dbOps interfaces.DatabaseOperation, tableName string) paymentpb.PaymentDomainServiceServer {
	if tableName == "" {
		tableName = "payment" // default fallback
	}
	return &PostgresPaymentRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreatePayment creates a new payment using common PostgreSQL operations
func (r *PostgresPaymentRepository) CreatePayment(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("payment data is required")
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
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	payment := &paymentpb.Payment{}
	if err := protojson.Unmarshal(resultJSON, payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentpb.CreatePaymentResponse{
		Data: []*paymentpb.Payment{payment},
	}, nil
}

// ReadPayment retrieves a payment using common PostgreSQL operations
func (r *PostgresPaymentRepository) ReadPayment(ctx context.Context, req *paymentpb.ReadPaymentRequest) (*paymentpb.ReadPaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	payment := &paymentpb.Payment{}
	if err := protojson.Unmarshal(resultJSON, payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentpb.ReadPaymentResponse{
		Data: []*paymentpb.Payment{payment},
	}, nil
}

// UpdatePayment updates a payment using common PostgreSQL operations
func (r *PostgresPaymentRepository) UpdatePayment(ctx context.Context, req *paymentpb.UpdatePaymentRequest) (*paymentpb.UpdatePaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required")
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
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	payment := &paymentpb.Payment{}
	if err := protojson.Unmarshal(resultJSON, payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &paymentpb.UpdatePaymentResponse{
		Data: []*paymentpb.Payment{payment},
	}, nil
}

// DeletePayment deletes a payment using common PostgreSQL operations
func (r *PostgresPaymentRepository) DeletePayment(ctx context.Context, req *paymentpb.DeletePaymentRequest) (*paymentpb.DeletePaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payment: %w", err)
	}

	return &paymentpb.DeletePaymentResponse{
		Success: true,
	}, nil
}

// ListPayments lists payments using common PostgreSQL operations
func (r *PostgresPaymentRepository) ListPayments(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var payments []*paymentpb.Payment
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		payment := &paymentpb.Payment{}
		if err := protojson.Unmarshal(resultJSON, payment); err != nil {
			// Log error and continue with next item
			continue
		}
		payments = append(payments, payment)
	}

	return &paymentpb.ListPaymentsResponse{
		Data: payments,
	}, nil
}

// GetPaymentListPageData retrieves a paginated, filtered, sorted, and searchable list of payments with invoice, payment_method, subscription, and client relationships
// This method uses CTEs (Common Table Expressions) to optimize query performance by loading all data in a single query
// TODO: Add unit tests for GetPaymentListPageData
func (r *PostgresPaymentRepository) GetPaymentListPageData(ctx context.Context, req *paymentpb.GetPaymentListPageDataRequest) (*paymentpb.GetPaymentListPageDataResponse, error) {
	// Extract pagination parameters with defaults
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100 // Cap at 100 items per page
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	// Extract search query
	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	// Extract sort parameters with defaults
	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 { // DESC enum value
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_payment_active ON payment(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_payment_subscription_id ON payment(subscription_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_payment_date_created ON payment(date_created DESC);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_active ON subscription(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_name_trgm ON subscription USING gin(name gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_client_id ON subscription(client_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_client_active ON client(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_client_user_id ON client(user_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_user_active ON "user"(active) WHERE active = true;

	// Build the CTE query following the translation plan pattern
	// Payment → Subscription → Client → User relationship chain
	query := `
		WITH
		-- CTE 1: Apply search filter on payment
		search_filtered AS (
			SELECT p.*
			FROM payment p
			LEFT JOIN subscription s ON p.subscription_id = s.id AND s.active = true
			LEFT JOIN client c ON s.client_id = c.id AND c.active = true
			LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
			WHERE p.active = true
				AND ($1::text = '' OR
					s.name ILIKE $1 OR
					(u.first_name || ' ' || u.last_name) ILIKE $1)
		),

		-- CTE 2: Join with subscription, client, and user
		enriched AS (
			SELECT
				sf.id,
				sf.name,
				sf.subscription_id,
				sf.amount,
				sf.status,
				sf.active,
				sf.date_created,
				sf.date_modified,
				jsonb_build_object(
					'id', s.id,
					'name', s.name,
					'client_id', s.client_id,
					'plan_id', s.plan_id,
					'date_start', s.date_start,
					'date_start_string', s.date_start_string,
					'date_end', s.date_end,
					'date_end_string', s.date_end_string,
					'active', s.active,
					'date_created', s.date_created,
					'date_modified', s.date_modified,
					'client', jsonb_build_object(
						'id', c.id,
						'user_id', c.user_id,
						'internal_id', c.internal_id,
						'active', c.active,
						'date_created', c.date_created,
						'date_modified', c.date_modified,
						'user', jsonb_build_object(
							'id', u.id,
							'first_name', u.first_name,
							'last_name', u.last_name,
							'email_address', u.email_address,
							'active', u.active,
							'date_created', u.date_created,
							'date_modified', u.date_modified
						)
					)
				) as subscription
			FROM search_filtered sf
			LEFT JOIN subscription s ON sf.subscription_id = s.id AND s.active = true
			LEFT JOIN client c ON s.client_id = c.id AND c.active = true
			LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
		),

		-- CTE 3: Apply sorting
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN $4 = 'name' AND $5 = 'ASC' THEN name END ASC,
				CASE WHEN $4 = 'name' AND $5 = 'DESC' THEN name END DESC,
				CASE WHEN $4 = 'amount' AND $5 = 'ASC' THEN amount END ASC,
				CASE WHEN $4 = 'amount' AND $5 = 'DESC' THEN amount END DESC,
				CASE WHEN $4 = 'status' AND $5 = 'ASC' THEN status END ASC,
				CASE WHEN $4 = 'status' AND $5 = 'DESC' THEN status END DESC,
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC
		),

		-- CTE 4: Calculate total count for pagination
		total_count AS (
			SELECT count(*) as total FROM sorted
		)

		-- Final SELECT with pagination
		SELECT
			s.id,
			s.name,
			s.subscription_id,
			s.amount,
			s.status,
			s.active,
			s.date_created,
			s.date_modified,
			s.subscription,
			tc.total as _total_count
		FROM sorted s
		CROSS JOIN total_count tc
		LIMIT $2 OFFSET $3
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	rows, err := db.GetDB().QueryContext(ctx, query,
		searchQuery,   // $1
		limit,         // $2
		offset,        // $3
		sortField,     // $4
		sortDirection, // $5
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetPaymentListPageData query: %w", err)
	}
	defer rows.Close()

	var payments []*paymentpb.Payment
	var totalCount int32

	for rows.Next() {
		var (
			id                 string
			name               string
			subscriptionID     string
			amount             float64
			status             string
			active             bool
			dateCreated        sql.NullInt64
			dateCreatedString  sql.NullString
			dateModified       sql.NullInt64
			dateModifiedString sql.NullString
			subscriptionJSON   []byte
			rowTotalCount      int32
		)

		err := rows.Scan(
			&id,
			&name,
			&subscriptionID,
			&amount,
			&status,
			&active,
			&dateCreated,
			&dateModified,
			&subscriptionJSON,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment row: %w", err)
		}

		totalCount = rowTotalCount

		// Build payment message
		payment := &paymentpb.Payment{
			Id:             id,
			Name:           name,
			SubscriptionId: subscriptionID,
			Amount:         amount,
			Status:         status,
			Active:         active,
		}

		if dateCreated.Valid {
			payment.DateCreated = &dateCreated.Int64
		}
		if dateCreatedString.Valid {
			payment.DateCreatedString = &dateCreatedString.String
		}
		if dateModified.Valid {
			payment.DateModified = &dateModified.Int64
		}
		if dateModifiedString.Valid {
			payment.DateModifiedString = &dateModifiedString.String
		}

		// Parse subscription JSON (with nested client and user)
		if len(subscriptionJSON) > 0 {
			var subscriptionData map[string]any
			if err := json.Unmarshal(subscriptionJSON, &subscriptionData); err == nil {
				subscriptionJSONBytes, _ := json.Marshal(subscriptionData)
				var subscription subscriptionpb.Subscription
				if err := protojson.Unmarshal(subscriptionJSONBytes, &subscription); err == nil {
					payment.Subscription = &subscription
				}
			}
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payment rows: %w", err)
	}

	// Build pagination response
	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	paginationResponse := &commonpb.PaginationResponse{
		TotalItems:  totalCount,
		CurrentPage: &page,
		TotalPages:  &totalPages,
		HasNext:     hasNext,
		HasPrev:     hasPrev,
	}

	return &paymentpb.GetPaymentListPageDataResponse{
		Success:     true,
		PaymentList: payments,
		Pagination:  paginationResponse,
	}, nil
}

// GetPaymentItemPageData retrieves a single payment with all related subscription, client, and user data expanded
// This method uses JOINs to load all related data in a single query
// TODO: Add unit tests for GetPaymentItemPageData
func (r *PostgresPaymentRepository) GetPaymentItemPageData(ctx context.Context, req *paymentpb.GetPaymentItemPageDataRequest) (*paymentpb.GetPaymentItemPageDataResponse, error) {
	if req.PaymentId == "" {
		return nil, fmt.Errorf("payment ID is required")
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_payment_id ON payment(id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_payment_subscription_id ON payment(subscription_id);

	// Build query to fetch payment with all related data
	query := `
		SELECT
			p.id,
			p.name,
			p.subscription_id,
			p.amount,
			p.status,
			p.active,
			p.date_created,
			p.date_modified,
			jsonb_build_object(
				'id', s.id,
				'name', s.name,
				'client_id', s.client_id,
				'plan_id', s.plan_id,
				'date_start', s.date_start,
				'date_start_string', s.date_start_string,
				'date_end', s.date_end,
				'date_end_string', s.date_end_string,
				'active', s.active,
				'date_created', s.date_created,
				'date_modified', s.date_modified,
				'client', jsonb_build_object(
					'id', c.id,
					'user_id', c.user_id,
					'internal_id', c.internal_id,
					'active', c.active,
					'date_created', c.date_created,
					'date_modified', c.date_modified,
					'user', jsonb_build_object(
						'id', u.id,
						'first_name', u.first_name,
						'last_name', u.last_name,
						'email_address', u.email_address,
						'active', u.active,
						'date_created', u.date_created,
						'date_modified', u.date_modified
					)
				)
			) as subscription
		FROM payment p
		LEFT JOIN subscription s ON p.subscription_id = s.id AND s.active = true
		LEFT JOIN client c ON s.client_id = c.id AND c.active = true
		LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
		WHERE p.id = $1 AND p.active = true
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	var (
		id                 string
		name               string
		subscriptionID     string
		amount             float64
		status             string
		active             bool
		dateCreated        sql.NullInt64
		dateCreatedString  sql.NullString
		dateModified       sql.NullInt64
		dateModifiedString sql.NullString
		subscriptionJSON   []byte
	)

	err := db.GetDB().QueryRowContext(ctx, query, req.PaymentId).Scan(
		&id,
		&name,
		&subscriptionID,
		&amount,
		&status,
		&active,
		&dateCreated,
		&dateModified,
		&subscriptionJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found with ID: %s", req.PaymentId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetPaymentItemPageData query: %w", err)
	}

	// Build payment message
	payment := &paymentpb.Payment{
		Id:             id,
		Name:           name,
		SubscriptionId: subscriptionID,
		Amount:         amount,
		Status:         status,
		Active:         active,
	}

	if dateCreated.Valid {
		payment.DateCreated = &dateCreated.Int64
	}
	if dateCreatedString.Valid {
		payment.DateCreatedString = &dateCreatedString.String
	}
	if dateModified.Valid {
		payment.DateModified = &dateModified.Int64
	}
	if dateModifiedString.Valid {
		payment.DateModifiedString = &dateModifiedString.String
	}

	// Parse subscription JSON (with nested client and user)
	if len(subscriptionJSON) > 0 {
		var subscriptionData map[string]any
		if err := json.Unmarshal(subscriptionJSON, &subscriptionData); err == nil {
			subscriptionJSONBytes, _ := json.Marshal(subscriptionData)
			var subscription subscriptionpb.Subscription
			if err := protojson.Unmarshal(subscriptionJSONBytes, &subscription); err == nil {
				payment.Subscription = &subscription
			}
		}
	}

	return &paymentpb.GetPaymentItemPageDataResponse{
		Success: true,
		Payment: payment,
	}, nil
}

// NewPaymentRepository creates a new PostgreSQL payment repository (old-style constructor)
func NewPaymentRepository(db *sql.DB, tableName string) paymentpb.PaymentDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPaymentRepository(dbOps, tableName)
}
