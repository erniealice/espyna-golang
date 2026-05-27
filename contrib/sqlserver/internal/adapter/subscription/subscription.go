//go:build sqlserver

// Package subscription provides the SQL Server adapter for the subscription domain.
//
// Dialect rules applied vs the postgres gold standard:
//   - $N → @pN  (all placeholders)
//   - "ident" → [ident]  (SQL Server bracket quoting)
//   - ILIKE → LIKE  (SQL Server CI collation)
//   - FILTER (WHERE …) → CASE WHEN … END  (no FILTER clause in T-SQL)
//   - CROSS JOIN total_count / COUNT(*) OVER()  (keep window; replace CROSS JOIN)
//   - LIMIT n OFFSET m → ORDER BY … OFFSET m ROWS FETCH NEXT n ROWS ONLY
//   - "user" → [user]  (SQL Server reserved word)
//   - active = true → active = 1  (SQL Server BIT)
//   - jsonb_build_object(…) → FOR JSON PATH subqueries
//   - WHERE workspace_id added wherever the postgres version was missing it
package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	espynactx "github.com/erniealice/espyna-golang/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// subscriptionSortableSQLCols lists SQL column names handled by the CASE WHEN
// sort branches. Any unrecognised column causes a loud error.
var subscriptionSortableSQLCols = []string{
	"name",
	"date_created",
	"date_time_start",
	"date_time_end",
	"client_name",
}

// subscriptionViewToSQLColMap translates view-facing sort keys to SQL column names.
var subscriptionViewToSQLColMap = map[string]string{
	"date_start": "date_time_start",
	"date_end":   "date_time_end",
	"client":     "client_name",
}

// SQLServerSubscriptionRepository implements subscription CRUD operations using SQL Server.
type SQLServerSubscriptionRepository struct {
	subscriptionpb.UnimplementedSubscriptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Subscription, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver subscription repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerSubscriptionRepository(dbOps, tableName), nil
	})
}

// NewSQLServerSubscriptionRepository creates a new SQL Server subscription repository.
func NewSQLServerSubscriptionRepository(dbOps interfaces.DatabaseOperation, tableName string) subscriptionpb.SubscriptionDomainServiceServer {
	if tableName == "" {
		tableName = "subscription"
	}
	var db *sql.DB
	if ep, ok := dbOps.(interface {
		GetExecutor(ctx context.Context) interfaces.DBExecutor
	}); ok {
		_ = ep // wired via GetExecutor; direct db extracted below
	}
	if ep, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ep.GetDB()
	}
	return &SQLServerSubscriptionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSubscription creates a new subscription using common SQL Server operations.
