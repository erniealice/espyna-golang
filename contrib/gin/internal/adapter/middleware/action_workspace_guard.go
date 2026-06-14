//go:build gin

package middleware

// action_workspace_guard.go — workspace-scoped action guard for Gin.
//
// TODO: Implement ActionWorkspaceGuardMiddleware for Gin. This middleware should
// mirror the vanilla net/http implementation
// (apps/service-admin/internal/infrastructure/input/http/middleware/action_workspace_guard.go):
//
// For unsafe methods (POST/PUT/PATCH/DELETE) on /action/* paths, it requires:
//   - _workspace_id      — the workspace_id captured at form-render time
//   - _workspace_id_sig  — HMAC over (_workspace_id + action_path + nonce)
//
// The middleware verifies:
//   1. The HMAC signature is valid for this action path.
//   2. _workspace_id matches the session's current workspace_id.
//
// On mismatch: 409 Conflict + HX-Refresh: true for HTMX clients.
//
// Public surface needed:
//   - WorkspaceFormSigner for sign-at-render (SignFields method)
//   - NewActionWorkspaceGuardMiddleware(cfg ActionWorkspaceGuardConfig) gin.HandlerFunc
//
// HMAC key hierarchy: WORKSPACE_FORM_HMAC_KEY -> PASSWORD_AUTH_RESET_TOKEN_SECRET
//
// Blocked on: Server API — requires the HMAC secret, workspace_id context
// accessor, and form parser to be wired through the Gin adapter.
