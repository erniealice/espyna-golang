// Package consumer — nav_resolver.go
//
// NavResolver bridges the compose engine's Nav system to the sidebar's
// types.SidebarItem / types.SidebarApp types. It resolves route keys from
// descriptors into concrete href URLs and optionally prepends a workspace
// slug for per-request sidebar generation.
//
// Moved from apps/service-admin/internal/composition/ — this is a framework
// concern with no app-internal dependencies.

package consumer

import (
	"strings"

	"github.com/erniealice/pyeza-golang/compose"
	"github.com/erniealice/pyeza-golang/types"
)

// NavResolver bridges the compose engine's Nav system to the sidebar's
// types.SidebarItem / types.SidebarApp types. It resolves route keys from
// descriptors into concrete href URLs and optionally prepends a workspace
// slug for per-request sidebar generation.
type NavResolver struct {
	result        *compose.Result
	workspaceSlug string
	clientID      string
}

// NewNavResolver wraps a compose.Result for sidebar href resolution.
func NewNavResolver(result *compose.Result) *NavResolver {
	return &NavResolver{result: result}
}

// WithWorkspace returns a copy of the NavResolver that prepends /w/{slug}
// (and optionally /as/{clientID}) to every resolved href. The original
// NavResolver is not mutated.
func (nav *NavResolver) WithWorkspace(slug, clientID string) *NavResolver {
	return &NavResolver{
		result:        nav.result,
		workspaceSlug: slug,
		clientID:      clientID,
	}
}

// Href resolves a single NavItem's route key to its final href URL. Returns
// an empty string if the unit, item, or route key is not found — sidebar
// construction treats empty hrefs as fail-closed entries that are omitted by
// filterSidebar downstream.
func (nav *NavResolver) Href(unitKey, itemKey string) string {
	href, ok := nav.result.ResolveNavItemHref(unitKey, itemKey)
	if !ok {
		return ""
	}
	return nav.applyWorkspace(href)
}

// AppURL resolves a unit's AppEntry route to a final href for the app
// switcher. Returns empty string if the unit has no AppEntry.
func (nav *NavResolver) AppURL(unitKey string) string {
	href, ok := nav.result.ResolveAppEntryURL(unitKey)
	if !ok {
		return ""
	}
	return nav.applyWorkspace(href)
}

// applyWorkspace prepends the workspace slug if configured.
func (nav *NavResolver) applyWorkspace(href string) string {
	if nav.workspaceSlug == "" || href == "" {
		return href
	}
	return PrependWorkspaceSlug(href, nav.workspaceSlug, nav.clientID)
}

// App resolves a unit's AppEntry into a types.SidebarApp ready for the
// sidebar apps[] array. Label and icon come from the descriptor's AppEntry
// defaults. The caller can override them from ExtendedSidebarLabels if a
// sidebar.json override exists.
func (nav *NavResolver) App(unitKey string) types.SidebarApp {
	nc, ok := nav.result.Nav[unitKey]
	if !ok || nc.AppEntry == nil {
		return types.SidebarApp{}
	}
	entry := nc.AppEntry
	return types.SidebarApp{
		Key:        entry.Key,
		Label:      entry.Label,
		Icon:       entry.Icon,
		URL:        nav.AppURL(unitKey),
		Permission: entry.Permission,
	}
}

// Pick resolves NavItems for the given unit and item keys into SidebarItems.
// Labels and icons use the descriptor Go defaults. The caller can override
// individual fields from ExtendedSidebarLabels for business-type overrides.
func (nav *NavResolver) Pick(unitKey string, itemKeys ...string) []types.SidebarItem {
	items := nav.result.PickNav(unitKey, itemKeys...)
	out := make([]types.SidebarItem, 0, len(items))
	for _, item := range items {
		href := nav.Href(unitKey, item.Key)
		if href == "" {
			continue
		}
		perm := item.Permission
		if perm == "" {
			if nc, ok := nav.result.Nav[unitKey]; ok {
				perm = nc.Permission
			}
		}
		out = append(out, types.SidebarItem{
			Key:        item.Key,
			Label:      nav.resolveLabel(item),
			Icon:       nav.resolveIcon(item),
			Href:       href,
			Permission: perm,
		})
	}
	return out
}

