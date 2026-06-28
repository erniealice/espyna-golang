package http

// finalize.go — Server.finalizeHTTPAdapter + the fail-loud UI-bundle asserts
// (Wave B D1.2).
//
// This is the relocation of the FRAMEWORK-GENERIC half of the app's old
// composition build() (apps/service-admin/internal/composition/container.go
// build():308-486). It reconstructs the EXACT 15-arg NewHTTPAdapter call from
// the now-populated *consumerapp.AppContext: the renderer / sidebars / labels /
// translations / route-rewriter come from the APP-SUPPLIED *pyeza.AppUIBundle
// (codex C4 + round-2 — stamped by Build() from WithUI), and the rest
// (cacheVersion / theme / font / businessType / perm+user+nav loaders) are
// Server-built from the Server's own use cases + the merged ComposeResult.
//
// Every UI-bundle slot is read through a fail-loud must* assert: a non-mock
// binary boot-FATALS rather than serve a page with a nil renderer / empty
// labels / nil translation table. The sole documented non-fatal is
// mustRewriter (nil RouteRewriter => no-op rewriter, the pre-P8 un-prefixed
// behaviour), since a host that does not use /w/{slug}/* routing legitimately
// supplies none.

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/erniealice/espyna-golang/consumer"
	consumerapp "github.com/erniealice/espyna-golang/consumer/app"
	compose "github.com/erniealice/espyna-golang/consumer/compose"
	"github.com/erniealice/pyeza-golang"
	pyezatypes "github.com/erniealice/pyeza-golang/types"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	securitypb "github.com/erniealice/esqyma/pkg/schema/v1/service/security"
)

