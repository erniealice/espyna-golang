//go:build !mock_auth

// Package rbac provides the production (non-mock) ports.Authorizer used as the
// Layer-4 use-case backstop beneath authcheck.Check. It answers every
// authorization question by membership-testing the EXACT same permission-code
// set the UI permission loader computes — the postgres PermissionQuery's
// ALLOW-minus-DENY union CTE, workspace-scoped. One authoritative resolver, no
// second SQL path, no split-brain with the UI gate.
//
// Build posture: //go:build !mock_auth — the prod-build sibling of
// secondary/auth/mock/fallback.go (AllowAllAuthService). mock_auth builds keep
// secondary/auth/mock/authorization.go (//go:build mock_auth) instead, so this
// file never compiles into a mock binary and the AllowAll/Mock symbols never
// compile into a prod binary.
//
// SHADOW MODE (rollout — mantra 3/3 reliability lens): enforcement is gated
// behind AUTHZ_ENFORCE (default OFF = shadow). In SHADOW mode every check
// computes the real verdict and, on a would-be DENY, LOGS the denial (userID,
// workspaceID, code, and whether the code exists in the seeded set) but RETURNS
// allow(true,nil) so nothing breaks. This measures the authcheck-code ↔
// seed-code parity gap BEFORE enforcement goes live. In ENFORCE mode the real
// verdict is returned. See w0-design.md §2 + decisions.md (Q-AWS2 = A+C).
//
// Binding scope (risk R2 in w0-design.md §2.3 — RESOLVED in Phase 0b): the UI
// loader passes a real binding hint and scopes to ONE binding. As of Phase 0b
// the espyna context ALSO carries the session's active binding: the session
// middleware resolves the session row via LookupSessionPrincipal and stamps
// (PrincipalType, PrincipalID, ActingAsClientID, ActingAsSupplierID) onto the
// RequestIdentity (shared/identity), readable here via
// contextutil.ExtractBindingFromContext. loadCodes therefore scopes to the
// active binding whenever the ctx carries a REAL one (kind != 0 && principalID
// != ""), so a user holding multiple bindings in one workspace (e.g. CLIENT +
// OPERATOR_STAFF) gets ONLY the active binding's codes — matching the UI gate
// and CLOSING the silent-elevation surface that the prior zero-pair union
// re-opened.
//
// Fail-closed / non-session preservation: when the ctx carries NO real binding
// (kind 0 — a service-to-service / authcheck-only context, or a pre-selection
// session whose binding could not be resolved) loadCodes passes the EXACT zero
// binding pair, which still routes to the legacy union-across-all-bindings CTE.
// This is deliberate: those non-session contexts have no single binding to
// scope to, and breaking them would be a worse failure than the (now
// session-gated) union. The session path — the one that previously leaked the
// union — is no longer affected.
package rbac

import (
	"context"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// permCacheTTL mirrors the UI loader's permissionCacheTTL
// (apps/service-admin/internal/infrastructure/input/http/permission_loader.go:10)
// so the use-case backstop and the UI gate accept the same ≤5-min staleness.
const permCacheTTL = 5 * time.Minute

// authzEnforceEnvVar is the runtime flag that flips the Authorizer from shadow
// (log-but-allow) to enforce (return the real verdict). Default OFF = shadow.
// Only the exact string "true" (case-insensitive via parseEnforce) enables
// enforcement, so any unset / typo'd value keeps the safe shadow posture.
const authzEnforceEnvVar = "AUTHZ_ENFORCE"

// permCacheKey is the composite cache key. As of Phase 0b the Authorizer scopes
// to the session's ACTIVE binding when the espyna context carries one (kind,
// principalID, acting-as), so the binding fields are part of the cache key —
// otherwise a user's CLIENT-binding code set could be served to that same
// user's STAFF-binding request (or vice versa) from a stale cache entry. When
// no binding is resolved (non-session / service-to-service context) all binding
// fields are zero and the key collapses to (userID, workspaceID) — the legacy
// union-scoped entry, exactly the w0-design.md §2.6 reduction.
type permCacheKey struct {
	userID             string
	workspaceID        string
	bindingKind        int32
	bindingID          string
	actingAsClientID   string
	actingAsSupplierID string
}

type cachedPerms struct {
	codes   []string
	expires time.Time
}

// permCache is the Authorizer's own small TTL cache, mirroring DBPermissionLoader
// (permission_loader.go:81-159): sync.RWMutex + map[key]cachedPerms, read-lock /
// refresh / write-lock shape. v1 has NO invalidation hook — the backstop accepts
// ≤5-min staleness because the UI loader already enforces freshness (OD-1/OD-2).
type permCache struct {
	mu    sync.RWMutex
	store map[permCacheKey]cachedPerms
}

func newPermCache() *permCache {
	return &permCache{store: make(map[permCacheKey]cachedPerms)}
}

// PermissionAuthorizer is the production ports.Authorizer backed by the
// authoritative PermissionQuery. All boolean methods delegate to hasCode, which
// loads the user's effective code set once (through the cache) and
// membership-tests with slices.Contains.
type PermissionAuthorizer struct {
	query   securityports.PermissionQuery
	cache   *permCache
	enforce bool
}

// compile-time assertion: PermissionAuthorizer satisfies ports.Authorizer.
var _ ports.Authorizer = (*PermissionAuthorizer)(nil)

// NewPermissionAuthorizer builds the RBAC Authorizer over the registered
// PermissionQuery. The enforcement mode is read once at construction from
// AUTHZ_ENFORCE (default shadow); restart the process to change modes.
func NewPermissionAuthorizer(q securityports.PermissionQuery) *PermissionAuthorizer {
	enforce := parseEnforce(os.Getenv(authzEnforceEnvVar))
	mode := "SHADOW (log-but-allow)"
	if enforce {
		mode = "ENFORCE (real verdict)"
	}
	log.Printf("🔐 RBAC Authorizer initialised — mode=%s (AUTHZ_ENFORCE=%q)", mode, os.Getenv(authzEnforceEnvVar))
	return &PermissionAuthorizer{
		query:   q,
		cache:   newPermCache(),
		enforce: enforce,
	}
}

// parseEnforce returns true only for an explicit truthy value. Anything else —
// unset, "", "0", "false", "shadow", a typo — keeps the safe shadow posture.
func parseEnforce(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	default:
		return false
	}
}

