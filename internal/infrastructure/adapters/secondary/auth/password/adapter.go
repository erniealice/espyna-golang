package password

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	dbinterfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterAuthProvider(
		"password",
		func() ports.AuthProvider {
			return newPasswordAuthAdapter()
		},
		transformConfig,
	)
	registry.RegisterAuthBuildFromEnv("password", buildFromEnv)
}

// buildFromEnv creates a PasswordAuthAdapter from environment variables.
// It reads AUTH_PASSWORD_RESET_TOKEN_SECRET, AUTH_PASSWORD_MAX_ATTEMPTS, and
// AUTH_PASSWORD_LOCKOUT_MINUTES but does NOT open a database connection —
// the connection is injected later via SetOperations (Phase 2 refactor).
func buildFromEnv() (ports.AuthProvider, error) {
	secret := os.Getenv("AUTH_PASSWORD_RESET_TOKEN_SECRET")
	if secret == "" {
		panic("FATAL: password provider requires AUTH_PASSWORD_RESET_TOKEN_SECRET to be set")
	}

	maxAttempts := defaultMaxAttempts
	if v := os.Getenv("AUTH_PASSWORD_MAX_ATTEMPTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxAttempts = parsed
		}
	}

	lockoutMinutes := defaultLockoutMinutes
	if v := os.Getenv("AUTH_PASSWORD_LOCKOUT_MINUTES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			lockoutMinutes = parsed
		}
	}

	protoConfig := &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "Password Auth",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "password",
			},
		},
	}

	adapter := &PasswordAuthAdapter{
		resetSecret:    secret,
		enabled:        false,
		maxAttempts:    maxAttempts,
		lockoutMinutes: lockoutMinutes,
	}
	if err := adapter.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("password: failed to initialize: %w", err)
	}
	return adapter, nil
}

// transformConfig converts a raw config map to the password proto config.
func transformConfig(rawConfig map[string]any) (*authpb.ProviderConfig, error) {
	return &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "Password Auth",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "password",
			},
		},
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

const (
	// defaultMaxAttempts is the default number of failed login attempts before lockout.
	defaultMaxAttempts = 5

	// defaultLockoutMinutes is the default lockout duration in minutes.
	defaultLockoutMinutes = 15
)

// PasswordAuthAdapter implements ports.AuthProvider and ports.AuthService
// using injected DatabaseOperation for credential storage and session management.
type PasswordAuthAdapter struct {
	ops             dbinterfaces.DatabaseOperation
	passwordService *PasswordService
	sessionService  *SessionService
	resetSecret     string
	enabled         bool
	maxAttempts     int
	lockoutMinutes  int
}

// newPasswordAuthAdapter creates an uninitialised PasswordAuthAdapter.
// The caller must invoke SetOperations and Initialize before use.
func newPasswordAuthAdapter() *PasswordAuthAdapter {
	return &PasswordAuthAdapter{
		enabled:        false,
		maxAttempts:    defaultMaxAttempts,
		lockoutMinutes: defaultLockoutMinutes,
	}
}

// Name returns the provider name.
func (a *PasswordAuthAdapter) Name() string {
	return "password"
}

// Initialize sets up the adapter with proto-based configuration.
// SetOperations must be called before any database query.
func (a *PasswordAuthAdapter) Initialize(config *authpb.ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	a.enabled = config.Enabled

	if a.resetSecret == "" {
		secret := os.Getenv("AUTH_PASSWORD_RESET_TOKEN_SECRET")
		if secret == "" {
			panic("FATAL: password provider requires AUTH_PASSWORD_RESET_TOKEN_SECRET to be set")
		}
		a.resetSecret = secret
	}

	if config.Enabled {
		log.Println("[OK] Password Auth provider initialized")
	} else {
		log.Println("[AUTH] Password Auth is disabled")
	}

	return nil
}

// SetOperations injects a DatabaseOperation implementation into the adapter and
// initialises the dependent services. This is the preferred entry point when the
// application container already manages the database connection.
func (a *PasswordAuthAdapter) SetOperations(ops dbinterfaces.DatabaseOperation) {
	a.ops = ops
	a.passwordService = NewPasswordService()
	a.sessionService = NewSessionService(ops)
}

// GetAuthService returns the authentication service (returns itself).
func (a *PasswordAuthAdapter) GetAuthService() ports.AuthService {
	if !a.enabled {
		return nil
	}
	return a
}