func (r *SQLServerSubscriptionRepository) CreateSubscription(ctx context.Context, req *subscriptionpb.CreateSubscriptionRequest) (*subscriptionpb.CreateSubscriptionResponse, error) {
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

// ReadSubscription retrieves a subscription using common SQL Server operations.
func (r *SQLServerSubscriptionRepository) ReadSubscription(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription: %w", err)
	}

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

// UpdateSubscription updates a subscription using common SQL Server operations.
func (r *SQLServerSubscriptionRepository) UpdateSubscription(ctx context.Context, req *subscriptionpb.UpdateSubscriptionRequest) (*subscriptionpb.UpdateSubscriptionResponse, error) {
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

// DeleteSubscription deletes a subscription using common SQL Server operations (soft delete).
func (r *SQLServerSubscriptionRepository) DeleteSubscription(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
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

// ListSubscriptions lists subscriptions using common SQL Server operations.
func (r *SQLServerSubscriptionRepository) ListSubscriptions(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// GetSubscriptionListPageData retrieves a paginated, filtered, sorted, and searchable list
// of subscriptions with client and plan relationships.
//
// SQL Server differences from the postgres gold standard:
//   - $N → @pN placeholders throughout.
//   - "user" → [user] (T-SQL reserved word).
//   - ILIKE → LIKE (SQL Server CI collation).
//   - CROSS JOIN total_count → COUNT(*) OVER () window function (removes the CROSS JOIN CTE).
//   - LIMIT n OFFSET m → ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
//   - jsonb_build_object(…) → FOR JSON PATH subselects.
//   - active = true → active = 1.
//   - Explicit workspace_id predicate on every CTE (A1 guard).
func (r *SQLServerSubscriptionRepository) GetSubscriptionListPageData(ctx context.Context, req *subscriptionpb.GetSubscriptionListPageDataRequest) (*subscriptionpb.GetSubscriptionListPageDataResponse, error) {
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

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	activeVal := 1
	if !activeFilter {
		activeVal = 0
	}

	// SQL Server translation:
	// - ILIKE → LIKE
	// - $N → @pN
	// - active = $7 → active = @p7 (bit comparison)
	// - CROSS JOIN total_count → COUNT(*) OVER () on the final SELECT
	// - LIMIT/OFFSET → OFFSET/FETCH NEXT
	// - jsonb_build_object → FOR JSON PATH subselects
	// - "user" → [user]
	// - workspace_id predicate on search_filtered CTE
	//
	// Sort CASE WHEN branches: SQL Server evaluates CASE WHEN @p4 = 'name' ...
	// the same way postgres does.
	query := `
		WITH
		search_filtered AS (
			SELECT s.*
			FROM subscription s
			WHERE s.active = @p7
				AND (@p8 = '' OR s.workspace_id = @p8)
				AND (@p1 = '' OR s.name LIKE @p1)
				AND (@p6 = '' OR s.client_id = @p6)
				AND (@p9 = '' OR s.price_plan_id = @p9)
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
				-- client_name: company name first, fall back to user full name.
				COALESCE(
					NULLIF(c.name, ''),
					NULLIF(LTRIM(RTRIM(COALESCE(u.first_name, '') + ' ' + COALESCE(u.last_name, ''))), '')
				) AS client_name,
				-- client JSON subselect (FOR JSON PATH WITHOUT_ARRAY_WRAPPER)
				(SELECT
					c.id,
					c.user_id,
					c.internal_id,
					c.name,
					c.active,
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', c.date_created) AS date_created,
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', c.date_modified) AS date_modified,
					u.id AS [user.id],
					u.first_name AS [user.first_name],
					u.last_name AS [user.last_name],
					u.email_address AS [user.email_address],
					u.active AS [user.active],
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', u.date_created) AS [user.date_created],
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', u.date_modified) AS [user.date_modified]
				FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS client,
				-- price_plan JSON subselect
				(SELECT
					pp.id,
					pp.plan_id,
					pp.name,
					pp.description,
					pp.active,
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', pp.date_created) AS date_created,
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', pp.date_modified) AS date_modified,
					p.id AS [plan.id],
					p.name AS [plan.name],
					p.description AS [plan.description],
					p.active AS [plan.active],
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', p.date_created) AS [plan.date_created],
					DATEDIFF_BIG(MILLISECOND, '1970-01-01', p.date_modified) AS [plan.date_modified]
				FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS price_plan
			FROM search_filtered sf
			LEFT JOIN client c ON sf.client_id = c.id AND c.active = 1
			LEFT JOIN [user] u ON c.user_id = u.id AND u.active = 1
			LEFT JOIN price_plan pp ON sf.price_plan_id = pp.id AND pp.active = 1
			LEFT JOIN plan p ON pp.plan_id = p.id AND p.active = 1
		),

		sorted AS (
			SELECT *
			FROM enriched
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
			COUNT(*) OVER () AS _total_count
		FROM sorted s
		ORDER BY
			CASE WHEN @p4 = 'name' AND @p5 = 'ASC' THEN s.name END ASC,
			CASE WHEN @p4 = 'name' AND @p5 = 'DESC' THEN s.name END DESC,
			CASE WHEN (@p4 = 'date_created' OR @p4 = '') AND @p5 = 'DESC' THEN s.date_created END DESC,
			CASE WHEN @p4 = 'date_created' AND @p5 = 'ASC' THEN s.date_created END ASC,
			CASE WHEN @p4 = 'date_time_start' AND @p5 = 'ASC' THEN s.date_time_start END ASC,
			CASE WHEN @p4 = 'date_time_start' AND @p5 = 'DESC' THEN s.date_time_start END DESC,
			CASE WHEN @p4 = 'date_time_end' AND @p5 = 'ASC' THEN s.date_time_end END ASC,
			CASE WHEN @p4 = 'date_time_end' AND @p5 = 'DESC' THEN s.date_time_end END DESC,
			CASE WHEN @p4 = 'client_name' AND @p5 = 'ASC' THEN s.client_name END ASC,
			CASE WHEN @p4 = 'client_name' AND @p5 = 'DESC' THEN s.client_name END DESC
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY
	`
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query,
		searchQuery,       // @p1
		limit,             // @p2
		offset,            // @p3
		sortField,         // @p4
		sortDirection,     // @p5
		clientIDFilter,    // @p6
		activeVal,         // @p7
		wsID,              // @p8
		pricePlanIDFilter, // @p9
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

// GetSubscriptionItemPageData retrieves a single subscription with all related data.
//
// SQL Server differences:
//   - $1/$2 → @p1/@p2.
//   - "user" → [user].
//   - LIMIT 1 → SELECT TOP 1 on outer query.
//   - jsonb_build_object → FOR JSON PATH WITHOUT_ARRAY_WRAPPER.
//   - workspace_id predicate enforced (A1 multi-tenancy guard).
func (r *SQLServerSubscriptionRepository) GetSubscriptionItemPageData(ctx context.Context, req *subscriptionpb.GetSubscriptionItemPageDataRequest) (*subscriptionpb.GetSubscriptionItemPageDataResponse, error) {
	if req.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)

	query := `
		SELECT TOP 1
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
			(SELECT
				c.id,
				c.user_id,
				c.internal_id,
				c.name,
				c.active,
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', c.date_created) AS date_created,
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', c.date_modified) AS date_modified,
				u.id AS [user.id],
				u.first_name AS [user.first_name],
				u.last_name AS [user.last_name],
				u.email_address AS [user.email_address],
				u.active AS [user.active],
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', u.date_created) AS [user.date_created],
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', u.date_modified) AS [user.date_modified]
			FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS client,
			(SELECT
				pp.id,
				pp.plan_id,
				pp.name,
				pp.description,
				pp.active,
				pp.billing_kind,
				pp.amount_basis,
				pp.billing_amount,
				pp.billing_currency,
				pp.billing_cycle_value,
				pp.billing_cycle_unit,
				pp.entitled_occurrences,
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', pp.date_created) AS date_created,
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', pp.date_modified) AS date_modified,
				p.id AS [plan.id],
				p.name AS [plan.name],
				p.description AS [plan.description],
				p.active AS [plan.active],
				p.job_template_id AS [plan.job_template_id],
				p.visits_per_cycle AS [plan.visits_per_cycle],
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', p.date_created) AS [plan.date_created],
				DATEDIFF_BIG(MILLISECOND, '1970-01-01', p.date_modified) AS [plan.date_modified]
			FOR JSON PATH, WITHOUT_ARRAY_WRAPPER) AS price_plan
		FROM subscription s
		LEFT JOIN client c ON s.client_id = c.id AND c.active = 1
		LEFT JOIN [user] u ON c.user_id = u.id AND u.active = 1
		LEFT JOIN price_plan pp ON s.price_plan_id = pp.id AND pp.active = 1
		LEFT JOIN plan p ON pp.plan_id = p.id AND p.active = 1
		WHERE s.id = @p1
		  AND (@p2 = '' OR s.workspace_id = @p2)
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.SubscriptionId, wsID)

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

	err := row.Scan(
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
//
// SQL Server differences:
//   - $N → @pN.
//   - active = TRUE → active = 1.
//   - ANY($2) → no array type; use IN with a VALUES subquery (looped binds).
//   - COUNT(*)::int → CAST(COUNT(*) AS INT).
//   - workspace_id guard preserved.
func (r *SQLServerSubscriptionRepository) CountActiveByClientIds(ctx context.Context, req *subscriptionpb.CountActiveByClientIdsRequest) (*subscriptionpb.CountActiveByClientIdsResponse, error) {
	wsID := espynactx.ExtractWorkspaceIDFromContext(ctx)
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)

	clientIDs := req.GetClientIds()

	var (
		rows *sql.Rows
		err  error
	)

	if len(clientIDs) > 0 {
		// Build IN (…) with @pN placeholders for each client ID.
		// @p1 = wsID; @p2..@p(N+1) = client IDs.
		placeholders := make([]string, len(clientIDs))
		args := make([]any, 0, len(clientIDs)+1)
		args = append(args, wsID)
		for i, id := range clientIDs {
			placeholders[i] = fmt.Sprintf("@p%d", i+2)
			args = append(args, id)
		}
		inList := ""
		for i, p := range placeholders {
			if i > 0 {
				inList += ", "
			}
			inList += p
		}
		query := fmt.Sprintf(`
			SELECT client_id, CAST(COUNT(*) AS INT) AS cnt
			  FROM subscription
			 WHERE active = 1
			   AND (@p1 = '' OR workspace_id = @p1)
			   AND client_id IN (%s)
			 GROUP BY client_id`, inList)
		rows, err = exec.QueryContext(ctx, query, args...)
	} else {
		rows, err = exec.QueryContext(ctx, `
			SELECT client_id, CAST(COUNT(*) AS INT) AS cnt
			  FROM subscription
			 WHERE active = 1
			   AND (@p1 = '' OR workspace_id = @p1)
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

// ListSubscriptionsByPricePlan returns subscriptions whose price_plan_id matches
// the request. Delegates to GetSubscriptionListPageData — same pattern as postgres.
func (r *SQLServerSubscriptionRepository) ListSubscriptionsByPricePlan(ctx context.Context, req *subscriptionpb.ListSubscriptionsByPricePlanRequest) (*subscriptionpb.ListSubscriptionsByPricePlanResponse, error) {
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

// NewSubscriptionRepository creates a new SQL Server subscription repository (old-style constructor).
func NewSubscriptionRepository(db *sql.DB, tableName string) subscriptionpb.SubscriptionDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerSubscriptionRepository(dbOps, tableName)
}
