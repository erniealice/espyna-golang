package http

// loaders.go — the per-request sidebar/RBAC loaders relocated into espyna
// consumer/http in Model-A Wave 3, merging the service-admin app's
// permission_loader.go + principal_loader.go + user_loader.go. Strong consumer
// types stay in-package (consumed by view_adapter in the SAME package); the
// proto-backed workspace loader stays app/entydad-side and reaches espyna only
// through the WorkspaceLoader interface (view_adapter.go).
//
// PrincipalType is aliased in view_adapter.go (= pyezarender.PrincipalType); its
// constants + String()/HomeRoute() methods are available package-wide.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	"github.com/erniealice/pyeza-golang/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Principal resolution (was principal_loader.go)
// ─────────────────────────────────────────────────────────────────────────────

// ErrNoBinding is returned by ResolveBindingInWorkspace when the user has
// no active binding in the requested workspace. Callers (workspace_path
// middleware) must treat this as a hard reject — the user is not allowed
// to access the workspace via URL navigation.
//
// Added 2026-05-22 per Phase P3+P7 of docs/plan/20260521-workspace-keyed-routing.
var ErrNoBinding = errors.New("principal_loader: no active binding in workspace")

// ErrAmbiguousBinding is returned by ResolveBindingInWorkspace when the user
// holds multiple active bindings in the requested workspace and the session
// principal hint does not uniquely identify one. Callers (workspace_path
// middleware, A3 — WKR-P0-3) must redirect to the workspace-role picker
// (/auth/select-workspace-role) rather than auto-electing by privilege.
//
// Added 2026-05-23 per Compartment A3 of docs/plan/20260522-codex-redteam-sweep.
var ErrAmbiguousBinding = errors.New("principal_loader: ambiguous binding (multiple bindings, no session principal match)")

// ActingAsTarget is one target a delegate may act for. A single
// CLIENT_DELEGATE principal may have N>1 acting-as options (one per
// DelegateClient row); the chooser picks one before entering the portal.
type ActingAsTarget struct {
	// ID is the underlying party id (client_id or supplier_id).
	ID string
	// WorkspaceID is the workspace the party lives in (resolved from the
	// client.workspace_id or supplier.workspace_id back-edge).
	WorkspaceID string
	// DisplayName is a human label (e.g. "Jane Smith").
	DisplayName string
}

// Principal is one resolved binding a User holds. The auth flow uses the
// list of Principals to decide post-login routing: 0 = no access, 1 = auto
// route, 2+ = present chooser.
//
// PrincipalID identifies the underlying grant row (WorkspaceUser.id /
// ClientPortalGrant.id / SupplierPortalGrant.id / Delegate.id) — this is
// the value stored in session.principal_id when the session row is
// established or rotated.
type Principal struct {
	Type            PrincipalType
	PrincipalID     string
	WorkspaceID     string
	DisplayName     string
	ActingAsTargets []ActingAsTarget // only populated for delegate principals
}

// HomeRoute returns the URL this principal should land on after login or
// after a switch. For a one-target delegate, the URL pre-selects the
// acting-as target via query parameter so the user auto-enters.
func (p Principal) HomeRoute() string {
	base := p.Type.HomeRoute()
	switch p.Type {
	case PrincipalTypeClientDelegate:
		if len(p.ActingAsTargets) == 1 {
			return fmt.Sprintf("%s?acting_as_client_id=%s", base, p.ActingAsTargets[0].ID)
		}
		return base + "select"
	case PrincipalTypeSupplierDelegate:
		if len(p.ActingAsTargets) == 1 {
			return fmt.Sprintf("%s?acting_as_supplier_id=%s", base, p.ActingAsTargets[0].ID)
		}
		return base + "select"
	}
	return base
}