// IsHealthy returns nil when the adapter is enabled and has operations injected.
// The adapter no longer owns the database connection, so it does not ping.
func (a *PasswordAuthAdapter) IsHealthy(ctx context.Context) error {
	if !a.enabled {
		return fmt.Errorf("password provider is not enabled")
	}
	if a.ops == nil {
		return fmt.Errorf("password: database operations not initialised")
	}
	return nil
}

// Close logs shutdown. The adapter no longer owns a database connection.
func (a *PasswordAuthAdapter) Close() error {
	if a.enabled {
		log.Println("[AUTH] Closing Password Auth provider")
		a.enabled = false
	}
	return nil
}

// IsEnabled returns whether the adapter is enabled.
func (a *PasswordAuthAdapter) IsEnabled() bool {
	return a.enabled
}

// =============================================================================
// AuthService interface
// =============================================================================

// VerifyToken validates a session token (not a JWT) and returns an Identity.
// password uses opaque session tokens stored in the "session" table rather than
// self-contained JWTs, so the token field of ValidateJwtTokenRequest carries
// the session token.
func (a *PasswordAuthAdapter) VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	if !a.enabled {
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

	if req.Token == "" {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Token is required",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_MALFORMED,
					Message: "Empty token",
				},
			},
		}, nil
	}

	userID, err := a.sessionService.ValidateSession(ctx, req.Token)
	if err != nil {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: err.Error(),
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_EXPIRED,
					Message: err.Error(),
				},
			},
		}, nil
	}

	identity, err := a.fetchIdentity(ctx, userID)
	if err != nil {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("failed to load user identity: %v", err),
		}, nil
	}

	now := time.Now()
	jwtToken := &authpb.JwtToken{
		Token:     req.Token,
		TokenType: "Bearer",
		IssuedAt:  timestamppb.New(now),
		Subject:   userID,
		Provider:  authpb.Provider_PROVIDER_CUSTOM,
	}

	return &authpb.ValidateJwtTokenResponse{
		IsValid:  true,
		Token:    jwtToken,
		Identity: identity,
	}, nil
}

// GetProviderName implements the AuthService interface.
func (a *PasswordAuthAdapter) GetProviderName() string {
	return "password"
}

// =============================================================================
// Extended methods for consumer adapters
// =============================================================================

