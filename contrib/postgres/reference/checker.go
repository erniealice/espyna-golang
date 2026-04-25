//go:build postgresql

package reference

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erniealice/espyna-golang/consumer"
	"github.com/lib/pq"
)

// Checker provides batch FK reference checking for deletable state.
// Each method returns a map where true = ID is in use and should NOT be deleted.
type Checker struct {
	db *sql.DB
}

func NewChecker(db *sql.DB) *Checker {
	return &Checker{db: db}
}

func (c *Checker) GetLocationInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT location_id AS ref_id FROM revenue WHERE location_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT location_id AS ref_id FROM expenditure WHERE location_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT location_id AS ref_id FROM inventory_item WHERE location_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT location_id AS ref_id FROM price_list WHERE location_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT location_id AS ref_id FROM price_schedule WHERE location_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
		) AS refs`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

func (c *Checker) GetRoleInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT role_id FROM workspace_user_role WHERE role_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetCategoryInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT category_id AS ref_id FROM client_category WHERE category_id = ANY($1) AND active = true
			UNION ALL
			SELECT category_id AS ref_id FROM supplier_category WHERE category_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetClientInUseIDs checks if clients are referenced in revenue or other client-linked records.
func (c *Checker) GetClientInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT DISTINCT client_id FROM revenue WHERE client_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

func (c *Checker) GetProductInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	// Model D: product_plan.product_id is now a catalog-level reference that
	// must block deletes. Without it, a product that's only in a catalog (not
	// yet invoiced, priced, or stocked) could be deleted out from under its
	// ProductPlan rows.
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT rli.product_id AS ref_id FROM revenue_line_item rli JOIN revenue r ON r.id = rli.revenue_id WHERE rli.product_id = ANY($1) AND rli.active = true AND ($2::text IS NULL OR r.workspace_id = $2)
			UNION ALL
			SELECT product_id AS ref_id FROM price_product WHERE product_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT product_id AS ref_id FROM inventory_item WHERE product_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT product_id AS ref_id FROM product_plan WHERE product_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

// GetProductVariantInUseIDs blocks deletion of product variants referenced by
// inventory stock, recorded revenue lines, variant-option pivots, or catalog
// product_plan rows. Covers the full Model D surface.
func (c *Checker) GetProductVariantInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT product_variant_id AS ref_id FROM inventory_item WHERE product_variant_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT rli.variant_id AS ref_id FROM revenue_line_item rli JOIN revenue r ON r.id = rli.revenue_id WHERE rli.variant_id = ANY($1) AND rli.active = true AND ($2::text IS NULL OR r.workspace_id = $2)
			UNION ALL
			SELECT product_variant_id AS ref_id FROM product_variant_option WHERE product_variant_id = ANY($1)
			UNION ALL
			SELECT product_variant_id AS ref_id FROM product_plan WHERE product_variant_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

// GetProductOptionValueInUseIDs blocks deletion of product_option_value rows
// that are referenced by product_variant_option pivots.
func (c *Checker) GetProductOptionValueInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT product_option_value_id FROM product_variant_option WHERE product_option_value_id = ANY($1)`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetProductOptionInUseIDs blocks deletion of a product_option when any of its
// child product_option_value rows are referenced by product_variant_option.
// Transitive check: we resolve "in use" by matching option_id → values → pivot.
func (c *Checker) GetProductOptionInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT pov.product_option_id AS ref_id
		FROM product_option_value pov
		WHERE pov.product_option_id = ANY($1)
		  AND EXISTS (
			SELECT 1 FROM product_variant_option pvo
			WHERE pvo.product_option_value_id = pov.id
		)`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetPlanInUseIDs checks if plans are referenced by product_plan or price_plan.
func (c *Checker) GetPlanInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT plan_id AS ref_id FROM product_plan WHERE plan_id = ANY($1)
			UNION ALL
			SELECT plan_id AS ref_id FROM price_plan WHERE plan_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetPriceListInUseIDs checks if price lists are referenced by price products.
func (c *Checker) GetPriceListInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT price_list_id FROM price_product WHERE price_list_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetPricePlanInUseIDs checks if price plans are referenced by active subscriptions.
// product_price_plan has ON DELETE CASCADE so it does not block deletes.
func (c *Checker) GetPricePlanInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT price_plan_id FROM subscription WHERE price_plan_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetPriceScheduleInUseIDs locks a PriceSchedule only when an active subscription
// references one of its (active) price_plans. Empty/draft schedules — even those
// with price_plan rows — remain deletable until money is flowing through them.
// Mirrors the PricePlan pricing-lock semantics one level up.
func (c *Checker) GetPriceScheduleInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT pp.price_schedule_id
		FROM price_plan pp
		JOIN subscription s ON s.price_plan_id = pp.id
		WHERE pp.price_schedule_id = ANY($1)
		  AND pp.active = true
		  AND s.active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetAssetCategoryInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT asset_category_id AS ref_id FROM asset WHERE asset_category_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetPaymentTermInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT payment_term_id AS ref_id FROM client WHERE payment_term_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT payment_term_id AS ref_id FROM supplier WHERE payment_term_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT payment_term_id AS ref_id FROM revenue WHERE payment_term_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT payment_term_id AS ref_id FROM expenditure WHERE payment_term_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
		) AS refs`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

