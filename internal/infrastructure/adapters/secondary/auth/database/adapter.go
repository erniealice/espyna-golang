//go:build db_auth

package database

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

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
		"db_auth",
		func() ports.AuthProvider {
			return newDatabaseAuthAdapter()
		},
		transformConfig,
	)
	registry.RegisterAuthBuildFromEnv("db_auth", buildFromEnv)
}

// buildFromEnv creates and initializes a DatabaseAuthAdapter from environment variables.
func buildFromEnv() (ports.AuthProvider, error) {
	secret := os.Getenv("DB_AUTH_RESET_TOKEN_SECRET")
	if secret == "" {
		panic("FATAL: db_auth provider requires DB_AUTH_RESET_TOKEN_SECRET to be set")
	}

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	dbName := os.Getenv("POSTGRES_NAME")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	sslMode := os.Getenv("POSTGRES_SSL_MODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, dbName, user, password, sslMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db_auth: failed to open database connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("db_auth: failed to ping database: %w", err)
	}

	protoConfig := &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "Database Auth",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "db_auth",
			},
		},
	}

	adapter := &DatabaseAuthAdapter{
		db:              db,
		passwordService: NewPasswordService(),
		sessionService:  NewSessionService(db),
		resetSecret:     secret,
		enabled:         false,
	}
	if err := adapter.Initialize(protoConfig); err != nil {
		db.Close()
		return nil, fmt.Errorf("db_auth: failed to initialize: %w", err)
	}
	return adapter, nil
}

// transformConfig converts a raw config map to the db_auth proto config.
func transformConfig(rawConfig map[string]any) (*authpb.ProviderConfig, error) {
	return &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "Database Auth",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "db_auth",
			},
		},
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// DatabaseAuthAdapter implements ports.AuthProvider and ports.AuthService
// using a PostgreSQL database for credential storage and session management.
type DatabaseAuthAdapter struct {
	db              *sql.DB
	passwordService *PasswordService
	sessionService  *SessionService
	resetSecret     string
	enabled         bool
}

// newDatabaseAuthAdapter creates an uninitialised DatabaseAuthAdapter.
// The caller must invoke Initialize before use.
func newDatabaseAuthAdapter() *DatabaseAuthAdapter {
	return &DatabaseAuthAdapter{enabled: false}
}

// Name returns the provider name.
func (a *DatabaseAuthAdapter) Name() string {
	return "db_auth"
}

// Initialize sets up the adapter with proto-based configuration.
// When used via buildFromEnv the DB is already attached; when called via the
// factory pattern the caller must supply a DB through SetDB before any query.
func (a *DatabaseAuthAdapter) Initialize(config *authpb.ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	a.enabled = config.Enabled

	if a.resetSecret == "" {
		secret := os.Getenv("DB_AUTH_RESET_TOKEN_SECRET")
		if secret == "" {
			panic("FATAL: db_auth provider requires DB_AUTH_RESET_TOKEN_SECRET to be set")
		}
		a.resetSecret = secret
	}

	if config.Enabled {
		log.Println("[OK] Database Auth provider initialized")
	} else {
		log.Println("[AUTH] Database Auth is disabled")
	}

	return nil
}

// SetDB injects an existing *sql.DB into the adapter and initialises the
// dependent services. This is the preferred entry point when the application
// container already manages the database connection.
func (a *DatabaseAuthAdapter) SetDB(db *sql.DB) {
	a.db = db
	a.passwordService = NewPasswordService()
	a.sessionService = NewSessionService(db)
}

// GetAuthService returns the authentication service (returns itself).
func (a *DatabaseAuthAdapter) GetAuthService() ports.AuthService {
	if !a.enabled {
		return nil
	}
	return a
}

// IsHealthy pings the underlying database.
func (a *DatabaseAuthAdapter) IsHealthy(ctx context.Context) error {
	if !a.enabled {
		return fmt.Errorf("db_auth provider is not enabled")
	}
	if a.db == nil {
		return fmt.Errorf("db_auth: database connection not initialised")
	}
	return a.db.PingContext(ctx)
}

