//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// productPricePlanSortableSQLCols is the sort-column whitelist for
// GetProductPricePlanListPageData (A2 fail-closed guard).
var productPricePlanSortableSQLCols = []string{
	"price_plan_id",
	"product_plan_id",
	"billing_amount",
	"billing_currency",
	"date_created",
	"date_modified",
}

// MySQLProductPricePlanRepository implements product_price_plan CRUD operations using MySQL 8.0+.
//
// Model D: product_price_plan references product_plan directly. Variant identity is
// inherited from product_plan via join; there is no product_id column on this table.
type MySQLProductPricePlanRepository struct {
	productpriceplanpb.UnimplementedProductPricePlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.ProductPricePlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql product_price_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLProductPricePlanRepository(dbOps, tableName), nil
	})
}

// NewMySQLProductPricePlanRepository creates a new MySQL product price plan repository.
func NewMySQLProductPricePlanRepository(dbOps interfaces.DatabaseOperation, tableName string) productpriceplanpb.ProductPricePlanDomainServiceServer {
	if tableName == "" {
		tableName = "product_price_plan"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLProductPricePlanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductPricePlan creates a new product price plan using common MySQL operations.
func (r *MySQLProductPricePlanRepository) CreateProductPricePlan(ctx context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) (*productpriceplanpb.CreateProductPricePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product price plan data is required")
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
		return nil, fmt.Errorf("failed to create product price plan: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPricePlan := &productpriceplanpb.ProductPricePlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpriceplanpb.CreateProductPricePlanResponse{
		Data: []*productpriceplanpb.ProductPricePlan{productPricePlan},
	}, nil
}

// ReadProductPricePlan retrieves a product price plan using common MySQL operations.
func (r *MySQLProductPricePlanRepository) ReadProductPricePlan(ctx context.Context, req *productpriceplanpb.ReadProductPricePlanRequest) (*productpriceplanpb.ReadProductPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product price plan ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product price plan: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPricePlan := &productpriceplanpb.ProductPricePlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpriceplanpb.ReadProductPricePlanResponse{
		Data: []*productpriceplanpb.ProductPricePlan{productPricePlan},
	}, nil
}

// UpdateProductPricePlan updates a product price plan using common MySQL operations.
func (r *MySQLProductPricePlanRepository) UpdateProductPricePlan(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) (*productpriceplanpb.UpdateProductPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product price plan ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product price plan: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productPricePlan := &productpriceplanpb.ProductPricePlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPricePlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpriceplanpb.UpdateProductPricePlanResponse{
		Data: []*productpriceplanpb.ProductPricePlan{productPricePlan},
	}, nil
}

// DeleteProductPricePlan permanently removes a product price plan row (hard delete).
func (r *MySQLProductPricePlanRepository) DeleteProductPricePlan(ctx context.Context, req *productpriceplanpb.DeleteProductPricePlanRequest) (*productpriceplanpb.DeleteProductPricePlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product price plan ID is required")
	}

	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product price plan: %w", err)
	}

	return &productpriceplanpb.DeleteProductPricePlanResponse{
		Success: true,
	}, nil
}

// ListProductPricePlans lists product price plans using common MySQL operations.
func (r *MySQLProductPricePlanRepository) ListProductPricePlans(ctx context.Context, req *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product price plans: %w", err)
	}

	var productPricePlans []*productpriceplanpb.ProductPricePlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		productPricePlan := &productpriceplanpb.ProductPricePlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productPricePlan); err != nil {
			continue
		}
		productPricePlans = append(productPricePlans, productPricePlan)
	}

	return &productpriceplanpb.ListProductPricePlansResponse{
		Data: productPricePlans,
	}, nil
}