func (nav *NavResolver) resolveLabel(item compose.NavItem) string {
	return item.Label
}

func (nav *NavResolver) resolveIcon(item compose.NavItem) string {
	return item.Icon
}

// RouteMapValue looks up a dot-notation route key directly in the compose
// engine's merged RouteMap and applies workspace prefixing. Returns empty
// string if the key is not found. Used for URLs that are in the RouteMap
// but have no corresponding Nav item (e.g., workspace.switch_url).
func (nav *NavResolver) RouteMapValue(routeKey string) string {
	if nav.result == nil {
		return ""
	}
	v, ok := nav.result.RouteMap[routeKey]
	if !ok {
		return ""
	}
	return nav.applyWorkspace(v)
}

// HrefWithQuery resolves a NavItem's route and appends a query string.
// Used for sidebar entries that add ?kind=X or ?status=X query params
// beyond what the NavItem's Params handle.
func (nav *NavResolver) HrefWithQuery(unitKey, itemKey, query string) string {
	href := nav.Href(unitKey, itemKey)
	if href == "" || query == "" {
		return href
	}
	if strings.Contains(href, "?") {
		return href + "&" + query
	}
	return href + "?" + query
}

// PrependWorkspaceSlug applies the /w/{slug}[/as/{clientID}] prefix to a URL,
// stripping any pre-existing /app/ prefix defensively (so the rewrite works
// both pre-P4 and post-P4). URLs in the pass-through list below are returned
// unchanged.
//
// Pass-through list (per Phase P6 of the workspace-keyed-routing plan):
//
//	/action/*  — session-bound mutations (Q-WS-3 -> A)
//	/auth/*    — pre-session (login, picker, logout)
//	/me/*      — cross-workspace personal surface (Q-WS-7 -> B)
//	/assets/*  — static asset mux
//	/static/*  — alternate static mux
//	/healthz   — health probe
//	/w/*       — already workspace-prefixed (idempotent rewrite)
//	external   — anything not starting with "/" (e.g. "https://...", "active",
//	             or nav identifiers like "service") is returned unchanged.
//	empty      — empty strings remain empty.
//
// Non-pass-through URLs have the /app/ prefix stripped (when present) and the
// /w/{slug}[/as/{clientID}] prefix prepended.
func PrependWorkspaceSlug(url, slug, clientID string) string {
	if url == "" {
		return url
	}
	// Anything that isn't a leading-slash absolute path is pass-through:
	// covers external URLs (https://...), bare nav identifiers
	// ("service", "active"), and {placeholder} fragments — none of which
	// should be workspace-prefixed.
	if !strings.HasPrefix(url, "/") {
		return url
	}
	// Namespace pass-through list.
	switch {
	case strings.HasPrefix(url, "/action/"),
		strings.HasPrefix(url, "/auth/"),
		strings.HasPrefix(url, "/me/"),
		strings.HasPrefix(url, "/assets/"),
		strings.HasPrefix(url, "/static/"),
		strings.HasPrefix(url, "/healthz"),
		strings.HasPrefix(url, "/w/"):
		return url
	}

	// Defensive /app/ strip (pre-P4 state). After P4 lands the URL
	// constants no longer carry /app/ and this branch becomes a no-op.
	stripped := url
	switch {
	case strings.HasPrefix(stripped, "/app/"):
		stripped = stripped[len("/app"):] // "/app/clients" -> "/clients"
	case stripped == "/app":
		stripped = "/"
	}

	prefix := "/w/" + slug
	if clientID != "" {
		prefix = prefix + "/as/" + clientID
	}
	if strings.HasPrefix(stripped, "/") {
		return prefix + stripped
	}
	return prefix + "/" + stripped
}
