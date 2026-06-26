//go:build postgresql

package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresCostPlanRepository implements cost_plan CRUD operations using PostgreSQL.
type PostgresCostPlanRepository struct {
	costplanpb.UnimplementedCostPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CostPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres cost_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCostPlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresCostPlanRepository creates a new PostgreSQL cost plan repository.
func NewPostgresCostPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) costplanpb.CostPlanDomainServiceServer {
	if tableName == "" {
		tableName = "cost_plan"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresCostPlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *PostgresCostPlanRepository) CreateCostPlan(ctx context.Context, req *costplanpb.CreateCostPlanRequest) (*costplanpb.CreateCostPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("cost plan data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// Empty optional FKs must arrive as SQL NULL.
	if v, ok := data["costScheduleId"].(string); ok && v == "" {
		data["costScheduleId"] = nil
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create cost plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cp := &costplanpb.CostPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &costplanpb.CreateCostPlanResponse{Data: []*costplanpb.CostPlan{cp}}, nil
}

func (r *PostgresCostPlanRepository) ReadCostPlan(ctx context.Context, req *costplanpb.ReadCostPlanRequest) (*costplanpb.ReadCostPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read cost plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cp := &costplanpb.CostPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &costplanpb.ReadCostPlanResponse{Data: []*costplanpb.CostPlan{cp}}, nil
}

func (r *PostgresCostPlanRepository) UpdateCostPlan(ctx context.Context, req *costplanpb.UpdateCostPlanRequest) (*costplanpb.UpdateCostPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost plan ID is required")
	}
	jsonData, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// Always include active flag — proto3 omits bool=false during JSON marshal.
	data["active"] = req.Data.GetActive()
	// Empty optional FK must reach the column as SQL NULL.
	if v, ok := data["costScheduleId"].(string); ok && v == "" {
		data["costScheduleId"] = nil
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update cost plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	cp := &costplanpb.CostPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &costplanpb.UpdateCostPlanResponse{Data: []*costplanpb.CostPlan{cp}}, nil
}

func (r *PostgresCostPlanRepository) DeleteCostPlan(ctx context.Context, req *costplanpb.DeleteCostPlanRequest) (*costplanpb.DeleteCostPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("cost plan ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete cost plan: %w", err)
	}
	return &costplanpb.DeleteCostPlanResponse{Success: true}, nil
}

func (r *PostgresCostPlanRepository) ListCostPlans(ctx context.Context, req *costplanpb.ListCostPlansRequest) (*costplanpb.ListCostPlansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list cost plans: %w", err)
	}
	var items []*costplanpb.CostPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		cp := &costplanpb.CostPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, cp); err != nil {
			continue
		}
		items = append(items, cp)
	}
	return &costplanpb.ListCostPlansResponse{Data: items}, nil
}

var costPlanSortableSQLCols = []string{
	"id", "name", "description", "active", "supplier_plan_id", "cost_schedule_id",
	"billing_kind", "amount_basis", "billing_amount", "billing_currency",
	"billing_cycle_value", "billing_cycle_unit", "default_term_value",
	"default_term_unit", "date_created", "date_modified",
}