// loadCodes returns the effective ALLOW-minus-DENY permission-code set for the
// (userID, workspaceID, binding) tuple through the cache, calling the
// authoritative PermissionQuery on a miss.
//
// Binding scope (Phase 0b — closes risk R2's silent-elevation surface): the
// session middleware stamps the session row's ACTIVE binding onto the
// RequestIdentity, which this reads via ExtractBindingFromContext. When the
// ctx carries a REAL binding (kind != 0 AND principalID != "") the lookup is
// scoped to that single binding — so a user holding multiple bindings in one
// workspace (e.g. CLIENT + OPERATOR_STAFF) gets ONLY the active binding's
// codes, matching the UI gate instead of the more-permissive union.
//
// FAIL-CLOSED / non-session preservation: when the ctx does NOT carry a real
// binding (kind 0 — service-to-service, no session, or a pre-selection session
// whose binding LookupSessionPrincipal could not resolve) we pass the EXACT
// zero binding pair, which buildPermissionQuerySQL routes to the legacy
// userRolesUnionCTE (union across every binding the user holds in this
// workspace). This deliberately preserves the prior backstop behaviour for
// non-session contexts so they are NOT broken by binding-scoping. A PARTIAL
// hint never reaches the query: we normalise any non-real binding to the full
// zero tuple, and buildPermissionQuerySQL itself fails closed (empty set) on
// any partial/ out-of-range combination it does receive.
func (a *PermissionAuthorizer) loadCodes(ctx context.Context, userID, workspaceID string) ([]string, error) {
	kind, principalID, actingAsClientID, actingAsSupplierID := contextutil.ExtractBindingFromContext(ctx)

	// Only a COMPLETE, real binding scopes the lookup. Anything else collapses
	// to the zero tuple → legacy union CTE (the documented non-session backstop).
	if !(kind != 0 && principalID != "") {
		kind, principalID, actingAsClientID, actingAsSupplierID = 0, "", "", ""
	}

	key := permCacheKey{
		userID:             userID,
		workspaceID:        workspaceID,
		bindingKind:        kind,
		bindingID:          principalID,
		actingAsClientID:   actingAsClientID,
		actingAsSupplierID: actingAsSupplierID,
	}

	a.cache.mu.RLock()
	if entry, ok := a.cache.store[key]; ok && time.Now().Before(entry.expires) {
		a.cache.mu.RUnlock()
		return entry.codes, nil
	}
	a.cache.mu.RUnlock()

	// Real binding → single-binding scope; zero tuple → userRolesUnionCTE
	// (union across every binding the user holds in this workspace), binding
	// []any{userID, workspaceID}. Returns ALLOW minus DENY, identical to the UI.
	codes, err := a.query.GetUserPermissionCodes(ctx, userID, workspaceID, kind, principalID, actingAsClientID, actingAsSupplierID)
	if err != nil {
		return nil, err
	}
	if codes == nil {
		codes = []string{}
	}

	a.cache.mu.Lock()
	a.cache.store[key] = cachedPerms{codes: codes, expires: time.Now().Add(permCacheTTL)}
	a.cache.mu.Unlock()

	return codes, nil
}

