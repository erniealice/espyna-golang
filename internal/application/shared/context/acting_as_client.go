package context

import "context"

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
func WithActingAsClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, keyActingAsClientID, clientID)
}

// GetActingAsClientIDFromContext returns the acting-as client scope, or "" when
// the caller is staff (workspace-wide) or no scope was set.
func GetActingAsClientIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(keyActingAsClientID).(string); ok {
		return v
	}
	return ""
}
