package job_phase

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// Approval action verbs. The view-layer gate (P3) and these use-case gates MUST
// cite the SAME verb (copya.md gate discipline — do not repeat job_activity's
// submit-gated-on-update mismatch). permission codes: job_phase:<verb>.
const (
	actionSubmit  = "submit"
	actionVerify  = "verify"
	actionPublish = "publish"
	actionReturn  = "return"
)

// transitionRepositories / transitionServices are shared by all four transition
// use cases. Transactor is REQUIRED (fail-closed, no non-transactional fallback);
// Authorizer powers the submit admin-override resolution.
type transitionRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}

type transitionServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// requireTransaction fail-closes when transaction support is absent. Approval
// transitions take FOR UPDATE locks that only serialize inside a transaction, so
// there is NO non-transactional fallback (codex "Exact lock and state contract").
func (s transitionServices) requireTransaction() error {
	if s.Transactor == nil || !s.Transactor.SupportsTransactions() {
		return errors.New("job_phase approval: transaction support is required for approval transitions")
	}
	return nil
}

// validateSheetRequest checks the sheet identity (template id + phase id).
func validateSheetRequest(templateID, phaseID string) error {
	if templateID == "" || phaseID == "" {
		return errors.New("job_phase approval: job_template_id and job_template_phase_id are required")
	}
	return nil
}

// strictAuthorizer is the deny-capable authorization capability the phase-approval
// transitions REQUIRE (codex FIX-FIRST 1 / §2 CRITICAL). The production RBAC
// Authorizer defaults to SHADOW mode, which returns allow(true) on a real DENY as
// a rollout measurement posture — that is unsafe for these security-critical
// verbs and for the publish-derived admin override, both of which would then be
// granted to a would-be-denied caller. This optional interface exposes a strict,
// deny-capable verdict independent of shadow, plus whether strict enforcement is
// active at all. The production *rbac.PermissionAuthorizer implements it; a nil
// authorizer, the mock AllowAll, or the no-op authorizer do NOT, so those builds
// fail closed (no override, transition refused) rather than silently allowing.
type strictAuthorizer interface {
	// StrictEnforcement reports whether AUTHZ_ENFORCE is active (real verdicts on
	// a would-be deny). When false the whole shadow posture allows-on-deny, so the
	// transitions must refuse to run.
	StrictEnforcement() bool
	// HasPermissionStrict returns the REAL allow/deny verdict independent of
	// shadow mode (a genuine deny returns false, never a masked allow). Cached
	// (≤5-min TTL) — used only as the pre-transaction fast pre-filter.
	HasPermissionStrict(ctx context.Context, userID, permission string) (bool, error)
	// HasPermissionStrictFresh is the AUTHORITATIVE deny-capable verdict: it
	// BYPASSES the TTL cache and reads on the ambient transaction executor, so a
	// permission revoked between request receipt and commit is honoured (codex P3
	// §A1). The four transition verb checks AND the publish-derived admin override
	// take their decisive verdict from this, INSIDE the locked transition tx.
	HasPermissionStrictFresh(ctx context.Context, userID, permission string) (bool, error)
}

// requireStrictAuthorizer fail-closes unless a strict-ENFORCING authorizer backs
// these gates. "refuse to expose phase-approval transitions unless strict
// authorization enforcement is active" (codex §2 fix). Returns the strict
// authorizer used for the verb re-check and the admin-override resolution.
func requireStrictAuthorizer(authz ports.Authorizer) (strictAuthorizer, error) {
	sa, ok := authz.(strictAuthorizer)
	if !ok {
		return nil, errors.New("job_phase approval: a strict-enforcing authorizer is required — shadow-mode/mock/no-op RBAC cannot gate approval transitions (fail closed)")
	}
	if !sa.StrictEnforcement() {
		return nil, errors.New("job_phase approval: strict authorization enforcement (AUTHZ_ENFORCE) must be active to run approval transitions (fail closed)")
	}
	return sa, nil
}

// requireStrictVerb strictly re-checks a job_phase:<verb> capability with a
// deny-capable verdict (independent of shadow). Used PRE-transaction as a fast
// fail-closed pre-filter (cached ≤5-min); the AUTHORITATIVE verdict is the
// in-transaction requireStrictVerbFresh below. Fails closed on a missing user, a
// lookup error, or a real deny.
func requireStrictVerb(ctx context.Context, sa strictAuthorizer, verb string) error {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return errors.New("job_phase approval: no trusted user in context (fail closed)")
	}
	ok, err := sa.HasPermissionStrict(ctx, userID, "job_phase:"+verb)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("job_phase approval: caller lacks job_phase:" + verb + " (strict verdict) — denied")
	}
	return nil
}

// requireStrictVerbFresh is the AUTHORITATIVE in-transaction verb gate (codex P3
// §A1): it re-checks job_phase:<verb> with a deny-capable verdict that BYPASSES
// the TTL cache and reads on the ambient transaction executor, so a revocation
// committed between request receipt and this point denies the transition. Called
// INSIDE ExecuteInTransaction, before the repository transition runs. Fails closed
// on a missing user, a lookup error, or a real deny.
func requireStrictVerbFresh(ctx context.Context, sa strictAuthorizer, verb string) error {
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return errors.New("job_phase approval: no trusted user in context (fail closed)")
	}
	ok, err := sa.HasPermissionStrictFresh(ctx, userID, "job_phase:"+verb)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("job_phase approval: caller lacks job_phase:" + verb + " (fresh in-tx verdict) — denied")
	}
	return nil
}

// resolveAdminOverride reports whether the acting user holds the D4 admin
// override — the separately proven job_phase:publish authority — using a STRICT,
// deny-capable verdict (never shadow's allow-on-deny). nil authorizer, missing
// user, lookup error, or a real deny all yield NO override (fail closed); the
// adapter's strict D7 all-task ownership check then runs. MUST be resolved INSIDE
// the transition transaction so a permission revocation between request receipt
// and commit is honoured — never a pre-tx allow bit carried across the boundary
// (codex §1 MEDIUM).
func resolveAdminOverride(ctx context.Context, sa strictAuthorizer) bool {
	if sa == nil {
		return false
	}
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return false
	}
	// FRESH, cache-bypassing, ambient-tx verdict (codex P3 §A1): a publish grant
	// revoked mid-request must not still mint the admin override from a warm cache.
	has, err := sa.HasPermissionStrictFresh(ctx, userID, "job_phase:publish")
	if err != nil {
		return false
	}
	return has
}
