// Package approvalctx carries the server-resolved authorization decision for a
// job-phase approval transition from the application use-case layer down to the
// postgres adapter, via context.
//
// Why context: the transition RPC requests carry ONLY the sheet identity
// (job_template_id, job_template_phase_id) — never an authz field. The admin
// override (D4: the acting user holds job_phase:publish authority) can only be
// resolved by the application-layer RBAC Authorizer, but it must be honoured
// INSIDE the adapter's locked transition transaction (after the full sheet set
// is locked). Context is the same in-process, non-spoofable channel that already
// carries identity — an HTTP request cannot inject a Go context value.
//
// INTERNAL by design (codex §2 hardening): this package lives under
// internal/application so ONLY code within the espyna module tree can mint a
// decision — an application use case (the intended minter) or espyna's own
// postgres adapter (the intended reader). A downstream app / HTTP handler /
// third-party package (all separate modules NOT rooted at espyna's internal
// tree) cannot import it, so the "only the use case sets AdminOverride"
// invariant is structural, not merely conventional. WithSubmitDecision was
// previously exported from shared/approvalctx where any in-process caller could
// mint an override.
//
// Fail-closed default: a missing decision (SubmitDecisionFromContext ok==false)
// means AdminOverride=false, so the adapter enforces the D7 all-task staff
// ownership check. Only the submit use case sets a decision; only the adapter's
// submit path reads it.
package approvalctx

import "context"

type contextKey struct{}

// spawnKey marks a context as originating from the trusted W-SPAWN materializer
// seam. Only the internal spawn seam mints it; the postgres job_phase adapter
// reads it to allow a template-backed generic Create (template_phase_id set) that
// would otherwise be rejected as a membership-phantom attack (codex §4 CRITICAL:
// membership anchors must not be freely creatable outside the parent-locked spawn
// seam). Being INTERNAL, no downstream module can forge it.
type spawnKey struct{}

// WithTrustedSpawn marks ctx as the trusted spawn seam (allowed to create a
// template-backed phase under the parent mutex).
func WithTrustedSpawn(ctx context.Context) context.Context {
	return context.WithValue(ctx, spawnKey{}, true)
}

// IsTrustedSpawn reports whether ctx was minted by the trusted spawn seam.
func IsTrustedSpawn(ctx context.Context) bool {
	v, _ := ctx.Value(spawnKey{}).(bool)
	return v
}

// recomputeKey marks a context as the trusted submit-time freshness-barrier
// recompute seam (codex P3 §A3). Only grade_compute.SheetRecompute — invoked
// EXCLUSIVELY by the job_phase submit transition post-lock/pre-flip, inside the
// transition transaction — mints it. The phase/job/task-outcome summary postgres
// reads honour it to (a) read on the AMBIENT transaction executor (so the job
// roll-up sees the phase summaries written earlier in the SAME tx) and (b) drop
// the per-request STAFF issued/recorded row scope (so a teacher submitting sees
// the FULL sheet's inputs, not just their own — the barrier runs as the system).
// Being INTERNAL, no downstream module can forge it; the reader additionally
// requires an ambient *sql.Tx before honouring it, so it can never become a
// pool-based staff-scope bypass.
type recomputeKey struct{}

// WithTrustedRecompute marks ctx as the trusted recompute seam.
func WithTrustedRecompute(ctx context.Context) context.Context {
	return context.WithValue(ctx, recomputeKey{}, true)
}

// IsTrustedRecompute reports whether ctx was minted by the trusted recompute seam.
func IsTrustedRecompute(ctx context.Context) bool {
	v, _ := ctx.Value(recomputeKey{}).(bool)
	return v
}

// SubmitDecision is the resolved authorization decision for a submit transition.
type SubmitDecision struct {
	// AdminOverride is true when the acting user holds job_phase:publish
	// authority (the D4 admin override: an admin can submit for an absent
	// teacher / a multi-deliverer sheet). When true the adapter SKIPS the D7
	// all-task staff-ownership requirement; when false the adapter enforces it.
	// Resolved with a STRICT (deny-capable) verdict INSIDE the transition
	// transaction — never carried across the tx boundary as a stale allow bit.
	AdminOverride bool
}

// WithSubmitDecision returns a context carrying the resolved submit decision.
func WithSubmitDecision(ctx context.Context, d SubmitDecision) context.Context {
	return context.WithValue(ctx, contextKey{}, d)
}

// SubmitDecisionFromContext reads the submit decision. ok==false means no
// decision was set — the caller MUST treat that as the fail-closed default
// (no admin override → enforce ownership).
func SubmitDecisionFromContext(ctx context.Context) (d SubmitDecision, ok bool) {
	d, ok = ctx.Value(contextKey{}).(SubmitDecision)
	return d, ok
}