// finalizeHTTPAdapter constructs the HTTP adapter from the populated AppContext
// (the framework-generic relocation of the app's build()), registers the routes
// the blocks collected into the registry, wires the catch-all, and assembles the
// fixed-order middleware chain. It returns the composed http.Handler.
func (s *Server) finalizeHTTPAdapter(appCtx *consumerapp.AppContext, routes *routeRegistry) (http.Handler, error) {
	ui := s.appUI // already nil-guarded in Build()

	// ── ComposeResult / NavResolver (Server-built, generic) ──────────────
	cr, _ := appCtx.ComposeResult.(*compose.Result)
	if cr == nil {
		cr = compose.NewResult()
	}
	// The renderer's {{route}} / {{routeWith}} template functions read the
	// merged RouteMap (Nav Phase 4 — single source of truth). The renderer is
	// app-supplied; we feed it the Server-merged route map.
	//
	// We ALSO install the workspace-form signer here, from the SAME secret the
	// ActionGuard VERIFIER is built from (s.resolveSecurity().secret, the source
	// of agCfg.Secret in finalizePreset). The signer is the symmetric render-side
	// half of the request-side guard: wiring both from one secret in the shared
	// Server means the {{actionForm}} signature and the guard verifier can NEVER
	// drift. Previously only school-admin wired the signer (per-app), so a
	// service-admin password+secret deploy rendered an empty _workspace_id and
	// every guarded /action/* POST 409'd. Pass-through-on-empty matches the guard.
	if r := mustRenderer(ui.Renderer); r != nil {
		r.SetRouteMap(cr.RouteMap)
		if signer := buildWorkspaceFormSigner(string(s.resolveSecurity().secret)); signer != nil {
			r.SetWorkspaceFormSigner(signer)
		}
	}

	// ── Loaders (Server-built, generic — no app/domain template FS) ──────
	permLoader := s.buildPermissionLoader()
	workspaceLoader := s.assertWorkspaceLoader(appCtx)
	userLoader := s.buildUserLoader(cr)

	// ── HTTP adapter — the EXACT 15-arg NewHTTPAdapter call ──────────────
	httpAdapter := NewHTTPAdapter(
		mustRenderer(ui.Renderer),             // 1  renderer        (APP-SUPPLIED)
		s.config.CacheVersion,                 // 2  cacheVersion    (Server)
		mustCommonLabels(ui.CommonLabels),     // 3 commonLabels   (APP-SUPPLIED)
		mustMessages(ui.Messages),             // 4  messages        (APP-SUPPLIED)
		mustRenderIcon(ui.RenderIcon),         // 5  renderIcon      (APP-SUPPLIED)
		mustSidebar(ui.SidebarBuilder),        // 6  sidebarBuilder  (APP-SUPPLIED)
		mustPortalSidebars(ui.PortalSidebars), // 7 portalSidebars (APP-SUPPLIED)
		mustBottomNav(ui.BottomNavBuilder),    // 8 bottomNavBuilder (APP-SUPPLIED)
		permLoader,                            // 9  permLoader      (Server)
		workspaceLoader,                       // 10 workspaceLoader (block slot)
		userLoader,                            // 11 userLoader      (Server)
		s.config.Theme,                        // 12 theme           (Server)
		s.config.Font,                         // 13 font            (Server)
		s.config.BusinessType,                 // 14 businessType    (Server)
		mustTranslations(ui.Translations),     // 15 translations  (APP-SUPPLIED)
	)

	// ── Per-request principal binding → binding-scoped RBAC (Phase 0) ────
	// Wire the permission loader's principal lookup so RBAC scopes to the
	// session's ACTIVE binding (kind, id, acting-as) instead of the legacy
	// union-across-all-bindings. Sourced from the session row via the same
	// LookupSessionPrincipal use case the workspace-path middleware uses
	// (server.go PrincipalLookup). Without this a multi-binding user (e.g. an
	// operator who is ALSO a staff principal) loads the UNION of every
	// binding's permissions — defeating per-principal scope. An empty hint (no
	// session binding) makes the view adapter install empty perms (fail-closed).
	if s.useCases != nil && s.useCases.Service != nil && s.useCases.Service.Auth != nil &&
		s.useCases.Service.Auth.LookupSessionPrincipal != nil {
		lookupUC := s.useCases.Service.Auth.LookupSessionPrincipal
		httpAdapter.SetPrincipalLookup(func(r *http.Request) PermissionBindingHint {
			token := consumer.GetSessionTokenFromContext(r.Context())
			if token == "" {
				return PermissionBindingHint{}
			}
			resp, err := lookupUC.Execute(r.Context(), &authpb.LookupSessionPrincipalRequest{Token: token})
			if err != nil || resp == nil {
				return PermissionBindingHint{}
			}
			return PermissionBindingHint{
				Kind:               PrincipalType(resp.GetKind()),
				BindingID:          resp.GetPrincipalId(),
				ActingAsClientID:   resp.GetActingAsClientId(),
				ActingAsSupplierID: resp.GetActingAsSupplierId(),
			}
		})
	}

	// ── Messages-URL: PRESERVE the {status}->open substitution ───────────
	// conversation.list resolves to /app/conversations/list/{status}; the
	// header Messages button needs the concrete inbox URL, so the {status}
	// placeholder is substituted to "open" exactly as the app did
	// (container.go:411). Without this the header Messages URL is broken.
	messagesURL := cr.RouteOrEmpty("conversation.list")
	httpAdapter.SetMessagesURL(replaceStatusOpen(messagesURL))

	// ── Workspace route rewriter (APP-SUPPLIED func; generic SET) ────────
	// The rewriter closure itself calls the app's nav.WithWorkspace; the
	// Server only performs the generic SetWorkspaceRouteRewriter invocation.
	// nil rewriter => no-op (the sole documented non-fatal).
	if rewriter := mustRewriter(ui.RouteRewriter); rewriter != nil {
		httpAdapter.SetWorkspaceRouteRewriter(rewriter)
	}

	// ── Register routes (generic) ────────────────────────────────────────
	// Auth registration is the entydad block's job (D2, register-direct);
	// here the Server registers everything the registry collected.
	for _, route := range routes.Routes() {
		httpAdapter.RegisterRoutes([]RouteConfig{{
			Method:  route.Method,
			Path:    route.Path,
			View:    route.View,
			Handler: route.Handler,
		}})
	}

	// ── Catch-all + reserved slugs + fixed-order chain (generic) ─────────
	s.WithCatchAll(httpAdapter.Handler())
	if s.assetsDir == "" {
		s.assetsDir = "assets"
	}
	return s.assembleHandler(), nil
}

// buildPermissionLoader constructs the binding-scoped permission loader from
// the Server's own service.Security.GetUserPermissionCodes use case. Returns
// nil (loader disabled — sidebar filtering skipped) when the use case is
// unavailable. Generic: no app/domain template FS.
func (s *Server) buildPermissionLoader() PermissionLoader {
	if s.useCases == nil || s.useCases.Service == nil || s.useCases.Service.Security == nil ||
		s.useCases.Service.Security.GetUserPermissionCodes == nil {
		log.Printf("  PermissionLoader: disabled (no Security use cases)")
		return nil
	}
	getCodesUC := s.useCases.Service.Security.GetUserPermissionCodes
	loader := NewDBPermissionLoader(serverPermissionQueryFunc(func(
		ctx context.Context,
		userID, workspaceID string,
		bindingKind PrincipalType,
		bindingID string,
		actingAsClientID, actingAsSupplierID string,
	) ([]string, error) {
		resp, err := getCodesUC.Execute(ctx, &securitypb.GetUserPermissionCodesRequest{
			UserId:             userID,
			WorkspaceId:        workspaceID,
			BindingKind:        principaltypepb.PrincipalType(bindingKind),
			BindingId:          bindingID,
			ActingAsClientId:   actingAsClientID,
			ActingAsSupplierId: actingAsSupplierID,
		})
		if err != nil {
			return nil, err
		}
		return resp.GetPermissionCodes(), nil
	}))
	log.Printf("  PermissionLoader: routed through service.Security.GetUserPermissionCodes use case (binding-scoped)")
	return loader
}

