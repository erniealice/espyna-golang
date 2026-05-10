//go:build postgresql

package reference

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erniealice/espyna-golang/consumer"
	topref "github.com/erniealice/espyna-golang/reference"
	"github.com/lib/pq"
)

// Compile-time guarantee that *Checker satisfies the top-level public
// reference.Checker interface. This assertion covers the full 21-method
// contract so any new method added to the interface is caught at build time.
var _ topref.Checker = (*Checker)(nil)

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

// GetPriceScheduleInUseIDs locks a PriceSchedule whenever any active
// price_plan references it. The earlier semantic (locked only by an active
// subscription via active price_plan) was leaky: deleting a schedule with
// dangling PricePlans orphaned the children. Now matches the symmetric
// PricePlan-locks-Subscription pattern one level up.
//
// Operators who genuinely want to wipe a schedule and its price_plans must
// either deactivate or delete the price_plans first. There is no
// ON DELETE CASCADE on price_plan.price_schedule_id, so this checker is the
// authoritative gate for both single and bulk delete handlers.
func (c *Checker) GetPriceScheduleInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT price_schedule_id
		FROM price_plan
		WHERE price_schedule_id = ANY($1)
		  AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetPlanClientScopeLockedIDs returns plan IDs whose client_id MUST NOT be
// changed because at least one of their price_plans is referenced by an active
// subscription. The semantic mirrors GetPricePlanInUseIDs (active-subscription-
// only) but bubbles up the lock from PricePlan to its parent Plan.
//
// Only meaningful when the operator is attempting to change client_id —
// callers that aren't editing client_id should ignore the result.
func (c *Checker) GetPlanClientScopeLockedIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT pp.plan_id
		FROM price_plan pp
		JOIN subscription s ON s.price_plan_id = pp.id
		WHERE pp.plan_id = ANY($1)
		  AND pp.active = true
		  AND s.active  = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetActiveSubscriptionCountForPricePlan returns the count of active
// subscriptions attached to a single PricePlan. Used by the N>1 confirm
// dialog gate on UpdatePricePlan when monetary fields change on a
// client-scoped PricePlan (see plan §3.5).
func (c *Checker) GetActiveSubscriptionCountForPricePlan(ctx context.Context, id string) (int, error) {
	var count int
	err := c.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM subscription
		WHERE price_plan_id = $1 AND active = true`, id).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("active subscription count query failed: %w", err)
	}
	return count, nil
}

func (c *Checker) GetAssetCategoryInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	query := `SELECT DISTINCT asset_category_id AS ref_id FROM asset WHERE asset_category_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetAssetInUseIDs blocks deletion of assets that have any asset_transaction
// row. Any posted transaction (ACQUISITION, DEPRECIATION, REVALUATION, etc.)
// makes the asset's financial history immutable — soft-delete must be refused.
// Workspace-scoped: only rows whose workspace_id matches the context workspace
// are counted (NULL workspace_id rows are excluded, consistent with Phase 1
// tenancy rules — they would not match any non-NULL workspace predicate).
func (c *Checker) GetAssetInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `
		SELECT DISTINCT asset_id AS ref_id
		FROM asset_transaction
		WHERE asset_id = ANY($1)
		  AND ($2::text IS NULL OR workspace_id = $2)`
	return queryInUseIDsWithWorkspace(ctx, c.db, query, ids, workspaceID)
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

// GetJobInUseIDs blocks deletion of a job when any of its child rows reference it.
// Checks: job_activity (job_id), job_phase (job_id).
// job_settlement has no direct job_id column — it links via job_activity_id (excluded).
// TODO: add revenue (job_id) when revenue.job_id column is added to the schema.
func (c *Checker) GetJobInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT job_id AS ref_id FROM job_activity WHERE job_id = ANY($1) AND active = true
			UNION ALL
			SELECT job_id AS ref_id FROM job_phase WHERE job_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetJobActivityInUseIDs blocks deletion of a job_activity when its typed sub-row
// exists (activity_labor / activity_material / activity_expense keyed by activity_id)
// OR when revenue_line_item.job_activity_id references it.
func (c *Checker) GetJobActivityInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT activity_id AS ref_id FROM activity_labor WHERE activity_id = ANY($1)
			UNION ALL
			SELECT activity_id AS ref_id FROM activity_material WHERE activity_id = ANY($1)
			UNION ALL
			SELECT activity_id AS ref_id FROM activity_expense WHERE activity_id = ANY($1)
			UNION ALL
			SELECT job_activity_id AS ref_id FROM revenue_line_item WHERE job_activity_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetJobPhaseInUseIDs blocks deletion of a job_phase when any job_task references
// it via job_phase_id.
func (c *Checker) GetJobPhaseInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	query := `SELECT DISTINCT job_phase_id AS ref_id FROM job_task WHERE job_phase_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetJobTaskInUseIDs blocks deletion of a job_task when any job_activity references
// it via job_task_id.
func (c *Checker) GetJobTaskInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	query := `SELECT DISTINCT job_task_id AS ref_id FROM job_activity WHERE job_task_id = ANY($1) AND active = true`
	return queryInUseIDs(ctx, c.db, query, ids)
}

// GetJobTemplateInUseIDs blocks deletion of a job_template when any job references
// it via job_template_id, or when any job_template_phase exists (FK has no CASCADE
// — verified by proto annotation absence; operator must delete phases first).
func (c *Checker) GetJobTemplateInUseIDs(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}
	query := `
		SELECT DISTINCT ref_id FROM (
			SELECT job_template_id AS ref_id FROM job WHERE job_template_id = ANY($1) AND active = true
			UNION ALL
			SELECT job_template_id AS ref_id FROM job_template_phase WHERE job_template_id = ANY($1) AND active = true
		) AS refs`
	return queryInUseIDs(ctx, c.db, query, ids)
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
