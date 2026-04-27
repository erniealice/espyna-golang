package infrastructure

import "context"

// ReferenceChecker is an application-layer port over the postgres-side
// reference.Checker. Use cases that need to query "is this row referenced
// elsewhere?" depend on this interface so they remain provider-agnostic.
//
// The concrete implementation lives in
// `contrib/postgres/reference/checker.go`. Tests can supply a stub.
//
// Methods are added on demand — only what use cases need is exposed here.
// All methods are safe to call with empty input slices; implementations
// short-circuit to an empty result.
type ReferenceChecker interface {
	// GetPlanClientScopeLockedIDs returns plan IDs whose client_id MUST NOT be
	// changed because at least one of their price_plans is referenced by an
	// active subscription. Used by UpdatePlan to gate client_id reassignment
	// (plan §3.1 — 20260427-plan-client-scope).
	GetPlanClientScopeLockedIDs(ctx context.Context, ids []string) (map[string]bool, error)

	// GetActiveSubscriptionCountForPricePlan returns the count of active
	// subscriptions attached to a single PricePlan. Used by UpdatePricePlan
	// to gate the N>1 confirm dialog when monetary fields change on a
	// client-scoped PricePlan (plan §3.5).
	GetActiveSubscriptionCountForPricePlan(ctx context.Context, id string) (int, error)
}

// NewNoOpReferenceChecker returns a checker that reports nothing in use.
// Useful as a sane default in non-postgres providers and tests that don't
// care about reference checks.
func NewNoOpReferenceChecker() ReferenceChecker {
	return &noOpReferenceChecker{}
}

type noOpReferenceChecker struct{}

func (n *noOpReferenceChecker) GetPlanClientScopeLockedIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOpReferenceChecker) GetActiveSubscriptionCountForPricePlan(_ context.Context, _ string) (int, error) {
	return 0, nil
}