// DelegateActingAsResolved reports whether a principal about to be handed to
// executePrincipalSwitch carries an UNAMBIGUOUS acting-as identity.
//
// codex RBC#1 — High-1 (2026-06-02). The sidebar/URL path closes the
// multi-target-delegate hole inside pickBindingForSession (ResolveBindingInWorkspace),
// but the SAME delegate bug class lives on the login auto-route and the
// chooser POST: both forward a resolved delegate Principal to
// executePrincipalSwitch with NO explicit acting-as target.
func DelegateActingAsResolved(p Principal, actingAsClientID, actingAsSupplierID string) bool {
	switch p.Type {
	case PrincipalTypeClientDelegate, PrincipalTypeSupplierDelegate:
		// fall through to the multi-target check below.
	default:
		return true // non-delegate: acting-as not applicable.
	}

	if len(p.ActingAsTargets) <= 1 {
		return true
	}

	want := strings.TrimSpace(actingAsClientID)
	if p.Type == PrincipalTypeSupplierDelegate {
		want = strings.TrimSpace(actingAsSupplierID)
	}
	if want == "" {
		return false
	}
	for _, t := range p.ActingAsTargets {
		if t.ID == want {
			return true
		}
	}
	return false
}

// PrincipalLoader is the read-side port for principal resolution.
// Currently has one implementation: DBPrincipalLoader.
type PrincipalLoader interface {
	Resolve(ctx context.Context, userID string) ([]Principal, error)
	IsEnabled() bool
}

// PrincipalResolveFunc is the function signature for resolving all of a
// user's principals across workspaces. Injected by the composition layer.
type PrincipalResolveFunc func(ctx context.Context, userID string) ([]Principal, error)

// BindingResolveFunc is the function signature for resolving a single
// binding in one workspace, applying the A3 resolution policy. Injected
// by the composition layer.
type BindingResolveFunc func(
	ctx context.Context,
	userID, workspaceID string,
	sessionPrincipalKind PrincipalType,
	sessionPrincipalID string,
) (*Principal, error)

// DBPrincipalLoader resolves a User's principal bindings by delegating to
// injected resolver functions. The *sql.DB dependency has been removed — all
// SQL lives in the postgres adapter at packages/espyna-golang/contrib/postgres/
// internal/adapter/entity/principal_resolver.go.
//
// Migrated 2026-06-14 per no-direct-sql-rule (docs/wiki/articles/
// no-direct-sql-rule.md).
type DBPrincipalLoader struct {
	resolveFn PrincipalResolveFunc
	bindingFn BindingResolveFunc
}

// NewDBPrincipalLoader creates a loader backed by injected resolver functions.
// Pass nil functions to disable — IsEnabled() will report false.
func NewDBPrincipalLoader(
	resolveFn PrincipalResolveFunc,
	bindingFn BindingResolveFunc,
) *DBPrincipalLoader {
	return &DBPrincipalLoader{
		resolveFn: resolveFn,
		bindingFn: bindingFn,
	}
}

// IsEnabled reports whether the loader has resolver functions.
func (l *DBPrincipalLoader) IsEnabled() bool {
	return l != nil && l.resolveFn != nil
}

// Resolve returns every active principal binding the given user has across
// the four grant tables. Returns an empty (non-nil) slice when no bindings
// exist — callers route empty results to /auth/no-access.
func (l *DBPrincipalLoader) Resolve(ctx context.Context, userID string) ([]Principal, error) {
	if !l.IsEnabled() {
		return nil, nil
	}
	if strings.TrimSpace(userID) == "" {
		return nil, nil
	}
	return l.resolveFn(ctx, userID)
}

// ResolveBindingInWorkspace returns the user's active binding in a specific
// workspace, applying the A3 resolution policy.
func (l *DBPrincipalLoader) ResolveBindingInWorkspace(
	ctx context.Context,
	userID, workspaceID string,
	sessionPrincipalKind PrincipalType,
	sessionPrincipalID string,
) (*Principal, error) {
	if !l.IsEnabled() || l.bindingFn == nil {
		return nil, ErrNoBinding
	}
	userID = strings.TrimSpace(userID)
	workspaceID = strings.TrimSpace(workspaceID)
	sessionPrincipalID = strings.TrimSpace(sessionPrincipalID)
	if userID == "" || workspaceID == "" {
		return nil, ErrNoBinding
	}
	return l.bindingFn(ctx, userID, workspaceID, sessionPrincipalKind, sessionPrincipalID)
}

// ── Pure functions kept for test coverage ───────────────────────────────────
// pickBindingForSession, groupDelegateRows, delegateTargetRow, and
// formatDelegateLabel remain as pure functions tested by
// principal_loader_test.go. The business logic has been MOVED to the espyna
// use case (serviceauth.PickBindingForSession) but these local wrappers
// preserve the existing test suite which exercises the local Principal types.

