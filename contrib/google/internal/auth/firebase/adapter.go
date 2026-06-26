package firebase

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterAuthProvider(
		"firebase",
		func() ports.AuthProvider {
			return NewAdapter()
		},
		transformConfig,
	)
	registry.RegisterAuthBuildFromEnv("firebase", buildFromEnv)
}

// buildFromEnv creates and initializes a Firebase auth provider from environment variables.
func buildFromEnv() (ports.AuthProvider, error) {
	projectID := os.Getenv("AUTH_FIREBASE_PROJECT_ID")
	protoConfig := &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_GCP,
		DisplayName: "Firebase",
		Config: &authpb.ProviderConfig_GcpConfig{
			GcpConfig: &authpb.GcpProviderConfig{
				ProjectId: projectID,
			},
		},
	}
	p := NewAdapter()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("firebase_auth: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to Firebase auth proto config.
func transformConfig(rawConfig map[string]any) (*authpb.ProviderConfig, error) {
	projectID, _ := rawConfig["project_id"].(string)
	return &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_GCP,
		DisplayName: "Firebase",
		Config: &authpb.ProviderConfig_GcpConfig{
			GcpConfig: &authpb.GcpProviderConfig{
				ProjectId: projectID,
			},
		},
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// FirebaseAuthAdapter implements ports.AuthProvider and ports.AuthService
// This adapter translates proto contracts to Firebase Auth SDK operations
// Following the same pattern as other adapters for consistency
type FirebaseAuthAdapter struct {
	config        *authpb.ProviderConfig
	enabled       bool
	timeout       time.Duration
	clientManager *FirebaseClientManager
}

// NewAdapter creates a new Firebase auth adapter
func NewAdapter() ports.AuthProvider {
	return &FirebaseAuthAdapter{
		enabled: false,
		timeout: 30 * time.Second,
	}
}

// Name returns the provider name
func (p *FirebaseAuthAdapter) Name() string {
	return "firebase"
}

// Initialize sets up Firebase auth with proto-based configuration
func (p *FirebaseAuthAdapter) Initialize(config *authpb.ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Verify provider type
	if config.Provider != authpb.Provider_PROVIDER_GCP {
		return fmt.Errorf("invalid provider type: expected GCP (Firebase), got %v", config.Provider)
	}

	// Extract GCP-specific config (Firebase uses GCP provider)
	gcpConfig := config.GetGcpConfig()
	if gcpConfig == nil {
		return fmt.Errorf("firebase auth requires GCP provider config")
	}

	// Initialize Firebase client manager using new pattern
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	// Create Firebase client manager
	manager, err := NewFirebaseClientManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Firebase client manager: %w", err)
	}

	p.clientManager = manager

	p.config = config
	p.enabled = config.Enabled

	log.Printf("[OK] Firebase Auth provider initialized (project: %s)", gcpConfig.ProjectId)
	return nil
}

// GetAuthService returns the authentication service (returns itself)
// Firebase provider implements both AuthProvider and AuthService interfaces
func (p *FirebaseAuthAdapter) GetAuthService() ports.AuthService {
	if !p.enabled {
		return nil
	}
	return p
}

// IsHealthy checks if Firebase auth is available
func (p *FirebaseAuthAdapter) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("firebase auth provider is not enabled")
	}

	// Test Firebase Auth client accessibility
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if p.clientManager == nil {
		return fmt.Errorf("firebase client manager not initialized")
	}

	authClient, err := p.clientManager.GetAuthClient(healthCtx)
	if err != nil {
		return fmt.Errorf("firebase auth client not available: %w", err)
	}
	if authClient == nil {
		return fmt.Errorf("firebase auth client not available")
	}

	return nil
}

// Close cleans up Firebase auth resources
func (p *FirebaseAuthAdapter) Close() error {
	if p.enabled {
		log.Println("[AUTH] Closing Firebase Auth provider")
		if p.clientManager != nil {
			if err := p.clientManager.Close(); err != nil {
				return fmt.Errorf("failed to close Firebase client manager: %w", err)
			}
		}
		p.enabled = false
	}
	return nil
}