func (c *Checker) GetLineInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT line_id AS ref_id FROM product WHERE line_id = ANY($1) AND active = true
			UNION ALL
			SELECT line_id AS ref_id FROM product_line WHERE line_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

func (c *Checker) GetLocationAreaInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT DISTINCT location_area_id AS ref_id FROM location WHERE location_area_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

// GetEventTagInUseIDs blocks deletion of an event_tag when any active
// event_tag_assignment row references it. Workspace-scoped via the join table.
func (c *Checker) GetEventTagInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `
		SELECT DISTINCT event_tag_id AS ref_id
		FROM event_tag_assignment
		WHERE event_tag_id = ANY($1)
		  AND active = true
		  AND ($2::text IS NULL OR workspace_id = $2)`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

// GetSubscriptionInUseIDs checks whether subscriptions are referenced by any of
// the tables that must block deletion: balance, invoice, license, payment,
// subscription_attribute, revenue, and operation.job (via subscription_id column).
func (c *Checker) GetSubscriptionInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT subscription_id AS ref_id FROM balance WHERE subscription_id = ANY($1)
			UNION ALL
			SELECT subscription_id AS ref_id FROM invoice WHERE subscription_id = ANY($1)
			UNION ALL
			SELECT subscription_id AS ref_id FROM license WHERE subscription_id = ANY($1)
			UNION ALL
			SELECT subscription_id AS ref_id FROM payment WHERE subscription_id = ANY($1)
			UNION ALL
			SELECT subscription_id AS ref_id FROM subscription_attribute WHERE subscription_id = ANY($1)
			UNION ALL
			SELECT subscription_id AS ref_id FROM revenue WHERE subscription_id = ANY($1) AND active = true
			UNION ALL
			SELECT subscription_id AS ref_id FROM job WHERE subscription_id = ANY($1)
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetSupplierInUseIDs checks if suppliers are referenced by expenditures or fulfillments.
func (c *Checker) GetSupplierInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT supplier_id AS ref_id FROM expenditure WHERE supplier_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
			UNION ALL
			SELECT supplier_id AS ref_id FROM fulfillment WHERE supplier_id = ANY($1) AND active = true AND ($2::text IS NULL OR workspace_id = $2)
		) AS refs`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
}

// queryInUseIDsWithWorkspace is like queryInUseIDs but passes a workspace_id as $2.
// The query must accept $1 = ids array and $2 = workspace_id (text or NULL).
func queryInUseIDsWithWorkspace(ctx context.Context, db *sql.DB, query string, ids []string, workspaceID string) (map[string]bool, error) {
	if len(ids) == 0 {
		return make(map[string]bool), nil
	}

	var wsArg any
	if workspaceID != "" {
		wsArg = workspaceID
	}

	rows, err := db.QueryContext(ctx, query, pq.Array(ids), wsArg)
	if err != nil {
		return nil, fmt.Errorf("reference check query failed: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("reference check scan failed: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}

func queryInUseIDs(ctx context.Context, db *sql.DB, query string, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return make(map[string]bool), nil
	}

	rows, err := db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("reference check query failed: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("reference check scan failed: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}