// assertWorkspaceLoader reads the block-provided workspace loader from the
// AppContext slot. The proto-backed impl imports workspacepb (illegal in
// espyna), so the block (entydad, D2) sets it; here we type-assert it. Returns
// nil (sidebar workspace switcher disabled) when unset.
func (s *Server) assertWorkspaceLoader(appCtx *consumerapp.AppContext) WorkspaceLoader {
	if appCtx.WorkspaceLoader == nil {
		log.Printf("  WorkspaceLoader: disabled (no block provided one)")
		return nil
	}
	wl, ok := appCtx.WorkspaceLoader.(WorkspaceLoader)
	if !ok {
		log.Fatalf("FATAL espyna finalizeHTTPAdapter: appCtx.WorkspaceLoader is %T, want "+
			"consumerhttp.WorkspaceLoader (the entydad block must set a structurally-"+
			"satisfying loader). Refusing to boot with a wrong-type workspace loader.",
			appCtx.WorkspaceLoader)
	}
	log.Printf("  WorkspaceLoader: enabled (block-provided; sidebar workspace switcher active)")
	return wl
}

// buildUserLoader constructs the sidebar profile-button user loader from the
// Server's own Entity.User.ReadUser use case. The profile URLs come from the
// merged compose route map. Returns nil (profile button disabled) when the use
// case is unavailable. Generic: no app/domain template FS.
func (s *Server) buildUserLoader(cr *compose.Result) UserLoader {
	if s.useCases == nil || s.useCases.Entity == nil || s.useCases.Entity.User == nil ||
		s.useCases.Entity.User.ReadUser == nil {
		log.Printf("  UserLoader: disabled (no User use case)")
		return nil
	}
	readUserUC := s.useCases.Entity.User.ReadUser
	loader := NewDBUserLoader(serverUserReaderFunc(func(
		ctx context.Context,
		userID string,
	) (UserDisplay, error) {
		resp, err := readUserUC.Execute(ctx, &userpb.ReadUserRequest{
			Data: &userpb.User{Id: userID},
		})
		if err != nil {
			return UserDisplay{}, err
		}
		data := resp.GetData()
		if len(data) == 0 || data[0] == nil {
			return UserDisplay{}, nil
		}
		u := data[0]
		return UserDisplay{
			FirstName: u.GetFirstName(),
			LastName:  u.GetLastName(),
			Email:     u.GetEmailAddress(),
			Active:    u.GetActive(),
		}, nil
	}), ProfileURLs{
		Profile:      cr.RouteOrEmpty("personal.profile"),
		Account:      cr.RouteOrEmpty("personal.account"),
		Billing:      cr.RouteOrEmpty("personal.billing"),
		Preferences:  cr.RouteOrEmpty("personal.preferences"),
		Logout:       "/auth/logout",
		LogoutAction: "/action/auth/logout",
	})
	log.Printf("  UserLoader: routed through Entity.User.ReadUser use case (sidebar profile button active)")
	return loader
}

// replaceStatusOpen substitutes the {status} placeholder in the
// conversation.list route with "open" so the header Messages button points at
// the concrete inbox URL (Wave B D1.2 — PRESERVES the app's container.go:411
// strings.ReplaceAll(messagesURL, "{status}", "open")). Without this the header
// Messages URL is broken (conversation.list = /app/conversations/list/{status}).
func replaceStatusOpen(url string) string {
	return strings.ReplaceAll(url, "{status}", "open")
}

// ── func-adapters to the loader interfaces (Server-internal) ────────────

// serverPermissionQueryFunc adapts a closure to the PermissionQuery interface.
type serverPermissionQueryFunc func(
	ctx context.Context,
	userID, workspaceID string,
	bindingKind PrincipalType,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) ([]string, error)

func (f serverPermissionQueryFunc) GetUserPermissionCodes(
	ctx context.Context,
	userID, workspaceID string,
	bindingKind PrincipalType,
	bindingID string,
	actingAsClientID, actingAsSupplierID string,
) ([]string, error) {
	return f(ctx, userID, workspaceID, bindingKind, bindingID, actingAsClientID, actingAsSupplierID)
}

// serverUserReaderFunc adapts a closure to the UserReader interface.
type serverUserReaderFunc func(ctx context.Context, userID string) (UserDisplay, error)

func (f serverUserReaderFunc) ReadUserDisplay(ctx context.Context, userID string) (UserDisplay, error) {
	return f(ctx, userID)
}

