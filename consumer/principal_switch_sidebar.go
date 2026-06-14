// Package consumer -- principal_switch_sidebar.go
//
// Portable sidebar workspace-switcher utilities. The appBuilder-dependent
// wiring (secureSidebarSwitchFn, secureSidebarSetSessionCookie) stays in
// the composition root; this file holds the framework-level helpers.
//
// Moved from apps/service-admin/internal/composition/.

package consumer

import (
	"net/http"
)

// SecureSidebarResolveUserID extracts the authenticated user_id from the
// request context (session middleware has already run).
func SecureSidebarResolveUserID(r *http.Request) string {
	return GetUserIDFromContext(r.Context())
}

// SecureSidebarSwitchWired reports whether the secure principal-switch
// primitive is available for this build/dialect.
// sessionMwAvail and switchPrincipalAvail are caller-provided nil checks
// for the session middleware and SwitchPrincipal/ResolvePrincipals use cases.
func SecureSidebarSwitchWired(sessionMwAvail, switchPrincipalAvail, resolvePrincipalsAvail bool) bool {
	return SecureSidebarPrimitiveAvailable() &&
		sessionMwAvail &&
		switchPrincipalAvail &&
		resolvePrincipalsAvail
}
