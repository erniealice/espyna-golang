//go:build postgresql

package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
	"github.com/lib/pq"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PostgresSupplierSubscriptionRepository implements supplier_subscription CRUD using PostgreSQL.
// NO Activate / Suspend / Cancel / Renew methods — lifecycle via UpdateSupplierSubscription.
type PostgresSupplierSubscriptionRepository struct {
	suppliersubscriptionpb.UnimplementedSupplierSubscriptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierSubscription, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_subscription repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierSubscriptionRepository(dbOps, tableName), nil
	})
}

// NewPostgresSupplierSubscriptionRepository creates a new PostgreSQL supplier subscription repository.
func NewPostgresSupplierSubscriptionRepository(dbOps interfaces.DatabaseOperation, tableName string) suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_subscription"
	}
	return &PostgresSupplierSubscriptionRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *PostgresSupplierSubscriptionRepository) CreateSupplierSubscription(ctx context.Context, req *suppliersubscriptionpb.CreateSupplierSubscriptionRequest) (*suppliersubscriptionpb.CreateSupplierSubscriptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier subscription data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// Manually inject code field
	if code := req.Data.GetCode(); code != "" {
		data["code"] = code
	}
	// Empty optional FKs must arrive as SQL NULL.
	for _, key := range []string{"locationId", "procurementRequestId"} {
		if v, ok := data[key].(string); ok && v == "" {
			data[key] = nil
		}
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier subscription: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ss := &suppliersubscriptionpb.SupplierSubscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ss); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &suppliersubscriptionpb.CreateSupplierSubscriptionResponse{Data: []*suppliersubscriptionpb.SupplierSubscription{ss}}, nil
}

func (r *PostgresSupplierSubscriptionRepository) ReadSupplierSubscription(ctx context.Context, req *suppliersubscriptionpb.ReadSupplierSubscriptionRequest) (*suppliersubscriptionpb.ReadSupplierSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier subscription ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier subscription: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ss := &suppliersubscriptionpb.SupplierSubscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ss); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &suppliersubscriptionpb.ReadSupplierSubscriptionResponse{Data: []*suppliersubscriptionpb.SupplierSubscription{ss}}, nil
}

func (r *PostgresSupplierSubscriptionRepository) UpdateSupplierSubscription(ctx context.Context, req *suppliersubscriptionpb.UpdateSupplierSubscriptionRequest) (*suppliersubscriptionpb.UpdateSupplierSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier subscription ID is required")
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
	for _, key := range []string{"locationId", "procurementRequestId"} {
		if v, ok := data[key].(string); ok && v == "" {
			data[key] = nil
		}
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier subscription: %w", err)
	}
	postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ss := &suppliersubscriptionpb.SupplierSubscription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ss); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &suppliersubscriptionpb.UpdateSupplierSubscriptionResponse{Data: []*suppliersubscriptionpb.SupplierSubscription{ss}}, nil
}

func (r *PostgresSupplierSubscriptionRepository) DeleteSupplierSubscription(ctx context.Context, req *suppliersubscriptionpb.DeleteSupplierSubscriptionRequest) (*suppliersubscriptionpb.DeleteSupplierSubscriptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier subscription ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier subscription: %w", err)
	}
	return &suppliersubscriptionpb.DeleteSupplierSubscriptionResponse{Success: true}, nil
}

func (r *PostgresSupplierSubscriptionRepository) ListSupplierSubscriptions(ctx context.Context, req *suppliersubscriptionpb.ListSupplierSubscriptionsRequest) (*suppliersubscriptionpb.ListSupplierSubscriptionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier subscriptions: %w", err)
	}
	var items []*suppliersubscriptionpb.SupplierSubscription
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToRFC3339(result, "date_time_start", "date_time_end")
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		ss := &suppliersubscriptionpb.SupplierSubscription{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ss); err != nil {
			continue
		}
		items = append(items, ss)
	}
	return &suppliersubscriptionpb.ListSupplierSubscriptionsResponse{Data: items}, nil
}

