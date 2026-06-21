package http

// fluent_compile_test.go — COMPILE-ONLY gate for the Wave-A fluent API (A.4).
//
// Asserts the locked reference chain
//
//	consumer.NewServer().WithApp(cfg).WithMiddleware(middleware.StandardAdmin()).
//	    WithBlocks(...).MustBuild()
//
// type-checks against the real method signatures. The body is NEVER executed
// (NewServer boots infra/DB); referencing it from a package-level var is enough
// for the compiler to verify the surface. The cookie builders + agnostic CSRF
// issuer are likewise referenced so their no-tag importability is gate-checked.

import (
	"context"
	"html/template"
	"net/http"

	consumerapp "github.com/erniealice/espyna-golang/consumer/app"
	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	"github.com/erniealice/pyeza-golang"
	pyezatypes "github.com/erniealice/pyeza-golang/types"
)

// fluentReferenceChain is the compile-only proof. A no-op block keeps the
// WithBlocks(...pyeza.AppOption) arm exercised.
func fluentReferenceChain() *Container {
	noopBlock := func(*consumerapp.AppContext) error { return nil }
	srv, err := NewServer()
	if err != nil {
		return nil
	}
	return srv.
		WithApp(AppConfig{
			ID:                     "service-admin",
			Name:                   "Service Admin",
			DefaultTheme:           "corporate-steel",
			DefaultFont:            "default",
			DefaultBusinessType:    "general",
			AssetRoot:              "assets",
			ReservedWorkspaceSlugs: []string{"auth", "me", "portal"},
			Features:               Features{SupplierPortalReady: false},
		}).
		WithMiddleware(consumermw.StandardAdmin()).
		WithBlocks(noopBlock).
		MustBuild()
}

// fluentReferenceChainWithUI is the Wave-B D1 compile-only proof: the END-state
// fluent chain
//
//	MustNewServer().WithApp(cfg).WithUI(ui).WithMiddleware(StandardAdmin()).
//	    WithBlocks(...).MustBuild()
//
// type-checks against the real method signatures, INCLUDING the new WithUI(...)
// option that seeds the COMPLETE app-supplied *consumerapp.AppUIBundle. The bundle's
// fields are populated with values whose concrete types match what the must*
// asserts in finalize.go expect (renderer / common+table labels / messages /
// renderIcon / sidebar / bottom-nav / portal map / translations / route
// rewriter), so this also gate-checks that the bundle's any-typed fields accept
// those concrete types. The body is NEVER executed (MustNewServer boots
// infra/DB); referencing it from a package-level var is enough.
func fluentReferenceChainWithUI() *Container {
	noopBlock := func(*consumerapp.AppContext) error { return nil }
	ui := buildReferenceUIBundle()
	return MustNewServer().
		WithApp(AppConfig{
			ID:                     "service-admin",
			Name:                   "Service Admin",
			DefaultTheme:           "corporate-steel",
			DefaultFont:            "default",
			DefaultBusinessType:    "general",
			AssetRoot:              "assets",
			ReservedWorkspaceSlugs: []string{"auth", "me", "portal"},
			Features:               Features{SupplierPortalReady: false},
		}).
		WithUI(ui).
		WithMiddleware(consumermw.StandardAdmin()).
		WithBlocks(noopBlock).
		MustBuild()
}

// buildReferenceUIBundle constructs an *consumerapp.AppUIBundle whose any-typed fields
// carry the concrete types the finalize.go must* asserts expect. Compile-only —
// the values are zero/empty placeholders (the renderer field is left as a typed
// nil *pyeza.HTMLRenderer only to gate the type, never dereferenced here).
func buildReferenceUIBundle() *consumerapp.AppUIBundle {
	var renderer *pyeza.HTMLRenderer // typed; never dereferenced in this compile-only proof
	return &consumerapp.AppUIBundle{
		Renderer:       renderer,
		RenderIcon:     func(string) template.HTML { return "" },
		CommonLabels:   pyeza.CommonLabels{},
		TableLabels:    pyezatypes.TableLabels{},
		Messages:       map[string]string{},
		Translations:   struct{}{}, // any non-nil value satisfies the non-nil translations guard
		SidebarBuilder: func(activeNav, activeSubNav string) any { return nil },
		BottomNavBuilder: func(activeNav string) ([]pyezatypes.BottomNavTab, []pyezatypes.AppGridItem, []pyezatypes.AppGridGroup) {
			return nil, nil, nil
		},
		PortalSidebars: map[PrincipalType]SidebarBuilder(nil),
		ExtLabels:      struct{}{},
		RouteRewriter:  func(ctx context.Context) context.Context { return ctx },
		AuthLabels:     struct{}{},
	}
}

// cookieBuilderReference proves the agnostic cookie builders + the no-tag CSRF
// issuer compile and are callable with explicit secure bools.
func cookieBuilderReference(w http.ResponseWriter) {
	_ = consumermw.SessionCookieSpec("ichizen_session", "t", 604800, true)
	_ = consumermw.SessionRotateCookieSpec("ichizen_session", "t", true)
	_ = consumermw.SessionClearCookieSpec("ichizen_session", true)
	_ = consumermw.SessionLogoutClearCookieSpec("ichizen_session", true)
	_ = consumermw.WorkspaceCSRFCookieSpec("tok", true)
	_ = consumermw.WorkspaceCSRFClearCookieSpec(true)
	_ = consumermw.WorkspaceCSRFCookieName
	_ = consumermw.IssueWorkspaceCSRFCookie(w, []byte("k"), "sess", "ws", true)
}

// _fluentRefs keeps the references live without executing them.
var _fluentRefs = []any{fluentReferenceChain, fluentReferenceChainWithUI, cookieBuilderReference}