// pickBindingForSession applies the A3 resolution policy to a list of
// already-enumerated bindings. Pure function — exercises the same decision
// matrix as the espyna use case's PickBindingForSession but operates on
// local types.
func pickBindingForSession(
	bindings []Principal,
	sessionPrincipalKind PrincipalType,
	sessionPrincipalID string,
) (*Principal, error) {
	if len(bindings) == 0 {
		return nil, ErrNoBinding
	}

	if sessionPrincipalKind != PrincipalTypeUnspecified && sessionPrincipalID != "" {
		for i := range bindings {
			if bindings[i].Type == sessionPrincipalKind &&
				bindings[i].PrincipalID == sessionPrincipalID {
				if len(bindings[i].ActingAsTargets) > 1 {
					return nil, ErrAmbiguousBinding
				}
				return &bindings[i], nil
			}
		}
		return nil, ErrAmbiguousBinding
	}

	if len(bindings) == 1 {
		if len(bindings[0].ActingAsTargets) > 1 {
			return nil, ErrAmbiguousBinding
		}
		return &bindings[0], nil
	}

	return nil, ErrAmbiguousBinding
}

// delegateTargetRow is one scanned (delegate.id, target) row. Kept for
// test coverage of groupDelegateRows.
type delegateTargetRow struct {
	DelegateID  string
	TargetID    string
	WorkspaceID string
	DisplayName string
}

// groupDelegateRows is the pure group-break that turns delegate join rows
// (ordered BY delegate.id) into one Principal per delegate.id. Kept for
// test coverage.
func groupDelegateRows(kind PrincipalType, roleLabel string, rows []delegateTargetRow) []Principal {
	fallbackName := "Client"
	if kind == PrincipalTypeSupplierDelegate {
		fallbackName = "Supplier"
	}

	out := make([]Principal, 0, 2)
	var (
		curDelegateID string
		curTargets    []ActingAsTarget
	)
	flush := func() {
		if curDelegateID != "" && len(curTargets) > 0 {
			out = append(out, Principal{
				Type:            kind,
				PrincipalID:     curDelegateID,
				WorkspaceID:     curTargets[0].WorkspaceID,
				DisplayName:     formatDelegateLabel("", roleLabel, curTargets),
				ActingAsTargets: curTargets,
			})
		}
	}
	for _, row := range rows {
		if row.DelegateID != curDelegateID {
			flush()
			curDelegateID = row.DelegateID
			curTargets = nil
		}
		display := strings.TrimSpace(row.DisplayName)
		if display == "" {
			display = fallbackName
		}
		curTargets = append(curTargets, ActingAsTarget{
			ID:          row.TargetID,
			WorkspaceID: strings.TrimSpace(row.WorkspaceID),
			DisplayName: display,
		})
	}
	flush()
	return out
}

