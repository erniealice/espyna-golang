// Package reference defines the public Checker contract for FK reference
// checking. It is intentionally free of any storage-backend imports so that
// builds targeting non-postgres drivers (firestore, in-memory, etc.) can
// depend on this package without dragging in database/sql or the pq driver.
//
// The postgres implementation lives in
// contrib/postgres/reference/checker.go and satisfies this interface via a
// compile-time assertion. Future drivers (firestore, etc.) implement the
// same interface independently.
//
// Internal callers that previously imported the narrow
// internal/application/ports/infrastructure.ReferenceChecker continue to
// work unchanged — that type is now a type alias to Checker.
package reference

import "context"

// Checker provides batch FK reference checking for deletable state.
// Each method returns a map where true = ID is in use and should NOT be
// deleted. All methods are safe to call with empty input slices;
// implementations short-circuit to an empty result.
type Checker interface {
	GetLocationInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)
	GetRoleInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)
	GetCategoryInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetClientInUseIDs checks if clients are referenced in revenue or other client-linked records.
	GetClientInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetProductInUseIDs blocks deletion of products referenced by revenue line
	// items, price products, inventory items, or product_plan catalog rows
	// (Model D: product_plan.product_id is a catalog-level reference that must
	// block deletes).
	GetProductInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetProductVariantInUseIDs blocks deletion of product variants referenced by
	// inventory stock, recorded revenue lines, variant-option pivots, or catalog
	// product_plan rows. Covers the full Model D surface.
	GetProductVariantInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetProductOptionValueInUseIDs blocks deletion of product_option_value rows
	// that are referenced by product_variant_option pivots.
	GetProductOptionValueInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetProductOptionInUseIDs blocks deletion of a product_option when any of its
	// child product_option_value rows are referenced by product_variant_option.
	// Transitive check: we resolve "in use" by matching option_id → values → pivot.
	GetProductOptionInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetPlanInUseIDs checks if plans are referenced by product_plan or price_plan.
	GetPlanInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetPriceListInUseIDs checks if price lists are referenced by price products.
	GetPriceListInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetPricePlanInUseIDs checks if price plans are referenced by active subscriptions.
	// product_price_plan has ON DELETE CASCADE so it does not block deletes.
	GetPricePlanInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetPriceScheduleInUseIDs locks a PriceSchedule only when an active subscription
	// references one of its (active) price_plans. Empty/draft schedules — even those
	// with price_plan rows — remain deletable until money is flowing through them.
	// Mirrors the PricePlan pricing-lock semantics one level up.
	GetPriceScheduleInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetPlanClientScopeLockedIDs returns plan IDs whose client_id MUST NOT be
	// changed because at least one of their price_plans is referenced by an active
	// subscription. The semantic mirrors GetPricePlanInUseIDs (active-subscription-
	// only) but bubbles up the lock from PricePlan to its parent Plan.
	//
	// Only meaningful when the operator is attempting to change client_id —
	// callers that aren't editing client_id should ignore the result.
	GetPlanClientScopeLockedIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetActiveSubscriptionCountForPricePlan returns the count of active
	// subscriptions attached to a single PricePlan. Used by the N>1 confirm
	// dialog gate on UpdatePricePlan when monetary fields change on a
	// client-scoped PricePlan (see plan §3.5).
	GetActiveSubscriptionCountForPricePlan(ctx context.Context, id string) (int, error)

	GetAssetCategoryInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetAssetInUseIDs blocks deletion of assets that have any asset_transaction
	// row (ACQUISITION, DEPRECIATION, REVALUATION, etc.). Any posted transaction
	// makes the asset's financial history immutable and prevents soft-delete.
	// Workspace-scoped via asset_transaction.workspace_id.
	GetAssetInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	GetPaymentTermInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)
	GetLineInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)
	GetLocationAreaInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetEventTagInUseIDs blocks deletion of an event_tag when any active
	// event_tag_assignment row references it. Workspace-scoped via the join table.
	GetEventTagInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetSubscriptionInUseIDs checks whether subscriptions are referenced by any of
	// the tables that must block deletion: balance, invoice, license, payment,
	// subscription_attribute, revenue, and operation.job (via subscription_id column).
	GetSubscriptionInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetSupplierInUseIDs checks if suppliers are referenced by expenditures or fulfillments.
	GetSupplierInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetJobInUseIDs blocks deletion of a job when any of its child rows
	// references it: job_activity, job_phase, job_settlement (via job_activity),
	// or revenue (via revenue.job_id). Workspace-scoped via job_activity.workspace_id.
	// Note: job_settlement has no direct job_id column; it links via job_activity_id.
	// Note: revenue has no job_id column as of this writing — omitted (TODO when added).
	GetJobInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetJobActivityInUseIDs blocks deletion of a job_activity when its
	// typed sub-row exists (activity_labor / activity_material / activity_expense
	// keyed by activity_id) OR when it has been posted (posting_status='POSTED')
	// OR when revenue_line_item.job_activity_id references it.
	GetJobActivityInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetJobPhaseInUseIDs blocks deletion of a job_phase when any job_task
	// references it via job_phase_id. job_activity.job_task_id chains
	// transitively but the direct job_task FK is sufficient — deleting
	// the tasks first is the operator-required precondition.
	GetJobPhaseInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetJobTaskInUseIDs blocks deletion of a job_task when any job_activity
	// references it via job_task_id.
	GetJobTaskInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetJobTemplateInUseIDs blocks deletion of a job_template when any job
	// references it via job_template_id. job_template_phase and job_template_task
	// rows do NOT CASCADE on delete (verified: proto FK annotations carry no
	// cascade option), so job_template_phase rows also block deletion.
	GetJobTemplateInUseIDs(ctx context.Context, ids []string) (map[string]bool, error)
}
