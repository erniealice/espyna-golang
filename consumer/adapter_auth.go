package consumer

import (
	"context"
	"fmt"

	serviceauthuc "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	"github.com/erniealice/espyna-golang/ports"
	dbinterfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
	svcauthpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
	pyezarender "github.com/erniealice/pyeza-golang/render"
)

// authProviderOperations is an alias for ports.AuthProvider so the consumer
// can talk to the underlying adapter without re-declaring the contract.
// Re-declaring would force return-type pinning (Go interface satisfaction
// requires exact return-type match) and silently fail at runtime when the
// upstream adapter returns ports.AuthService rather than a local mirror type.
type authProviderOperations = ports.AuthProvider

// databaseAuthOperations defines the extended operations available with the
// password provider.
type databaseAuthOperations interface {
	Register(ctx context.Context, email, password, firstName, lastName, mobileNumber string) (string, error)
	Login(ctx context.Context, email, password string) (string, *authpb.Identity, error)
	RequestPasswordReset(ctx context.Context, email string) (string, error)
	ExecutePasswordReset(ctx context.Context, token, newPassword string) error
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	HashPassword(password string) (string, error)
	CreateSession(ctx context.Context, userID string) (string, error)
	ValidateSession(ctx context.Context, token string) (string, error)
	InvalidateSession(ctx context.Context, token string) error
	GetSessionWorkspaceContext(ctx context.Context, token string) (wsUserID, wsID string)
}

// authServiceOperations is an alias for ports.AuthService — same reason as above.
type authServiceOperations = ports.AuthService

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Auth Adapter

Provides direct access to authentication operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY auth provider (Firebase, Password, Mock)
based on your CONFIG_AUTH_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewAuthAdapterFromContainer(container)

	// Verify JWT token
	result, err := adapter.VerifyToken(ctx, "Bearer eyJ...")

	// Check if auth is enabled
	if adapter.IsEnabled() {
	    // Auth is available
	}
*/

// AuthAdapter provides technology-agnostic access to authentication services.
// It wraps the AuthProvider interface and works with Firebase, JWT, Mock, etc.
type AuthAdapter struct {
	provider  authProviderOperations
	service   authServiceOperations
	container *Container
}

// NewAuthAdapterFromContainer creates an AuthAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's provider.
func NewAuthAdapterFromContainer(container *Container) *AuthAdapter {
	if container == nil {
		return nil
	}

	// Get auth provider from container
	providerContract := container.GetAuthProvider()
	if providerContract == nil {
		return nil
	}

	// The composition layer wraps providers in a generic ProviderWrapper to
	// satisfy contracts.Provider. The wrapper does NOT implement the auth
	// service surface (GetAuthService / IsHealthy / IsEnabled), so we must
	// unwrap to reach the concrete adapter before type-asserting.
	var raw any = providerContract
	if w, ok := providerContract.(interface{ Provider() interface{} }); ok {
		if inner := w.Provider(); inner != nil {
			raw = inner
		}
	}

	provider, ok := raw.(authProviderOperations)
	if !ok {
		return nil
	}

	// If the auth provider wants database operations (e.g. the password adapter),
	// inject them from the container. This keeps providers tech-agnostic — the
	// adapter asks for a DatabaseOperation, and the container supplies whichever
	// backend is active (postgres/firestore/mock).
	type operationsSettable interface {
		SetOperations(ops dbinterfaces.DatabaseOperation)
	}
	if settable, ok := provider.(operationsSettable); ok {
		if ops, ok := container.GetDatabaseOperations().(dbinterfaces.DatabaseOperation); ok {
			settable.SetOperations(ops)
		}
	}

	// Get auth service from provider
	service := provider.GetAuthService()

	return &AuthAdapter{
		provider:  provider,
		service:   service,
		container: container,
	}
}

