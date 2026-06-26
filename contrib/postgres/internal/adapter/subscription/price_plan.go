//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresPricePlanRepository implements price_plan CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_price_plan_active ON price_plan(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_price_plan_plan_id ON price_plan(plan_id) - Filter by plan
//   - CREATE INDEX idx_price_plan_billing_amount ON price_plan(billing_amount) - Sort/filter by price
//   - CREATE INDEX idx_price_plan_date_created ON price_plan(date_created DESC) - Default sorting
type PostgresPricePlanRepository struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PricePlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres price_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPricePlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresPricePlanRepository creates a new PostgreSQL price plan repository
func NewPostgresPricePlanRepository(dbOps interfaces.DatabaseOperation, tableName string) priceplanpb.PricePlanDomainServiceServer {
	if tableName == "" {
		tableName = "price_plan" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPricePlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePricePlan creates a new price plan using common PostgreSQL operations
func (r *PostgresPricePlanRepository) CreatePricePlan(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price plan data is required")
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

	// Empty optional FK ("" from a blank picker) must arrive at postgres as
	// SQL NULL — both client_id and the schedule-scoped FK columns. Mirrors
	// the UpdatePricePlan normalisation below.
	if v, ok := data["client_id"].(string); ok && v == "" {
		data["client_id"] = nil
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pricePlan := &priceplanpb.PricePlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceplanpb.CreatePricePlanResponse{
		Data: []*priceplanpb.PricePlan{pricePlan},
	}, nil
}

// ReadPricePlan retrieves a price plan using common PostgreSQL operations
func (r *PostgresPricePlanRepository) ReadPricePlan(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price plan: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pricePlan := &priceplanpb.PricePlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceplanpb.ReadPricePlanResponse{
		Data: []*priceplanpb.PricePlan{pricePlan},
	}, nil
}

// UpdatePricePlan updates a price plan using common PostgreSQL operations
func (r *PostgresPricePlanRepository) UpdatePricePlan(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
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

	// Always include active flag — proto3 omits bool=false during JSON marshal,
	// which would silently skip deactivation via the form toggle.
	data["active"] = req.Data.GetActive()

	// Always include client_id so cascades from Plan client_id changes (espyna
	// update_plan §3.2) actually clear the FK column. protojson omits nil
	// optional strings (silent skip) and serializes &"" as "" (FK violation).
	// Force-write: empty/nil → SQL NULL; non-empty → the value.
	if cid := req.Data.GetClientId(); cid == "" {
		data["client_id"] = nil
	} else {
		data["client_id"] = cid
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pricePlan := &priceplanpb.PricePlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &priceplanpb.UpdatePricePlanResponse{
		Data: []*priceplanpb.PricePlan{pricePlan},
	}, nil
}

// DeletePricePlan permanently removes a price plan row. Matches the semantics of
// its parent PriceSchedule: activate/deactivate owns the soft-delete slot, so
// Delete must be a true hard delete. Child ProductPricePlan rows cascade via the
// price_plan_cascade_delete migration.
func (r *PostgresPricePlanRepository) DeletePricePlan(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) (*priceplanpb.DeletePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete price plan: %w", err)
	}

	return &priceplanpb.DeletePricePlanResponse{
		Success: true,
	}, nil
}

var pricePlanSortableSQLCols = []string{
	"id", "active", "plan_id", "name", "description", "billing_amount",
	"billing_currency", "price_schedule_id", "billing_kind", "amount_basis",
	"billing_cycle_value", "billing_cycle_unit", "default_term_value", "default_term_unit",
	"client_id", "date_created", "date_modified",
}

var pricePlanSortSpec = espynahttp.SortSpec{AllowedCols: pricePlanSortableSQLCols}

// ListPricePlans lists price plans using common PostgreSQL operations
func (r *PostgresPricePlanRepository) ListPricePlans(ctx context.Context, req *priceplanpb.ListPricePlansRequest) (*priceplanpb.ListPricePlansResponse, error) {
	if err := espynahttp.ValidateSortColumns(pricePlanSortSpec, req.GetSort(), "price_plan"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list price plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var pricePlans []*priceplanpb.PricePlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		pricePlan := &priceplanpb.PricePlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pricePlan); err != nil {
			// Log error and continue with next item
			continue
		}
		pricePlans = append(pricePlans, pricePlan)
	}

	return &priceplanpb.ListPricePlansResponse{
		Data: pricePlans,
	}, nil
}

// GetPricePlanListPageData retrieves paginated price plan list data with CTE
func (r *PostgresPricePlanRepository) GetPricePlanListPageData(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) (*priceplanpb.GetPricePlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}
	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}
	// A2: route the caller-supplied sort column through the fail-closed
	// whitelist helper instead of interpolating it raw.
	orderBy, err := postgresCore.BuildOrderBy(pricePlanSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for price plan list: %w", err)
	}

	// A1: price_plan has no workspace_id column of its own (verified against the
	// baseline schema); tenancy is inherited through its plan FK (plan carries
	// workspace_id), so the predicate scopes on the joined plan's workspace_id.
	// The query is wrapped in an enriched CTE projecting exactly the price_plan
	// columns (qualified pp.*) so the BuildOrderBy fragment — which emits an
	// unqualified, quoted column — resolves unambiguously against the CTE output
	// and the scan order stays identical. Empty wsID = service-to-service call.
	wsID := identity.Must(ctx).WorkspaceID
	query := `WITH enriched AS (SELECT pp.id, pp.plan_id, pp.billing_amount, pp.billing_currency, pp.name, pp.description, pp.active, pp.date_created, pp.date_modified, pp.price_schedule_id, pp.billing_kind, pp.amount_basis, pp.billing_cycle_value, pp.billing_cycle_unit, pp.default_term_value, pp.default_term_unit FROM price_plan pp LEFT JOIN plan pl ON pp.plan_id = pl.id WHERE pp.active = true AND ($4::text = '' OR pl.workspace_id = $4::text) AND ($1::text IS NULL OR $1::text = '' OR pp.plan_id ILIKE $1 OR pp.billing_currency ILIKE $1)) SELECT * FROM enriched ` + orderBy + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var pricePlans []*priceplanpb.PricePlan
	var totalCount int64
	for rows.Next() {
		var id, planId, billingCurrency string
		var name, description sql.NullString
		var billingAmount int64
		var active bool
		var dateCreated, dateModified time.Time
		var priceScheduleId sql.NullString
		var billingKindRaw, amountBasisRaw sql.NullInt32
		var billingCycleValue, defaultTermValue sql.NullInt32
		var billingCycleUnit, defaultTermUnit sql.NullString
		if err := rows.Scan(&id, &planId, &billingAmount, &billingCurrency, &name, &description, &active, &dateCreated, &dateModified, &priceScheduleId, &billingKindRaw, &amountBasisRaw, &billingCycleValue, &billingCycleUnit, &defaultTermValue, &defaultTermUnit); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount++
		pricePlan := &priceplanpb.PricePlan{Id: id, PlanId: planId, BillingAmount: billingAmount, BillingCurrency: billingCurrency, Active: active}
		if name.Valid {
			pricePlan.Name = &name.String
		}
		if description.Valid {
			pricePlan.Description = &description.String
		}
		if priceScheduleId.Valid && priceScheduleId.String != "" {
			pricePlan.PriceScheduleId = &priceScheduleId.String
		}
		if billingKindRaw.Valid {
			pricePlan.BillingKind = priceplanpb.BillingKind(billingKindRaw.Int32)
		}
		if amountBasisRaw.Valid {
			pricePlan.AmountBasis = priceplanpb.AmountBasis(amountBasisRaw.Int32)
		}
		if billingCycleValue.Valid {
			v := billingCycleValue.Int32
			pricePlan.BillingCycleValue = &v
		}
		if billingCycleUnit.Valid {
			pricePlan.BillingCycleUnit = &billingCycleUnit.String
		}
		if defaultTermValue.Valid {
			v := defaultTermValue.Int32
			pricePlan.DefaultTermValue = &v
		}
		if defaultTermUnit.Valid {
			pricePlan.DefaultTermUnit = &defaultTermUnit.String
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			pricePlan.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			pricePlan.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			pricePlan.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			pricePlan.DateModifiedString = &dmStr
		}
		pricePlans = append(pricePlans, pricePlan)
	}
	return &priceplanpb.GetPricePlanListPageDataResponse{PricePlanList: pricePlans, Success: true}, nil
}