// ── Fail-loud UI-bundle type-asserts (Wave B D1.2) ──────────────────────
// Each reads a `any` AppUIBundle slot and boot-FATALS on a missing or
// wrong-type value, so a non-mock binary can NEVER serve pages with a nil
// renderer / empty labels / nil translation table. mustRewriter is the sole
// documented non-fatal (nil => no-op rewriter).

func mustRenderer(v any) *pyeza.HTMLRenderer {
	r, ok := v.(*pyeza.HTMLRenderer)
	if !ok || r == nil {
		log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.Renderer is %T, want non-nil "+
			"*pyeza.HTMLRenderer (call WithUI with buildRenderer()). Refusing to boot with no renderer.", v)
	}
	return r
}

func mustCommonLabels(v any) pyeza.CommonLabels {
	l, ok := v.(pyeza.CommonLabels)
	if !ok {
		log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.CommonLabels is %T, want pyeza.CommonLabels "+
			"(loadTranslations().Common). Refusing to boot with no shared UI labels.", v)
	}
	return l
}

func mustTableLabels(v any) pyeza.TableLabels {
	l, ok := v.(pyeza.TableLabels)
	if !ok {
		log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.TableLabels is %T, want pyeza.TableLabels "+
			"(pyeza.MapTableLabels(common)). Refusing to boot with no table/grid labels.", v)
	}
	return l
}

func mustMessages(v any) map[string]string {
	m, ok := v.(map[string]string)
	if !ok || m == nil {
		log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.Messages is %T, want non-nil map[string]string "+
			"(loadTranslations().Messages). Refusing to boot with no flat-message table.", v)
	}
	return m
}

func mustRenderIcon(v any) IconRenderer {
	switch fn := v.(type) {
	case IconRenderer:
		if fn != nil {
			return fn
		}
	case func(string) template.HTML:
		if fn != nil {
			return IconRenderer(fn)
		}
	}
	log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.RenderIcon is %T, want non-nil "+
		"func(string) template.HTML (renderer.RenderIcon). Refusing to boot with no icon renderer.", v)
	return nil
}

func mustTranslations(v any) any {
	// The HTTP adapter takes translations as `any` (the ViewAdapter stores it
	// as `any` and never imports lyngua), so espyna asserts only non-nil-ness
	// here — keeping espyna free of a lyngua/app dependency while still
	// fail-loud-guarding a blank per-request translation table.
	if v == nil {
		log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.Translations is nil, want a non-nil " +
			"*lynguaV1.TranslationProvider. Refusing to boot with no per-request translation table.")
	}
	return v
}

func mustSidebar(v any) SidebarBuilder {
	switch fn := v.(type) {
	case SidebarBuilder:
		if fn != nil {
			return fn
		}
	case func(activeNav, activeSubNav string) any:
		if fn != nil {
			return SidebarBuilder(fn)
		}
	}
	log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.SidebarBuilder is %T, want non-nil "+
		"consumerhttp.SidebarBuilder (buildStaffSidebarBuilder). Refusing to boot with no sidebar builder.", v)
	return nil
}

func mustBottomNav(v any) BottomNavBuilder {
	switch fn := v.(type) {
	case BottomNavBuilder:
		if fn != nil {
			return fn
		}
	case func(activeNav string) ([]pyezatypes.BottomNavTab, []pyezatypes.AppGridItem, []pyezatypes.AppGridGroup):
		if fn != nil {
			return BottomNavBuilder(fn)
		}
	}
	log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.BottomNavBuilder is %T, want non-nil "+
		"consumerhttp.BottomNavBuilder (buildBottomNavBuilder). Refusing to boot with no bottom-nav builder.", v)
	return nil
}

func mustPortalSidebars(v any) map[PrincipalType]SidebarBuilder {
	// PortalSidebars is optional (nil => staff-only behaviour, the pre-D2
	// default) — but a wrong-type value is a programming error and boot-FATALS.
	if v == nil {
		return nil
	}
	if m, ok := v.(map[PrincipalType]SidebarBuilder); ok {
		return m
	}
	log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.PortalSidebars is %T, want "+
		"map[consumerhttp.PrincipalType]consumerhttp.SidebarBuilder. Refusing to boot with a wrong-type portal map.", v)
	return nil
}

func mustRewriter(v any) WorkspaceRouteRewriter {
	// The ONE deliberate non-fatal: a host that does not use /w/{slug}/*
	// workspace-keyed routing legitimately supplies none; an absent rewriter
	// degrades to un-prefixed routes (pre-P8), not a blank page.
	if v == nil {
		return nil
	}
	switch fn := v.(type) {
	case WorkspaceRouteRewriter:
		return fn
	case func(context.Context) context.Context:
		return WorkspaceRouteRewriter(fn)
	}
	log.Fatalf("FATAL espyna finalizeHTTPAdapter: UI.RouteRewriter is %T, want "+
		"func(context.Context) context.Context (the app nav.WithWorkspace closure) or nil.", v)
	return nil
}