// formatDelegateLabel builds a human-readable display string for a delegate
// principal card. Kept for test coverage.
func formatDelegateLabel(userName, role string, targets []ActingAsTarget) string {
	switch len(targets) {
	case 0:
		return role
	case 1:
		return fmt.Sprintf("%s · %s", targets[0].DisplayName, role)
	default:
		return fmt.Sprintf("%d targets · %s", len(targets), role)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Permission loading (was permission_loader.go)
// ─────────────────────────────────────────────────────────────────────────────

const permissionCacheTTL = 5 * time.Minute

type cachedPerms struct {
	codes   []string
	expires time.Time
}

// PermissionQuery is the narrow RBAC permission-code lookup contract
// consumed by DBPermissionLoader. It is satisfied by a thin closure that
// invokes the service-driven use case at
// uc.Service.Security.GetUserPermissionCodes — see container.go for the
// wiring.
//
// Binding-scoped lookup (A2 / WKR-P0-2 — 2026-05-24): the bindingKind
// (PrincipalType integer) and bindingID hint flow through to the underlying
// PermissionQuery so the adapter can restrict the grant chain to the SINGLE
// selected binding row instead of UNIONing across every binding the user
// holds in the workspace.
//
// Delegate target scoping (A2-followup / codex A2-P0-1 — 2026-05-24):
// for CLIENT_DELEGATE / SUPPLIER_DELEGATE bindings the actingAsClientID /
// actingAsSupplierID identifies the per-target delegate_client /
// delegate_supplier row the delegate is currently acting through. These
// are ignored for non-delegate kinds; when a delegate kind is supplied
// without its acting-as id, the postgres adapter fails closed (empty
// permission set).
//
// Fail-closed posture (codex A2-P1-1): only the EXACT zero pair
// (bindingKind=Unspecified, bindingID="") triggers the legacy
// union-across-all-bindings backend path. Partial / malformed hints
// (kind set without id, id set with UNSPECIFIED, out-of-range kind)
// return an empty permission set from the underlying adapter.
type PermissionQuery interface {
	GetUserPermissionCodes(
		ctx context.Context,
		userID, workspaceID string,
		bindingKind PrincipalType,
		bindingID string,
		actingAsClientID, actingAsSupplierID string,
	) ([]string, error)
}

// DBPermissionLoader caches permission-code lookups over an injected
// PermissionQuery — usually backed by the
// uc.Service.Security.GetUserPermissionCodes use case.
//
// Cache key composition (A2 — 2026-05-24, extended A2-followup):
// the cache key is a struct comprising (userID, workspaceID, bindingKind,
// bindingID, actingAsClientID, actingAsSupplierID). Struct keys are
// collision-safe by construction (no delimiter games). Switching binding —
// or for delegate users, switching acting-as target — does not yield
// stale cached entries from a different binding.
type DBPermissionLoader struct {
	query PermissionQuery
	mu    sync.RWMutex
	cache map[permissionCacheKey]cachedPerms
}

// permissionCacheKey is the struct-based composite cache key for
// DBPermissionLoader. Each distinct (user, workspace, binding, acting-as)
// tuple is a distinct map key — no delimiter encoding, no collision possible
// from ':' characters embedded in string fields (codex A2-P1-2 fix).
type permissionCacheKey struct {
	userID             string
	workspaceID        string
	bindingKind        PrincipalType
	bindingID          string
	actingAsClientID   string
	actingAsSupplierID string
}

// NewDBPermissionLoader wraps a PermissionQuery with a per-(user,workspace,
// binding,acting-as) cache. Pass nil to disable — IsEnabled() will report
// false.
func NewDBPermissionLoader(query PermissionQuery) *DBPermissionLoader {
	return &DBPermissionLoader{
		query: query,
		cache: make(map[permissionCacheKey]cachedPerms),
	}
}

// GetUserPermissionCodes returns all effective ALLOW permission codes
// for a user within a workspace, restricted to the grant chain identified
// by (bindingKind, bindingID, actingAs*), honouring DENY-wins semantics
// from the underlying PermissionQuery. Cached for permissionCacheTTL.
//
// Fail-closed contract — see PermissionQuery doc. Production callers MUST
// supply a complete hint; only legacy bootstrap and test paths reach the
// union fallback.
func (l *DBPermissionLoader) GetUserPermissionCodes(
	ctx context.Context,
	userID string,
	workspaceID string,
	bindingKind PrincipalType,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) ([]string, error) {
	key := permissionCacheKey{
		userID:             userID,
		workspaceID:        workspaceID,
		bindingKind:        bindingKind,
		bindingID:          bindingID,
		actingAsClientID:   actingAsClientID,
		actingAsSupplierID: actingAsSupplierID,
	}
	l.mu.RLock()
	if entry, ok := l.cache[key]; ok && time.Now().Before(entry.expires) {
		l.mu.RUnlock()
		return entry.codes, nil
	}
	l.mu.RUnlock()

	codes, err := l.query.GetUserPermissionCodes(
		ctx, userID, workspaceID, bindingKind, bindingID,
		actingAsClientID, actingAsSupplierID,
	)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	l.cache[key] = cachedPerms{codes: codes, expires: time.Now().Add(permissionCacheTTL)}
	l.mu.Unlock()

	log.Printf("PermissionLoader: loaded %d codes for user %s workspace %s binding %s/%s acting=(%s,%s)",
		len(codes), userID, workspaceID, bindingKind, bindingID,
		actingAsClientID, actingAsSupplierID)
	return codes, nil
}

// InvalidateUser clears all cached permissions for a user across all workspaces
// and bindings (call after role changes).
func (l *DBPermissionLoader) InvalidateUser(userID string) {
	l.mu.Lock()
	for key := range l.cache {
		if key.userID == userID {
			delete(l.cache, key)
		}
	}
	l.mu.Unlock()
}

// IsEnabled returns true when a PermissionQuery backend is configured.
func (l *DBPermissionLoader) IsEnabled() bool {
	return l.query != nil
}

// ─────────────────────────────────────────────────────────────────────────────
// User display loading (was user_loader.go)
// ─────────────────────────────────────────────────────────────────────────────

// UserReader is the narrow "fetch a user's display fields by id" contract
// consumed by DBUserLoader. It is satisfied by a thin closure that invokes
// the espyna entity-layer use case at uc.Entity.User.ReadUser — see
// container.go for the wiring.
//
// active = true contract: the previous raw query filtered
// `WHERE id = $1 AND active = true`. The espyna ReadUser path is a primary-key
// lookup that does NOT filter on active, so the loader enforces it here —
// an inactive (or missing) user yields a zero-value SidebarCurrentUser, exactly
// as the old `LIMIT 1` + no-row path did.
type UserReader interface {
	// ReadUserDisplay returns the user's first name, last name, email, and
	// active flag for the given id. A non-nil error or an empty/zero result
	// is treated by the loader as "no displayable user".
	ReadUserDisplay(ctx context.Context, userID string) (UserDisplay, error)
}

// UserDisplay carries the read-only display fields the sidebar profile button
// needs. It deliberately mirrors the three columns the old raw query selected
// (first_name, last_name, email_address) plus the active flag used to preserve
// the active = true filter at the loader boundary.
type UserDisplay struct {
	FirstName string
	LastName  string
	Email     string
	Active    bool
}

// DBUserLoader loads the authenticated user's display data (first/last name,
// email) for the sidebar profile button by delegating to a UserReader — usually
// backed by the uc.Entity.User.ReadUser use case. It holds no database handle
// and runs no SQL; the adapter layer owns the query.
type DBUserLoader struct {
	reader      UserReader
	profileURLs ProfileURLs
}

// ProfileURLs carries the per-app URL conventions for the bottom-of-sidebar
// menu. service-admin populates these once at startup; the loader passes
// them through unchanged on every request.
type ProfileURLs struct {
	Profile      string
	Account      string
	Billing      string
	Preferences  string
	Logout       string
	LogoutAction string
}

// NewDBUserLoader creates a UserLoader backed by the given UserReader.
// Pass nil to disable — IsEnabled() will report false.
// Pass the profileURLs you want the sidebar menu to point at.
func NewDBUserLoader(reader UserReader, profileURLs ProfileURLs) *DBUserLoader {
	return &DBUserLoader{reader: reader, profileURLs: profileURLs}
}

// LoadCurrentUser returns the SidebarCurrentUser for the currently
// authenticated request. Returns a zero-value SidebarCurrentUser when there
// is no session, no reader, or the user is missing/inactive — the template
// renders nothing in that case.
func (l *DBUserLoader) LoadCurrentUser(ctx context.Context) types.SidebarCurrentUser {
	userID := consumer.GetUserIDFromContext(ctx)
	if userID == "" {
		userID = consumer.ExtractUserIDFromContext(ctx)
	}
	if userID == "" || l.reader == nil {
		return types.SidebarCurrentUser{}
	}

	display, err := l.reader.ReadUserDisplay(ctx, userID)
	if err != nil {
		log.Printf("UserLoader: failed to load user %s: %v", userID, err)
		return types.SidebarCurrentUser{}
	}
	// Preserve the old `AND active = true` filter at the loader boundary:
	// an inactive user renders no sidebar profile, same as the prior no-row case.
	if !display.Active {
		return types.SidebarCurrentUser{}
	}

	return types.SidebarCurrentUser{
		UserID:          userID,
		FirstName:       display.FirstName,
		LastName:        display.LastName,
		Email:           display.Email,
		ProfileURL:      l.profileURLs.Profile,
		AccountURL:      l.profileURLs.Account,
		BillingURL:      l.profileURLs.Billing,
		PreferencesURL:  l.profileURLs.Preferences,
		LogoutURL:       l.profileURLs.Logout,
		LogoutActionURL: l.profileURLs.LogoutAction,
	}
}

// IsEnabled returns true when this loader has a UserReader backend configured.
func (l *DBUserLoader) IsEnabled() bool {
	return l.reader != nil
}
