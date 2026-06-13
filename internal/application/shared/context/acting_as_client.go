package context

import (
	"context"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// keyActingAsClientID is the context key under which the session's acting-as
// client scope is stored. For a client-portal principal this is the client_id
// the caller is currently acting as; for a staff principal it is empty.
//
// Communication directive Q-MSG-5 / invariant I1: the conversation IDOR anchor
// (conversation.client_id) is stamped from THIS value, never from the request
// body. Auth/portal middleware populates it; use cases read it via
// GetActingAsClientIDFromContext.
const keyActingAsClientID contextKey = "acting_as_client_id"

// WithActingAsClientID stores the acting-as client scope on the context.
// Legacy writer — after N9 P1a, middleware calls identity.WithRequestIdentity
// instead and this becomes dead code.
func WithActingAsClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, keyActingAsClientID, clientID)
}

// GetActingAsClientIDFromContext returns the acting-as client scope, or "" when
// the caller is staff (workspace-wide) or no scope was set.
//
// Bridge (N9 P1): reads from the RequestIdentity struct first, falling back to
// the legacy per-key context value during the P1a transition.
func GetActingAsClientIDFromContext(ctx context.Context) string {
	if id, ok := identity.FromContext(ctx); ok {
		return id.ActingAsClientID
	}
	if v, ok := ctx.Value(keyActingAsClientID).(string); ok {
		return v
	}
	return ""
}
