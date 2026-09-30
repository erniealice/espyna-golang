package domain

import (
	"context"

	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

// Charge policy hand-written ports (20260927-usage-and-pass-through-charges, Slice A).
// The charge_policy use cases type-assert these on the generated repositories, mirroring
// the rating_description_set lifecycle-port pattern. Every method requires a trusted
// workspace on ctx and fails closed (not-found) for a foreign or missing row.

// ChargePolicyLocker takes the policy row FOR UPDATE inside the ambient transaction and
// returns its fresh state. Serializes retire / clone / approve / delete on one policy.
type ChargePolicyLocker interface {
	LockChargePolicyForUpdate(ctx context.Context, id string) (*policypb.ChargePolicy, error)
}

// ChargePolicyVersionLocker takes the version row FOR UPDATE inside the ambient transaction
// and returns its fresh state. Draft-only component/posting writes and approval lock the
// parent version first so an approval and a concurrent draft edit serialize.
type ChargePolicyVersionLocker interface {
	LockChargePolicyVersionForUpdate(ctx context.Context, id string) (*versionpb.ChargePolicyVersion, error)
}

// ChargePolicyReferenceReader answers which of the given policy ids are referenced by at
// least one product_price_plan row of the caller's workspace.
type ChargePolicyReferenceReader interface {
	ChargePolicyIDsReferencedByPricePlans(ctx context.Context, policyIDs []string) (map[string]bool, error)
}

// PricePlanLocker serializes the charge policy opt-in guard against a concurrent write of the
// other side (parent price plan billing_kind / amount_basis vs. a line opting in). price_plan
// carries no workspace_id, so the adapter locks the row's workspace-bearing parent plan FOR UPDATE
// (workspace predicate inside the lock statement, C5) inside the ambient transaction. Fails closed
// without a transaction or trusted workspace; a missing or foreign-workspace price plan wraps
// ErrLockedRowNotFound.
type PricePlanLocker interface {
	LockPricePlanForUpdate(ctx context.Context, id string) error
}
