//go:build postgres

package subscription

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
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// PostgresSubscriptionRepository implements subscription CRUD operations using PostgreSQL
type PostgresSubscriptionRepository struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "subscription", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresSubscriptionRepository(dbOps, tableName), nil
	})
}

// NewPostgresSubscriptionRepository creates a new PostgreSQL subscription repository
func NewPostgresSubscriptionRepository(dbOps interfaces.DatabaseOperation, tableName string) subscriptionpb.SubscriptionDomainServiceServer {
	if tableName == "" {
		tableName = "subscription" // default fallback
	}
	return &PostgresSubscriptionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateSubscription creates a new subscription using common PostgreSQL operations
func (r *PostgresSubscriptionRepository) CreateSubscription(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription data is required")
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
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscription := &subscriptionpb.Subscription{}
	if err := protojson.Unmarshal(resultJSON, subscription); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionpb.CreateSubscriptionResponse{
		Data: []*subscriptionpb.Subscription{subscription},
	}, nil
}

// ReadSubscription retrieves a subscription using common PostgreSQL operations
func (r *PostgresSubscriptionRepository) ReadSubscription(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscription := &subscriptionpb.Subscription{}
	if err := protojson.Unmarshal(resultJSON, subscription); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionpb.ReadSubscriptionResponse{
		Data: []*subscriptionpb.Subscription{subscription},
	}, nil
}

// UpdateSubscription updates a subscription using common PostgreSQL operations
func (r *PostgresSubscriptionRepository) UpdateSubscription(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
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
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscription := &subscriptionpb.Subscription{}
	if err := protojson.Unmarshal(resultJSON, subscription); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &subscriptionpb.UpdateSubscriptionResponse{
		Data: []*subscriptionpb.Subscription{subscription},
	}, nil
}

// DeleteSubscription deletes a subscription using common PostgreSQL operations
func (r *PostgresSubscriptionRepository) DeleteSubscription(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete subscription: %w", err)
	}

	return &subscriptionpb.DeleteSubscriptionResponse{
		Success: true,
	}, nil
}

// ListSubscriptions lists subscriptions using common PostgreSQL operations
func (r *PostgresSubscriptionRepository) ListSubscriptions(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var subscriptions []*subscriptionpb.Subscription
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		subscription := &subscriptionpb.Subscription{}
		if err := protojson.Unmarshal(resultJSON, subscription); err != nil {
			// Log error and continue with next item
			continue
		}
		subscriptions = append(subscriptions, subscription)
	}

	return &subscriptionpb.ListSubscriptionsResponse{
		Data: subscriptions,
	}, nil
}

