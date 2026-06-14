//go:build gin

package middleware

// workspace_path.go — URL-driven workspace context resolution for Gin.
//
// TODO: Implement WorkspacePathMiddleware for Gin. This middleware should mirror
// the vanilla net/http implementation
// (apps/service-admin/internal/infrastructure/input/http/middleware/workspace_path.go):
//
//  1. Match /w/{workspace_slug}[/as/{client_id}][/rest...] via regex.
//  2. Validate slug format (^[a-z0-9]+(?:-[a-z0-9]+)*$ length 3-30) and
//     reject reserved-word collisions.
//  3. CSRF preflight (Sec-Fetch-Site + Sec-Fetch-Mode).
//  4. Slug -> workspace_id resolution via LRU cache (5 min TTL).
//  5. Read session context from upstream session middleware.
//  6. Active-binding validation via BindingResolver.
//  7. Slug-not-found and binding-missing -> 303 to /auth/select-workspace-role.
//  8. Optional /as/{client_id} extraction.
//  9. Per-user rotation rate limit (10/min default).
// 10. URL-driven session rotation when URL workspace differs from session workspace.
// 11. SameSite=Strict cookie rewrite on rotation.
// 12. Bind workspace_id into request context.
// 13. Strip /w/{slug}[/as/{client_id}] prefix and dispatch.
//
// Blocked on: Server API — requires the composition layer's ExecuteSwitch
// primitive, BindingResolver, session cookie writer, and slug-to-workspace
// resolver to be wired through the Gin adapter's Initialize method.