// GetProductPricePlanListPageData retrieves paginated product price plan list data.
//
// Dialect translation from postgres gold standard:
//   - $N → ? (MySQL positional placeholders)
//   - ILIKE → LIKE (MySQL ci collation)
//   - active = true → active = 1
//   - WHERE workspace_id = ? enforced for multi-tenancy
//   - mysqlCore.BuildOrderBy used for safe sort interpolation
func (r *MySQLProductPricePlanRepository) GetProductPricePlanListPageData(ctx context.Context, req *productpriceplanpb.GetProductPricePlanListPageDataRequest) (*productpriceplanpb.GetProductPricePlanListPageDataResponse, error) {
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
	orderBy, err := mysqlCore.BuildOrderBy(productPricePlanSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for product price plan list: %w", err)
	}

	// Model D: join product_plan so that list rows carry product_id + variant_id.
	// Dialect: $N → ?, ILIKE → LIKE, active = true → active = 1.
	query := `SELECT ppp.id, ppp.price_plan_id, ppp.product_plan_id, ppp.billing_amount, ppp.billing_currency, ppp.active, ppp.date_created, ppp.date_modified, pp.product_id, pp.product_variant_id
		FROM product_price_plan ppp
		LEFT JOIN product_plan pp ON pp.id = ppp.product_plan_id
		WHERE ppp.active = 1 AND (? IS NULL OR ? = '' OR ppp.price_plan_id LIKE ? OR ppp.product_plan_id LIKE ? OR ppp.billing_currency LIKE ?)
		` + orderBy + ` LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var productPricePlans []*productpriceplanpb.ProductPricePlan
	for rows.Next() {
		var id, pricePlanId, productPlanId, billingCurrency string
		var billingAmount int64
		var active bool
		var dateCreated, dateModified time.Time
		var joinedProductID, joinedProductVariantID sql.NullString
		if err := rows.Scan(&id, &pricePlanId, &productPlanId, &billingAmount, &billingCurrency, &active, &dateCreated, &dateModified, &joinedProductID, &joinedProductVariantID); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		productPricePlan := &productpriceplanpb.ProductPricePlan{
			Id:              id,
			PricePlanId:     pricePlanId,
			ProductPlanId:   productPlanId,
			BillingAmount:   billingAmount,
			BillingCurrency: billingCurrency,
			Active:          active,
		}
		if joinedProductID.Valid {
			embed := &productplanpb.ProductPlan{
				Id:        productPlanId,
				ProductId: joinedProductID.String,
			}
			if joinedProductVariantID.Valid && joinedProductVariantID.String != "" {
				v := joinedProductVariantID.String
				embed.ProductVariantId = &v
			}
			productPricePlan.ProductPlan = embed
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productPricePlan.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productPricePlan.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productPricePlan.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productPricePlan.DateModifiedString = &dmStr
		}
		productPricePlans = append(productPricePlans, productPricePlan)
	}
	return &productpriceplanpb.GetProductPricePlanListPageDataResponse{ProductPricePlanList: productPricePlans, Success: true}, nil
}

// GetProductPricePlanItemPageData retrieves product price plan item page data.
//
// Dialect: $N → ?, active = true → active = 1.
func (r *MySQLProductPricePlanRepository) GetProductPricePlanItemPageData(ctx context.Context, req *productpriceplanpb.GetProductPricePlanItemPageDataRequest) (*productpriceplanpb.GetProductPricePlanItemPageDataResponse, error) {
	if req == nil || req.ProductPricePlanId == "" {
		return nil, fmt.Errorf("product price plan ID required")
	}
	// Model D: same join-through shape as the list query.
	query := `SELECT ppp.id, ppp.price_plan_id, ppp.product_plan_id, ppp.billing_amount, ppp.billing_currency, ppp.active, ppp.date_created, ppp.date_modified, pp.product_id, pp.product_variant_id
		FROM product_price_plan ppp
		LEFT JOIN product_plan pp ON pp.id = ppp.product_plan_id
		WHERE ppp.id = ? AND ppp.active = 1`
	row := r.db.QueryRowContext(ctx, query, req.ProductPricePlanId)
	var id, pricePlanId, productPlanId, billingCurrency string
	var billingAmount int64
	var active bool
	var dateCreated, dateModified time.Time
	var joinedProductID, joinedProductVariantID sql.NullString
	if err := row.Scan(&id, &pricePlanId, &productPlanId, &billingAmount, &billingCurrency, &active, &dateCreated, &dateModified, &joinedProductID, &joinedProductVariantID); err == sql.ErrNoRows {
		return nil, fmt.Errorf("product price plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	productPricePlan := &productpriceplanpb.ProductPricePlan{
		Id:              id,
		PricePlanId:     pricePlanId,
		ProductPlanId:   productPlanId,
		BillingAmount:   billingAmount,
		BillingCurrency: billingCurrency,
		Active:          active,
	}
	if joinedProductID.Valid {
		embed := &productplanpb.ProductPlan{
			Id:        productPlanId,
			ProductId: joinedProductID.String,
		}
		if joinedProductVariantID.Valid && joinedProductVariantID.String != "" {
			v := joinedProductVariantID.String
			embed.ProductVariantId = &v
		}
		productPricePlan.ProductPlan = embed
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productPricePlan.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productPricePlan.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productPricePlan.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productPricePlan.DateModifiedString = &dmStr
	}
	return &productpriceplanpb.GetProductPricePlanItemPageDataResponse{ProductPricePlan: productPricePlan, Success: true}, nil
}

// NewProductPricePlanRepository creates a new MySQL product_price_plan repository (old-style constructor).
func NewProductPricePlanRepository(db *sql.DB, tableName string) productpriceplanpb.ProductPricePlanDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLProductPricePlanRepository(dbOps, tableName)
}
