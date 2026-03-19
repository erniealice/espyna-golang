//go:build db_auth

package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

const (
	// defaultSessionExpiry is the default session duration (7 days).
	defaultSessionExpiry = 7 * 24 * time.Hour

	// sessionTokenBytes is the number of random bytes for session tokens.
	sessionTokenBytes = 32
)

// SessionService handles session creation, validation, and invalidation.
type SessionService struct {
	db     *sql.DB
	expiry time.Duration
}

// NewSessionService creates a new SessionService.
// Reads DB_AUTH_SESSION_EXPIRY from environment (default: 168h).
func NewSessionService(db *sql.DB) *SessionService {
	expiry := defaultSessionExpiry
	if envExpiry := os.Getenv("DB_AUTH_SESSION_EXPIRY"); envExpiry != "" {
		if parsed, err := time.ParseDuration(envExpiry); err == nil && parsed > 0 {
			expiry = parsed
		}
	}
	return &SessionService{db: db, expiry: expiry}
}

// generateToken creates a cryptographically secure random token.
func generateToken() (string, error) {
	bytes := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session for the given user and returns the token.
func (s *SessionService) CreateSession(ctx context.Context, userID string) (string, error) {
	id := uuid.New().String()
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(s.expiry)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO "session" ("id", "user_id", "token", "created_at", "expires_at", "active")
		 VALUES ($1, $2, $3, NOW(), $4, true)`,
		id, userID, token, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// ValidateSession checks if a session token is valid and returns the user ID.
// A session is valid if it exists, is active, and has not expired.
func (s *SessionService) ValidateSession(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("empty session token")
	}

	var userID string
	err := s.db.QueryRowContext(ctx,
		`SELECT "user_id" FROM "session"
		 WHERE "token" = $1 AND "active" = true AND "expires_at" > NOW()`,
		token,
	).Scan(&userID)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid or expired session")
	}
	if err != nil {
		return "", fmt.Errorf("failed to validate session: %w", err)
	}

	return userID, nil
}

// InvalidateSession marks a single session as inactive.
func (s *SessionService) InvalidateSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE "session" SET "active" = false WHERE "token" = $1`,
		token,
	)
	if err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}
	return nil
}

// InvalidateAllUserSessions marks all sessions for a user as inactive.
// Used when password is changed or account is compromised.
func (s *SessionService) InvalidateAllUserSessions(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE "session" SET "active" = false WHERE "user_id" = $1 AND "active" = true`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to invalidate user sessions: %w", err)
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions from the database.
// This can be called periodically (e.g., via a cron job).
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM "session" WHERE "expires_at" < NOW() OR "active" = false`,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup sessions: %w", err)
	}
	return result.RowsAffected()
}

// GetSessionExpiry returns the configured session duration.
func (s *SessionService) GetSessionExpiry() time.Duration {
	return s.expiry
}