// IsEnabled returns whether Firebase auth is enabled
func (p *FirebaseAuthAdapter) IsEnabled() bool {
	return p.enabled
}

// VerifyToken implements the AuthService interface using proto types
// This method handles JWT token verification against Firebase Auth
func (p *FirebaseAuthAdapter) VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	if !p.enabled {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Authentication service is disabled",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: "Service disabled",
				},
			},
		}, nil
	}

	// Get Firebase auth client from client manager
	if p.clientManager == nil {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Firebase client manager not initialized",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: "Client manager not initialized",
				},
			},
		}, nil
	}

	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Failed to get Firebase auth client",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: err.Error(),
				},
			},
		}, nil
	}
	if authClient == nil {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Firebase auth client not available",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: "Client not initialized",
				},
			},
		}, nil
	}

	// Verify the ID token
	firebaseToken, err := authClient.VerifyIDToken(ctx, req.Token)
	if err != nil {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Invalid or expired token",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_INVALID_SIGNATURE,
					Message: err.Error(),
				},
			},
		}, nil
	}

	// Convert Firebase token to proto types
	identity := &authpb.Identity{
		Id:          firebaseToken.UID,
		Type:        authpb.IdentityType_IDENTITY_TYPE_USER,
		Provider:    authpb.Provider_PROVIDER_GCP, // Firebase = GCP
		Email:       getStringClaim(firebaseToken.Claims, "email"),
		DisplayName: getStringClaim(firebaseToken.Claims, "name"),
		IsActive:    true,
		// Provider-specific identity
		ProviderIdentity: &authpb.Identity_GcpIdentity{
			GcpIdentity: &authpb.GcpIdentity{
				GoogleId: firebaseToken.UID,
			},
		},
	}

	jwtToken := &authpb.JwtToken{
		Token:     req.Token,
		TokenType: "Bearer",
		ExpiresAt: timestamppb.New(time.Unix(firebaseToken.Expires, 0)),
		IssuedAt:  timestamppb.New(time.Unix(firebaseToken.IssuedAt, 0)),
		Issuer:    firebaseToken.Issuer,
		Subject:   firebaseToken.UID,
		Provider:  authpb.Provider_PROVIDER_GCP,
	}

	return &authpb.ValidateJwtTokenResponse{
		IsValid:  true,
		Token:    jwtToken,
		Identity: identity,
	}, nil
}

// GetProviderName implements the AuthService interface
func (p *FirebaseAuthAdapter) GetProviderName() string {
	return "firebase"
}

// VerifyIDTokenWithProvider verifies a Firebase ID token and returns the
// signed-in user's email plus the `firebase.sign_in_provider` claim (e.g.
// "microsoft.com", "google.com", "password"). This is the Layer-5 enforcement
// support path: the web login endpoint checks the returned method against the
// configured allow-list. Email is sourced from the standard `email` claim;
// the provider is nested under the `firebase` claim object.
func (p *FirebaseAuthAdapter) VerifyIDTokenWithProvider(ctx context.Context, idToken string) (email, signInProvider string, err error) {
	if !p.enabled || p.clientManager == nil {
		return "", "", fmt.Errorf("firebase auth provider is not enabled")
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return "", "", fmt.Errorf("firebase auth client not available: %w", err)
	}
	if authClient == nil {
		return "", "", fmt.Errorf("firebase auth client not available")
	}
	// CheckRevoked (not plain VerifyIDToken): rejects tokens whose user has been
	// disabled or whose sessions were revoked, within the token's 1h validity —
	// a Defensibility hardening over signature+expiry alone. One extra call, at
	// LOGIN only (never per-request), so the cost is negligible.
	token, err := authClient.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid, expired, or revoked token: %w", err)
	}
	if fb, ok := token.Claims["firebase"].(map[string]interface{}); ok {
		signInProvider = getStringClaim(fb, "sign_in_provider")
	}
	email = getStringClaim(token.Claims, "email")
	// DEFENSIBILITY (account-takeover guard): the email is the join key to the DB
	// user, so an UNVERIFIED email must NEVER match. Federated providers
	// (microsoft.com / google.com) set email_verified=true by construction;
	// a self-registered Firebase email/password account claiming someone else's
	// address has email_verified=false and is rejected here. Return the provider
	// for the caller's log, but no email — so the login fails closed as "invalid".
	verified, _ := token.Claims["email_verified"].(bool)
	if !verified {
		if requireVerifiedEmail() {
			return "", signInProvider, fmt.Errorf("email not verified (provider=%s, email=%s)", signInProvider, email)
		}
		log.Printf("[AUTH] WARNING: accepting UNVERIFIED email %s (provider=%s) — AUTH_FIREBASE_VERIFIED_EMAILS_ONLY=false (INSECURE escape hatch; re-opens the email-match account-takeover surface — use only with a trusted/closed user set, e.g. bootstrap)", email, signInProvider)
	}
	return email, signInProvider, nil
}