// Close closes the auth adapter.
// Note: If created from container, this does NOT close the container.
func (a *AuthAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetProvider returns the underlying auth provider for advanced operations.
func (a *AuthAdapter) GetProvider() authProviderOperations {
	return a.provider
}

// GetService returns the underlying auth service for direct access.
func (a *AuthAdapter) GetService() authServiceOperations {
	return a.service
}

// Name returns the name of the underlying auth provider (e.g., "firebase", "password", "mock")
func (a *AuthAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the auth provider is enabled
func (a *AuthAdapter) IsEnabled() bool {
	if a.service == nil {
		return false
	}
	return a.service.IsEnabled()
}

// --- Auth Operations ---

// VerifyToken validates a JWT token and returns the validation result.
// The token should be the full token string (with or without "Bearer " prefix).
func (a *AuthAdapter) VerifyToken(ctx context.Context, token string) (*authpb.ValidateJwtTokenResponse, error) {
	if a.service == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}

	req := &authpb.ValidateJwtTokenRequest{
		Token: token,
	}

	return a.service.VerifyToken(ctx, req)
}

// VerifyTokenProto validates a JWT token using the protobuf request type directly.
// Use this for full control over all validation parameters.
func (a *AuthAdapter) VerifyTokenProto(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	if a.service == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}
	return a.service.VerifyToken(ctx, req)
}

// signInProviderVerifier is the optional surface a provider implements to
// expose the federated sign-in method (firebase.sign_in_provider claim)
// alongside the verified email. Satisfied by the Firebase adapter.
type signInProviderVerifier interface {
	VerifyIDTokenWithProvider(ctx context.Context, idToken string) (email, signInProvider string, err error)
}

// authCapabilityProvider is the optional surface a provider implements to report
// a user's sign-in methods (firebase ProviderUserInfo / password_hash presence).
// Satisfied by the Firebase and Password adapters.
type authCapabilityProvider interface {
	GetUserAuthCapability(ctx context.Context, userID string) (ports.AuthCapability, error)
}

// VerifyTokenWithSignInProvider verifies an ID token and returns the signed-in
// email plus the sign-in method (e.g. "microsoft.com"). When the active
// provider does not expose the richer surface it falls back to VerifyToken and
// returns an empty signInProvider (enforcement then no-ops for that token).
// Used by the Firebase web-login endpoint for the allow-list check (Layer 5).
func (a *AuthAdapter) VerifyTokenWithSignInProvider(ctx context.Context, idToken string) (email, signInProvider string, err error) {
	if v, ok := a.service.(signInProviderVerifier); ok && a.service != nil {
		return v.VerifyIDTokenWithProvider(ctx, idToken)
	}
	if v, ok := a.provider.(signInProviderVerifier); ok && a.provider != nil {
		return v.VerifyIDTokenWithProvider(ctx, idToken)
	}
	// Fallback: standard verification, email only, no method claim.
	resp, vErr := a.VerifyToken(ctx, idToken)
	if vErr != nil {
		return "", "", vErr
	}
	if resp == nil || !resp.IsValid || resp.Identity == nil {
		msg := "token validation failed"
		if resp != nil && resp.ErrorMessage != "" {
			msg = resp.ErrorMessage
		}
		return "", "", fmt.Errorf("%s", msg)
	}
	return resp.Identity.GetEmail(), "", nil
}

// GetUserAuthCapability reports the user's sign-in methods when the active
// provider exposes the optional capability. When neither the service nor the
// provider implements it (e.g. mock/noop), it fails CLOSED — HasPassword:false —
// so the UI hides the local reset form and the handler guard rejects a reset for
// a provider whose credential model is unknown. (DECISION: fail-closed default —
// see MISMATCH M-5.)
func (a *AuthAdapter) GetUserAuthCapability(ctx context.Context, userID string) (ports.AuthCapability, error) {
	if v, ok := a.service.(authCapabilityProvider); ok && a.service != nil {
		return v.GetUserAuthCapability(ctx, userID)
	}
	if v, ok := a.provider.(authCapabilityProvider); ok && a.provider != nil {
		return v.GetUserAuthCapability(ctx, userID)
	}
	return ports.AuthCapability{HasPassword: false}, nil
}