// Close releases the database connection when the adapter owns it.
func (a *DatabaseAuthAdapter) Close() error {
	if a.enabled {
		log.Println("[AUTH] Closing Database Auth provider")
		a.enabled = false
	}
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// IsEnabled returns whether the adapter is enabled.
func (a *DatabaseAuthAdapter) IsEnabled() bool {
	return a.enabled
}

// =============================================================================
// AuthService interface
// =============================================================================

// VerifyToken validates a session token (not a JWT) and returns an Identity.
// db_auth uses opaque session tokens stored in the "session" table rather than
// self-contained JWTs, so the token field of ValidateJwtTokenRequest carries
// the session token.
func (a *DatabaseAuthAdapter) VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
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
func (a *DatabaseAuthAdapter) GetProviderName() string {
	return "db_auth"
}

// =============================================================================
// Extended methods for consumer adapters
// =============================================================================

// Register creates a new user account and returns the new user ID.
func (a *DatabaseAuthAdapter) Register(ctx context.Context, email, password, firstName, lastName, mobileNumber string) (string, error) {
	// Check for duplicate email
	var exists bool
	err := a.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM "user" WHERE email_address = $1)`,
		email,
	).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if exists {
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
	_, err = a.db.ExecContext(ctx,
		`INSERT INTO "user" ("id", "email_address", "password_hash", "first_name", "last_name", "mobile_number", "active", "created_at", "updated_at")
		 VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())`,
		userID, email, hash, firstName, lastName, mobileNumber,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// Login authenticates a user and returns a session token and identity.
func (a *DatabaseAuthAdapter) Login(ctx context.Context, email, password string) (string, *authpb.Identity, error) {
	var (
		userID       string
		passwordHash string
		firstName    string
		lastName     string
		emailAddress string
	)

	err := a.db.QueryRowContext(ctx,
		`SELECT "id", "password_hash", "first_name", "last_name", "email_address"
		 FROM "user"
		 WHERE "email_address" = $1 AND "active" = true`,
		email,
	).Scan(&userID, &passwordHash, &firstName, &lastName, &emailAddress)
	if err == sql.ErrNoRows {
		return "", nil, fmt.Errorf("invalid email or password")
	}
	if err != nil {
		return "", nil, fmt.Errorf("failed to look up user: %w", err)
	}

	if err := a.passwordService.VerifyPassword(passwordHash, password); err != nil {
		return "", nil, fmt.Errorf("invalid email or password")
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
func (a *DatabaseAuthAdapter) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	var userID string
	err := a.db.QueryRowContext(ctx,
		`SELECT "id" FROM "user" WHERE "email_address" = $1 AND "active" = true`,
		email,
	).Scan(&userID)
	if err == sql.ErrNoRows {
		// Return no error to avoid leaking whether the email exists
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to look up user: %w", err)
	}

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

	_, err = a.db.ExecContext(ctx,
		`UPDATE "user"
		 SET "password_reset_token" = $1,
		     "password_reset_expires" = $2
		 WHERE "id" = $3`,
		tokenHash, expiresAtTime, userID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	return rawToken, nil
}

// ExecutePasswordReset verifies the signed reset token, updates the password,
// and invalidates all existing sessions.
func (a *DatabaseAuthAdapter) ExecutePasswordReset(ctx context.Context, token, newPassword string) error {
	// Split token into payload and signature
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

	// Compare stored hash
	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(tokenHashBytes[:])

	var storedHash string
	var storedExpiry time.Time
	err = a.db.QueryRowContext(ctx,
		`SELECT COALESCE("password_reset_token", ''), COALESCE("password_reset_expires", 'epoch'::timestamptz)
		 FROM "user"
		 WHERE "id" = $1 AND "active" = true`,
		payload.UserID,
	).Scan(&storedHash, &storedExpiry)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user not found")
	}
	if err != nil {
		return fmt.Errorf("failed to retrieve user for password reset: %w", err)
	}

	if storedHash == "" || !hmac.Equal([]byte(storedHash), []byte(tokenHash)) {
		return fmt.Errorf("invalid or already-used reset token")
	}
	if time.Now().After(storedExpiry) {
		return fmt.Errorf("reset token has expired")
	}

	newHash, err := a.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = a.db.ExecContext(ctx,
		`UPDATE "user"
		 SET "password_hash" = $1,
		     "password_reset_token" = NULL,
		     "password_reset_expires" = NULL,
		     "updated_at" = NOW()
		 WHERE "id" = $2`,
		newHash, payload.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return a.sessionService.InvalidateAllUserSessions(ctx, payload.UserID)
}

// CreateSession creates a new session for the given user ID.
func (a *DatabaseAuthAdapter) CreateSession(ctx context.Context, userID string) (string, error) {
	return a.sessionService.CreateSession(ctx, userID)
}

// ValidateSession validates a session token and returns the associated user ID.
func (a *DatabaseAuthAdapter) ValidateSession(ctx context.Context, token string) (string, error) {
	return a.sessionService.ValidateSession(ctx, token)
}

// InvalidateSession marks a single session as inactive.
func (a *DatabaseAuthAdapter) InvalidateSession(ctx context.Context, token string) error {
	return a.sessionService.InvalidateSession(ctx, token)
}

// =============================================================================
// Internal helpers
// =============================================================================

// fetchIdentity queries the user table and builds an Identity protobuf.
func (a *DatabaseAuthAdapter) fetchIdentity(ctx context.Context, userID string) (*authpb.Identity, error) {
	var (
		firstName    string
		lastName     string
		emailAddress string
		active       bool
		createdAt    time.Time
	)

	err := a.db.QueryRowContext(ctx,
		`SELECT "first_name", "last_name", "email_address", "active", "created_at"
		 FROM "user"
		 WHERE "id" = $1`,
		userID,
	).Scan(&firstName, &lastName, &emailAddress, &active, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
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

// Compile-time checks that DatabaseAuthAdapter satisfies both interfaces.
var _ ports.AuthProvider = (*DatabaseAuthAdapter)(nil)
var _ ports.AuthService = (*DatabaseAuthAdapter)(nil)