// requireVerifiedEmail reports whether an unverified token email must be
// REJECTED. Defaults to TRUE (secure): the email is the DB join key, so an
// unverified address is an account-takeover surface. Set
// AUTH_FIREBASE_VERIFIED_EMAILS_ONLY=false ONLY as a temporary escape hatch —
// e.g. an Azure/Microsoft tenant whose tokens carry email_verified=false — and
// pair it with a trusted, closed user set. Unset/any-other value ⇒ secure.
func requireVerifiedEmail() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_FIREBASE_VERIFIED_EMAILS_ONLY"))) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

// ChangePassword is not supported by the Firebase adapter; password updates flow through Firebase Auth SDK directly.
func (p *FirebaseAuthAdapter) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	return fmt.Errorf("change password not supported by firebase auth provider; use Firebase Admin SDK")
}

// =============================================================================
// Admin user-lifecycle effects at the IdP (§4 adapter matrix — firebase column)
// =============================================================================
//
// The DB user row carries NO firebase UID — the DB↔firebase join key is the
// email address (the login flow in VerifyIDTokenWithProvider keys on `email`).
// The firebase adapter therefore resolves the firebase UserRecord by email.
// The use case (the DB-owning caller) passes the user's email_address as the
// `userID` argument for firebase-backed deployments; resolveFirebaseUID also
// tolerates a raw firebase UID (it tries GetUser first, then GetUserByEmail),
// so the adapter is correct whichever identifier the caller supplies.

// resolveFirebaseUID maps the caller-supplied identifier to a firebase UID.
// If the identifier looks like an email (contains '@') it resolves via
// GetUserByEmail; otherwise it tries GetUser(identifier) (already a firebase
// UID) and falls back to GetUserByEmail. Returns the canonical firebase UID
// and the record's email (used by PasswordResetLink).
func (p *FirebaseAuthAdapter) resolveFirebaseUID(ctx context.Context, identifier string) (uid, email string, err error) {
	if !p.enabled || p.clientManager == nil {
		return "", "", fmt.Errorf("firebase auth provider is not enabled")
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return "", "", fmt.Errorf("firebase auth client not available: %w", err)
	}
	if authClient == nil {
		return "", "", fmt.Errorf("firebase auth client not available")
	}

	if strings.Contains(identifier, "@") {
		rec, gerr := authClient.GetUserByEmail(ctx, identifier)
		if gerr != nil {
			return "", "", fmt.Errorf("firebase: resolve user by email %q: %w", identifier, gerr)
		}
		return rec.UID, rec.Email, nil
	}

	// Not an email — try as a firebase UID first, then fall back to email lookup.
	if rec, gerr := authClient.GetUser(ctx, identifier); gerr == nil {
		return rec.UID, rec.Email, nil
	}
	rec, gerr := authClient.GetUserByEmail(ctx, identifier)
	if gerr != nil {
		return "", "", fmt.Errorf("firebase: resolve user %q (not a known UID or email): %w", identifier, gerr)
	}
	return rec.UID, rec.Email, nil
}