// IsHealthy checks if the auth provider is healthy and available.
func (a *AuthAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("auth provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// GetProviderName returns the name of the auth provider (for logging/debugging).
func (a *AuthAdapter) GetProviderName() string {
	if a.service == nil {
		return ""
	}
	return a.service.GetProviderName()
}

// --- Convenience Methods ---

// ValidateAndExtractUserID validates a token and extracts the user ID if valid.
// Returns the user ID on success, or an error if validation fails.
func (a *AuthAdapter) ValidateAndExtractUserID(ctx context.Context, token string) (string, error) {
	resp, err := a.VerifyToken(ctx, token)
	if err != nil {
		return "", err
	}

	if !resp.IsValid {
		return "", fmt.Errorf("token validation failed: %s", resp.ErrorMessage)
	}

	if resp.Identity == nil {
		return "", fmt.Errorf("no identity in token")
	}

	return resp.Identity.Id, nil
}

// ValidateAndExtractIdentity validates a token and extracts the identity if valid.
// Returns the identity on success, or an error if validation fails.
func (a *AuthAdapter) ValidateAndExtractIdentity(ctx context.Context, token string) (*authpb.Identity, error) {
	resp, err := a.VerifyToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if !resp.IsValid {
		return nil, fmt.Errorf("token validation failed: %s", resp.ErrorMessage)
	}

	return resp.Identity, nil
}

// ValidateAndExtractToken validates a token and extracts the decoded token if valid.
// Returns the decoded token on success, or an error if validation fails.
func (a *AuthAdapter) ValidateAndExtractToken(ctx context.Context, token string) (*authpb.JwtToken, error) {
	resp, err := a.VerifyToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if !resp.IsValid {
		return nil, fmt.Errorf("token validation failed: %s", resp.ErrorMessage)
	}

	return resp.Token, nil
}

// --- Database Auth Methods ---

// Register creates a new user account with the given credentials.
// Only supported by password provider. Returns ErrNotSupported for other providers.
func (a *AuthAdapter) Register(ctx context.Context, email, password, firstName, lastName, mobileNumber string) (string, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", fmt.Errorf("register not supported by %s provider", a.Name())
	}
	return dbAuth.Register(ctx, email, password, firstName, lastName, mobileNumber)
}

// Login authenticates a user with email/password and returns a session token + identity.
// Only supported by password provider.
func (a *AuthAdapter) Login(ctx context.Context, email, password string) (string, *authpb.Identity, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", nil, fmt.Errorf("login not supported by %s provider", a.Name())
	}
	return dbAuth.Login(ctx, email, password)
}

// RequestPasswordReset generates a reset token for the given email.
// Returns the raw token (caller sends it via email). Only supported by password provider.
func (a *AuthAdapter) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", fmt.Errorf("password reset not supported by %s provider", a.Name())
	}
	return dbAuth.RequestPasswordReset(ctx, email)
}

// ExecutePasswordReset validates a reset token and sets a new password.
// Only supported by password provider.
func (a *AuthAdapter) ExecutePasswordReset(ctx context.Context, token, newPassword string) error {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return fmt.Errorf("password reset not supported by %s provider", a.Name())
	}
	return dbAuth.ExecutePasswordReset(ctx, token, newPassword)
}

// sessionFallbackUC returns the provider-AGNOSTIC service-auth session use
// cases (the same ones MockSessionMiddleware drives), or nil when they are
// unavailable. Consulted ONLY for providers that do NOT implement
// databaseAuthOperations (e.g. firebase): the password provider keeps its own
// DB-backed SessionService, so for it the fallback is never reached. This is
// what makes server-side sessions provider-independent — a Firebase user (who
// authenticated by ID-token verification, with no DB password) still gets a
// real opaque session row in the SAME `session` table, created/validated/
// invalidated through the espyna application use cases.
//
// It is ALSO the generic accessor for the provider-agnostic session use cases
// that have no databaseAuthOperations equivalent at all (e.g.
// LookupSessionPrincipal, which reads the shared session row for EVERY
// provider including password) — those callers consult it unconditionally.
func (a *AuthAdapter) sessionFallbackUC() *serviceauthuc.UseCases {
	if a.container == nil {
		return nil
	}
	uc := a.container.GetUseCases()
	if uc == nil || uc.Service == nil {
		return nil
	}
	return uc.Service.Auth
}

// CreateSession creates a new session for the given user.
// Password provider mints via its own SessionService; every other provider
// (firebase) mints through the agnostic IssueSession use case (same table).
func (a *AuthAdapter) CreateSession(ctx context.Context, userID string) (string, error) {
	if dbAuth, ok := a.provider.(databaseAuthOperations); ok {
		return dbAuth.CreateSession(ctx, userID)
	}
	uc := a.sessionFallbackUC()
	if uc == nil || uc.IssueSession == nil {
		return "", fmt.Errorf("session management not available for %s provider", a.Name())
	}
	resp, err := uc.IssueSession.Execute(ctx, &svcauthpb.IssueSessionRequest{UserId: userID})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.GetToken() == "" {
		return "", fmt.Errorf("issue session: empty token")
	}
	return resp.GetToken(), nil
}

