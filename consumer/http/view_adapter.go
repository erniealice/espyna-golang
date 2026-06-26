package http

// view_adapter.go — ViewAdapter (view → http.HandlerFunc) + the fail-CLOSED
// permission install.
//
// Relocated from the service-admin app (internal/infrastructure/input/http/
// view_adapter.go) into espyna consumer/http in Model-A Wave 3. PRESERVED
// BYTE-FOR-BEHAVIOUR: the fail-closed empty-perms install on a missing session
// binding hint (Adapt: hint.Empty() → types.NewEmptyUserPermissions()) and the
// EnsureUserPermissionsInContext backstop before any view runs. Do NOT regress
// to fail-open.

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/consumer"
	pyezarender "github.com/erniealice/pyeza-golang/render"
	"github.com/erniealice/pyeza-golang/types"
	"github.com/erniealice/pyeza-golang/view"
)

// ── Type re-exports ───────────────────────────────────────────────────────────
//
// These type aliases expose the pyeza/render types through this package so the
// loaders + adapter (same package) reference one name. They re-export pyeza
// render, the canonical home (espyna → pyeza arrow).

// IconRenderer is a function that renders an icon template name to HTML.
// Re-exported from pyeza/render.
type IconRenderer = pyezarender.IconRenderer

// SidebarBuilder creates a SidebarConfig for the given active navigation state.
// Re-exported from pyeza/render.
type SidebarBuilder = pyezarender.SidebarBuilder

// BottomNavBuilder creates bottom navigation tabs, the all-apps grid for mobile,
// and grouped app data for the bottom sheet.
// Re-exported from pyeza/render.
type BottomNavBuilder = pyezarender.BottomNavBuilder

// PrincipalType is the presentation-layer principal kind.
// Re-exported from pyeza/render.
type PrincipalType = pyezarender.PrincipalType

const (
	PrincipalTypeUnspecified      = pyezarender.PrincipalTypeUnspecified
	PrincipalTypeOperatorOwner    = pyezarender.PrincipalTypeOperatorOwner
	PrincipalTypeOperatorStaff    = pyezarender.PrincipalTypeOperatorStaff
	PrincipalTypeClient           = pyezarender.PrincipalTypeClient
	PrincipalTypeClientDelegate   = pyezarender.PrincipalTypeClientDelegate
	PrincipalTypeSupplier         = pyezarender.PrincipalTypeSupplier
	PrincipalTypeSupplierDelegate = pyezarender.PrincipalTypeSupplierDelegate
)

// PermissionLoader loads permission codes for a user scoped to a workspace
// and a specific session binding.
//
// The bindingKind + bindingID hint (A2 / WKR-P0-2 — 2026-05-24) restricts
// the underlying RBAC query to the SINGLE selected binding row from the
// session, closing the silent privilege-elevation hole where a user
// holding multiple bindings in one workspace would receive the UNION of
// permissions across every binding.
//
// Delegate target scoping (A2-followup / codex A2-P0-1 — 2026-05-24):
// for CLIENT_DELEGATE / SUPPLIER_DELEGATE bindings the actingAs* id
// scopes resolution to the per-target delegate_client / delegate_supplier
// row.
//
// Fail-closed posture: only the EXACT zero pair (UNSPECIFIED, "") plus
// empty acting-as values triggers the legacy union fall-back behaviour;
// partial / malformed hints return an empty permission set.
//
// This interface satisfies pyeza/render.PermissionLoader.
type PermissionLoader interface {
	GetUserPermissionCodes(
		ctx context.Context,
		userID string,
		workspaceID string,
		bindingKind PrincipalType,
		bindingID string,
		actingAsClientID, actingAsSupplierID string,
	) ([]string, error)
	IsEnabled() bool
}

// PermissionBindingHint is the full binding identification surfaced from
// the session row for the active request — used by the permission loader
// to scope RBAC resolution.
// Re-exported from pyeza/render.
type PermissionBindingHint = pyezarender.PermissionBindingHint

// PermissionPrincipalLookup reads the full PermissionBindingHint for the
// session attached to the request — sourced from the session row's
// principal_type + principal_id + acting_as_* columns. Returns an Empty
// hint when no session row is bound (pre-login / auth shell routes).
//
// Composition wires this to composition.lookupSessionPrincipalFull so the
// hint flows from the same source as the A3 BindingResolver hint.
type PermissionPrincipalLookup func(r *http.Request) PermissionBindingHint