// GetSubscriptionListPageData retrieves a paginated, filtered, sorted, and searchable list of subscriptions with client and plan relationships
// This method uses CTEs (Common Table Expressions) to optimize query performance by loading all data in a single query
// TODO: Add unit tests for GetSubscriptionListPageData
func (r *PostgresSubscriptionRepository) GetSubscriptionListPageData(ctx context.Context, req *subscriptionpb.GetSubscriptionListPageDataRequest) (*subscriptionpb.GetSubscriptionListPageDataResponse, error) {
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

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_active ON subscription(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_name_trgm ON subscription USING gin(name gin_trgm_ops);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_client_id ON subscription(client_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_price_plan_id ON subscription(price_plan_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_date_created ON subscription(date_created DESC);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_client_active ON client(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_client_user_id ON client(user_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_price_plan_active ON price_plan(active) WHERE active = true;
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_user_active ON "user"(active) WHERE active = true;

	// Build the CTE query following the translation plan pattern
	query := `
		WITH
		-- CTE 1: Apply search filter on subscription
		search_filtered AS (
			SELECT s.*
			FROM subscription s
			WHERE s.active = true
				AND ($1::text = '' OR
					s.name ILIKE $1)
		),

		-- CTE 2: Join with client and user
		enriched AS (
			SELECT
				sf.id,
				sf.name,
				sf.client_id,
				sf.price_plan_id,
				sf.date_start,
				sf.date_start_string,
				sf.date_end,
				sf.date_end_string,
				sf.active,
				sf.date_created,
				sf.date_modified,
				jsonb_build_object(
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
				) as client,
				jsonb_build_object(
					'id', pp.id,
					'name', pp.name,
					'description', pp.description,
					'active', pp.active,
					'date_created', pp.date_created,
					'date_modified', pp.date_modified
				) as price_plan
			FROM search_filtered sf
			LEFT JOIN client c ON sf.client_id = c.id AND c.active = true
			LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
			LEFT JOIN price_plan pp ON sf.price_plan_id = pp.id AND pp.active = true
		),

		-- CTE 3: Apply sorting
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN $4 = 'name' AND $5 = 'ASC' THEN name END ASC,
				CASE WHEN $4 = 'name' AND $5 = 'DESC' THEN name END DESC,
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC,
				CASE WHEN $4 = 'date_start' AND $5 = 'ASC' THEN date_start END ASC,
				CASE WHEN $4 = 'date_start' AND $5 = 'DESC' THEN date_start END DESC,
				CASE WHEN $4 = 'date_end' AND $5 = 'ASC' THEN date_end END ASC,
				CASE WHEN $4 = 'date_end' AND $5 = 'DESC' THEN date_end END DESC
		),

		-- CTE 4: Calculate total count for pagination
		total_count AS (
			SELECT count(*) as total FROM sorted
		)

		-- Final SELECT with pagination
		SELECT
			s.id,
			s.name,
			s.client_id,
			s.price_plan_id,
			s.date_start,
			s.date_start_string,
			s.date_end,
			s.date_end_string,
			s.active,
			s.date_created,
			s.date_modified,
			s.client,
			s.price_plan,
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
		return nil, fmt.Errorf("failed to execute GetSubscriptionListPageData query: %w", err)
	}
	defer rows.Close()

	var subscriptions []*subscriptionpb.Subscription
	var totalCount int32

	for rows.Next() {
		var (
			id                 string
			name               string
			clientID           string
			pricePlanID        string
			dateStart          sql.NullInt64
			dateStartString    sql.NullString
			dateEnd            sql.NullInt64
			dateEndString      sql.NullString
			active             bool
			dateCreated        sql.NullInt64
			dateCreatedString  sql.NullString
			dateModified       sql.NullInt64
			dateModifiedString sql.NullString
			clientJSON         []byte
			pricePlanJSON      []byte
			rowTotalCount      int32
		)

		err := rows.Scan(
			&id,
			&name,
			&clientID,
			&pricePlanID,
			&dateStart,
			&dateStartString,
			&dateEnd,
			&dateEndString,
			&active,
			&dateCreated,
			&dateModified,
			&clientJSON,
			&pricePlanJSON,
			&rowTotalCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription row: %w", err)
		}

		totalCount = rowTotalCount

		// Build subscription message
		subscription := &subscriptionpb.Subscription{
			Id:          id,
			Name:        name,
			ClientId:    clientID,
			PricePlanId: pricePlanID,
			Active:      active,
		}

		if dateStart.Valid {
			subscription.DateStart = &dateStart.Int64
		}
		if dateStartString.Valid {
			subscription.DateStartString = &dateStartString.String
		}
		if dateEnd.Valid {
			subscription.DateEnd = &dateEnd.Int64
		}
		if dateEndString.Valid {
			subscription.DateEndString = &dateEndString.String
		}
		if dateCreated.Valid {
			subscription.DateCreated = &dateCreated.Int64
		}
		if dateCreatedString.Valid {
			subscription.DateCreatedString = &dateCreatedString.String
		}
		if dateModified.Valid {
			subscription.DateModified = &dateModified.Int64
		}
		if dateModifiedString.Valid {
			subscription.DateModifiedString = &dateModifiedString.String
		}

		// Parse client JSON
		if len(clientJSON) > 0 {
			var clientData map[string]any
			if err := json.Unmarshal(clientJSON, &clientData); err == nil {
				clientJSONBytes, _ := json.Marshal(clientData)
				var client clientpb.Client
				if err := protojson.Unmarshal(clientJSONBytes, &client); err == nil {
					subscription.Client = &client
				}
			}
		}

		// Parse price_plan JSON
		if len(pricePlanJSON) > 0 {
			var pricePlanData map[string]any
			if err := json.Unmarshal(pricePlanJSON, &pricePlanData); err == nil {
				pricePlanJSONBytes, _ := json.Marshal(pricePlanData)
				var pricePlan priceplanpb.PricePlan
				if err := protojson.Unmarshal(pricePlanJSONBytes, &pricePlan); err == nil {
					subscription.PricePlan = &pricePlan
				}
			}
		}

		subscriptions = append(subscriptions, subscription)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subscription rows: %w", err)
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

	return &subscriptionpb.GetSubscriptionListPageDataResponse{
		Success:          true,
		SubscriptionList: subscriptions,
		Pagination:       paginationResponse,
	}, nil
}

// GetSubscriptionItemPageData retrieves a single subscription with all related client, user, and plan data expanded
// This method uses CTEs (Common Table Expressions) to load all related data in a single query
// TODO: Add unit tests for GetSubscriptionItemPageData
func (r *PostgresSubscriptionRepository) GetSubscriptionItemPageData(ctx context.Context, req *subscriptionpb.GetSubscriptionItemPageDataRequest) (*subscriptionpb.GetSubscriptionItemPageDataResponse, error) {
	if req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_id ON subscription(id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_client_id ON subscription(client_id);
	// PERFORMANCE INDEX REQUIRED: CREATE INDEX idx_subscription_price_plan_id ON subscription(price_plan_id);

	// Build CTE query to fetch subscription with all related data
	query := `
		SELECT
			s.id,
			s.name,
			s.client_id,
			s.price_plan_id,
			s.date_start,
			s.date_start_string,
			s.date_end,
			s.date_end_string,
			s.active,
			s.date_created,
			s.date_modified,
			jsonb_build_object(
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
			) as client,
			jsonb_build_object(
				'id', pp.id,
				'name', pp.name,
				'description', pp.description,
				'active', pp.active,
				'date_created', pp.date_created,
				'date_modified', pp.date_modified
			) as price_plan
		FROM subscription s
		LEFT JOIN client c ON s.client_id = c.id AND c.active = true
		LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
		LEFT JOIN price_plan pp ON s.price_plan_id = pp.id AND pp.active = true
		WHERE s.id = $1 AND s.active = true
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
		clientID           string
		pricePlanID        string
		dateStart          sql.NullInt64
		dateStartString    sql.NullString
		dateEnd            sql.NullInt64
		dateEndString      sql.NullString
		active             bool
		dateCreated        sql.NullInt64
		dateCreatedString  sql.NullString
		dateModified       sql.NullInt64
		dateModifiedString sql.NullString
		clientJSON         []byte
		pricePlanJSON      []byte
	)

	err := db.GetDB().QueryRowContext(ctx, query, req.SubscriptionId).Scan(
		&id,
		&name,
		&clientID,
		&pricePlanID,
		&dateStart,
		&dateStartString,
		&dateEnd,
		&dateEndString,
		&active,
		&dateCreated,
		&dateModified,
		&clientJSON,
		&pricePlanJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subscription not found with ID: %s", req.SubscriptionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetSubscriptionItemPageData query: %w", err)
	}

	// Build subscription message
	subscription := &subscriptionpb.Subscription{
		Id:          id,
		Name:        name,
		ClientId:    clientID,
		PricePlanId: pricePlanID,
		Active:      active,
	}

	if dateStart.Valid {
		subscription.DateStart = &dateStart.Int64
	}
	if dateStartString.Valid {
		subscription.DateStartString = &dateStartString.String
	}
	if dateEnd.Valid {
		subscription.DateEnd = &dateEnd.Int64
	}
	if dateEndString.Valid {
		subscription.DateEndString = &dateEndString.String
	}
	if dateCreated.Valid {
		subscription.DateCreated = &dateCreated.Int64
	}
	if dateCreatedString.Valid {
		subscription.DateCreatedString = &dateCreatedString.String
	}
	if dateModified.Valid {
		subscription.DateModified = &dateModified.Int64
	}
	if dateModifiedString.Valid {
		subscription.DateModifiedString = &dateModifiedString.String
	}

	// Parse client JSON
	if len(clientJSON) > 0 {
		var clientData map[string]any
		if err := json.Unmarshal(clientJSON, &clientData); err == nil {
			clientJSONBytes, _ := json.Marshal(clientData)
			var client clientpb.Client
			if err := protojson.Unmarshal(clientJSONBytes, &client); err == nil {
				subscription.Client = &client
			}
		}
	}

	// Parse price_plan JSON
	if len(pricePlanJSON) > 0 {
		var pricePlanData map[string]any
		if err := json.Unmarshal(pricePlanJSON, &pricePlanData); err == nil {
			pricePlanJSONBytes, _ := json.Marshal(pricePlanData)
			var pricePlan priceplanpb.PricePlan
			if err := protojson.Unmarshal(pricePlanJSONBytes, &pricePlan); err == nil {
				subscription.PricePlan = &pricePlan
			}
		}
	}

	return &subscriptionpb.GetSubscriptionItemPageDataResponse{
		Success:      true,
		Subscription: subscription,
	}, nil
}

// NewSubscriptionRepository creates a new PostgreSQL subscription repository (old-style constructor)
func NewSubscriptionRepository(db *sql.DB, tableName string) subscriptionpb.SubscriptionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresSubscriptionRepository(dbOps, tableName)
}