// ValidateSession checks if a session token is valid and returns the user ID.
// Password provider validates via its own SessionService; every other provider
// (firebase) validates through the agnostic AuthenticateSession use case.
func (a *AuthAdapter) ValidateSession(ctx context.Context, token string) (string, error) {
	if dbAuth, ok := a.provider.(databaseAuthOperations); ok {
		return dbAuth.ValidateSession(ctx, token)
	}
	uc := a.sessionFallbackUC()
	if uc == nil || uc.AuthenticateSession == nil {
		return "", fmt.Errorf("session management not available for %s provider", a.Name())
	}
	resp, err := uc.AuthenticateSession.Execute(ctx, &svcauthpb.AuthenticateSessionRequest{Token: token})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.GetIdentity() == nil || resp.GetIdentity().GetUserId() == "" {
		return "", fmt.Errorf("invalid session")
	}
	return resp.GetIdentity().GetUserId(), nil
}

// InvalidateSession marks a session as inactive.
// Password provider invalidates via its own SessionService; every other
// provider (firebase) invalidates through the agnostic InvalidateSession use case.
func (a *AuthAdapter) InvalidateSession(ctx context.Context, token string) error {
	if dbAuth, ok := a.provider.(databaseAuthOperations); ok {
		return dbAuth.InvalidateSession(ctx, token)
	}
	uc := a.sessionFallbackUC()
	if uc == nil || uc.InvalidateSession == nil {
		return fmt.Errorf("session management not available for %s provider", a.Name())
	}
	_, err := uc.InvalidateSession.Execute(ctx, &svcauthpb.InvalidateSessionRequest{Token: token})
	return err
}

// HashPassword hashes a plaintext password using bcrypt.
// Delegates to the password adapter's PasswordService so that consumers do not
// need to import bcrypt directly.
// Only supported by the password provider.
func (a *AuthAdapter) HashPassword(password string) (string, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", fmt.Errorf("hash password not supported by %s provider", a.Name())
	}
	return dbAuth.HashPassword(password)
}

// ChangePassword updates the password for an authenticated user.
// oldPassword must match the stored hash; newPassword is validated and hashed.
// The caller's current session is preserved (only the password_hash is updated).
// Returns a specific error when oldPassword is wrong — the user is authenticated
// so there is no enumeration risk.
// Only supported by the password provider.
func (a *AuthAdapter) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return fmt.Errorf("change password not supported by %s provider", a.Name())
	}
	return dbAuth.ChangePassword(ctx, userID, oldPassword, newPassword)
}

// GetSessionWorkspaceContext returns the workspace_user_id and workspace_id stored on the session.
// Password provider reads via its own SessionService; every other provider
// (firebase) reads through the agnostic AuthenticateSession use case. Returns
// empty strings when the token does not name a valid session (NOT an error —
// a valid session with no workspace selected yet also returns empties).
func (a *AuthAdapter) GetSessionWorkspaceContext(ctx context.Context, token string) (wsUserID, wsID string) {
	if dbAuth, ok := a.provider.(databaseAuthOperations); ok {
		return dbAuth.GetSessionWorkspaceContext(ctx, token)
	}
	uc := a.sessionFallbackUC()
	if uc == nil || uc.AuthenticateSession == nil {
		return "", ""
	}
	resp, err := uc.AuthenticateSession.Execute(ctx, &svcauthpb.AuthenticateSessionRequest{Token: token})
	if err != nil || resp == nil || resp.GetIdentity() == nil {
		return "", ""
	}
	id := resp.GetIdentity()
	return id.GetWorkspaceUserId(), id.GetWorkspaceId()
}

// SessionIdentity is the coherent identity snapshot resolved from a single
// session token: the authenticated user plus the workspace binding that token
// currently points at.
//
// Security-critical (P3-W3 — stale-binding carry-over on principal switch):
// every field here is keyed by the SAME live cookie token, so the workspace
// binding ALWAYS reflects the post-switch session row, never a cached or
// torn pre-switch value. See AuthAdapter.ResolveSessionIdentity.
type SessionIdentity struct {
	UserID          string
	WorkspaceID     string
	WorkspaceUserID string
}