// WorkspaceLoader loads workspace data for the current user.
// Called per-request to populate the sidebar workspace switcher.
//
// This interface satisfies pyeza/render.WorkspaceLoader. The proto-backed impl
// (DBWorkspaceLoader, which imports workspacepb) stays app/entydad-side; espyna
// takes only this interface so no esqyma proto leaks into the framework runtime.
type WorkspaceLoader interface {
	// LoadWorkspaces returns all workspaces and the current workspace for the user.
	// Returns (available, current). Returns nil slice + zero value when unavailable.
	LoadWorkspaces(ctx context.Context) (available []types.SidebarWorkspace, current types.SidebarWorkspace)
	IsEnabled() bool
}

// UserLoader loads the authenticated user's display data for the bottom-of-
// sidebar profile button + popover menu. Called per-request.
//
// This interface satisfies pyeza/render.UserLoader.
type UserLoader interface {
	LoadCurrentUser(ctx context.Context) types.SidebarCurrentUser
	IsEnabled() bool
}

// WorkspaceRouteRewriter is the per-request hook invoked by the view adapter
// to install a workspace-rewritten RouteResult into the request context.
// Re-exported from pyeza/render.
type WorkspaceRouteRewriter = pyezarender.WorkspaceRouteRewriter

// WithRequestSidebarBuilder binds a per-request SidebarBuilder closure into ctx.
// Forwarded to pyeza/render.WithRequestSidebarBuilder.
func WithRequestSidebarBuilder(ctx context.Context, fn SidebarBuilder) context.Context {
	return pyezarender.WithRequestSidebarBuilder(ctx, fn)
}

// RequestSidebarBuilderFromContext returns the per-request SidebarBuilder
// bound by WithRequestSidebarBuilder, or nil when none is bound.
// Forwarded to pyeza/render.RequestSidebarBuilderFromContext.
func RequestSidebarBuilderFromContext(ctx context.Context) SidebarBuilder {
	return pyezarender.RequestSidebarBuilderFromContext(ctx)
}

// WithRouteMap stores a per-request workspace-prefixed route map in ctx.
// Forwarded to pyeza/render.WithRouteMap.
func WithRouteMap(ctx context.Context, m map[string]string) context.Context {
	return pyezarender.WithRouteMap(ctx, m)
}

// ViewAdapter transforms application views into HTTP handlers
type ViewAdapter struct {
	renderer               TemplateRenderer
	cacheVersion           string
	commonLabels           any
	messages               map[string]string
	renderIcon             IconRenderer
	sidebarBuilder         SidebarBuilder                   // staff (OPERATOR_OWNER / OPERATOR_STAFF) builder
	portalSidebarBuilders  map[PrincipalType]SidebarBuilder // portal builders keyed by principal type; nil = portal-dispatch disabled
	bottomNavBuilder       BottomNavBuilder                 // nil = no bottom nav
	permLoader             PermissionLoader                 // nil = no permission filtering
	workspaceLoader        WorkspaceLoader                  // nil = no workspace switcher
	userLoader             UserLoader                       // nil = no sidebar profile button
	workspaceRouteRewriter WorkspaceRouteRewriter           // nil = no per-request workspace-prefix rewrite
	principalLookup        PermissionPrincipalLookup        // nil = no binding hint (loader falls back to legacy union)
	defaultTheme           string
	defaultFont            string
	businessType           string
	translations           any // *lynguaV1.TranslationProvider
	// messagesURL is the staff secure-messaging inbox URL (Plan-4). When set,
	// injectUserPermissions populates PageData.HasMessages / .MessagesURL for any
	// principal holding conversation:list, lighting the header Messages button.
	messagesURL string

	// pipeline is the framework-agnostic render pipeline (pyeza/render.Pipeline).
	// It holds the inject/filter logic that operates on context + reflection only.
	pipeline *pyezarender.Pipeline
}

// buildPipeline constructs the pyeza/render.Pipeline from the ViewAdapter's
// current configuration. Called by NewViewAdapter and after any setter that
// changes pipeline-relevant config.
func (a *ViewAdapter) buildPipeline() {
	a.pipeline = &pyezarender.Pipeline{
		GetUserID: func(ctx context.Context) string {
			if uid := consumer.GetUserIDFromContext(ctx); uid != "" {
				return uid
			}
			return consumer.ExtractUserIDFromContext(ctx)
		},
		GetWorkspaceID:        consumer.GetWorkspaceIDFromContext,
		PermLoader:            a.permLoader,
		WorkspaceLoader:       a.workspaceLoader,
		UserLoader:            a.userLoader,
		SidebarBuilder:        a.sidebarBuilder,
		PortalSidebarBuilders: a.portalSidebarBuilders,
		BottomNavBuilder:      a.bottomNavBuilder,
		RenderIcon:            a.renderIcon,
		CommonLabels:          a.commonLabels,
		Messages:              a.messages,
		DefaultTheme:          a.defaultTheme,
		DefaultFont:           a.defaultFont,
		MessagesURL:           a.messagesURL,
		DevMode:               isDevMode(),
	}
}