func (r *PostgresCostPlanRepository) GetCostPlanListPageData(ctx context.Context, req *costplanpb.GetCostPlanListPageDataRequest) (*costplanpb.GetCostPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}
	limit, offset := int32(50), int32(0)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if op := req.Pagination.GetOffset(); op != nil && op.Page > 0 {
			offset = (op.Page - 1) * limit
		}
	}
	// Sort — fail-closed against the per-entity whitelist (A2 guard). The whitelist
	// holds only the raw cost_plan (cp) columns, each surfaced as a distinct output
	// column name, so the emitted ORDER BY "<col>" resolves unambiguously. The
	// joined sp.name AS supplier_plan_name alias is intentionally excluded
	// (not cp.-qualifiable). An unknown column errors instead of being interpolated.
	orderByClause, err := postgresCore.BuildOrderBy(costPlanSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}
	query := `SELECT cp.id, cp.name, cp.description, cp.active, cp.supplier_plan_id, cp.cost_schedule_id,
	                 cp.billing_kind, cp.amount_basis, cp.billing_amount, cp.billing_currency,
	                 cp.billing_cycle_value, cp.billing_cycle_unit, cp.default_term_value, cp.default_term_unit,
	                 cp.date_created, cp.date_modified,
	                 sp.name AS supplier_plan_name
	          FROM cost_plan cp
	          LEFT JOIN supplier_plan sp ON cp.supplier_plan_id = sp.id
	          WHERE cp.active = true
	            AND ($1::text IS NULL OR $1::text = '' OR cp.name ILIKE $1 OR cp.description ILIKE $1)
	          ` + orderByClause + ` LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*costplanpb.CostPlan
	for rows.Next() {
		var id, name, supplierPlanID, billingCurrency string
		var description, costScheduleID, supplierPlanName sql.NullString
		var active bool
		var billingKindRaw, amountBasisRaw sql.NullInt32
		var billingAmount sql.NullInt64
		var billingCycleValue, defaultTermValue sql.NullInt32
		var billingCycleUnit, defaultTermUnit sql.NullString
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &name, &description, &active, &supplierPlanID, &costScheduleID,
			&billingKindRaw, &amountBasisRaw, &billingAmount, &billingCurrency,
			&billingCycleValue, &billingCycleUnit, &defaultTermValue, &defaultTermUnit,
			&dateCreated, &dateModified, &supplierPlanName); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		cp := &costplanpb.CostPlan{Id: id, Name: &name, Active: active, SupplierPlanId: supplierPlanID, BillingCurrency: billingCurrency}
		if description.Valid {
			cp.Description = &description.String
		}
		if costScheduleID.Valid && costScheduleID.String != "" {
			cp.CostScheduleId = &costScheduleID.String
		}
		if billingKindRaw.Valid {
			cp.BillingKind = costplanpb.CostPlanBillingKind(billingKindRaw.Int32)
		}
		if amountBasisRaw.Valid {
			cp.AmountBasis = costplanpb.CostPlanAmountBasis(amountBasisRaw.Int32)
		}
		if billingAmount.Valid {
			cp.BillingAmount = billingAmount.Int64
		}
		if billingCycleValue.Valid {
			v := billingCycleValue.Int32
			cp.BillingCycleValue = &v
		}
		if billingCycleUnit.Valid {
			cp.BillingCycleUnit = &billingCycleUnit.String
		}
		if defaultTermValue.Valid {
			v := defaultTermValue.Int32
			cp.DefaultTermValue = &v
		}
		if defaultTermUnit.Valid {
			cp.DefaultTermUnit = &defaultTermUnit.String
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			cp.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			cp.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			cp.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			cp.DateModifiedString = &dmStr
		}
		items = append(items, cp)
	}
	return &costplanpb.GetCostPlanListPageDataResponse{CostPlanList: items, Success: true}, nil
}

func (r *PostgresCostPlanRepository) GetCostPlanItemPageData(ctx context.Context, req *costplanpb.GetCostPlanItemPageDataRequest) (*costplanpb.GetCostPlanItemPageDataResponse, error) {
	if req == nil || req.CostPlanId == "" {
		return nil, fmt.Errorf("cost plan ID required")
	}
	query := `SELECT cp.id, cp.name, cp.description, cp.active, cp.supplier_plan_id, cp.cost_schedule_id,
	                 cp.billing_kind, cp.amount_basis, cp.billing_amount, cp.billing_currency,
	                 cp.billing_cycle_value, cp.billing_cycle_unit, cp.default_term_value, cp.default_term_unit,
	                 cp.date_created, cp.date_modified
	          FROM cost_plan cp WHERE cp.id = $1`
	row := r.db.QueryRowContext(ctx, query, req.CostPlanId)
	var id, name, supplierPlanID, billingCurrency string
	var description, costScheduleID sql.NullString
	var active bool
	var billingKindRaw, amountBasisRaw sql.NullInt32
	var billingAmount sql.NullInt64
	var billingCycleValue, defaultTermValue sql.NullInt32
	var billingCycleUnit, defaultTermUnit sql.NullString
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &name, &description, &active, &supplierPlanID, &costScheduleID,
		&billingKindRaw, &amountBasisRaw, &billingAmount, &billingCurrency,
		&billingCycleValue, &billingCycleUnit, &defaultTermValue, &defaultTermUnit,
		&dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("cost plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	cp := &costplanpb.CostPlan{Id: id, Name: &name, Active: active, SupplierPlanId: supplierPlanID, BillingCurrency: billingCurrency}
	if description.Valid {
		cp.Description = &description.String
	}
	if costScheduleID.Valid && costScheduleID.String != "" {
		cp.CostScheduleId = &costScheduleID.String
	}
	if billingKindRaw.Valid {
		cp.BillingKind = costplanpb.CostPlanBillingKind(billingKindRaw.Int32)
	}
	if amountBasisRaw.Valid {
		cp.AmountBasis = costplanpb.CostPlanAmountBasis(amountBasisRaw.Int32)
	}
	if billingAmount.Valid {
		cp.BillingAmount = billingAmount.Int64
	}
	if billingCycleValue.Valid {
		v := billingCycleValue.Int32
		cp.BillingCycleValue = &v
	}
	if billingCycleUnit.Valid {
		cp.BillingCycleUnit = &billingCycleUnit.String
	}
	if defaultTermValue.Valid {
		v := defaultTermValue.Int32
		cp.DefaultTermValue = &v
	}
	if defaultTermUnit.Valid {
		cp.DefaultTermUnit = &defaultTermUnit.String
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		cp.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		cp.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		cp.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		cp.DateModifiedString = &dmStr
	}
	return &costplanpb.GetCostPlanItemPageDataResponse{CostPlan: cp, Success: true}, nil
}
