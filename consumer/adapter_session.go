package consumer

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/usecases/auth"
)

// SessionAdapter is the narrow public surface for session lifecycle
// (authenticate cookie, issue session, invalidate session). It wraps the
// usecases/auth use cases so downstream modules (HTTP middleware, login
// handlers, logout flows) never import espyna internals and never name
// internal request/response types.
//
// SessionAdapter is provider-independent: it works in mock_auth, password,
// or any other auth configuration as long as the Session + User proto
// repositories were wired during container initialization.
type SessionAdapter struct {
	useCases *UseCases
}

// SessionIdentity is the authenticated principal produced by AuthenticateCookie.
// Defined in the consumer package (not aliased from internal) so callers
// have a stable public type.
type SessionIdentity struct {
	UserID          string
	Email           string
	WorkspaceUserID string
	WorkspaceID     string
	Token           string
	ExpiresAtUnixMs int64
}

// NewSessionAdapter wires the adapter from a use case aggregate.
// Returns nil when useCases is nil or Auth use cases are missing.
func NewSessionAdapter(useCases *UseCases) *SessionAdapter {
	if useCases == nil || useCases.Auth == nil {
		return nil
	}
	return &SessionAdapter{useCases: useCases}
}

// NewSessionAdapterFromContainer pulls the use case aggregate off a container.
func NewSessionAdapterFromContainer(container *Container) *SessionAdapter {
	if container == nil {
		return nil
	}
	return NewSessionAdapter(container.GetUseCases())
}

// IsEnabled reports whether the adapter has a working AuthenticateSession
// and IssueSession use case pair wired.
func (a *SessionAdapter) IsEnabled() bool {
	return a != nil &&
		a.useCases != nil &&
		a.useCases.Auth != nil &&
		a.useCases.Auth.AuthenticateSession != nil &&
		a.useCases.Auth.IssueSession != nil
}

// AuthenticateCookie resolves an opaque session token into a fully hydrated
// SessionIdentity (user id, email, workspace context, expiry). Returns
// (_, false) on any failure — invalid token, expired session, unknown user.
// Never returns a partial identity.
func (a *SessionAdapter) AuthenticateCookie(ctx context.Context, token string) (*SessionIdentity, bool) {
	if a == nil || a.useCases == nil || a.useCases.Auth == nil || a.useCases.Auth.AuthenticateSession == nil || token == "" {
		return nil, false
	}
	resp, err := a.useCases.Auth.AuthenticateSession.Execute(ctx, &auth.AuthenticateSessionRequest{Token: token})
	if err != nil || resp == nil {
		return nil, false
	}
	return &SessionIdentity{
		UserID:          resp.Identity.UserID,
		Email:           resp.Identity.Email,
		WorkspaceUserID: resp.Identity.WorkspaceUserID,
		WorkspaceID:     resp.Identity.WorkspaceID,
		Token:           resp.Identity.Token,
		ExpiresAtUnixMs: resp.Identity.ExpiresAtUnixMs,
	}, true
}

// IssueSession mints a new session for the given user with optional
// workspace context. Pass empty strings for wsUserID / wsID when the user
// has no workspace binding yet.
func (a *SessionAdapter) IssueSession(ctx context.Context, userID, wsUserID, wsID string) (string, error) {
	if a == nil || a.useCases == nil || a.useCases.Auth == nil || a.useCases.Auth.IssueSession == nil {
		return "", errSessionAdapterNotWired
	}
	resp, err := a.useCases.Auth.IssueSession.Execute(ctx, &auth.IssueSessionRequest{
		UserID:          userID,
		WorkspaceUserID: wsUserID,
		WorkspaceID:     wsID,
	})
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

// InvalidateSession terminates the session addressed by an opaque token
// (logout semantics). Invalidating an unknown token is a no-op.
func (a *SessionAdapter) InvalidateSession(ctx context.Context, token string) error {
	if a == nil || a.useCases == nil || a.useCases.Auth == nil || a.useCases.Auth.InvalidateSession == nil {
		return errSessionAdapterNotWired
	}
	_, err := a.useCases.Auth.InvalidateSession.Execute(ctx, &auth.InvalidateSessionRequest{Token: token})
	return err
}

type sessionAdapterError string

func (e sessionAdapterError) Error() string { return string(e) }

const errSessionAdapterNotWired sessionAdapterError = "session adapter: auth use cases not wired"