// Note: Pagination removed - not available in current protobuf schema

// GetPricePlanItemPageData retrieves price plan item page data
func (r *PostgresPricePlanRepository) GetPricePlanItemPageData(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) (*priceplanpb.GetPricePlanItemPageDataResponse, error) {
	if req == nil || req.PricePlanId == "" {
		return nil, fmt.Errorf("price plan ID required")
	}
	query := `SELECT id, plan_id, billing_amount, billing_currency, name, description, active, date_created, date_modified, price_schedule_id, billing_kind, amount_basis, billing_cycle_value, billing_cycle_unit, default_term_value, default_term_unit FROM price_plan WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.PricePlanId)
	var id, planId, billingCurrency string
	var name, description sql.NullString
	var billingAmount int64
	var active bool
	var dateCreated, dateModified time.Time
	var priceScheduleId sql.NullString
	var billingKindRaw, amountBasisRaw sql.NullInt32
	var billingCycleValue, defaultTermValue sql.NullInt32
	var billingCycleUnit, defaultTermUnit sql.NullString
	if err := row.Scan(&id, &planId, &billingAmount, &billingCurrency, &name, &description, &active, &dateCreated, &dateModified, &priceScheduleId, &billingKindRaw, &amountBasisRaw, &billingCycleValue, &billingCycleUnit, &defaultTermValue, &defaultTermUnit); err == sql.ErrNoRows {
		return nil, fmt.Errorf("price plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	pricePlan := &priceplanpb.PricePlan{Id: id, PlanId: planId, BillingAmount: billingAmount, BillingCurrency: billingCurrency, Active: active}
	if name.Valid {
		pricePlan.Name = &name.String
	}
	if description.Valid {
		pricePlan.Description = &description.String
	}
	if priceScheduleId.Valid && priceScheduleId.String != "" {
		pricePlan.PriceScheduleId = &priceScheduleId.String
	}
	if billingKindRaw.Valid {
		pricePlan.BillingKind = priceplanpb.BillingKind(billingKindRaw.Int32)
	}
	if amountBasisRaw.Valid {
		pricePlan.AmountBasis = priceplanpb.AmountBasis(amountBasisRaw.Int32)
	}
	if billingCycleValue.Valid {
		v := billingCycleValue.Int32
		pricePlan.BillingCycleValue = &v
	}
	if billingCycleUnit.Valid {
		pricePlan.BillingCycleUnit = &billingCycleUnit.String
	}
	if defaultTermValue.Valid {
		v := defaultTermValue.Int32
		pricePlan.DefaultTermValue = &v
	}
	if defaultTermUnit.Valid {
		pricePlan.DefaultTermUnit = &defaultTermUnit.String
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		pricePlan.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		pricePlan.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		pricePlan.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		pricePlan.DateModifiedString = &dmStr
	}
	return &priceplanpb.GetPricePlanItemPageDataResponse{PricePlan: pricePlan, Success: true}, nil
}

// Note: Duplicate operations.ParseTimestamp function removed - use the one from common operations package

// NewPricePlanRepository creates a new PostgreSQL price_plan repository (old-style constructor)
func NewPricePlanRepository(db *sql.DB, tableName string) priceplanpb.PricePlanDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPricePlanRepository(dbOps, tableName)
}