// GetSupplierSubscriptionListPageData retrieves a paginated, filtered, sorted, searchable list
// of supplier subscriptions with supplier and cost plan relationships.
func (r *PostgresSupplierSubscriptionRepository) GetSupplierSubscriptionListPageData(ctx context.Context, req *suppliersubscriptionpb.GetSupplierSubscriptionListPageDataRequest) (*suppliersubscriptionpb.GetSupplierSubscriptionListPageDataResponse, error) {
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

	supplierIDFilter := ""
	costPlanIDFilter := ""
	activeFilter := true
	if req.Filters != nil {
		for _, f := range req.Filters.Filters {
			switch f.GetField() {
			case "supplier_id":
				if sf := f.GetStringFilter(); sf != nil {
					supplierIDFilter = sf.GetValue()
				}
			case "cost_plan_id":
				if sf := f.GetStringFilter(); sf != nil {
					costPlanIDFilter = sf.GetValue()
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

	query := `
		WITH search_filtered AS (
			SELECT s.*
			FROM supplier_subscription s
			WHERE s.active = $7
				AND ($8::text = '' OR s.workspace_id = $8::text)
				AND ($1::text = '' OR s.name ILIKE $1)
				AND ($6::text = '' OR s.supplier_id = $6)
				AND ($9::text = '' OR s.cost_plan_id = $9)
		),
		enriched AS (
			SELECT
				sf.id, sf.name, sf.supplier_id, sf.cost_plan_id,
				sf.date_time_start, sf.date_time_end,
				sf.active, sf.date_created, sf.date_modified,
				jsonb_build_object(
					'id', cp.id,
					'name', cp.name,
					'billing_kind', cp.billing_kind,
					'billing_currency', cp.billing_currency,
					'active', cp.active,
					'date_created', (EXTRACT(EPOCH FROM cp.date_created) * 1000)::bigint,
					'date_modified', (EXTRACT(EPOCH FROM cp.date_modified) * 1000)::bigint
				) as cost_plan
			FROM search_filtered sf
			LEFT JOIN cost_plan cp ON sf.cost_plan_id = cp.id AND cp.active = true
		),
		sorted AS (
			SELECT * FROM enriched
			ORDER BY
				CASE WHEN $4 = 'name' AND $5 = 'ASC' THEN name END ASC,
				CASE WHEN $4 = 'name' AND $5 = 'DESC' THEN name END DESC,
				CASE WHEN ($4 = 'date_created' OR $4 = '') AND $5 = 'DESC' THEN date_created END DESC,
				CASE WHEN $4 = 'date_created' AND $5 = 'ASC' THEN date_created END ASC,
				CASE WHEN $4 = 'date_time_start' AND $5 = 'ASC' THEN date_time_start END ASC,
				CASE WHEN $4 = 'date_time_start' AND $5 = 'DESC' THEN date_time_start END DESC,
				CASE WHEN $4 = 'date_time_end' AND $5 = 'ASC' THEN date_time_end END ASC,
				CASE WHEN $4 = 'date_time_end' AND $5 = 'DESC' THEN date_time_end END DESC
		),
		total_count AS (
			SELECT count(*) as total FROM sorted
		)
		SELECT
			s.id, s.name, s.supplier_id, s.cost_plan_id,
			s.date_time_start, s.date_time_end, s.active,
			s.date_created, s.date_modified,
			s.cost_plan,
			tc.total as _total_count
		FROM sorted s
		CROSS JOIN total_count tc
		LIMIT $2 OFFSET $3
	`

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	wsID := identity.Must(ctx).WorkspaceID
	rows, err := db.GetDB().QueryContext(ctx, query,
		searchQuery,      // $1
		limit,            // $2
		offset,           // $3
		sortField,        // $4
		sortDirection,    // $5
		supplierIDFilter, // $6
		activeFilter,     // $7
		wsID,             // $8
		costPlanIDFilter, // $9
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetSupplierSubscriptionListPageData query: %w", err)
	}
	defer rows.Close()

	var subs []*suppliersubscriptionpb.SupplierSubscription
	var totalCount int32

	for rows.Next() {
		var (
			id            string
			name          string
			supplierID    string
			costPlanID    string
			dateTimeStart sql.NullTime
			dateTimeEnd   sql.NullTime
			active        bool
			dateCreated   sql.NullTime
			dateModified  sql.NullTime
			costPlanJSON  []byte
			rowTotalCount int32
		)
		if err := rows.Scan(&id, &name, &supplierID, &costPlanID, &dateTimeStart, &dateTimeEnd, &active, &dateCreated, &dateModified, &costPlanJSON, &rowTotalCount); err != nil {
			return nil, fmt.Errorf("failed to scan supplier subscription row: %w", err)
		}
		totalCount = rowTotalCount
		ss := &suppliersubscriptionpb.SupplierSubscription{
			Id: id, Name: name, SupplierId: supplierID, CostPlanId: costPlanID, Active: active,
		}
		if dateTimeStart.Valid {
			ss.DateTimeStart = timestamppb.New(dateTimeStart.Time)
		}
		if dateTimeEnd.Valid {
			ss.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
		}
		if dateCreated.Valid {
			ts := dateCreated.Time.UnixMilli()
			ss.DateCreated = &ts
		}
		if dateModified.Valid {
			ts := dateModified.Time.UnixMilli()
			ss.DateModified = &ts
		}
		if len(costPlanJSON) > 0 {
			var cpData map[string]any
			if err := json.Unmarshal(costPlanJSON, &cpData); err == nil {
				cpJSONBytes, _ := json.Marshal(cpData)
				var cp costplanpb.CostPlan
				if err := protojson.Unmarshal(cpJSONBytes, &cp); err == nil {
					ss.CostPlan = &cp
				}
			}
		}
		subs = append(subs, ss)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating supplier subscription rows: %w", err)
	}

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

	return &suppliersubscriptionpb.GetSupplierSubscriptionListPageDataResponse{
		Success:                  true,
		SupplierSubscriptionList: subs,
		Pagination:               paginationResponse,
	}, nil
}

// GetSupplierSubscriptionItemPageData retrieves a single supplier subscription with related data.
func (r *PostgresSupplierSubscriptionRepository) GetSupplierSubscriptionItemPageData(ctx context.Context, req *suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataRequest) (*suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataResponse, error) {
	if req.SupplierSubscriptionId == "" {
		return nil, fmt.Errorf("supplier subscription ID is required")
	}

	query := `
		SELECT
			s.id, s.name, s.supplier_id, s.cost_plan_id, s.code,
			s.date_time_start, s.date_time_end, s.active,
			s.date_created, s.date_modified,
			jsonb_build_object(
				'id', cp.id,
				'name', cp.name,
				'billing_kind', cp.billing_kind,
				'billing_currency', cp.billing_currency,
				'billing_amount', cp.billing_amount,
				'amount_basis', cp.amount_basis,
				'billing_cycle_value', cp.billing_cycle_value,
				'billing_cycle_unit', cp.billing_cycle_unit,
				'default_term_value', cp.default_term_value,
				'default_term_unit', cp.default_term_unit,
				'active', cp.active,
				'date_created', (EXTRACT(EPOCH FROM cp.date_created) * 1000)::bigint,
				'date_modified', (EXTRACT(EPOCH FROM cp.date_modified) * 1000)::bigint
			) as cost_plan
		FROM supplier_subscription s
		LEFT JOIN cost_plan cp ON s.cost_plan_id = cp.id AND cp.active = true
		WHERE s.id = $1
		  AND ($2::text = '' OR s.workspace_id = $2::text)
	`

	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	var (
		id            string
		name          string
		supplierID    string
		costPlanID    string
		code          sql.NullString
		dateTimeStart sql.NullTime
		dateTimeEnd   sql.NullTime
		active        bool
		dateCreated   sql.NullTime
		dateModified  sql.NullTime
		costPlanJSON  []byte
	)

	wsID := identity.Must(ctx).WorkspaceID
	err := db.GetDB().QueryRowContext(ctx, query, req.SupplierSubscriptionId, wsID).Scan(
		&id, &name, &supplierID, &costPlanID, &code,
		&dateTimeStart, &dateTimeEnd, &active,
		&dateCreated, &dateModified, &costPlanJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("supplier subscription not found with ID: %s", req.SupplierSubscriptionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetSupplierSubscriptionItemPageData query: %w", err)
	}

	ss := &suppliersubscriptionpb.SupplierSubscription{
		Id: id, Name: name, SupplierId: supplierID, CostPlanId: costPlanID, Active: active,
	}
	if code.Valid && code.String != "" {
		c := code.String
		ss.Code = &c
	}
	if dateTimeStart.Valid {
		ss.DateTimeStart = timestamppb.New(dateTimeStart.Time)
	}
	if dateTimeEnd.Valid {
		ss.DateTimeEnd = timestamppb.New(dateTimeEnd.Time)
	}
	if dateCreated.Valid {
		ts := dateCreated.Time.UnixMilli()
		ss.DateCreated = &ts
	}
	if dateModified.Valid {
		ts := dateModified.Time.UnixMilli()
		ss.DateModified = &ts
	}
	if len(costPlanJSON) > 0 {
		var cpData map[string]any
		if err := json.Unmarshal(costPlanJSON, &cpData); err == nil {
			cpJSONBytes, _ := json.Marshal(cpData)
			var cp costplanpb.CostPlan
			if err := protojson.Unmarshal(cpJSONBytes, &cp); err == nil {
				ss.CostPlan = &cp
			}
		}
	}

	return &suppliersubscriptionpb.GetSupplierSubscriptionItemPageDataResponse{
		Success:              true,
		SupplierSubscription: ss,
	}, nil
}

// CountActiveBySupplierIds counts active supplier subscriptions grouped by supplier ID.
func (r *PostgresSupplierSubscriptionRepository) CountActiveBySupplierIds(ctx context.Context, req *suppliersubscriptionpb.CountActiveBySupplierIdsRequest) (*suppliersubscriptionpb.CountActiveBySupplierIdsResponse, error) {
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	wsID := identity.Must(ctx).WorkspaceID
	var (
		rows *sql.Rows
		err  error
	)

	supplierIDs := req.GetSupplierIds()
	if len(supplierIDs) > 0 {
		rows, err = db.GetDB().QueryContext(ctx,
			`SELECT supplier_id, COUNT(*)::int AS cnt
			   FROM supplier_subscription
			  WHERE active = TRUE
			    AND ($1::text = '' OR workspace_id = $1::text)
			    AND supplier_id = ANY($2)
			  GROUP BY supplier_id`,
			wsID, pq.Array(supplierIDs),
		)
	} else {
		rows, err = db.GetDB().QueryContext(ctx,
			`SELECT supplier_id, COUNT(*)::int AS cnt
			   FROM supplier_subscription
			  WHERE active = TRUE
			    AND ($1::text = '' OR workspace_id = $1::text)
			  GROUP BY supplier_id`,
			wsID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("CountActiveBySupplierIds query failed: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int32)
	for rows.Next() {
		var sid string
		var n int32
		if scanErr := rows.Scan(&sid, &n); scanErr != nil {
			return nil, fmt.Errorf("CountActiveBySupplierIds scan failed: %w", scanErr)
		}
		if sid != "" {
			counts[sid] = n
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CountActiveBySupplierIds rows error: %w", err)
	}

	return &suppliersubscriptionpb.CountActiveBySupplierIdsResponse{Counts: counts}, nil
}

// ListSupplierSubscriptionsByCostPlan returns supplier subscriptions filtered by cost_plan_id.
// Delegates to GetSupplierSubscriptionListPageData to reuse the CTE hydration logic.
func (r *PostgresSupplierSubscriptionRepository) ListSupplierSubscriptionsByCostPlan(ctx context.Context, req *suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanRequest) (*suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanResponse, error) {
	if req == nil || req.CostPlanId == "" {
		return nil, fmt.Errorf("cost_plan_id is required")
	}

	activeOnly := true
	if req.ActiveOnly != nil {
		activeOnly = *req.ActiveOnly
	}

	filters := []*commonpb.TypedFilter{{
		Field: "cost_plan_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    req.CostPlanId,
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

	pageReq := &suppliersubscriptionpb.GetSupplierSubscriptionListPageDataRequest{
		Filters:    &commonpb.FilterRequest{Filters: filters},
		Pagination: req.Pagination,
		Sort:       req.Sort,
	}

	pageResp, err := r.GetSupplierSubscriptionListPageData(ctx, pageReq)
	if err != nil {
		return nil, err
	}

	return &suppliersubscriptionpb.ListSupplierSubscriptionsByCostPlanResponse{
		SupplierSubscriptionList: pageResp.GetSupplierSubscriptionList(),
		Pagination:               pageResp.GetPagination(),
		Success:                  pageResp.GetSuccess(),
		Error:                    pageResp.Error,
	}, nil
}
