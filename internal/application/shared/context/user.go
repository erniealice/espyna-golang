package context

import (
	"context"
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

func ExtractUserIDFromContext(ctx context.Context) string {
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
	if v, ok := ctx.Value(keyWorkspaceID).(string); ok && v != "" {
		return v
	}
	return ""
}

func ExtractWorkspaceUserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(keyWorkspaceUserID).(string); ok && v != "" {
		return v
	}
	return ""
}

func ExtractEmailFromContext(ctx context.Context) string {
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
