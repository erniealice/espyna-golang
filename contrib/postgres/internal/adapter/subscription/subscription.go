//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"github.com/lib/pq"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// subscriptionSortableSQLCols lists the SQL column names that the sorted CTE
// handles via CASE WHEN branches. These are SQL-side names (after ColMap
// translation). Any unrecognised column triggers a loud error instead of
// silently producing no ORDER BY.
var subscriptionSortableSQLCols = []string{
	"name",
	"date_created",
	"date_time_start",
	"date_time_end",
	"client_name",
}

// subscriptionViewToSQLColMap translates view-facing sort column keys (as
// sent by the browser via ParseTableParamsFromSpec) to the SQL column names
// used in the enriched CTE. Columns absent from the map pass through unchanged.
var subscriptionViewToSQLColMap = map[string]string{
	"date_start": "date_time_start",
	"date_end":   "date_time_end",
	"client":     "client_name",
}

// PostgresSubscriptionRepository implements subscription CRUD operations using PostgreSQL
type PostgresSubscriptionRepository struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Subscription, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
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

	// Manually inject code field (proto descriptor may not include it yet)
	if code := req.Data.GetCode(); code != "" {
		data["code"] = code
	}

	// Create document using common operations.
	// date_time_start / date_time_end arrive as RFC3339 strings (protojson
	// representation of google.protobuf.Timestamp). PostgreSQL TIMESTAMPTZ
	// accepts that directly, so no manual conversion is needed.
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// date_time_start / date_time_end come back as int64 unix-millis from
	// normalizeValue; convert to RFC3339 so protojson can decode the Timestamp.
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscription := &subscriptionpb.Subscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscription); err != nil {
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

	// Same date_time_start/date_time_end conversion as CreateSubscription — see comment there.
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscription := &subscriptionpb.Subscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscription); err != nil {
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

	// Manually inject code field (see CreateSubscription comment)
	if code := req.Data.GetCode(); code != "" {
		data["code"] = code
	}

	// Update document using common operations.
	// date_time_start / date_time_end arrive as RFC3339 strings (protojson
	// representation of google.protobuf.Timestamp). PostgreSQL TIMESTAMPTZ
	// accepts that directly, so no manual conversion is needed.
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// date_time_start / date_time_end come back as int64 unix-millis from
	// normalizeValue; convert to RFC3339 so protojson can decode the Timestamp.
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	subscription := &subscriptionpb.Subscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscription); err != nil {
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
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	// Convert results to protobuf slice using protojson.
	// DenormalizeKeys converts snake_case DB column names to camelCase
	// so protojson can map them to the correct protobuf fields.
	var subscriptions []*subscriptionpb.Subscription
	for _, result := range listResult.Data {
		// Same date_time_start/date_time_end conversion as CreateSubscription — see comment there.
		postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		subscription := &subscriptionpb.Subscription{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscription); err != nil {
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

	// Extract client_id, price_plan_id, and active filters
	clientIDFilter := ""
	pricePlanIDFilter := ""
	activeFilter := true // default: active only
	hasActiveFilter := false
	if req.Filters != nil {
		for _, f := range req.Filters.Filters {
			switch f.GetField() {
			case "client_id":
				if sf := f.GetStringFilter(); sf != nil {
					clientIDFilter = sf.GetValue()
				}
			case "price_plan_id":
				if sf := f.GetStringFilter(); sf != nil {
					pricePlanIDFilter = sf.GetValue()
				}
			case "s.active", "active":
				if bf := f.GetBooleanFilter(); bf != nil {
					activeFilter = bf.GetValue()
					hasActiveFilter = true
				}
			}
		}
	}
	_ = hasActiveFilter

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

	// Translate view-facing column key to SQL column name via ColMap.
	if mapped, ok := subscriptionViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	// Loud-failure guard: reject any sort column not handled by the CASE WHEN
	// chain. This defends against direct API callers that bypass the HTTP layer.
	// Empty sortField is allowed (treated as "date_created" default above).
	if sortField != "" && !slices.Contains(subscriptionSortableSQLCols, sortField) {
		return nil, fmt.Errorf("invalid sort column %q for subscription list (allowed SQL cols: %v)", sortField, subscriptionSortableSQLCols)
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
			WHERE s.active = $7
				AND ($8::text = '' OR s.workspace_id = $8::text)
				AND ($1::text = '' OR
					s.name ILIKE $1)
				AND ($6::text = '' OR s.client_id = $6)
				AND ($9::text = '' OR s.price_plan_id = $9)
		),

		-- CTE 2: Join with client, user, price_plan, and plan
		enriched AS (
			SELECT
				sf.id,
				sf.name,
				sf.client_id,
				sf.price_plan_id,
				sf.date_time_start,
				sf.date_time_end,
				sf.active,
				sf.date_created,
				sf.date_modified,
				-- client_name is the top-level sortable column for client sort.
				-- Prefer c.name (company name); fall back to first_name || last_name
				-- for individual clients without a company name.
				COALESCE(
					NULLIF(c.name, ''),
					NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), '')
				) AS client_name,
				jsonb_build_object(
					'id', c.id,
					'user_id', c.user_id,
					'internal_id', c.internal_id,
					'name', c.name,
					'active', c.active,
					'date_created', (EXTRACT(EPOCH FROM c.date_created) * 1000)::bigint,
					'date_modified', (EXTRACT(EPOCH FROM c.date_modified) * 1000)::bigint,
					'user', jsonb_build_object(
						'id', u.id,
						'first_name', u.first_name,
						'last_name', u.last_name,
						'email_address', u.email_address,
						'active', u.active,
						'date_created', (EXTRACT(EPOCH FROM u.date_created) * 1000)::bigint,
						'date_modified', (EXTRACT(EPOCH FROM u.date_modified) * 1000)::bigint
					)
				) as client,
				jsonb_build_object(
					'id', pp.id,
					'plan_id', pp.plan_id,
					'name', pp.name,
					'description', pp.description,
					'active', pp.active,
					'date_created', (EXTRACT(EPOCH FROM pp.date_created) * 1000)::bigint,
					'date_modified', (EXTRACT(EPOCH FROM pp.date_modified) * 1000)::bigint,
					'plan', jsonb_build_object(
						'id', p.id,
						'name', p.name,
						'description', p.description,
						'active', p.active,
						'date_created', (EXTRACT(EPOCH FROM p.date_created) * 1000)::bigint,
						'date_modified', (EXTRACT(EPOCH FROM p.date_modified) * 1000)::bigint
					)
				) as price_plan
			FROM search_filtered sf
			LEFT JOIN client c ON sf.client_id = c.id AND c.active = true
			LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
			LEFT JOIN price_plan pp ON sf.price_plan_id = pp.id AND pp.active = true
			LEFT JOIN plan p ON pp.plan_id = p.id AND p.active = true
		)

		-- Final SELECT with sorting, window count, and pagination.
		-- A10: COUNT(*) OVER () replaces the prior total-count CTE + CROSS JOIN,
		-- computed over the full enriched set before LIMIT/OFFSET. The parameterized
		-- CASE WHEN sort (guarded above by subscriptionSortableSQLCols) moves into
		-- this SELECT (over enriched, which still projects client_name) so the
		-- window count spans every filtered row.
		SELECT
			e.id,
			e.name,
			e.client_id,
			e.price_plan_id,
			e.date_time_start,
			e.date_time_end,
			e.active,
			e.date_created,
			e.date_modified,
			e.client,
			e.price_plan,
			COUNT(*) OVER () as _total_count
		FROM enriched e
		ORDER BY
			CASE WHEN $4 = 'name' AND $5 = 'ASC' THEN e.name END ASC,
			CASE WHEN $4 = 'name' AND $5 = 'DESC' THEN e.name END DESC,
			CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN e.date_created END DESC,
			CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN e.date_created END ASC,
			CASE WHEN $4 = 'date_time_start' AND $5 = 'ASC' THEN e.date_time_start END ASC,
			CASE WHEN $4 = 'date_time_start' AND $5 = 'DESC' THEN e.date_time_start END DESC,
			CASE WHEN $4 = 'date_time_end' AND $5 = 'ASC' THEN e.date_time_end END ASC,
			CASE WHEN $4 = 'date_time_end' AND $5 = 'DESC' THEN e.date_time_end END DESC,
			CASE WHEN $4 = 'client_name' AND $5 = 'ASC' THEN e.client_name END ASC,
			CASE WHEN $4 = 'client_name' AND $5 = 'DESC' THEN e.client_name END DESC
		LIMIT $2 OFFSET $3
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Workspace isolation: this method bypasses the WorkspaceAwareOperations
	// decorator (raw SQL via db.GetDB()), so we extract workspace_id from
	// context and filter explicitly. Empty wsID = service-to-service call.
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	// Execute query
	rows, err := db.GetDB().QueryContext(ctx, query,
		searchQuery,       // $1
		limit,             // $2
		offset,            // $3
		sortField,         // $4
		sortDirection,     // $5
		clientIDFilter,    // $6
		activeFilter,      // $7
		wsID,              // $8
		pricePlanIDFilter, // $9
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetSubscriptionListPageData query: %w", err)
	}
	defer rows.Close()

	var subscriptions []*subscriptionpb.Subscription
	var totalCount int32

	for rows.Next() {
		var (
			id            string
			name          string
			clientID      string
			pricePlanID   string
			dateTimeStart sql.NullTime
			dateTimeEnd   sql.NullTime
			active        bool
			dateCreated   sql.NullTime
			dateModified  sql.NullTime
			clientJSON    []byte
			pricePlanJSON []byte
			rowTotalCount int32
		)

		err := rows.Scan(
			&id,
			&name,
			&clientID,
			&pricePlanID,
			&dateTimeStart,
			&dateTimeEnd,
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

		if dateTimeStart.Valid {
			subscription.DateTimeStart = timestamppb.New(dateTimeStart.Time)
		}
		if dateTimeEnd.Valid {
			subscription.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.UnixMilli()
			subscription.DateCreated = &ts
		}
		if dateModified.Valid {
			ts := dateModified.Time.UnixMilli()
			subscription.DateModified = &ts
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
			s.code,
			s.date_time_start,
			s.date_time_end,
			s.active,
			s.date_created,
			s.date_modified,
			jsonb_build_object(
				'id', c.id,
				'user_id', c.user_id,
				'internal_id', c.internal_id,
				'name', c.name,
				'active', c.active,
				'date_created', (EXTRACT(EPOCH FROM c.date_created) * 1000)::bigint,
				'date_modified', (EXTRACT(EPOCH FROM c.date_modified) * 1000)::bigint,
				'user', jsonb_build_object(
					'id', u.id,
					'first_name', u.first_name,
					'last_name', u.last_name,
					'email_address', u.email_address,
					'active', u.active,
					'date_created', (EXTRACT(EPOCH FROM u.date_created) * 1000)::bigint,
					'date_modified', (EXTRACT(EPOCH FROM u.date_modified) * 1000)::bigint
				)
			) as client,
			jsonb_build_object(
				'id', pp.id,
				'plan_id', pp.plan_id,
				'name', pp.name,
				'description', pp.description,
				'active', pp.active,
				'billing_kind', pp.billing_kind,
				'amount_basis', pp.amount_basis,
				'billing_amount', pp.billing_amount,
				'billing_currency', pp.billing_currency,
				'billing_cycle_value', pp.billing_cycle_value,
				'billing_cycle_unit', pp.billing_cycle_unit,
				'entitled_occurrences', pp.entitled_occurrences,
				'date_created', (EXTRACT(EPOCH FROM pp.date_created) * 1000)::bigint,
				'date_modified', (EXTRACT(EPOCH FROM pp.date_modified) * 1000)::bigint,
				'plan', jsonb_build_object(
					'id', p.id,
					'name', p.name,
					'description', p.description,
					'active', p.active,
					'job_template_id', p.job_template_id,
					'visits_per_cycle', p.visits_per_cycle,
					'date_created', (EXTRACT(EPOCH FROM p.date_created) * 1000)::bigint,
					'date_modified', (EXTRACT(EPOCH FROM p.date_modified) * 1000)::bigint
				)
			) as price_plan
		FROM subscription s
		LEFT JOIN client c ON s.client_id = c.id AND c.active = true
		LEFT JOIN "user" u ON c.user_id = u.id AND u.active = true
		LEFT JOIN price_plan pp ON s.price_plan_id = pp.id AND pp.active = true
		LEFT JOIN plan p ON pp.plan_id = p.id AND p.active = true
		-- Active filter intentionally omitted: detail-page lookups must work
		-- for deactivated subscriptions too (so operators can review/restore).
		-- Active scoping belongs at the LIST level, not the by-id lookup.
		WHERE s.id = $1
		  AND ($2::text = '' OR s.workspace_id = $2::text)
	`

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	// Execute query
	var (
		id            string
		name          string
		clientID      string
		pricePlanID   string
		code          sql.NullString
		dateTimeStart sql.NullTime
		dateTimeEnd   sql.NullTime
		active        bool
		dateCreated   sql.NullTime
		dateModified  sql.NullTime
		clientJSON    []byte
		pricePlanJSON []byte
	)

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	err := db.GetDB().QueryRowContext(ctx, query, req.SubscriptionId, wsID).Scan(
		&id,
		&name,
		&clientID,
		&pricePlanID,
		&code,
		&dateTimeStart,
		&dateTimeEnd,
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
	if code.Valid && code.String != "" {
		c := code.String
		subscription.Code = &c
	}

	if dateTimeStart.Valid {
		subscription.DateTimeStart = timestamppb.New(dateTimeStart.Time)
	}
	if dateTimeEnd.Valid {
		subscription.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
	}
	if dateCreated.Valid {
		ts := dateCreated.Time.UnixMilli()
		subscription.DateCreated = &ts
	}
	if dateModified.Valid {
		ts := dateModified.Time.UnixMilli()
		subscription.DateModified = &ts
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

// CountActiveByClientIds counts active subscriptions grouped by client ID.
// If req.ClientIds is non-empty the count is restricted to those clients.
// Workspace isolation is applied automatically from context.
func (r *PostgresSubscriptionRepository) CountActiveByClientIds(ctx context.Context, req *subscriptionpb.CountActiveByClientIdsRequest) (*subscriptionpb.CountActiveByClientIdsResponse, error) {
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	var (
		rows *sql.Rows
		err  error
	)

	clientIDs := req.GetClientIds()
	if len(clientIDs) > 0 {
		rows, err = db.GetDB().QueryContext(ctx,
			`SELECT client_id, COUNT(*)::int AS cnt
			   FROM subscription
			  WHERE active = TRUE
			    AND ($1::text = '' OR workspace_id = $1::text)
			    AND client_id = ANY($2)
			  GROUP BY client_id`,
			wsID, pq.Array(clientIDs),
		)
	} else {
		rows, err = db.GetDB().QueryContext(ctx,
			`SELECT client_id, COUNT(*)::int AS cnt
			   FROM subscription
			  WHERE active = TRUE
			    AND ($1::text = '' OR workspace_id = $1::text)
			  GROUP BY client_id`,
			wsID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("CountActiveByClientIds query failed: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int32)
	for rows.Next() {
		var cid string
		var n int32
		if scanErr := rows.Scan(&cid, &n); scanErr != nil {
			return nil, fmt.Errorf("CountActiveByClientIds scan failed: %w", scanErr)
		}
		if cid != "" {
			counts[cid] = n
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CountActiveByClientIds rows error: %w", err)
	}

	return &subscriptionpb.CountActiveByClientIdsResponse{Counts: counts}, nil
}

// ListSubscriptionsByPricePlan returns the subscriptions whose price_plan_id
// matches the request. It is a thin delegator over GetSubscriptionListPageData
// — the existing CTE-based query already JOINs and hydrates Client + PricePlan
// + Plan, so no new SQL is introduced here. Callers (the centymo price-plan
// detail "Engagements" tab) get fully-hydrated rows in a single query.
func (r *PostgresSubscriptionRepository) ListSubscriptionsByPricePlan(ctx context.Context, req *subscriptionpb.ListSubscriptionsByPricePlanRequest) (*subscriptionpb.ListSubscriptionsByPricePlanResponse, error) {
	if req == nil || req.PricePlanId == "" {
		return nil, fmt.Errorf("price_plan_id is required")
	}

	activeOnly := true
	if req.ActiveOnly != nil {
		activeOnly = *req.ActiveOnly
	}

	filters := []*commonpb.TypedFilter{{
		Field: "price_plan_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    req.PricePlanId,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}}
	if activeOnly {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "active",
			FilterType: &commonpb.TypedFilter_BooleanFilter{
				BooleanFilter: &commonpb.BooleanFilter{Value: true},
			},
		})
	}

	pageReq := &subscriptionpb.GetSubscriptionListPageDataRequest{
		Filters:    &commonpb.FilterRequest{Filters: filters},
		Pagination: req.Pagination,
		Sort:       req.Sort,
	}

	pageResp, err := r.GetSubscriptionListPageData(ctx, pageReq)
	if err != nil {
		return nil, err
	}

	return &subscriptionpb.ListSubscriptionsByPricePlanResponse{
		SubscriptionList: pageResp.GetSubscriptionList(),
		Pagination:       pageResp.GetPagination(),
		Success:          pageResp.GetSuccess(),
		Error:            pageResp.Error,
	}, nil
}

// NewSubscriptionRepository creates a new PostgreSQL subscription repository (old-style constructor)
func NewSubscriptionRepository(db *sql.DB, tableName string) subscriptionpb.SubscriptionDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresSubscriptionRepository(dbOps, tableName)
}
