package consumer

import (
	"context"
	"fmt"
	"log"
	"net/http"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	sharedidentity "github.com/erniealice/espyna-golang/shared/identity"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// MockSessionIdentity holds the identity fields resolved from a mock session.
// This is a superset of the production SessionIdentity — the mock needs
// Email and Token for dev-mode bootstrapping.
type MockSessionIdentity struct {
	UserID          string
	Email           string
	WorkspaceUserID string
	WorkspaceID     string
	Token           string
	ExpiresAtUnixMs int64
}

// MockSessionMiddleware provides automatic session management for dev mode.
// On each request it checks for a session cookie; if absent, it auto-creates
// a user and session through espyna's application use cases, sets the
// cookie, and injects the identity into the request context.
//
// This middleware belongs in consumer/ (the framework boundary) rather than
// in service-admin's middleware package because it couples tightly to espyna
// use cases (useCases.Service.Auth.*, useCases.Entity.User.*) and must use
// the same cookie name and context keys as the production SessionMiddleware.
//
// Fidelity fixes over the prior service-admin copy:
//   - Uses DefaultSessionCookieName ("ichizen_session") instead of "session_token"
//   - Writes ContextKeySessionToken to context (workspace_path middleware reads it)
type MockSessionMiddleware struct {
	useCases            *UseCases
	testUserID          string
	testEmail           string
	testWorkspaceUserID string
	defaultWorkspaceID  string

	// Populated by bootstrapSession after resolving the user's actual
	// workspace_user row. Consumed by the outer Handle to inject the correct
	// workspace into the request context for the just-bootstrapped request.
	lastResolvedWorkspaceUserID string
	lastResolvedWorkspaceID     string
}

// NewMockSessionMiddleware wires the middleware from the espyna use case
// aggregate. Auth use cases (AuthenticateSession, IssueSession,
// InvalidateSession) are reached through useCases.Service.Auth.* via the
// service-driven sub-aggregate.
func NewMockSessionMiddleware(
	useCases *UseCases,
	testUserID, testEmail, testWorkspaceUserID, defaultWorkspaceID string,
) *MockSessionMiddleware {
	return &MockSessionMiddleware{
		useCases:            useCases,
		testUserID:          testUserID,
		testEmail:           testEmail,
		testWorkspaceUserID: testWorkspaceUserID,
		defaultWorkspaceID:  defaultWorkspaceID,
	}
}

// Handle is the middleware handler that ensures every request has a valid session.
func (m *MockSessionMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isMockSessionStaticAsset(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Cookie path: authenticate via the auth use case directly.
		if cookie, err := r.Cookie(DefaultSessionCookieName); err == nil && cookie.Value != "" {
			if identity, ok := m.authenticateCookie(r.Context(), cookie.Value); ok {
				ctx := m.injectIdentity(r.Context(), identity)
				log.Printf("[mock-session] cookie-resolved uid=%s email=%s path=%s",
					identity.UserID, identity.Email, r.URL.Path)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			log.Printf("[mock-session] cookie present but invalid (token=%s...), falling back to test user",
				cookie.Value[:min(8, len(cookie.Value))])
		}

		// No valid cookie: bootstrap the dev identity and issue a fresh session.
		token, err := m.bootstrapSession(r.Context())
		if err != nil {
			log.Printf("session middleware: failed to create session: %v", err)
			http.Error(w, "Failed to initialize session", http.StatusInternalServerError)
			return
		}

		// 1-year dev session cookie via the agnostic builder. secure=FALSE to
		// preserve the historical Secure-absent bytes (the dev mock is
		// CONFIG_AUTH_PROVIDER=mock, never prod — A.2.2 MOCK Secure NOTE). The
		// builder writes Secure:false which serializes identically to the prior
		// Secure-field-omitted cookie.
		http.SetCookie(w, consumermw.SessionCookieSpec(DefaultSessionCookieName, token, 86400*365, false))

		// Prefer the workspace_user the bootstrap just resolved over the env
		// default — the user may belong to a different workspace than
		// DEFAULT_WORKSPACE_ID, and the freshly-issued session already stores
		// the resolved values.
		resolvedWsUserID := m.lastResolvedWorkspaceUserID
		resolvedWsID := m.lastResolvedWorkspaceID
		if resolvedWsUserID == "" {
			resolvedWsUserID = m.testWorkspaceUserID
		}
		if resolvedWsID == "" {
			resolvedWsID = m.defaultWorkspaceID
		}
		ctx := m.injectIdentity(r.Context(), &MockSessionIdentity{
			UserID:          m.testUserID,
			Email:           m.testEmail,
			WorkspaceUserID: resolvedWsUserID,
			WorkspaceID:     resolvedWsID,
			Token:           token,
		})
		log.Printf("[mock-session] auto-created uid=%s workspace=%s path=%s", m.testUserID, resolvedWsID, r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateCookie resolves an opaque session token into a MockSessionIdentity,
// or returns (nil, false) on any failure.
func (m *MockSessionMiddleware) authenticateCookie(ctx context.Context, token string) (*MockSessionIdentity, bool) {
	if m.useCases == nil || m.useCases.Service == nil || m.useCases.Service.Auth == nil ||
		m.useCases.Service.Auth.AuthenticateSession == nil || token == "" {
		return nil, false
	}
	resp, err := m.useCases.Service.Auth.AuthenticateSession.Execute(ctx, &authpb.AuthenticateSessionRequest{Token: token})
	if err != nil || resp == nil || resp.GetIdentity() == nil {
		return nil, false
	}
	identity := resp.GetIdentity()
	return &MockSessionIdentity{
		UserID:          identity.GetUserId(),
		Email:           identity.GetEmail(),
		WorkspaceUserID: identity.GetWorkspaceUserId(),
		WorkspaceID:     identity.GetWorkspaceId(),
		Token:           identity.GetToken(),
		ExpiresAtUnixMs: identity.GetExpiresAtUnixMs(),
	}, true
}

// issueSession mints a new session for the given user/workspace context
// via the service-driven Auth sub-aggregate.
func (m *MockSessionMiddleware) issueSession(ctx context.Context, userID, wsUserID, wsID string) (string, error) {
	if m.useCases == nil || m.useCases.Service == nil || m.useCases.Service.Auth == nil ||
		m.useCases.Service.Auth.IssueSession == nil {
		return "", fmt.Errorf("mock session middleware: IssueSession use case not wired")
	}
	resp, err := m.useCases.Service.Auth.IssueSession.Execute(ctx, &authpb.IssueSessionRequest{
		UserId:          userID,
		WorkspaceUserId: wsUserID,
		WorkspaceId:     wsID,
	})
	if err != nil {
		return "", err
	}
	return resp.GetToken(), nil
}

// injectIdentity places the MockSessionIdentity into the request context using
// the espyna-owned context keys.
//
// Fidelity fix: writes ContextKeySessionToken so downstream middleware
// (workspace_path) can read the session token via GetSessionTokenFromContext.
// The prior service-admin copy omitted this, causing silent failures.
func (m *MockSessionMiddleware) injectIdentity(ctx context.Context, id *MockSessionIdentity) context.Context {
	wsUserID := id.WorkspaceUserID
	wsID := id.WorkspaceID
	if wsUserID == "" {
		wsUserID = m.testWorkspaceUserID
	}
	if wsID == "" {
		wsID = m.defaultWorkspaceID
	}
	ctx = WithSessionIdentity(ctx, id.UserID, wsID, wsUserID, id.Email)
	ctx = context.WithValue(ctx, ContextKeyUserID, id.UserID)
	ctx = context.WithValue(ctx, ContextKeySessionToken, id.Token)

	// Stamp the session token onto the RequestIdentity that
	// WithSessionIdentity just stored. The struct is stored by pointer,
	// so mutating the retrieved pointer updates the context value in place.
	if rid, ok := sharedidentity.FromContext(ctx); ok {
		rid.SessionToken = id.Token
	}

	return ctx
}

// bootstrapSession guarantees the dev user + workspace_user exist, then
// mints a fresh session via the auth use case.
//
// The CRUD use cases we dispatch into (ReadUser, CreateUser, ReadWorkspaceUser,
// CreateWorkspaceUser) enforce authcheck. We therefore inject the test
// identity into the bootstrap context so authcheck resolves the current
// user as the superadmin seed — who, in the RBAC seed data, holds every
// permission. Dev-mode only; never reachable outside CONFIG_AUTH_PROVIDER=mock.
func (m *MockSessionMiddleware) bootstrapSession(ctx context.Context) (string, error) {
	if m.useCases == nil || m.useCases.Service == nil || m.useCases.Service.Auth == nil {
		return "", fmt.Errorf("mock session middleware: service-driven auth use cases not wired")
	}

	bootstrapCtx := WithSessionIdentity(ctx, m.testUserID, m.defaultWorkspaceID, m.testWorkspaceUserID, m.testEmail)

	if err := m.ensureUser(bootstrapCtx); err != nil {
		return "", fmt.Errorf("ensure user: %w", err)
	}

	wsUserID, wsID, err := m.resolveOrCreateWorkspaceUser(bootstrapCtx)
	if err != nil {
		log.Printf("[mock-session] resolveOrCreateWorkspaceUser failed (continuing with env default): %v", err)
		wsUserID, wsID = m.testWorkspaceUserID, m.defaultWorkspaceID
	}

	token, err := m.issueSession(bootstrapCtx, m.testUserID, wsUserID, wsID)
	if err != nil {
		return "", fmt.Errorf("issue session: %w", err)
	}

	// Cache on the middleware so the outer handler can inject the resolved
	// workspace into the request context for the current request (otherwise
	// the first request after a fresh bootstrap still runs with the env
	// default because injectIdentity is called with m.defaultWorkspaceID).
	m.lastResolvedWorkspaceUserID = wsUserID
	m.lastResolvedWorkspaceID = wsID

	log.Printf("Mock session created for user %s workspace=%s (token: %s...)",
		m.testUserID, wsID, token[:min(8, len(token))])
	return token, nil
}

// ensureUser creates the dev test user if it doesn't already exist, routing
// through ReadUser -> CreateUser. Each call enforces its own authcheck; in
// mock mode that check is a no-op.
func (m *MockSessionMiddleware) ensureUser(ctx context.Context) error {
	if m.useCases == nil || m.useCases.Entity == nil || m.useCases.Entity.User == nil {
		return fmt.Errorf("user use cases unavailable")
	}
	userUC := m.useCases.Entity.User

	readResp, err := userUC.ReadUser.Execute(ctx, &userpb.ReadUserRequest{
		Data: &userpb.User{Id: m.testUserID},
	})
	if err == nil && readResp != nil && len(readResp.Data) > 0 {
		return nil
	}

	_, err = userUC.CreateUser.Execute(ctx, &userpb.CreateUserRequest{
		Data: &userpb.User{
			Id:           m.testUserID,
			FirstName:    "Super",
			LastName:     "Admin",
			EmailAddress: m.testEmail,
			MobileNumber: "+639000000000",
		},
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	log.Printf("Mock user created: %s (%s)", m.testUserID, m.testEmail)
	return nil
}

// resolveOrCreateWorkspaceUser returns the (workspace_user_id, workspace_id)
// pair the session should adopt. It prefers an existing workspace_user row for
// the authenticated user, and only creates a new one when the user has no
// workspace_user rows at all. Preference order among existing rows:
//  1. A row whose workspace_id matches defaultWorkspaceID (env preference).
//  2. The first active row returned (deterministic because ListWorkspaceUsers
//     orders by date_created DESC — most recent wins).
func (m *MockSessionMiddleware) resolveOrCreateWorkspaceUser(ctx context.Context) (string, string, error) {
	if m.testWorkspaceUserID == "" || m.defaultWorkspaceID == "" {
		return m.testWorkspaceUserID, m.defaultWorkspaceID, nil
	}
	if m.useCases == nil || m.useCases.Entity == nil || m.useCases.Entity.WorkspaceUser == nil {
		return "", "", fmt.Errorf("workspace_user use cases unavailable")
	}
	wsUserUC := m.useCases.Entity.WorkspaceUser

	listResp, err := wsUserUC.ListWorkspaceUsers.Execute(ctx, &workspaceuserpb.ListWorkspaceUsersRequest{})
	if err == nil && listResp != nil {
		var fallback *workspaceuserpb.WorkspaceUser
		for _, wu := range listResp.GetData() {
			if wu == nil || wu.GetUserId() != m.testUserID || !wu.GetActive() {
				continue
			}
			if wu.GetWorkspaceId() == m.defaultWorkspaceID {
				return wu.GetId(), wu.GetWorkspaceId(), nil
			}
			if fallback == nil {
				fallback = wu
			}
		}
		if fallback != nil {
			return fallback.GetId(), fallback.GetWorkspaceId(), nil
		}
	}

	_, err = wsUserUC.CreateWorkspaceUser.Execute(ctx, &workspaceuserpb.CreateWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{
			Id:          m.testWorkspaceUserID,
			WorkspaceId: m.defaultWorkspaceID,
			UserId:      m.testUserID,
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("create workspace_user: %w", err)
	}
	return m.testWorkspaceUserID, m.defaultWorkspaceID, nil
}

// isMockSessionStaticAsset checks if the path is for a static asset (skip session check).
func isMockSessionStaticAsset(path string) bool {
	return len(path) >= 8 && path[:8] == "/assets/" || path == "/favicon.ico"
}
