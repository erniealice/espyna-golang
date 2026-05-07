//go:build postgresql

package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	supplierproductcostplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_cost_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresSupplierProductCostPlanRepository implements supplier_product_cost_plan CRUD using PostgreSQL.
type PostgresSupplierProductCostPlanRepository struct {
	supplierproductcostplanpb.UnimplementedSupplierProductCostPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierProductCostPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_product_cost_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierProductCostPlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresSupplierProductCostPlanRepository creates a new PostgreSQL supplier product cost plan repository.
func NewPostgresSupplierProductCostPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) supplierproductcostplanpb.SupplierProductCostPlanDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_product_cost_plan"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierProductCostPlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *PostgresSupplierProductCostPlanRepository) CreateSupplierProductCostPlan(ctx context.Context, req *supplierproductcostplanpb.CreateSupplierProductCostPlanRequest) (*supplierproductcostplanpb.CreateSupplierProductCostPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier product cost plan data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier product cost plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	spcp := &supplierproductcostplanpb.SupplierProductCostPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spcp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierproductcostplanpb.CreateSupplierProductCostPlanResponse{Data: []*supplierproductcostplanpb.SupplierProductCostPlan{spcp}}, nil
}

func (r *PostgresSupplierProductCostPlanRepository) ReadSupplierProductCostPlan(ctx context.Context, req *supplierproductcostplanpb.ReadSupplierProductCostPlanRequest) (*supplierproductcostplanpb.ReadSupplierProductCostPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier product cost plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier product cost plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	spcp := &supplierproductcostplanpb.SupplierProductCostPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spcp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierproductcostplanpb.ReadSupplierProductCostPlanResponse{Data: []*supplierproductcostplanpb.SupplierProductCostPlan{spcp}}, nil
}

func (r *PostgresSupplierProductCostPlanRepository) UpdateSupplierProductCostPlan(ctx context.Context, req *supplierproductcostplanpb.UpdateSupplierProductCostPlanRequest) (*supplierproductcostplanpb.UpdateSupplierProductCostPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier product cost plan ID is required")
	}
	jsonData, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	data["active"] = req.Data.GetActive()
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier product cost plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	spcp := &supplierproductcostplanpb.SupplierProductCostPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spcp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierproductcostplanpb.UpdateSupplierProductCostPlanResponse{Data: []*supplierproductcostplanpb.SupplierProductCostPlan{spcp}}, nil
}

func (r *PostgresSupplierProductCostPlanRepository) DeleteSupplierProductCostPlan(ctx context.Context, req *supplierproductcostplanpb.DeleteSupplierProductCostPlanRequest) (*supplierproductcostplanpb.DeleteSupplierProductCostPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier product cost plan ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier product cost plan: %w", err)
	}
	return &supplierproductcostplanpb.DeleteSupplierProductCostPlanResponse{Success: true}, nil
}

func (r *PostgresSupplierProductCostPlanRepository) ListSupplierProductCostPlans(ctx context.Context, req *supplierproductcostplanpb.ListSupplierProductCostPlansRequest) (*supplierproductcostplanpb.ListSupplierProductCostPlansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier product cost plans: %w", err)
	}
	var items []*supplierproductcostplanpb.SupplierProductCostPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		spcp := &supplierproductcostplanpb.SupplierProductCostPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spcp); err != nil {
			continue
		}
		items = append(items, spcp)
	}
	return &supplierproductcostplanpb.ListSupplierProductCostPlansResponse{Data: items}, nil
}

func (r *PostgresSupplierProductCostPlanRepository) GetSupplierProductCostPlanListPageData(ctx context.Context, req *supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataRequest) (*supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
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
	query := `SELECT id, active, cost_plan_id, supplier_product_plan_id, billing_treatment, billing_amount, date_created, date_modified
	          FROM supplier_product_cost_plan
	          WHERE active = true
	          ORDER BY date_created DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*supplierproductcostplanpb.SupplierProductCostPlan
	for rows.Next() {
		var id, costPlanID, supplierProductPlanID string
		var active bool
		var billingTreatmentRaw sql.NullInt32
		var billingAmount sql.NullInt64
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &active, &costPlanID, &supplierProductPlanID, &billingTreatmentRaw, &billingAmount, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		spcp := &supplierproductcostplanpb.SupplierProductCostPlan{Id: id, Active: active, CostPlanId: costPlanID, SupplierProductPlanId: supplierProductPlanID}
		if billingTreatmentRaw.Valid {
			spcp.BillingTreatment = supplierproductcostplanpb.SupplierProductCostPlanBillingTreatment(billingTreatmentRaw.Int32)
		}
		if billingAmount.Valid {
			spcp.BillingAmount = billingAmount.Int64
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			spcp.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			spcp.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			spcp.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			spcp.DateModifiedString = &dmStr
		}
		items = append(items, spcp)
	}
	return &supplierproductcostplanpb.GetSupplierProductCostPlanListPageDataResponse{SupplierProductCostPlanList: items, Success: true}, nil
}

func (r *PostgresSupplierProductCostPlanRepository) GetSupplierProductCostPlanItemPageData(ctx context.Context, req *supplierproductcostplanpb.GetSupplierProductCostPlanItemPageDataRequest) (*supplierproductcostplanpb.GetSupplierProductCostPlanItemPageDataResponse, error) {
	if req == nil || req.SupplierProductCostPlanId == "" {
		return nil, fmt.Errorf("supplier product cost plan ID required")
	}
	query := `SELECT id, active, cost_plan_id, supplier_product_plan_id, billing_treatment, billing_amount, date_created, date_modified
	          FROM supplier_product_cost_plan WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, req.SupplierProductCostPlanId)
	var id, costPlanID, supplierProductPlanID string
	var active bool
	var billingTreatmentRaw sql.NullInt32
	var billingAmount sql.NullInt64
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &active, &costPlanID, &supplierProductPlanID, &billingTreatmentRaw, &billingAmount, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("supplier product cost plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	spcp := &supplierproductcostplanpb.SupplierProductCostPlan{Id: id, Active: active, CostPlanId: costPlanID, SupplierProductPlanId: supplierProductPlanID}
	if billingTreatmentRaw.Valid {
		spcp.BillingTreatment = supplierproductcostplanpb.SupplierProductCostPlanBillingTreatment(billingTreatmentRaw.Int32)
	}
	if billingAmount.Valid {
		spcp.BillingAmount = billingAmount.Int64
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		spcp.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		spcp.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		spcp.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		spcp.DateModifiedString = &dmStr
	}
	return &supplierproductcostplanpb.GetSupplierProductCostPlanItemPageDataResponse{SupplierProductCostPlan: spcp, Success: true}, nil
}
