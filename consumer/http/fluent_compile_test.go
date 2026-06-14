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
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	"github.com/erniealice/pyeza-golang"
)

// fluentReferenceChain is the compile-only proof. A no-op block keeps the
// WithBlocks(...pyeza.AppOption) arm exercised.
func fluentReferenceChain() *Container {
	noopBlock := func(*pyeza.AppContext) error { return nil }
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
var _fluentRefs = []any{fluentReferenceChain, cookieBuilderReference}