// SetMessagesURL installs the secure-messaging inbox URL used to populate the
// header Messages button's PageData fields (Plan-4). Composition wires this
// after building the ViewAdapter so the constructor signature stays stable.
// Empty disables the header Messages button.
func (a *ViewAdapter) SetMessagesURL(url string) {
	a.messagesURL = url
	a.pipeline.MessagesURL = url
}

// SetWorkspaceRouteRewriter installs the per-request workspace route rewriter.
// Composition wires this after building the ViewAdapter so the constructor
// signature stays stable (15 positional params already). Nil clears the hook.
//
// Added 2026-05-22 per Phase P6 of docs/plan/20260521-workspace-keyed-routing.
func (a *ViewAdapter) SetWorkspaceRouteRewriter(fn WorkspaceRouteRewriter) {
	a.workspaceRouteRewriter = fn
}

// SetPrincipalLookup installs the per-request (bindingKind, bindingID)
// lookup used by the permission loader to scope grants to the active
// binding (A2 / WKR-P0-2 — 2026-05-24). Nil clears the hook and the
// loader falls back to its legacy union-across-all-bindings behaviour.
func (a *ViewAdapter) SetPrincipalLookup(fn PermissionPrincipalLookup) {
	a.principalLookup = fn
}

// NewViewAdapter creates a new ViewAdapter.
//
// portalSidebarBuilders may be nil; when non-nil, requests under
// /portal/{kind}/ are dispatched to the matching builder rather than
// the staff sidebarBuilder. Pass the result of composition.PortalSidebarBuilders()
// at startup. Nil disables portal dispatch (staff builder handles all routes).
func NewViewAdapter(renderer TemplateRenderer, cacheVersion string, commonLabels any, messages map[string]string, renderIcon IconRenderer, sidebarBuilder SidebarBuilder, portalSidebarBuilders map[PrincipalType]SidebarBuilder, bottomNavBuilder BottomNavBuilder, permLoader PermissionLoader, workspaceLoader WorkspaceLoader, userLoader UserLoader, defaultTheme string, defaultFont string, businessType string, translations any) *ViewAdapter {
	a := &ViewAdapter{
		renderer:              renderer,
		cacheVersion:          cacheVersion,
		commonLabels:          commonLabels,
		messages:              messages,
		renderIcon:            renderIcon,
		sidebarBuilder:        sidebarBuilder,
		portalSidebarBuilders: portalSidebarBuilders,
		bottomNavBuilder:      bottomNavBuilder,
		permLoader:            permLoader,
		workspaceLoader:       workspaceLoader,
		userLoader:            userLoader,
		defaultTheme:          defaultTheme,
		defaultFont:           defaultFont,
		businessType:          businessType,
		translations:          translations,
	}
	a.buildPipeline()
	return a
}

// isDevMode reports whether the server is running in development mode.
// Development mode is detected by APP_ENV=development OR CONFIG_AUTH_PROVIDER=mock.
// In dev mode, nil UserPermissions in context causes a panic so missing wiring is
// caught immediately rather than silently granting all permissions.
func isDevMode() bool {
	if os.Getenv("APP_ENV") == "development" {
		return true
	}
	return os.Getenv("CONFIG_AUTH_PROVIDER") == "mock"
}

// getUserIDFromContext extracts the authenticated user ID from the request context.
func getUserIDFromContext(ctx context.Context) string {
	if uid := consumer.GetUserIDFromContext(ctx); uid != "" {
		return uid
	}
	return consumer.ExtractUserIDFromContext(ctx)
}

