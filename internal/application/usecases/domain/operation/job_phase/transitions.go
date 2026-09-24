package job_phase

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/shared/identity"
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

// Approval-policy capability codes (plan 20260924-approval-role-workflow D3/D4).
// Generic codes; business vocabulary lives only in role names and labels.
const (
	permApprovalScopeWorkspace = "approval_scope:workspace"
	permVerifyOwn              = "job_phase:verify_own"
	permPublishUnverified      = "job_phase:publish_unverified"
)

// Principal kinds, mirroring esqyma domain.entity.v1.PrincipalType (the values
// the session binding stamps on identity.PrincipalType; contrib/postgres
// principalscope carries the same constants — the application layer cannot
// import that adapter package).
const (
	principalTypeOperatorOwner int32 = 1
	principalTypeOperatorStaff int32 = 2
	principalTypeStaff         int32 = 7
)

// sessionKind returns the acting session's principal kind (0 when absent). A
// malformed binding (kind set without a principal id) reports 0 so it can never
// take the staff or operator branch (review wave-2 #1).
func sessionKind(ctx context.Context) int32 {
	if id, ok := identity.FromContext(ctx); ok && id != nil && id.PrincipalID != "" {
		return id.PrincipalType
	}
	return 0
}

// hasFresh is the fail-closed fresh strict verdict for one code: nil authorizer,
// missing user, lookup error, or a real deny all yield false.
func hasFresh(ctx context.Context, sa strictAuthorizer, code string) bool {
	if sa == nil {
		return false
	}
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return false
	}
	has, err := sa.HasPermissionStrictFresh(ctx, userID, code)
	if err != nil {
		return false
	}
	return has
}

// resolveAdminOverride reports whether the acting user may SKIP the D7 all-task
// ownership check on submit, using STRICT, FRESH, in-transaction verdicts (never
// shadow's allow-on-deny, never a pre-tx allow bit — codex §1 MEDIUM / P3 §A1).
//
//   - STAFF session (kind 7): only approval_scope:workspace (the Principal). Since
//     2026-09-24 job_phase:publish is staff-holdable, and publish must no longer
//     imply "submit any sheet" in the staff persona (plan D3).
//   - operator session (kinds 1/2): job_phase:publish (the original D4 admin
//     override, preserved — owner D-RUN-3) or approval_scope:workspace.
//   - any other kind (unresolved 0, portal 3-6): never.
//
// Anything else → false, and the adapter's strict ownership check runs.
func resolveAdminOverride(ctx context.Context, sa strictAuthorizer) bool {
	switch sessionKind(ctx) {
	case principalTypeStaff:
		return hasFresh(ctx, sa, permApprovalScopeWorkspace)
	case principalTypeOperatorOwner, principalTypeOperatorStaff:
		return hasFresh(ctx, sa, permApprovalScopeWorkspace) || hasFresh(ctx, sa, "job_phase:publish")
	default:
		// Unresolved (0) and portal (3-6) kinds never get the override.
		return false
	}
}

// resolveScopeDecision resolves the approval-policy capabilities for a verify /
// publish / return transition INSIDE its transaction (fresh strict verdicts).
// wantVerifyOwn / wantPublishUnverified limit the lookups to the ones the verb
// consults. Every field fails closed to false.
func resolveScopeDecision(ctx context.Context, sa strictAuthorizer, wantVerifyOwn, wantPublishUnverified bool) approvalctx.ScopeDecision {
	switch sessionKind(ctx) {
	case principalTypeStaff, principalTypeOperatorOwner, principalTypeOperatorStaff:
	default:
		return approvalctx.ScopeDecision{} // unresolved / portal / malformed: nothing
	}
	d := approvalctx.ScopeDecision{WorkspaceScope: hasFresh(ctx, sa, permApprovalScopeWorkspace)}
	if wantVerifyOwn {
		d.VerifyOwn = hasFresh(ctx, sa, permVerifyOwn)
	}
	if wantPublishUnverified {
		d.PublishUnverified = hasFresh(ctx, sa, permPublishUnverified)
	}
	return d
}