// ResolveSessionIdentity validates the session token and resolves the
// workspace binding it points at, in that order, keyed by a SINGLE token.
//
// Why this exists (P3-W3 — eliminate stale-permission carry-over on principal
// switch): the prior SessionMiddleware flow issued ValidateSession(token) and
// GetSessionWorkspaceContext(token) as two independent reads. After a principal
// switch A→B the cookie token is either rotated (workspace change) or the
// in-place session row is mutated (same workspace, new binding). In both cases
// the binding the request must use is the one the CURRENT cookie token resolves
// to. Splitting the resolution across two reads opened a torn-read window: a
// switch committing between the two reads could stamp the request with a userID
// from one snapshot and a workspace binding from another.
//
// Collapsing resolution behind one accessor keyed on one token closes that
// window. The workspace binding is read ONLY after ValidateSession confirms the
// token names a live, non-expired, active session — so a session that was
// rotated out from under this request (old token → active=false) fails the
// validate step and never reaches the workspace read. There is no per-request
// cache here: every request re-resolves from the cookie token, so B's request
// can never observe A's binding.
//
// Returns an error only when the token does not name a valid session (the
// caller redirects to login). A valid session with no workspace selected yet
// returns empty WorkspaceID/WorkspaceUserID — that is a legitimate
// pre-selection state, NOT a denial, and is preserved exactly as before.
func (a *AuthAdapter) ResolveSessionIdentity(ctx context.Context, token string) (SessionIdentity, error) {
	userID, err := a.ValidateSession(ctx, token)
	if err != nil {
		return SessionIdentity{}, err
	}
	// Workspace binding is keyed by the same validated token, so it reflects
	// the post-switch session row. NULL/empty columns (e.g. workspace_user_id
	// after an operator→client in-place switch) surface as empty strings, which
	// is the correct post-switch value — not a carry-over from the old binding.
	wsUserID, wsID := a.GetSessionWorkspaceContext(ctx, token)
	return SessionIdentity{
		UserID:          userID,
		WorkspaceID:     wsID,
		WorkspaceUserID: wsUserID,
	}, nil
}

// LookupSessionPrincipal resolves the session row's ACTIVE binding (kind,
// principal_id, acting_as_*) for the given token. Provider-AGNOSTIC: the
// service-auth LookupSessionPrincipal use case reads the shared `session`
// table via its PrincipalResolver adapter, so it works identically for the
// password and firebase providers (unlike the databaseAuthOperations branch,
// which only the password adapter implements).
//
// Returns a ZERO SessionPrincipal (UNSPECIFIED kind, empty IDs) on any
// miss/error — the same fail-closed "no hint" sentinel the workspace-path
// PrincipalLookup and the view-adapter SetPrincipalLookup already treat as
// "no resolved binding". The session middleware stamps this onto the
// RequestIdentity via identity.WithSessionBinding so the Layer-4 RBAC backstop
// can scope to the active binding instead of unioning across all bindings.
func (a *AuthAdapter) LookupSessionPrincipal(ctx context.Context, token string) SessionPrincipal {
	if token == "" {
		return SessionPrincipal{}
	}
	uc := a.sessionFallbackUC()
	if uc == nil || uc.LookupSessionPrincipal == nil {
		return SessionPrincipal{}
	}
	resp, err := uc.LookupSessionPrincipal.Execute(ctx, &svcauthpb.LookupSessionPrincipalRequest{Token: token})
	if err != nil || resp == nil {
		return SessionPrincipal{}
	}
	return SessionPrincipal{
		Kind:               pyezarender.PrincipalType(resp.GetKind()),
		PrincipalID:        resp.GetPrincipalId(),
		ActingAsClientID:   resp.GetActingAsClientId(),
		ActingAsSupplierID: resp.GetActingAsSupplierId(),
	}
}

// --- Re-export error codes for consumer convenience ---

const (
	// AuthErrCodeMissingToken indicates no token was provided
	AuthErrCodeMissingToken = "AUTH_MISSING_TOKEN"
	// AuthErrCodeInvalidToken indicates the token format is invalid
	AuthErrCodeInvalidToken = "AUTH_INVALID_TOKEN"
	// AuthErrCodeExpiredToken indicates the token has expired
	AuthErrCodeExpiredToken = "AUTH_EXPIRED_TOKEN"
	// AuthErrCodeServiceDown indicates the auth service is unavailable
	AuthErrCodeServiceDown = "AUTH_SERVICE_DOWN"
	// AuthErrCodeUnauthorized indicates authorization was denied
	AuthErrCodeUnauthorized = "AUTH_UNAUTHORIZED"
)
