//go:build sqlserver

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// SQLServerPricePlanRepository implements price_plan CRUD operations using SQL Server.
type SQLServerPricePlanRepository struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PricePlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver price_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPricePlanRepository(dbOps, tableName), nil
	})
}

// NewSQLServerPricePlanRepository creates a new SQL Server price plan repository.
func NewSQLServerPricePlanRepository(dbOps interfaces.DatabaseOperation, tableName string) priceplanpb.PricePlanDomainServiceServer {
	if tableName == "" {
		tableName = "price_plan"
	}
	return &SQLServerPricePlanRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreatePricePlan creates a new price plan using common SQL Server operations.
func (r *SQLServerPricePlanRepository) CreatePricePlan(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("price plan data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	if v, ok := data["client_id"].(string); ok && v == "" {
		data["client_id"] = nil
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create price plan: %w", err)
	}

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

// ReadPricePlan retrieves a price plan using common SQL Server operations.
func (r *SQLServerPricePlanRepository) ReadPricePlan(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read price plan: %w", err)
	}

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

// UpdatePricePlan updates a price plan using common SQL Server operations.
func (r *SQLServerPricePlanRepository) UpdatePricePlan(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	data["active"] = req.Data.GetActive()

	if cid := req.Data.GetClientId(); cid == "" {
		data["client_id"] = nil
	} else {
		data["client_id"] = cid
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update price plan: %w", err)
	}

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

// DeletePricePlan permanently removes a price plan row (hard delete).
func (r *SQLServerPricePlanRepository) DeletePricePlan(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) (*priceplanpb.DeletePricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
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

// ListPricePlans lists price plans using common SQL Server operations.
func (r *SQLServerPricePlanRepository) ListPricePlans(ctx context.Context, req *priceplanpb.ListPricePlansRequest) (*priceplanpb.ListPricePlansResponse, error) {
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

	var pricePlans []*priceplanpb.PricePlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		pricePlan := &priceplanpb.PricePlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pricePlan); err != nil {
			continue
		}
		pricePlans = append(pricePlans, pricePlan)
	}

	return &priceplanpb.ListPricePlansResponse{
		Data: pricePlans,
	}, nil
}

// GetPricePlanListPageData retrieves paginated price plan list data.
//
// SQL Server differences:
//   - $1/$2/$3 → @p1/@p2/@p3.
//   - active = true → active = 1.
//   - ILIKE → LIKE.
//   - LIMIT/OFFSET → OFFSET/FETCH NEXT.
//   - sortField validated against allowlist before interpolation.
func (r *SQLServerPricePlanRepository) GetPricePlanListPageData(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) (*priceplanpb.GetPricePlanListPageDataResponse, error) {
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
	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sf := req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == 1 {
			sortOrder = "DESC"
		} else {
			sortOrder = "ASC"
		}
		// Validate against allowlist.
		for _, col := range pricePlanSortableSQLCols {
			if col == sf {
				sortField = sf
				break
			}
		}
	}

	// SQL Server: LIKE; OFFSET/FETCH; active = 1; @pN placeholders.
	// sortField is author-controlled (validated above) — safe to interpolate.
	query := fmt.Sprintf(`
		SELECT id, plan_id, billing_amount, billing_currency, name, description, active,
		       date_created, date_modified, price_schedule_id, billing_kind, amount_basis,
		       billing_cycle_value, billing_cycle_unit, default_term_value, default_term_unit
		FROM price_plan
		WHERE active = 1
		  AND (@p1 IS NULL OR @p1 = '' OR plan_id LIKE @p1 OR billing_currency LIKE @p1)
		ORDER BY [%s] %s
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY
	`, sortField, sortOrder)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var pricePlans []*priceplanpb.PricePlan
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
	_ = page // pagination response below
	return &priceplanpb.GetPricePlanListPageDataResponse{PricePlanList: pricePlans, Success: true}, nil
}

// GetPricePlanItemPageData retrieves price plan item page data.
//
// SQL Server differences: $1 → @p1; active = true → active = 1.
func (r *SQLServerPricePlanRepository) GetPricePlanItemPageData(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) (*priceplanpb.GetPricePlanItemPageDataResponse, error) {
	if req == nil || req.PricePlanId == "" {
		return nil, fmt.Errorf("price plan ID required")
	}

	query := `SELECT id, plan_id, billing_amount, billing_currency, name, description, active,
		date_created, date_modified, price_schedule_id, billing_kind, amount_basis,
		billing_cycle_value, billing_cycle_unit, default_term_value, default_term_unit
		FROM price_plan WHERE id = @p1 AND active = 1`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.PricePlanId)

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

// NewPricePlanRepository creates a new SQL Server price_plan repository (old-style constructor).
func NewPricePlanRepository(db *sql.DB, tableName string) priceplanpb.PricePlanDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerPricePlanRepository(dbOps, tableName)
}