// DisableUserAtProvider disables the firebase account (UpdateUser{Disabled:true}).
func (p *FirebaseAuthAdapter) DisableUserAtProvider(ctx context.Context, userID string) error {
	uid, _, err := p.resolveFirebaseUID(ctx, userID)
	if err != nil {
		return err
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return fmt.Errorf("firebase auth client not available: %w", err)
	}
	if _, err := authClient.UpdateUser(ctx, uid, (&auth.UserToUpdate{}).Disabled(true)); err != nil {
		return fmt.Errorf("firebase: disable user %s: %w", uid, err)
	}
	return nil
}

// EnableUserAtProvider re-enables the firebase account (UpdateUser{Disabled:false}).
func (p *FirebaseAuthAdapter) EnableUserAtProvider(ctx context.Context, userID string) error {
	uid, _, err := p.resolveFirebaseUID(ctx, userID)
	if err != nil {
		return err
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return fmt.Errorf("firebase auth client not available: %w", err)
	}
	if _, err := authClient.UpdateUser(ctx, uid, (&auth.UserToUpdate{}).Disabled(false)); err != nil {
		return fmt.Errorf("firebase: enable user %s: %w", uid, err)
	}
	return nil
}

// UpdateEmailAtProvider syncs the firebase account email (UpdateUser{Email}).
// Resolving by the OLD email is correct: the use case updates the DB then calls
// this with the prior identifier, and the firebase record still carries the old
// address until this write lands.
func (p *FirebaseAuthAdapter) UpdateEmailAtProvider(ctx context.Context, userID, newEmail string) error {
	uid, _, err := p.resolveFirebaseUID(ctx, userID)
	if err != nil {
		return err
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return fmt.Errorf("firebase auth client not available: %w", err)
	}
	if _, err := authClient.UpdateUser(ctx, uid, (&auth.UserToUpdate{}).Email(newEmail)); err != nil {
		return fmt.Errorf("firebase: update email for user %s: %w", uid, err)
	}
	return nil
}

// AdminSetPassword sets a new password at firebase (UpdateUser{Password}).
func (p *FirebaseAuthAdapter) AdminSetPassword(ctx context.Context, userID, newPassword string) error {
	uid, _, err := p.resolveFirebaseUID(ctx, userID)
	if err != nil {
		return err
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return fmt.Errorf("firebase auth client not available: %w", err)
	}
	if _, err := authClient.UpdateUser(ctx, uid, (&auth.UserToUpdate{}).Password(newPassword)); err != nil {
		return fmt.Errorf("firebase: set password for user %s: %w", uid, err)
	}
	return nil
}

// GeneratePasswordResetLink returns a firebase-issued reset link (PasswordResetLink(email)).
func (p *FirebaseAuthAdapter) GeneratePasswordResetLink(ctx context.Context, userID string) (string, error) {
	_, email, err := p.resolveFirebaseUID(ctx, userID)
	if err != nil {
		return "", err
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return "", fmt.Errorf("firebase auth client not available: %w", err)
	}
	link, err := authClient.PasswordResetLink(ctx, email)
	if err != nil {
		return "", fmt.Errorf("firebase: generate password reset link for %s: %w", email, err)
	}
	return link, nil
}

// RevokeUserTokens revokes the user's outstanding refresh tokens (RevokeRefreshTokens(uid)).
func (p *FirebaseAuthAdapter) RevokeUserTokens(ctx context.Context, userID string) error {
	uid, _, err := p.resolveFirebaseUID(ctx, userID)
	if err != nil {
		return err
	}
	authClient, err := p.clientManager.GetAuthClient(ctx)
	if err != nil {
		return fmt.Errorf("firebase auth client not available: %w", err)
	}
	if err := authClient.RevokeRefreshTokens(ctx, uid); err != nil {
		return fmt.Errorf("firebase: revoke tokens for user %s: %w", uid, err)
	}
	return nil
}

// Helper function to safely extract string claims
func getStringClaim(claims map[string]interface{}, key string) string {
	if val, ok := claims[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Compile-time checks that FirebaseAuthAdapter implements both interfaces
var _ ports.AuthProvider = (*FirebaseAuthAdapter)(nil)
var _ ports.AuthService = (*FirebaseAuthAdapter)(nil)