// injectRequestContext runs the per-request workspace route rewrite and loads
// the acting user's permissions into the request context, returning the
// updated request. It is the shared front-half of both Adapt (view routes) and
// WrapHandler (raw JSON/handler routes) so that raw http.HandlerFunc routes —
// e.g. the workspace_user_role search-roles autocomplete JSON endpoint —
// observe the same RBAC permission context as view routes. Without this, a raw
// handler's view.GetUserPermissions(ctx) returns empty perms and any
// perms.Can(...) gate fails closed with a spurious 403.
func (a *ViewAdapter) injectRequestContext(r *http.Request) *http.Request {
	ctx := r.Context()

	// Phase P6 — per-request workspace route rewrite.
	if a.workspaceRouteRewriter != nil {
		ctx = a.workspaceRouteRewriter(ctx)
		r = r.WithContext(ctx)
	}

	// Load permissions into context BEFORE handler execution.
	if a.permLoader != nil && a.permLoader.IsEnabled() {
		if userID := getUserIDFromContext(ctx); userID != "" {
			workspaceID := consumer.GetWorkspaceIDFromContext(ctx)
			var hint PermissionBindingHint
			if a.principalLookup != nil {
				hint = a.principalLookup(r)
				ctx = consumer.WithActingAsClientID(ctx, hint.ActingAsClientID)
				r = r.WithContext(ctx)
				if hint.Empty() {
					ctx = view.WithUserPermissions(ctx, types.NewEmptyUserPermissions())
					r = r.WithContext(ctx)
					log.Printf("[rbac] uid=%s no-session-binding-hint path=%s — installed empty perms (fail-closed)",
						userID, r.URL.Path)
				} else {
					codes, err := a.permLoader.GetUserPermissionCodes(ctx, userID, workspaceID,
						hint.Kind, hint.BindingID,
						hint.ActingAsClientID, hint.ActingAsSupplierID)
					if err == nil {
						perms := types.NewUserPermissions(codes)
						ctx = view.WithUserPermissions(ctx, perms)
						r = r.WithContext(ctx)
						log.Printf("[rbac] uid=%s perms=%d path=%s binding=%s/%s acting=(%s,%s)",
							userID, len(codes), r.URL.Path, hint.Kind, hint.BindingID,
							hint.ActingAsClientID, hint.ActingAsSupplierID)
					} else {
						log.Printf("[rbac] uid=%s perm-load-error=%v path=%s", userID, err, r.URL.Path)
					}
				}
			} else {
				codes, err := a.permLoader.GetUserPermissionCodes(ctx, userID, workspaceID,
					PrincipalTypeUnspecified, "", "", "")
				if err == nil {
					perms := types.NewUserPermissions(codes)
					ctx = view.WithUserPermissions(ctx, perms)
					r = r.WithContext(ctx)
					log.Printf("[rbac] uid=%s perms=%d path=%s (no principalLookup wired — legacy path)",
						userID, len(codes), r.URL.Path)
				} else {
					log.Printf("[rbac] uid=%s perm-load-error=%v path=%s", userID, err, r.URL.Path)
				}
			}
		} else {
			log.Printf("[rbac] no uid in context, skipping permissions path=%s", r.URL.Path)
		}
	}

	// P4 safety net: guarantee non-nil UserPermissions before any handler runs.
	ctx = pyezarender.EnsureUserPermissionsInContext(ctx, r.URL.Path, a.pipeline.DevMode)
	return r.WithContext(ctx)
}

// Adapt creates an http.HandlerFunc that delegates to the given view
func (a *ViewAdapter) Adapt(v view.View) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = a.injectRequestContext(r)
		viewCtx := a.buildViewContext(r)
		result := v.Handle(r.Context(), viewCtx)
		a.handleResult(w, r, result)
	}
}

// WrapHandler wraps a raw http.HandlerFunc with the same workspace-route-rewrite
// and RBAC permission-injection front-half that Adapt applies to view routes.
// Used for non-view routes (e.g. JSON autocomplete endpoints) that still gate on
// view.GetUserPermissions(ctx).
func (a *ViewAdapter) WrapHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = a.injectRequestContext(r)
		h(w, r)
	}
}

// buildViewContext extracts context data from the HTTP request
func (a *ViewAdapter) buildViewContext(r *http.Request) *view.ViewContext {
	queryParams := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	pathParams := make(map[string]string)

	return &view.ViewContext{
		Request:      r,
		Lang:         getLang(r),
		CurrentPath:  r.URL.Path,
		IsHTMX:       isHTMX(r),
		PathParams:   pathParams,
		QueryParams:  queryParams,
		CacheVersion: a.cacheVersion,
		Messages:     a.messages,
		BusinessType: a.businessType,
		Translations: a.translations,
	}
}

// handleResult processes the ViewResult and writes the appropriate response
func (a *ViewAdapter) handleResult(w http.ResponseWriter, r *http.Request, result view.ViewResult) {
	if result.Error != nil {
		a.handleError(w, r, result)
		return
	}

	if result.Redirect != "" {
		a.handleRedirect(w, r, result)
		return
	}

	a.handleRender(w, r, result)
}

