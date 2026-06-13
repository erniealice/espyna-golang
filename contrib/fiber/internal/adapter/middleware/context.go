//go:build fiber

package middleware

import (
	"context"

	"github.com/erniealice/espyna-golang/shared/identity"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
)

// contextKey is a private type for context keys defined in this package.
// The vanilla net/http reference uses bare string keys; we use a typed key to
// avoid collisions while carrying the identical email / identity / expires
// claims that the authentication middleware sets and downstream code reads.
type contextKey string

const (
	ctxKeyIdentity contextKey = "identity"
	ctxKeyExpires  contextKey = "expires"
)

// contextWithValue is a thin wrapper around context.WithValue using the
// package-local typed key, keeping the call sites readable.
func contextWithValue(ctx context.Context, key contextKey, val any) context.Context {
	return context.WithValue(ctx, key, val)
}

// GetUserFromContext extracts user information from the request user context.
// Mirrors vanilla contrib/http GetUserFromContext: uid comes from the shared
// context util, email from the email claim. ok is false when uid is empty.
func GetUserFromContext(ctx context.Context) (uid string, email string, ok bool) {
	id, found := identity.FromContext(ctx)
	if !found || id.UserID == "" {
		return "", "", false
	}
	return id.UserID, id.Email, true
}

// GetIdentityFromContext extracts the full identity from the request user context.
// Mirrors vanilla contrib/http GetIdentityFromContext.
func GetIdentityFromContext(ctx context.Context) (*authpb.Identity, bool) {
	authID, ok := ctx.Value(ctxKeyIdentity).(*authpb.Identity)
	return authID, ok
}

// GetWorkspaceFromContext extracts the workspace ID from the request user context.
// Mirrors vanilla contrib/http GetWorkspaceFromContext (delegates to the shared
// context util so the workspace ID set by the session layer is honored).
func GetWorkspaceFromContext(ctx context.Context) string {
	return identity.Must(ctx).WorkspaceID
}
