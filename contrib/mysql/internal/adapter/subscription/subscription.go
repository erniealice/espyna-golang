//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/erniealice/espyna-golang/shared/identity"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// subscriptionSortableSQLCols lists the SQL column names that are allowed for
// ORDER BY. Any unrecognised column triggers a loud error instead of silently
// producing no ORDER BY.
var subscriptionSortableSQLCols = []string{
	"name",
	"date_created",
	"date_time_start",
	"date_time_end",
	"client_name",
}

// subscriptionViewToSQLColMap translates view-facing sort column keys to the
// SQL column names used in the enriched CTE.
var subscriptionViewToSQLColMap = map[string]string{
	"date_start": "date_time_start",
	"date_end":   "date_time_end",
	"client":     "client_name",
}

// MySQLSubscriptionRepository implements subscription CRUD operations using MySQL 8.0+.
type MySQLSubscriptionRepository struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Subscription, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql subscription repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLSubscriptionRepository(dbOps, tableName), nil
	})
}

// NewMySQLSubscriptionRepository creates a new MySQL subscription repository.
func NewMySQLSubscriptionRepository(dbOps interfaces.DatabaseOperation, tableName string) subscriptionpb.SubscriptionDomainServiceServer {
	if tableName == "" {
		tableName = "subscription"
	}
	return &MySQLSubscriptionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateSubscription creates a new subscription using common MySQL operations.
func (r *MySQLSubscriptionRepository) CreateSubscription(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	if code := req.Data.GetCode(); code != "" {
		data["code"] = code
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

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

// ReadSubscription retrieves a subscription using common MySQL operations.
func (r *MySQLSubscriptionRepository) ReadSubscription(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription: %w", err)
	}

	mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

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

// UpdateSubscription updates a subscription using common MySQL operations.
func (r *MySQLSubscriptionRepository) UpdateSubscription(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	if code := req.Data.GetCode(); code != "" {
		data["code"] = code
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")

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

// DeleteSubscription deletes a subscription using common MySQL operations (soft delete).
func (r *MySQLSubscriptionRepository) DeleteSubscription(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription: %w", err)
	}

	return &subscriptionpb.DeleteSubscriptionResponse{
		Success: true,
	}, nil
}

// ListSubscriptions lists subscriptions using common MySQL operations.
func (r *MySQLSubscriptionRepository) ListSubscriptions(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var subscriptions []*subscriptionpb.Subscription
	for _, result := range listResult.Data {
		mysqlCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		subscription := &subscriptionpb.Subscription{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, subscription); err != nil {
			continue
		}
		subscriptions = append(subscriptions, subscription)
	}

	return &subscriptionpb.ListSubscriptionsResponse{
		Data: subscriptions,
	}, nil
}

// GetSubscriptionListPageData retrieves a paginated, filtered, sorted, and
// searchable list of subscriptions with client and plan relationships.
//
// Dialect translation from postgres gold standard:
//   - $N → ? (MySQL positional placeholders, args in same left-to-right order)
//   - "user" → `user` (backtick-quoted reserved word)
//   - ILIKE → LIKE (MySQL ci collation handles case-insensitivity)
//   - jsonb_build_object → JSON_OBJECT
//   - CROSS JOIN total_count → COUNT(*) OVER () window function (MySQL 8.0+ supported)
//   - active = true → active = 1 (MySQL TINYINT(1) boolean)
//   - WHERE s.workspace_id = ? enforced for multi-tenancy
func (r *MySQLSubscriptionRepository) GetSubscriptionListPageData(ctx context.Context, req *subscriptionpb.GetSubscriptionListPageDataRequest) (*subscriptionpb.GetSubscriptionListPageDataResponse, error) {
	limit := int32(20)
	page := int32(1)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
		if req.Pagination.GetOffset() != nil {
			page = req.Pagination.GetOffset().Page
			if page < 1 {
				page = 1
			}
		}
	}
	offset := (page - 1) * limit

	searchQuery := ""
	if req.Search != nil && req.Search.Query != "" {
		searchQuery = "%" + req.Search.Query + "%"
	}

	clientIDFilter := ""
	pricePlanIDFilter := ""
	activeFilter := true
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
				}
			}
		}
	}

	sortField := "date_created"
	sortDirection := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortDirection = "DESC"
		} else {
			sortDirection = "ASC"
		}
	}

	if mapped, ok := subscriptionViewToSQLColMap[sortField]; ok {
		sortField = mapped
	}

	if sortField != "" && !slices.Contains(subscriptionSortableSQLCols, sortField) {
		return nil, fmt.Errorf("invalid sort column %q for subscription list (allowed SQL cols: %v)", sortField, subscriptionSortableSQLCols)
	}

	wsID := identity.Must(ctx).WorkspaceID

	// Dialect: $N → ?, ILIKE → LIKE, "user" → `user`, active = true → active = 1,
	// CROSS JOIN total_count → COUNT(*) OVER (), jsonb_build_object → JSON_OBJECT,
	// WHERE workspace_id added.
	//
	// Sort is handled inline via CASE WHEN (positional ? not allowed in ORDER BY),
	// which mirrors the postgres gold pattern exactly.
	query := fmt.Sprintf(`
		WITH
		search_filtered AS (
			SELECT s.*
			FROM subscription s
			WHERE s.active = ?
				AND (? = '' OR s.workspace_id = ?)
				AND (? = '' OR s.name LIKE ?)
				AND (? = '' OR s.client_id = ?)
				AND (? = '' OR s.price_plan_id = ?)
		),
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
				COALESCE(
					NULLIF(c.name, ''),
					NULLIF(TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))), '')
				) AS client_name,
				JSON_OBJECT(
					'id', c.id,
					'user_id', c.user_id,
					'internal_id', c.internal_id,
					'name', c.name,
					'active', c.active,
					'date_created', UNIX_TIMESTAMP(c.date_created) * 1000,
					'date_modified', UNIX_TIMESTAMP(c.date_modified) * 1000,
					'user', JSON_OBJECT(
						'id', u.id,
						'first_name', u.first_name,
						'last_name', u.last_name,
						'email_address', u.email_address,
						'active', u.active,
						'date_created', UNIX_TIMESTAMP(u.date_created) * 1000,
						'date_modified', UNIX_TIMESTAMP(u.date_modified) * 1000
					)
				) as client,
				JSON_OBJECT(
					'id', pp.id,
					'plan_id', pp.plan_id,
					'name', pp.name,
					'description', pp.description,
					'active', pp.active,
					'date_created', UNIX_TIMESTAMP(pp.date_created) * 1000,
					'date_modified', UNIX_TIMESTAMP(pp.date_modified) * 1000,
					'plan', JSON_OBJECT(
						'id', p.id,
						'name', p.name,
						'description', p.description,
						'active', p.active,
						'date_created', UNIX_TIMESTAMP(p.date_created) * 1000,
						'date_modified', UNIX_TIMESTAMP(p.date_modified) * 1000
					)
				) as price_plan,
				COUNT(*) OVER () AS _total_count
			FROM search_filtered sf
			LEFT JOIN client c ON sf.client_id = c.id AND c.active = 1
			LEFT JOIN `+"`user`"+` u ON c.user_id = u.id AND u.active = 1
			LEFT JOIN price_plan pp ON sf.price_plan_id = pp.id AND pp.active = 1
			LEFT JOIN plan p ON pp.plan_id = p.id AND p.active = 1
		),
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN '%s' = 'name' AND '%s' = 'ASC' THEN name END ASC,
				CASE WHEN '%s' = 'name' AND '%s' = 'DESC' THEN name END DESC,
				CASE WHEN ('%s' = 'date_created' OR '%s' = '') AND '%s' = 'DESC' THEN date_created END DESC,
				CASE WHEN '%s' = 'date_created' AND '%s' = 'ASC' THEN date_created END ASC,
				CASE WHEN '%s' = 'date_time_start' AND '%s' = 'ASC' THEN date_time_start END ASC,
				CASE WHEN '%s' = 'date_time_start' AND '%s' = 'DESC' THEN date_time_start END DESC,
				CASE WHEN '%s' = 'date_time_end' AND '%s' = 'ASC' THEN date_time_end END ASC,
				CASE WHEN '%s' = 'date_time_end' AND '%s' = 'DESC' THEN date_time_end END DESC,
				CASE WHEN '%s' = 'client_name' AND '%s' = 'ASC' THEN client_name END ASC,
				CASE WHEN '%s' = 'client_name' AND '%s' = 'DESC' THEN client_name END DESC
		)
		SELECT
			s.id,
			s.name,
			s.client_id,
			s.price_plan_id,
			s.date_time_start,
			s.date_time_end,
			s.active,
			s.date_created,
			s.date_modified,
			s.client,
			s.price_plan,
			s._total_count
		FROM sorted s
		LIMIT ? OFFSET ?
	`,
		// sort field/dir interpolated into CASE WHEN (safe: whitelist-validated above)
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
		sortField, sortDirection,
	)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query,
		activeFilter, // s.active = ?
		wsID, wsID,   // workspace_id check: (? = '' OR s.workspace_id = ?)
		searchQuery, searchQuery, // name LIKE
		clientIDFilter, clientIDFilter, // client_id
		pricePlanIDFilter, pricePlanIDFilter, // price_plan_id
		limit, offset,
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

	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	return &subscriptionpb.GetSubscriptionListPageDataResponse{
		Success:          true,
		SubscriptionList: subscriptions,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalCount,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// GetSubscriptionItemPageData retrieves a single subscription with all related
// client, user, and plan data expanded.
//
// Dialect: $1/$2 → ?, "user" → `user`, jsonb_build_object → JSON_OBJECT,
// UNIX_TIMESTAMP replaces EXTRACT(EPOCH FROM ...) * 1000,
// WHERE workspace_id = ? added (missing from postgres gold — added here per brief).
func (r *MySQLSubscriptionRepository) GetSubscriptionItemPageData(ctx context.Context, req *subscriptionpb.GetSubscriptionItemPageDataRequest) (*subscriptionpb.GetSubscriptionItemPageDataResponse, error) {
	if req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	wsID := identity.Must(ctx).WorkspaceID

	// Dialect changes vs postgres:
	//   - $1,$2 → ? (positional)
	//   - "user" → `user`
	//   - jsonb_build_object → JSON_OBJECT
	//   - EXTRACT(EPOCH FROM x) * 1000 → UNIX_TIMESTAMP(x) * 1000
	//   - active = true → active = 1
	//   - WHERE s.workspace_id = ? added (postgres was missing it)
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
			JSON_OBJECT(
				'id', c.id,
				'user_id', c.user_id,
				'internal_id', c.internal_id,
				'name', c.name,
				'active', c.active,
				'date_created', UNIX_TIMESTAMP(c.date_created) * 1000,
				'date_modified', UNIX_TIMESTAMP(c.date_modified) * 1000,
				'user', JSON_OBJECT(
					'id', u.id,
					'first_name', u.first_name,
					'last_name', u.last_name,
					'email_address', u.email_address,
					'active', u.active,
					'date_created', UNIX_TIMESTAMP(u.date_created) * 1000,
					'date_modified', UNIX_TIMESTAMP(u.date_modified) * 1000
				)
			) as client,
			JSON_OBJECT(
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
				'date_created', UNIX_TIMESTAMP(pp.date_created) * 1000,
				'date_modified', UNIX_TIMESTAMP(pp.date_modified) * 1000,
				'plan', JSON_OBJECT(
					'id', p.id,
					'name', p.name,
					'description', p.description,
					'active', p.active,
					'job_template_id', p.job_template_id,
					'visits_per_cycle', p.visits_per_cycle,
					'date_created', UNIX_TIMESTAMP(p.date_created) * 1000,
					'date_modified', UNIX_TIMESTAMP(p.date_modified) * 1000
				)
			) as price_plan
		FROM subscription s
		LEFT JOIN client c ON s.client_id = c.id AND c.active = 1
		LEFT JOIN ` + "`user`" + ` u ON c.user_id = u.id AND u.active = 1
		LEFT JOIN price_plan pp ON s.price_plan_id = pp.id AND pp.active = 1
		LEFT JOIN plan p ON pp.plan_id = p.id AND p.active = 1
		WHERE s.id = ?
		  AND (? = '' OR s.workspace_id = ?)
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

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

	// Arg order: subscriptionId, wsID (empty-OR check), wsID.
	err := exec.QueryRowContext(ctx, query, req.SubscriptionId, wsID, wsID).Scan(
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
// Dialect: pq.Array → manual IN clause with ? placeholders.
func (r *MySQLSubscriptionRepository) CountActiveByClientIds(ctx context.Context, req *subscriptionpb.CountActiveByClientIdsRequest) (*subscriptionpb.CountActiveByClientIdsResponse, error) {
	wsID := identity.Must(ctx).WorkspaceID
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	var (
		sqlRows *sql.Rows
		err     error
	)

	clientIDs := req.GetClientIds()
	if len(clientIDs) > 0 {
		placeholders := strings.Repeat("?,", len(clientIDs))
		placeholders = placeholders[:len(placeholders)-1] // trim trailing comma
		args := []any{wsID, wsID}
		for _, id := range clientIDs {
			args = append(args, id)
		}
		sqlRows, err = exec.QueryContext(ctx,
			`SELECT client_id, COUNT(*) AS cnt
			   FROM subscription
			  WHERE active = 1
			    AND (? = '' OR workspace_id = ?)
			    AND client_id IN (`+placeholders+`)
			  GROUP BY client_id`,
			args...,
		)
	} else {
		sqlRows, err = exec.QueryContext(ctx,
			`SELECT client_id, COUNT(*) AS cnt
			   FROM subscription
			  WHERE active = 1
			    AND (? = '' OR workspace_id = ?)
			  GROUP BY client_id`,
			wsID, wsID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("CountActiveByClientIds query failed: %w", err)
	}
	defer sqlRows.Close()

	counts := make(map[string]int32)
	for sqlRows.Next() {
		var cid string
		var n int32
		if scanErr := sqlRows.Scan(&cid, &n); scanErr != nil {
			return nil, fmt.Errorf("CountActiveByClientIds scan failed: %w", scanErr)
		}
		if cid != "" {
			counts[cid] = n
		}
	}
	if err := sqlRows.Err(); err != nil {
		return nil, fmt.Errorf("CountActiveByClientIds rows error: %w", err)
	}

	return &subscriptionpb.CountActiveByClientIdsResponse{Counts: counts}, nil
}

// ListSubscriptionsByPricePlan returns the subscriptions whose price_plan_id
// matches the request, delegating to GetSubscriptionListPageData.
func (r *MySQLSubscriptionRepository) ListSubscriptionsByPricePlan(ctx context.Context, req *subscriptionpb.ListSubscriptionsByPricePlanRequest) (*subscriptionpb.ListSubscriptionsByPricePlanResponse, error) {
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

// NewSubscriptionRepository creates a new MySQL subscription repository (old-style constructor).
func NewSubscriptionRepository(db *sql.DB, tableName string) subscriptionpb.SubscriptionDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLSubscriptionRepository(dbOps, tableName)
}