func (a *ViewAdapter) handleError(w http.ResponseWriter, r *http.Request, result view.ViewResult) {
	statusCode := result.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	log.Printf("View error: %v", result.Error)
	http.Error(w, result.Error.Error(), statusCode)
}

func (a *ViewAdapter) handleRedirect(w http.ResponseWriter, r *http.Request, result view.ViewResult) {
	statusCode := result.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusSeeOther
	}

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", result.Redirect)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, result.Redirect, statusCode)
	}
}

func (a *ViewAdapter) handleRender(w http.ResponseWriter, r *http.Request, result view.ViewResult) {
	ctx := r.Context()
	a.pipeline.InjectPageData(ctx, result.Data, r.URL.Path)
	a.pipeline.InjectSessionUser(ctx, result.Data)
	a.pipeline.InjectWorkspaceData(ctx, result.Data)
	a.pipeline.InjectUserData(ctx, result.Data)
	a.pipeline.InjectPostRotationBanner(ctx, result.Data)
	a.pipeline.InjectUserPermissions(ctx, result.Data)
	statusCode := result.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// Write custom headers (for HTMX triggers, etc.)
	for key, value := range result.Headers {
		w.Header().Set(key, value)
	}

	// If no template but headers set, just return with status (for HTMX responses)
	if result.Template == "" && len(result.Headers) > 0 {
		w.WriteHeader(statusCode)
		return
	}

	if statusCode != http.StatusOK {
		// Set Content-Type BEFORE committing the status.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
	}

	// Use context-aware rendering when available (workspace route map override).
	// The ContextRenderer interface is satisfied by pyeza.HTMLRenderer; the type
	// assertion avoids a hard dependency so the TemplateRenderer interface stays
	// backward-compatible for callers that don't need per-request route maps.
	ctxRenderer, hasCtxRenderer := a.renderer.(ContextRenderer)

	template := result.Template

	if isHTMX(r) {
		hxTarget := r.Header.Get("HX-Target")

		if hxTarget == "main-content" {
			contentTemplate := template + "-content"
			if hasCtxRenderer {
				if err := ctxRenderer.RenderBufferedWithContext(w, ctx, contentTemplate, result.Data); err == nil {
					ctxRenderer.RenderWithContext(w, ctx, "header-oob", result.Data)
					ctxRenderer.RenderWithContext(w, ctx, "help-pane-oob", result.Data)
					return
				} else {
					log.Printf("Buffered render failed for %s: %v", contentTemplate, err)
				}
			} else {
				if err := a.renderer.RenderBuffered(w, contentTemplate, result.Data); err == nil {
					a.renderer.Render(w, "header-oob", result.Data)
					a.renderer.Render(w, "help-pane-oob", result.Data)
					return
				} else {
					log.Printf("Buffered render failed for %s: %v", contentTemplate, err)
				}
			}
		}

		partialTemplate := template + "-partial"
		if hasCtxRenderer {
			if err := ctxRenderer.RenderBufferedWithContext(w, ctx, partialTemplate, result.Data); err != nil {
				if err := ctxRenderer.RenderWithContext(w, ctx, template, result.Data); err != nil {
					log.Printf("Template rendering error: %v", err)
					http.Error(w, "Failed to render page", http.StatusInternalServerError)
				}
			}
		} else {
			if err := a.renderer.RenderBuffered(w, partialTemplate, result.Data); err != nil {
				if err := a.renderer.Render(w, template, result.Data); err != nil {
					log.Printf("Template rendering error: %v", err)
					http.Error(w, "Failed to render page", http.StatusInternalServerError)
				}
			}
		}
	} else {
		if hasCtxRenderer {
			if err := ctxRenderer.RenderWithContext(w, ctx, template, result.Data); err != nil {
				log.Printf("Template rendering error: %v", err)
				http.Error(w, "Failed to render page", http.StatusInternalServerError)
			}
		} else {
			if err := a.renderer.Render(w, template, result.Data); err != nil {
				log.Printf("Template rendering error: %v", err)
				http.Error(w, "Failed to render page", http.StatusInternalServerError)
			}
		}
	}
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func getLang(r *http.Request) string {
	if lang, ok := r.Context().Value("lang").(string); ok {
		return lang
	}

	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		parts := strings.Split(acceptLang, ",")
		if len(parts) > 0 {
			lang := strings.TrimSpace(strings.Split(parts[0], ";")[0])
			if len(lang) >= 2 {
				return lang[:2]
			}
		}
	}

	return "en"
}
