package context

import "context"

// keyConfirm carries an explicit "user has confirmed the dangerous action"
// signal through the request context. The handler layer sets it from a query
// param / form field (e.g. `confirm=true`); use cases that gate on N>1
// invariants (see plan §3.5 in 20260427-plan-client-scope) read it via
// IsConfirmed.
const keyConfirm contextKey = "confirm"

// WithConfirm marks the context as confirmed.
func WithConfirm(ctx context.Context, confirmed bool) context.Context {
	return context.WithValue(ctx, keyConfirm, confirmed)
}

// IsConfirmed reports whether the caller has signalled confirmation.
// Defaults to false when the key is absent.
func IsConfirmed(ctx context.Context) bool {
	if v, ok := ctx.Value(keyConfirm).(bool); ok {
		return v
	}
	return false
}
