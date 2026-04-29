package context

import "context"

// SpawnJobsOverrideKey is the public plain-string context key that carries the
// operator-facing "Spawn Jobs on Create" toggle value (per
// auto-spawn-jobs-from-subscription plan §5.1) from the view-layer subscription
// create handler down through CreateSubscriptionUseCase into the
// JobTemplateInstantiator port.
//
// Plain-string (not a typed contextKey) on purpose: sibling packages outside
// the espyna module (centymo-golang) cannot import this `internal` package.
// They write the same key directly via context.WithValue, mirroring the
// existing cross-package convention used for "businessType" (see business.go).
//
// Tri-state semantics on the stored *bool:
//   - missing key  → unset (use the legacy default of true so existing flows
//     keep spawning Jobs when a JobTemplate resolves)
//   - *bool true   → operator explicitly opted in (or default-on)
//   - *bool false  → operator explicitly opted out (must skip spawn)
//
// Using *bool avoids the bool-false-vs-absent ambiguity — relevant here
// because an HTML checkbox sends nothing when unchecked, so the centymo form
// branch already distinguishes presence.
const SpawnJobsOverrideKey = "spawn_jobs_override"

// WithSpawnJobsOverride attaches the operator's spawn-jobs toggle decision
// (true = opt-in, false = opt-out) to ctx so downstream use cases can read it.
func WithSpawnJobsOverride(ctx context.Context, spawnJobs bool) context.Context {
	v := spawnJobs
	return context.WithValue(ctx, SpawnJobsOverrideKey, &v)
}

// ExtractSpawnJobsOverride returns the operator's toggle decision and a bool
// indicating whether it was set. When `set == false`, callers should fall back
// to their own default (typically true, matching the pre-toggle behavior).
func ExtractSpawnJobsOverride(ctx context.Context) (spawnJobs bool, set bool) {
	if v, ok := ctx.Value(SpawnJobsOverrideKey).(*bool); ok && v != nil {
		return *v, true
	}
	return false, false
}
