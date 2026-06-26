package http

// adapter.go — HTTPAdapter + RouteConfig + the renderer interfaces.
//
// Relocated from the service-admin app (internal/infrastructure/input/http/
// adapter.go) into espyna consumer/http in Model-A Wave 3. The framework HTTP
// runtime (adapter + view_adapter + loaders + permission_filter) is now
// espyna-owned; the app composes it via consumer "…/consumer/http". Moved
// verbatim — mux wiring + the HTTPAdapter/RouteConfig/TemplateRenderer/
// ContextRenderer contract are unchanged.

import (
	"context"
	"net/http"

	"github.com/erniealice/pyeza-golang/view"
)

// TemplateRenderer defines the interface for rendering templates.
// pyeza.HTMLRenderer satisfies this interface.
type TemplateRenderer interface {
	Render(w http.ResponseWriter, name string, data interface{}) error
	RenderBuffered(w http.ResponseWriter, name string, data interface{}) error
}

// ContextRenderer is an optional interface that template renderers may
// implement to support per-request route map overrides. When the renderer
// satisfies this interface, the ViewAdapter uses the context-aware methods
// for workspace requests so that {{route}} / {{routeWith}} resolve against
// the workspace-prefixed route map stored in context.
//
// pyeza.HTMLRenderer implements this interface via RenderWithContext and
// RenderBufferedWithContext.
type ContextRenderer interface {
	RenderWithContext(w http.ResponseWriter, ctx context.Context, name string, data interface{}) error
	RenderBufferedWithContext(w http.ResponseWriter, ctx context.Context, name string, data interface{}) error
}

// RouteConfig defines a single route configuration (mirrors composition.RouteConfig)
type RouteConfig struct {
	Method      string
	Path        string
	View        view.View
	Handler     http.HandlerFunc
	Middlewares []string
	Name        string
}

// HTTPAdapter adapts view routes to HTTP handlers
type HTTPAdapter struct {
	mux         *http.ServeMux
	viewAdapter *ViewAdapter
}

// NewHTTPAdapter creates a new HTTP adapter.
//
// portalSidebarBuilders maps PrincipalType to the corresponding portal sidebar
// builder. Pass nil to keep the pre-D2 staff-only behaviour. Pass the result
// of composition.PortalSidebarBuilders() for full portal dispatch.
func NewHTTPAdapter(renderer TemplateRenderer, cacheVersion string, commonLabels any, messages map[string]string, renderIcon IconRenderer, sidebarBuilder SidebarBuilder, portalSidebarBuilders map[PrincipalType]SidebarBuilder, bottomNavBuilder BottomNavBuilder, permLoader PermissionLoader, workspaceLoader WorkspaceLoader, userLoader UserLoader, defaultTheme string, defaultFont string, businessType string, translations any) *HTTPAdapter {
	return &HTTPAdapter{
		mux:         http.NewServeMux(),
		viewAdapter: NewViewAdapter(renderer, cacheVersion, commonLabels, messages, renderIcon, sidebarBuilder, portalSidebarBuilders, bottomNavBuilder, permLoader, workspaceLoader, userLoader, defaultTheme, defaultFont, businessType, translations),
	}
}

// SetWorkspaceRouteRewriter sets the per-request workspace-route rewriter
// hook on the embedded view adapter. The rewriter inspects the request
// context for a URL-canonical workspace slug (set by workspace_path
// middleware) and binds a workspace-prefixed RouteResult into context for
// downstream consumers (sidebar dispatch + view rendering).
//
// Per Phase P8 of docs/plan/20260521-workspace-keyed-routing/.
func (a *HTTPAdapter) SetWorkspaceRouteRewriter(fn WorkspaceRouteRewriter) {
	if a == nil || a.viewAdapter == nil {
		return
	}
	a.viewAdapter.SetWorkspaceRouteRewriter(fn)
}

// SetPrincipalLookup sets the per-request (bindingKind, bindingID) lookup
// used by the permission loader to scope grants to the active binding
// (A2 / WKR-P0-2 — 2026-05-24). Composition wires this to a closure that
// reads the session token from context and queries
// composition.lookupSessionPrincipal — the same source A3's BindingResolver
// hint uses. Nil clears the hook (loader falls back to legacy union
// behaviour).
func (a *HTTPAdapter) SetPrincipalLookup(fn PermissionPrincipalLookup) {
	if a == nil || a.viewAdapter == nil {
		return
	}
	a.viewAdapter.SetPrincipalLookup(fn)
}

// SetMessagesURL installs the secure-messaging inbox URL (Plan-4) used to
// populate the header Messages button's PageData fields for principals holding
// conversation:list. Empty disables the button.
func (a *HTTPAdapter) SetMessagesURL(url string) {
	if a == nil || a.viewAdapter == nil {
		return
	}
	a.viewAdapter.SetMessagesURL(url)
}

// RegisterRoutes registers all routes from the given slice
func (a *HTTPAdapter) RegisterRoutes(routes []RouteConfig) {
	for _, route := range routes {
		var handler http.HandlerFunc

		if route.View != nil {
			handler = a.viewAdapter.Adapt(route.View)
		} else if route.Handler != nil {
			// Raw handlers (e.g. JSON autocomplete endpoints) still gate on
			// view.GetUserPermissions(ctx); wrap them so they observe the same
			// RBAC permission context as view routes (else they fail closed 403).
			handler = a.viewAdapter.WrapHandler(route.Handler)
		} else {
			continue
		}

		pattern := route.Path
		if route.Method != "" {
			pattern = route.Method + " " + route.Path
		}

		a.mux.HandleFunc(pattern, handler)
	}
}

// Handler returns the HTTP handler
func (a *HTTPAdapter) Handler() http.Handler {
	return a.mux
}
