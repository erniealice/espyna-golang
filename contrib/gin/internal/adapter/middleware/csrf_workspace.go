//go:build gin

package middleware

// csrf_workspace.go — workspace-claim CSRF middleware for Gin.
//
// TODO: Implement WorkspaceCSRFMiddleware for Gin. This middleware should mirror
// the vanilla net/http implementation
// (apps/service-admin/internal/infrastructure/input/http/middleware/csrf_workspace.go):
//
// Token format: v1.<base64url(sessionToken|workspaceID|nonce)>.<base64url(HMAC-SHA256)>
//
// The HMAC key follows the same env-var hierarchy as action_workspace_guard:
// WORKSPACE_FORM_HMAC_KEY -> PASSWORD_AUTH_RESET_TOKEN_SECRET.
//
// Middleware chain position:
//   session -> workspace_path -> CSRF -> action_workspace_guard -> timezone -> mux
//
// The existing csrf.go in this package implements the legacy opaque-token CSRF
// check. This file should add the workspace-claim extension on top: embed
// sessionToken and workspaceID in the CSRF token, sign with HMAC-SHA256, and
// verify the claims on unsafe methods.
//
// Public surface needed:
//   - IssueWorkspaceCSRFCookie(c *gin.Context, sessionToken, workspaceID string)
//   - NewWorkspaceCSRFMiddleware(cfg WorkspaceCSRFConfig) gin.HandlerFunc
//
// Blocked on: Server API — requires the session middleware to provide the
// session token and workspace_id via gin.Context or request context, and the
// HMAC secret to be wired through the adapter's Initialize method.
