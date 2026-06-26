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
	supplierproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_product_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresSupplierProductPlanRepository implements supplier_product_plan CRUD using PostgreSQL.
type PostgresSupplierProductPlanRepository struct {
	supplierproductplanpb.UnimplementedSupplierProductPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierProductPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_product_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierProductPlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresSupplierProductPlanRepository creates a new PostgreSQL supplier product plan repository.
func NewPostgresSupplierProductPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) supplierproductplanpb.SupplierProductPlanDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_product_plan"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierProductPlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *PostgresSupplierProductPlanRepository) CreateSupplierProductPlan(ctx context.Context, req *supplierproductplanpb.CreateSupplierProductPlanRequest) (*supplierproductplanpb.CreateSupplierProductPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier product plan data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	// Empty optional FK (product_variant_id) must arrive as SQL NULL.
	if v, ok := data["productVariantId"].(string); ok && v == "" {
		data["productVariantId"] = nil
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier product plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	spp := &supplierproductplanpb.SupplierProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierproductplanpb.CreateSupplierProductPlanResponse{Data: []*supplierproductplanpb.SupplierProductPlan{spp}}, nil
}

func (r *PostgresSupplierProductPlanRepository) ReadSupplierProductPlan(ctx context.Context, req *supplierproductplanpb.ReadSupplierProductPlanRequest) (*supplierproductplanpb.ReadSupplierProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier product plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier product plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	spp := &supplierproductplanpb.SupplierProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierproductplanpb.ReadSupplierProductPlanResponse{Data: []*supplierproductplanpb.SupplierProductPlan{spp}}, nil
}

func (r *PostgresSupplierProductPlanRepository) UpdateSupplierProductPlan(ctx context.Context, req *supplierproductplanpb.UpdateSupplierProductPlanRequest) (*supplierproductplanpb.UpdateSupplierProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier product plan ID is required")
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
	if v, ok := data["productVariantId"].(string); ok && v == "" {
		data["productVariantId"] = nil
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier product plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	spp := &supplierproductplanpb.SupplierProductPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierproductplanpb.UpdateSupplierProductPlanResponse{Data: []*supplierproductplanpb.SupplierProductPlan{spp}}, nil
}

func (r *PostgresSupplierProductPlanRepository) DeleteSupplierProductPlan(ctx context.Context, req *supplierproductplanpb.DeleteSupplierProductPlanRequest) (*supplierproductplanpb.DeleteSupplierProductPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier product plan ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier product plan: %w", err)
	}
	return &supplierproductplanpb.DeleteSupplierProductPlanResponse{Success: true}, nil
}

func (r *PostgresSupplierProductPlanRepository) ListSupplierProductPlans(ctx context.Context, req *supplierproductplanpb.ListSupplierProductPlansRequest) (*supplierproductplanpb.ListSupplierProductPlansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier product plans: %w", err)
	}
	var items []*supplierproductplanpb.SupplierProductPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		spp := &supplierproductplanpb.SupplierProductPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, spp); err != nil {
			continue
		}
		items = append(items, spp)
	}
	return &supplierproductplanpb.ListSupplierProductPlansResponse{Data: items}, nil
}

var supplierProductPlanSortableSQLCols = []string{
	"id", "name", "active", "supplier_plan_id", "product_id",
	"product_variant_id", "date_created", "date_modified",
}

