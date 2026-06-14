package middleware

import impl "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/middleware"

// WorkspacePathConfig configures the WorkspacePath middleware wrapper.
type WorkspacePathConfig = impl.WorkspacePathConfig

// WorkspaceSwitchResult is the outcome of a URL-driven principal switch.
type WorkspaceSwitchResult = impl.WorkspaceSwitchResult

// WorkspacePath returns a MiddlewareFunc that parses /w/{slug}/* URL paths,
// resolves workspace slugs to workspace IDs, validates user bindings, and
// optionally rotates sessions on cross-workspace navigation.
func WorkspacePath(cfg WorkspacePathConfig) MiddlewareFunc { return impl.WorkspacePath(cfg) }