// Register creates a new user account and returns the new user ID.
func (a *PasswordAuthAdapter) Register(ctx context.Context, email, password, firstName, lastName, mobileNumber string) (string, error) {
	if a.ops == nil {
		return "", fmt.Errorf("password: database operations not initialised")
	}

	// Uniqueness check: if a user with this email exists, reject.
	dup, err := a.ops.QueryOne(ctx, "user", dbinterfaces.NewQueryBuilder().WhereEqualTo("email_address", email))
	if err != nil {
		return "", fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if dup != nil {
		return "", fmt.Errorf("email address is already registered")
	}

	if err := a.passwordService.ValidatePasswordStrength(password); err != nil {
		return "", err
	}

	hash, err := a.passwordService.HashPassword(password)
	if err != nil {
		return "", err
	}

	userID := uuid.New().String()
	_, err = a.ops.Create(ctx, "user", map[string]any{
		"id":            userID,
		"email_address": email,
		"password_hash": hash,
		"first_name":    firstName,
		"last_name":     lastName,
		"mobile_number": mobileNumber,
		"active":        true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// errInvalidCredentials is the single generic error returned for any failed
// login attempt (unknown user, wrong password, or account locked). Keeping it
// as a package-level var ensures byte-identical comparison across all code paths.
var errInvalidCredentials = fmt.Errorf("invalid email or password")

// Login authenticates a user and returns a session token and identity.
// Rate-limiting is applied: failed attempts increment a counter; after
// maxAttempts failures the account is locked for lockoutMinutes minutes.
// A locked account returns the same generic error as a bad-credential
// attempt — callers cannot distinguish lockout from wrong password.
func (a *PasswordAuthAdapter) Login(ctx context.Context, email, password string) (string, *authpb.Identity, error) {
	if a.ops == nil {
		return "", nil, fmt.Errorf("password: database operations not initialised")
	}

	row, err := a.ops.QueryOne(ctx, "user",
		dbinterfaces.NewQueryBuilder().
			WhereEqualTo("email_address", email).
			WhereEqualTo("active", true),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to look up user: %w", err)
	}
	if row == nil {
		return "", nil, errInvalidCredentials
	}

	userID, _ := row["id"].(string)
	passwordHash, _ := row["password_hash"].(string)
	firstName, _ := row["first_name"].(string)
	lastName, _ := row["last_name"].(string)
	emailAddress, _ := row["email_address"].(string)

	// --- Rate-limit precheck ---
	// Extract failed_login_attempts (INT column).
	var failedAttempts int
	switch v := row["failed_login_attempts"].(type) {
	case int64:
		failedAttempts = int(v)
	case int32:
		failedAttempts = int(v)
	case int:
		failedAttempts = v
	case float64:
		failedAttempts = int(v)
	}

	// Extract locked_until (TIMESTAMPTZ column — may be nil/null).
	var lockedUntil time.Time
	switch v := row["locked_until"].(type) {
	case time.Time:
		lockedUntil = v
	case string:
		lockedUntil, _ = time.Parse(time.RFC3339, v)
	}

	// If the account is currently locked, reject without calling bcrypt.
	if !lockedUntil.IsZero() && time.Now().Before(lockedUntil) {
		return "", nil, errInvalidCredentials
	}

	// --- Credential verification ---
	if err := a.passwordService.VerifyPassword(passwordHash, password); err != nil {
		// Increment the failed-attempt counter and lock the account once the
		// configured threshold is crossed. This MUST be atomic at the DB level:
		// the `failedAttempts` value read above is only a precheck snapshot, and
		// under a concurrent bad-password burst a read-modify-write (read N,
		// write N+1 in Go) lets multiple goroutines compute the same N+1, so the
		// stored counter lags the real attempt count and lockout can slip past
		// the threshold. recordFailedAttempt performs a single
		// `UPDATE ... SET failed_login_attempts = failed_login_attempts + 1`,
		// reading the post-increment value back from the DB so the lockout
		// decision is made on the true count even under contention.
		a.recordFailedAttempt(ctx, userID, failedAttempts)
		return "", nil, errInvalidCredentials
	}

	// --- Success path: reset counter ---
	if _, updateErr := a.ops.Update(ctx, "user", userID, map[string]any{
		"failed_login_attempts": 0,
		"locked_until":          nil,
	}); updateErr != nil {
		log.Printf("[AUTH] failed to reset login attempt counter for user %s: %v", userID, updateErr)
	}

	token, err := a.sessionService.CreateSession(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create session: %w", err)
	}

	identity := &authpb.Identity{
		Id:          userID,
		Type:        authpb.IdentityType_IDENTITY_TYPE_USER,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		Email:       emailAddress,
		DisplayName: fmt.Sprintf("%s %s", firstName, lastName),
		IsActive:    true,
	}

	return token, identity, nil
}

// requestPasswordResetPayload is the JSON payload embedded in the HMAC reset token.
type requestPasswordResetPayload struct {
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
	Nonce     string `json:"nonce"`
}

// RequestPasswordReset generates a signed reset token for the given email.
// The returned raw token is intended to be included in a password-reset link
// sent to the user; it is NOT stored in plaintext.
func (a *PasswordAuthAdapter) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	if a.ops == nil {
		return "", fmt.Errorf("password: database operations not initialised")
	}

	row, err := a.ops.QueryOne(ctx, "user",
		dbinterfaces.NewQueryBuilder().
			WhereEqualTo("email_address", email).
			WhereEqualTo("active", true),
	)
	if err != nil {
		return "", fmt.Errorf("failed to look up user: %w", err)
	}
	if row == nil {
		// Return no error to avoid leaking whether the email exists.
		return "", nil
	}

	userID, _ := row["id"].(string)

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	payload := requestPasswordResetPayload{
		UserID:    userID,
		ExpiresAt: expiresAt,
		Nonce:     nonce,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal reset payload: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(a.resetSecret))
	mac.Write(payloadJSON)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	rawToken := base64.RawURLEncoding.EncodeToString(payloadJSON) + "." + sig

	// Store a hash of the token so we can verify it later without keeping the
	// plaintext in the database.
	tokenHashBytes := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(tokenHashBytes[:])
	expiresAtTime := time.Unix(expiresAt, 0)

	_, err = a.ops.Update(ctx, "user", userID, map[string]any{
		"password_reset_token":   tokenHash,
		"password_reset_expires": expiresAtTime,
	})
	if err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	return rawToken, nil
}

// ExecutePasswordReset verifies the signed reset token, updates the password,
// and invalidates all existing sessions.
func (a *PasswordAuthAdapter) ExecutePasswordReset(ctx context.Context, token, newPassword string) error {
	if a.ops == nil {
		return fmt.Errorf("password: database operations not initialised")
	}

	// Split token into payload and signature.
	dotIdx := -1
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx < 0 {
		return fmt.Errorf("malformed reset token")
	}

	payloadB64 := token[:dotIdx]
	sigB64 := token[dotIdx+1:]

	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return fmt.Errorf("malformed reset token: invalid payload encoding")
	}

	expectedSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("malformed reset token: invalid signature encoding")
	}

	mac := hmac.New(sha256.New, []byte(a.resetSecret))
	mac.Write(payloadJSON)
	if !hmac.Equal(mac.Sum(nil), expectedSig) {
		return fmt.Errorf("invalid reset token: signature mismatch")
	}

	var payload requestPasswordResetPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return fmt.Errorf("malformed reset token: cannot decode payload")
	}

	if time.Now().Unix() > payload.ExpiresAt {
		return fmt.Errorf("reset token has expired")
	}

	// Compare stored hash using DatabaseOperation.Read.
	row, err := a.ops.Read(ctx, "user", payload.UserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user for password reset: %w", err)
	}
	if row == nil {
		return fmt.Errorf("user not found")
	}

	// Verify active flag.
	if active, ok := row["active"].(bool); ok && !active {
		return fmt.Errorf("user not found")
	}

	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(tokenHashBytes[:])

	storedHash, _ := row["password_reset_token"].(string)
	if storedHash == "" || !hmac.Equal([]byte(storedHash), []byte(tokenHash)) {
		return fmt.Errorf("invalid or already-used reset token")
	}

	// Check stored expiry (belt-and-suspenders on top of the HMAC payload check).
	var storedExpiry time.Time
	switch v := row["password_reset_expires"].(type) {
	case time.Time:
		storedExpiry = v
	case string:
		storedExpiry, _ = time.Parse(time.RFC3339, v)
	}
	if !storedExpiry.IsZero() && time.Now().After(storedExpiry) {
		return fmt.Errorf("reset token has expired")
	}

	newHash, err := a.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = a.ops.Update(ctx, "user", payload.UserID, map[string]any{
		"password_hash":          newHash,
		"password_reset_token":   nil,
		"password_reset_expires": nil,
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return a.sessionService.InvalidateAllUserSessions(ctx, payload.UserID)
}

// ChangePassword updates the password for an authenticated user.
// Verifies oldPassword against the stored hash, then writes a new bcrypt hash.
// The user's current session is NOT invalidated — only the password_hash is updated.
// Returns a specific error (not the generic login error) when oldPassword is wrong,
// because the caller is already authenticated and there is no enumeration risk.
func (a *PasswordAuthAdapter) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	if a.ops == nil {
		return fmt.Errorf("password: database operations not initialised")
	}

	row, err := a.ops.Read(ctx, "user", userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}
	if row == nil {
		return fmt.Errorf("user not found")
	}

	storedHash, _ := row["password_hash"].(string)
	if err := a.passwordService.VerifyPassword(storedHash, oldPassword); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	newHash, err := a.passwordService.HashPassword(newPassword)
	if err != nil {
		// HashPassword returns "password must be at least N characters" on short input.
		return err
	}

	_, err = a.ops.Update(ctx, "user", userID, map[string]any{
		"password_hash": newHash,
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// =============================================================================
// Admin user-lifecycle effects at the IdP (§4 adapter matrix — password column)
// =============================================================================
//
// For the password provider the DATABASE is the source of truth: account
// status (user.active), credentials (user.password_hash) and session validity
// (the session table + the per-request user.active guard) all live in Postgres
// and are mutated by the domain use cases directly. The IdP-effect methods are
// therefore no-ops here, EXCEPT AdminSetPassword, which is the password
// provider's actual reset mechanism (bcrypt + write user.password_hash).

// DisableUserAtProvider is a no-op for the password provider: the use case sets
// user.active=false in the DB, which the per-request guard enforces. There is no
// external IdP to update.
func (a *PasswordAuthAdapter) DisableUserAtProvider(ctx context.Context, userID string) error {
	return nil
}

// EnableUserAtProvider is a no-op for the password provider: the use case sets
// user.active=true in the DB. No external IdP to update.
func (a *PasswordAuthAdapter) EnableUserAtProvider(ctx context.Context, userID string) error {
	return nil
}

// UpdateEmailAtProvider is a no-op for the password provider: the DB user.email_address
// updated by the use case is authoritative. No external IdP to sync.
func (a *PasswordAuthAdapter) UpdateEmailAtProvider(ctx context.Context, userID, newEmail string) error {
	return nil
}

// AdminSetPassword sets a new password WITHOUT the old one (admin-initiated reset).
// This is the password provider's real reset path: validate strength, bcrypt-hash,
// and write user.password_hash. Existing sessions are NOT invalidated here — the
// use case orchestrates session revocation separately (RevokeUserSessionsUseCase),
// matching ChangePassword's session-preserving contract.
func (a *PasswordAuthAdapter) AdminSetPassword(ctx context.Context, userID, newPassword string) error {
	if a.ops == nil {
		return fmt.Errorf("password: database operations not initialised")
	}

	row, err := a.ops.Read(ctx, "user", userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}
	if row == nil {
		return fmt.Errorf("user not found")
	}

	newHash, err := a.passwordService.HashPassword(newPassword)
	if err != nil {
		// HashPassword enforces minimum length / strength.
		return err
	}

	if _, err := a.ops.Update(ctx, "user", userID, map[string]any{
		"password_hash": newHash,
	}); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// GeneratePasswordResetLink is not supported by the password provider: there is
// no external IdP to issue a link. Admins reset directly via AdminSetPassword,
// and self-service users use RequestPasswordReset (HMAC token). Returns "" + err.
func (a *PasswordAuthAdapter) GeneratePasswordResetLink(ctx context.Context, userID string) (string, error) {
	return "", fmt.Errorf("password provider does not issue reset links; use AdminSetPassword (admin) or RequestPasswordReset (self-service)")
}

// RevokeUserTokens is a no-op for the password provider: there are no IdP refresh
// tokens. Session rows (invalidated by the session use case) and the per-request
// user.active guard are authoritative.
func (a *PasswordAuthAdapter) RevokeUserTokens(ctx context.Context, userID string) error {
	return nil
}

// GetUserAuthCapability reports whether the user has a local password credential.
// HasPassword is (password_hash != ""); the password provider is always the
// single "password" provider.
func (a *PasswordAuthAdapter) GetUserAuthCapability(ctx context.Context, userID string) (ports.AuthCapability, error) {
	if a.ops == nil {
		return ports.AuthCapability{}, fmt.Errorf("password: database operations not initialised")
	}
	row, err := a.ops.Read(ctx, "user", userID)
	if err != nil {
		return ports.AuthCapability{}, fmt.Errorf("failed to retrieve user: %w", err)
	}
	if row == nil {
		return ports.AuthCapability{}, fmt.Errorf("user not found")
	}
	hash, _ := row["password_hash"].(string)
	return ports.AuthCapability{HasPassword: hash != "", Providers: []string{"password"}}, nil
}

// CreateSession creates a new session for the given user ID.
func (a *PasswordAuthAdapter) CreateSession(ctx context.Context, userID string) (string, error) {
	return a.sessionService.CreateSession(ctx, userID)
}

// ValidateSession validates a session token and returns the associated user ID.
func (a *PasswordAuthAdapter) ValidateSession(ctx context.Context, token string) (string, error) {
	return a.sessionService.ValidateSession(ctx, token)
}

// InvalidateSession marks a single session as inactive.
func (a *PasswordAuthAdapter) InvalidateSession(ctx context.Context, token string) error {
	return a.sessionService.InvalidateSession(ctx, token)
}

// GetSessionWorkspaceContext returns the workspace_user_id and workspace_id for an active session.
func (a *PasswordAuthAdapter) GetSessionWorkspaceContext(ctx context.Context, token string) (wsUserID, wsID string) {
	return a.sessionService.GetSessionWorkspaceContext(ctx, token)
}

// HashPassword hashes a plaintext password using bcrypt via the PasswordService.
// Exposed so that the composition root can delegate password hashing to the auth
// adapter layer instead of importing bcrypt directly.
func (a *PasswordAuthAdapter) HashPassword(password string) (string, error) {
	if a.passwordService == nil {
		return "", fmt.Errorf("password: service not initialised (call SetOperations first)")
	}
	return a.passwordService.HashPassword(password)
}

// =============================================================================
// Internal helpers
// =============================================================================

// executorProvider is satisfied by the postgres DatabaseOperation (and its
// workspace-aware wrapper), which expose a transaction-aware raw SQL executor
// via GetExecutor. The interface is declared locally (matching the convention
// in contrib/postgres/.../entity/entity_executor.go) so the adapter can reach a
// raw executor without depending on the concrete adapter type. The mock
// DatabaseOperation used in non-postgres builds and unit tests does NOT
// implement this, so recordFailedAttempt transparently falls back to the
// read-modify-write path for those backends.
type executorProvider interface {
	GetExecutor(ctx context.Context) sqlexec.DBExecutor
}

// recordFailedAttempt increments the failed-login counter for userID and locks
// the account once the configured threshold is reached.
//
// When the underlying DatabaseOperation exposes a raw SQL executor (the
// production postgres path), the increment and lockout decision are performed
// in a SINGLE atomic statement:
//
//	UPDATE "user"
//	   SET failed_login_attempts = failed_login_attempts + 1,
//	       locked_until = CASE
//	           WHEN failed_login_attempts + 1 >= $maxAttempts
//	               THEN $lockUntil
//	           ELSE locked_until
//	       END
//	 WHERE id = $userID
//	RETURNING failed_login_attempts
//
// Because Postgres evaluates the SET expressions against the row under a write
// lock, two concurrent bad-password attempts can never read the same stale
// counter and both write the same N+1 — each increment is serialized, so the
// stored counter always equals the true number of failed attempts and the
// lockout fires at EXACTLY maxAttempts. The CASE keeps the lockout-window write
// in the same statement (atomic with the increment) and never clears an
// existing locked_until below the threshold.
//
// snapshotAttempts is the precheck value read earlier in Login; it is used only
// for the non-atomic fallback (mock / non-executor backends) and for diagnostic
// logging — it is NEVER the basis for the lockout decision on the atomic path.
func (a *PasswordAuthAdapter) recordFailedAttempt(ctx context.Context, userID string, snapshotAttempts int) {
	lockUntil := time.Now().Add(time.Duration(a.lockoutMinutes) * time.Minute)

	if ep, ok := a.ops.(executorProvider); ok {
		const q = `UPDATE "user"
			SET failed_login_attempts = failed_login_attempts + 1,
			    locked_until = CASE
			        WHEN failed_login_attempts + 1 >= $2 THEN $3
			        ELSE locked_until
			    END
			WHERE id = $1
			RETURNING failed_login_attempts`
		var newCount int
		err := ep.GetExecutor(ctx).
			QueryRowContext(ctx, q, userID, a.maxAttempts, lockUntil).
			Scan(&newCount)
		if err != nil {
			log.Printf("[AUTH] failed to atomically update login attempt counter for user %s: %v", userID, err)
		}
		return
	}

	// Fallback: backends without a raw executor (mock DB, non-postgres builds,
	// unit tests) keep the original read-modify-write. Not atomic, but these
	// backends are single-process / test-only and not subject to the concurrent
	// production burst this fix targets.
	newCount := snapshotAttempts + 1
	updateData := map[string]any{
		"failed_login_attempts": newCount,
	}
	if newCount >= a.maxAttempts {
		updateData["locked_until"] = lockUntil
	}
	if _, updateErr := a.ops.Update(ctx, "user", userID, updateData); updateErr != nil {
		log.Printf("[AUTH] failed to update login attempt counter for user %s: %v", userID, updateErr)
	}
}

// fetchIdentity queries the user table and builds an Identity protobuf.
func (a *PasswordAuthAdapter) fetchIdentity(ctx context.Context, userID string) (*authpb.Identity, error) {
	if a.ops == nil {
		return nil, fmt.Errorf("password: database operations not initialised")
	}

	row, err := a.ops.Read(ctx, "user", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if row == nil {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	firstName, _ := row["first_name"].(string)
	lastName, _ := row["last_name"].(string)
	emailAddress, _ := row["email_address"].(string)
	active, _ := row["active"].(bool)

	var createdAt time.Time
	switch v := row["created_at"].(type) {
	case time.Time:
		createdAt = v
	case string:
		createdAt, _ = time.Parse(time.RFC3339, v)
	}

	return &authpb.Identity{
		Id:          userID,
		Type:        authpb.IdentityType_IDENTITY_TYPE_USER,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		Email:       emailAddress,
		DisplayName: fmt.Sprintf("%s %s", firstName, lastName),
		IsActive:    active,
		CreatedAt:   timestamppb.New(createdAt),
	}, nil
}

// Compile-time checks that PasswordAuthAdapter satisfies both interfaces.
var _ ports.AuthProvider = (*PasswordAuthAdapter)(nil)
var _ ports.AuthService = (*PasswordAuthAdapter)(nil)