// hasCode is the single membership-test helper every boolean method delegates
// to. It loads the code set once and tests with slices.Contains.
//
// Shadow mode: when enforcement is OFF, a real-verdict DENY is LOGGED (with
// whether the code exists in the loaded set — distinguishing "user lacks an
// existing code" from "code format/casing mismatch vs the seed catalog", risk
// R6) but the method RETURNS allow(true,nil) so nothing breaks. A real lookup
// ERROR is propagated in BOTH modes — fail-closed is the safe direction and an
// error means we could not compute a verdict at all.
func (a *PermissionAuthorizer) hasCode(ctx context.Context, userID, workspaceID, code string) (bool, error) {
	codes, err := a.loadCodes(ctx, userID, workspaceID)
	if err != nil {
		// Lookup failure is propagated unconditionally — even in shadow mode we
		// surface "could not resolve permissions" rather than silently masking a
		// broken RBAC backend. authcheck.Check maps this to authorization_failed.
		log.Printf("AUTHZ_RBAC_ERROR | user=%s | workspace=%s | code=%s | error=%v", userID, workspaceID, code, err)
		return false, err
	}

	if slices.Contains(codes, code) {
		return true, nil
	}

	// Would-be DENY. We have only THIS user's effective set here, so the most
	// useful parity signal we can log is the size of that set: a would-be deny
	// against a NON-empty set is a genuine "user wasn't granted this code",
	// while a would-be deny against an EMPTY set (userSetSize=0) flags either a
	// genuinely permission-less principal OR a resolution gap (wrong workspace
	// in ctx, unseeded user) — distinguishing real denials from the
	// code-format/casing parity gap of risk R6 once these are aggregated.
	if a.enforce {
		log.Printf("AUTHZ_RBAC_DENY | mode=ENFORCE | user=%s | workspace=%s | code=%s | inUserSet=false | userSetSize=%d",
			userID, workspaceID, code, len(codes))
		return false, nil
	}

	// SHADOW: log the would-be deny, but allow so nothing breaks. This row is
	// the parity-gap measurement: every AUTHZ_RBAC_SHADOW_DENY is a site where
	// ENFORCE mode WOULD have denied. Grep these in prod logs before flipping
	// AUTHZ_ENFORCE on.
	log.Printf("AUTHZ_RBAC_SHADOW_DENY | mode=SHADOW(allowed) | user=%s | workspace=%s | code=%s | inUserSet=false | userSetSize=%d",
		userID, workspaceID, code, len(codes))
	return true, nil
}

// HasPermission checks the permission against the workspace in the espyna
// context. `permission` is the already-built "<entity>:<action>" string from
// ports.EntityPermission (authcheck.go:60).
func (a *PermissionAuthorizer) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	ws := contextutil.ExtractWorkspaceIDFromContext(ctx)
	return a.hasCode(ctx, userID, ws, permission)
}

// HasGlobalPermission is v1-identical to HasPermission (the interface doc at
// authorization.go:12-13 says "equivalent to HasPermission"). It deliberately
// uses the ctx workspace, NOT workspaceID="" — the union CTE filters
// wu.workspace_id = $2, so an empty workspace would return no roles → spurious
// deny (risk R5 / OD-4). OD-4 verified: superadmin-001 is seeded as a normal
// workspace_user under default-workspace (wu-superadmin → role-admin, 841 ALLOW
// grants, zero DENY in role_permission.csv), so it resolves through the
// workspace-scoped union CTE and is NOT silently denied as long as the ctx
// workspace matches the grant's workspace.
func (a *PermissionAuthorizer) HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error) {
	ws := contextutil.ExtractWorkspaceIDFromContext(ctx)
	return a.hasCode(ctx, userID, ws, permission)
}

// HasPermissionInWorkspace is the only method given an explicit workspaceID;
// it is preferred over the ctx workspace when the caller supplies one.
func (a *PermissionAuthorizer) HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error) {
	return a.hasCode(ctx, userID, workspaceID, permission)
}

// GetUserRoles is stubbed empty for v1: the union CTE returns codes, not role
// names, and authcheck.Check never calls this (OD-6). Confirmed no other
// ports.Authorizer consumer needs roles.
func (a *PermissionAuthorizer) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}

// GetUserRolesInWorkspace is stubbed empty for v1 (OD-6) — same rationale as
// GetUserRoles.
func (a *PermissionAuthorizer) GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error) {
	return []string{}, nil
}

// GetUserWorkspaces is stubbed empty for v1 (OD-6) — not consulted by
// authcheck.Check.
func (a *PermissionAuthorizer) GetUserWorkspaces(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}

// GetUserPermissionCodes is the UI-adaptation pass-through (authorization.go:27-29).
// It returns the raw code set for the ctx workspace, scoped to the session's
// active binding when the ctx carries one (loadCodes reads it via
// ExtractBindingFromContext) and falling back to the zero-pair union otherwise —
// shadow/enforce mode does NOT apply here (this is a bulk read, not a gate).
func (a *PermissionAuthorizer) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	ws := contextutil.ExtractWorkspaceIDFromContext(ctx)
	return a.loadCodes(ctx, userID, ws)
}

// IsEnabled MUST return true. authcheck.Check short-circuits
// `if !IsEnabled() { return nil }` (allow all, authcheck.go:49). If this ever
// returned false, all ~957 use-case sites would silently allow — the exact
// AllowAll failure mode this whole wave removes. The boot-fail guard in
// getServices (usecases.go) runs BEFORE this and never relies on IsEnabled() to
// fail closed (risk R3). NOTE: IsEnabled() is about whether authcheck RUNS at
// all; shadow mode (which allows on a would-be deny) is a separate, internal
// concern — shadow mode still RUNS the real lookup, so IsEnabled() is true in
// both shadow and enforce.
func (a *PermissionAuthorizer) IsEnabled() bool {
	return true
}
