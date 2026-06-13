// Package context owns the request-scoped values that flow through every use
// case Execute body — user ID, workspace ID, workspace-user ID, email, business
// type, confirmation flags, translation handles, and the spawn-jobs/schedule
// hints. The context keys are unexported so callers MUST go through the typed
// With*/Require*/Get* helpers; the application layer owns the keyspace and no
// adapter may stuff or read raw context values directly. It is a Layer-3 leaf
// (see hexagonal-rules.md §5) sitting beneath the use case layer.
//
// # Internal-reads bridge (N9 Phase 1)
//
// As of the N9 P1 migration, every reader helper first checks for the typed
// RequestIdentity struct (stored by shared/identity.WithRequestIdentity). If
// the struct is present, the field is read directly from it — fail-CLOSED: a
// missing struct on a middleware-protected route is a programming error. If the
// struct is NOT present, the helpers fall back to the legacy per-key context
// values for backward compatibility during the transition period (P1a migrates
// the writers). This dual-read ensures the ~988 importer files keep compiling
// and working correctly regardless of whether the write side has been migrated.
//
// Charter — this package MUST NOT import:
//   - proto entity types (esqyma/...)
//   - DB drivers or adapter packages
//   - anything under internal/application/usecases/...
//
// Depends only on the Go standard library plus internal/application/ports
// (for the Translator handle carried on the context), plus shared/identity
// (for the RequestIdentity struct — a dependency-free public leaf).
//
// Consumers (keep in sync):
//   - The full use case layer: usecases/domain/<X>/** and usecases/service/<X>/**
//     read user/workspace identity at the top of every Execute and emit
//     context-aware translated errors (~976 .go files import this package).
//   - The driven adapters that translate transport context into application
//     context: contrib/{postgres,mysql,sqlserver,fiber,http} request setup.
//   - internal/application/shared/{authcheck,testutil} — authcheck reads the
//     user ID for permission checks; testutil seeds user/business-type for tests.
//   - internal/orchestration/engine and the grpc/rbac infrastructure adapters.
//
// Adding a new caller is expected (this is the canonical request-context leaf);
// adding a new context KEY is the change that needs review — keep the keyspace
// small and the helpers typed.
package context

import (
	"context"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// contextKey is unexported — forces usage through helpers (hexagonal: application layer owns keys)
type contextKey string

const (
	keyUserID          contextKey = "user_id"
	keyWorkspaceID     contextKey = "workspace_id"
	keyWorkspaceUserID contextKey = "workspace_user_id"
	keyEmail           contextKey = "email"
)

// --- Writers ---
//
// These legacy writers remain for backward compatibility. After N9 P1a
// migrates the write side, middleware will call identity.WithRequestIdentity
// instead, and these individual writers will be dead code (deletable).

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, keyUserID, userID)
}

func WithWorkspaceID(ctx context.Context, wsID string) context.Context {
	return context.WithValue(ctx, keyWorkspaceID, wsID)
}

func WithWorkspaceUserID(ctx context.Context, wsUserID string) context.Context {
	return context.WithValue(ctx, keyWorkspaceUserID, wsUserID)
}

func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, keyEmail, email)
}

func WithSessionIdentity(ctx context.Context, userID, workspaceID, workspaceUserID, email string) context.Context {
	ctx = WithUserID(ctx, userID)
	if workspaceID != "" {
		ctx = WithWorkspaceID(ctx, workspaceID)
	}
	if workspaceUserID != "" {
		ctx = WithWorkspaceUserID(ctx, workspaceUserID)
	}
	if email != "" {
		ctx = WithEmail(ctx, email)
	}
	return ctx
}

// --- Readers ---
//
// Bridge pattern (N9 P1): check the RequestIdentity struct first. If present,
// read from the struct (fail-CLOSED). If absent, fall back to legacy per-key
// context values (backward compat during the P1a transition).

func ExtractUserIDFromContext(ctx context.Context) string {
	// Bridge: prefer the typed RequestIdentity struct
	if id, ok := identity.FromContext(ctx); ok {
		return id.UserID
	}
	// Legacy fallback: per-key context values
	if v, ok := ctx.Value(keyUserID).(string); ok && v != "" {
		return v
	}
	// Backward compat: legacy plain string keys
	if v, ok := ctx.Value("uid").(string); ok && v != "" {
		return v
	}
	if v, ok := ctx.Value("user_id").(string); ok && v != "" {
		return v
	}
	return ""
}

func ExtractWorkspaceIDFromContext(ctx context.Context) string {
	if id, ok := identity.FromContext(ctx); ok {
		return id.WorkspaceID
	}
	if v, ok := ctx.Value(keyWorkspaceID).(string); ok && v != "" {
		return v
	}
	return ""
}

func ExtractWorkspaceUserIDFromContext(ctx context.Context) string {
	if id, ok := identity.FromContext(ctx); ok {
		return id.WorkspaceUserID
	}
	if v, ok := ctx.Value(keyWorkspaceUserID).(string); ok && v != "" {
		return v
	}
	return ""
}

func ExtractEmailFromContext(ctx context.Context) string {
	if id, ok := identity.FromContext(ctx); ok {
		return id.Email
	}
	if v, ok := ctx.Value(keyEmail).(string); ok && v != "" {
		return v
	}
	if v, ok := ctx.Value("email").(string); ok && v != "" {
		return v
	}
	return ""
}

func RequireUserIDFromContext(ctx context.Context) (string, error) {
	uid := ExtractUserIDFromContext(ctx)
	if uid == "" {
		return "", ErrUserNotFoundInContext
	}
	return uid, nil
}

func HasUserInContext(ctx context.Context) bool {
	return ExtractUserIDFromContext(ctx) != ""
}