func (r *PostgresSupplierProductPlanRepository) GetSupplierProductPlanListPageData(ctx context.Context, req *supplierproductplanpb.GetSupplierProductPlanListPageDataRequest) (*supplierproductplanpb.GetSupplierProductPlanListPageDataResponse, error) {
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
	// Sort — fail-closed against the per-entity whitelist (A2 guard). Route the
	// caller-supplied sort column through core.BuildOrderBy so an unknown column
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(supplierProductPlanSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`SELECT id, name, active, supplier_plan_id, product_id, product_variant_id, date_created, date_modified
	          FROM supplier_product_plan
	          WHERE active = true
	            AND ($1::text IS NULL OR $1::text = '' OR name ILIKE $1)
	          %s LIMIT $2 OFFSET $3`, orderByClause)
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*supplierproductplanpb.SupplierProductPlan
	for rows.Next() {
		var id, name, supplierPlanID, productID string
		var productVariantID sql.NullString
		var active bool
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &name, &active, &supplierPlanID, &productID, &productVariantID, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		spp := &supplierproductplanpb.SupplierProductPlan{Id: id, Name: name, Active: active, SupplierPlanId: supplierPlanID, ProductId: productID}
		if productVariantID.Valid && productVariantID.String != "" {
			spp.ProductVariantId = &productVariantID.String
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			spp.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			spp.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			spp.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			spp.DateModifiedString = &dmStr
		}
		items = append(items, spp)
	}
	return &supplierproductplanpb.GetSupplierProductPlanListPageDataResponse{SupplierProductPlanList: items, Success: true}, nil
}

func (r *PostgresSupplierProductPlanRepository) GetSupplierProductPlanItemPageData(ctx context.Context, req *supplierproductplanpb.GetSupplierProductPlanItemPageDataRequest) (*supplierproductplanpb.GetSupplierProductPlanItemPageDataResponse, error) {
	if req == nil || req.SupplierProductPlanId == "" {
		return nil, fmt.Errorf("supplier product plan ID required")
	}
	query := `SELECT id, name, active, supplier_plan_id, product_id, product_variant_id, date_created, date_modified
	          FROM supplier_product_plan WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, req.SupplierProductPlanId)
	var id, name, supplierPlanID, productID string
	var productVariantID sql.NullString
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &name, &active, &supplierPlanID, &productID, &productVariantID, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("supplier product plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	spp := &supplierproductplanpb.SupplierProductPlan{Id: id, Name: name, Active: active, SupplierPlanId: supplierPlanID, ProductId: productID}
	if productVariantID.Valid && productVariantID.String != "" {
		spp.ProductVariantId = &productVariantID.String
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		spp.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		spp.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		spp.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		spp.DateModifiedString = &dmStr
	}
	return &supplierproductplanpb.GetSupplierProductPlanItemPageDataResponse{SupplierProductPlan: spp, Success: true}, nil
}

// ListBySupplierPlan returns all supplier product plans for a given supplier plan ID.
func (r *PostgresSupplierProductPlanRepository) ListBySupplierPlan(ctx context.Context, req *supplierproductplanpb.ListSupplierProductPlansBySupplierPlanRequest) (*supplierproductplanpb.ListSupplierProductPlansBySupplierPlanResponse, error) {
	if req == nil || req.SupplierPlanId == "" {
		return nil, fmt.Errorf("supplier_plan_id is required")
	}
	var params *interfaces.ListParams
	filterReq := &interfaces.ListParams{
		Filters: nil,
	}
	_ = filterReq
	_ = params

	query := `SELECT id, name, active, supplier_plan_id, product_id, product_variant_id, date_created, date_modified
	          FROM supplier_product_plan
	          WHERE supplier_plan_id = $1 AND active = true
	          ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query, req.SupplierPlanId)
	if err != nil {
		return nil, fmt.Errorf("ListBySupplierPlan query failed: %w", err)
	}
	defer rows.Close()
	var items []*supplierproductplanpb.SupplierProductPlan
	for rows.Next() {
		var id, name, supplierPlanID, productID string
		var productVariantID sql.NullString
		var active bool
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &name, &active, &supplierPlanID, &productID, &productVariantID, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		spp := &supplierproductplanpb.SupplierProductPlan{Id: id, Name: name, Active: active, SupplierPlanId: supplierPlanID, ProductId: productID}
		if productVariantID.Valid && productVariantID.String != "" {
			spp.ProductVariantId = &productVariantID.String
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			spp.DateCreated = &ts
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			spp.DateModified = &ts
		}
		items = append(items, spp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListBySupplierPlan rows error: %w", err)
	}
	return &supplierproductplanpb.ListSupplierProductPlansBySupplierPlanResponse{SupplierProductPlans: items, Success: true}, nil
}
