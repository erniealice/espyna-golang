package password

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	dbinterfaces "github.com/erniealice/espyna-golang/database/interfaces"
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
	ops    dbinterfaces.DatabaseOperation
	expiry time.Duration
}

// NewSessionService creates a new SessionService.
// Reads PASSWORD_AUTH_SESSION_EXPIRY from environment (default: 168h).
func NewSessionService(ops dbinterfaces.DatabaseOperation) *SessionService {
	expiry := defaultSessionExpiry
	if envExpiry := os.Getenv("PASSWORD_AUTH_SESSION_EXPIRY"); envExpiry != "" {
		if parsed, err := time.ParseDuration(envExpiry); err == nil && parsed > 0 {
			expiry = parsed
		}
	}
	return &SessionService{ops: ops, expiry: expiry}
}

// generateToken creates a cryptographically secure random token.
func generateToken() (string, error) {
	bytes := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// resolveWorkspaceUser looks up the default workspace_user for a user.
func (s *SessionService) resolveWorkspaceUser(ctx context.Context, userID string) (wsUserID, wsID string) {
	if s.ops == nil {
		return "", ""
	}
	q := dbinterfaces.NewQueryBuilder().
		WhereEqualTo("user_id", userID).
		WhereEqualTo("active", true).
		OrderBy("date_created", true).
		Limit(1)
	results, err := s.ops.Query(ctx, "workspace_user", q)
	if err != nil || len(results) == 0 {
		return "", ""
	}
	row := results[0]
	wsUserID, _ = row["id"].(string)
	wsID, _ = row["workspace_id"].(string)
	return
}

// CreateSession creates a new session for the given user and returns the token.
func (s *SessionService) CreateSession(ctx context.Context, userID string) (string, error) {
	if s.ops == nil {
		return "", fmt.Errorf("session service: database operations not initialised")
	}

	id := uuid.New().String()
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	// Resolve workspace_user for default workspace
	wsUserID, wsID := s.resolveWorkspaceUser(ctx, userID)

	// expires_at is BIGINT (unix milliseconds) per the session migration.
	expiresAtMs := time.Now().Add(s.expiry).UnixMilli()

	data := map[string]any{
		"id":         id,
		"user_id":    userID,
		"token":      token,
		"expires_at": expiresAtMs,
		"active":     true,
	}
	// workspace_user_id and workspace_id are nullable — only include when non-empty.
	if wsUserID != "" {
		data["workspace_user_id"] = wsUserID
	}
	if wsID != "" {
		data["workspace_id"] = wsID
	}

	_, err = s.ops.Create(ctx, "session", data)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// GetSessionWorkspaceContext returns the workspace_user_id and workspace_id for an active session.
func (s *SessionService) GetSessionWorkspaceContext(ctx context.Context, token string) (wsUserID, wsID string) {
	if s.ops == nil {
		return "", ""
	}
	q := dbinterfaces.NewQueryBuilder().
		WhereEqualTo("token", token).
		WhereEqualTo("active", true)
	row, err := s.ops.QueryOne(ctx, "session", q)
	if err != nil || row == nil {
		return "", ""
	}
	// expires_at check in Go — QueryBuilder has no WhereGreaterThan.
	// If the session is expired, return empty context.
	if isSessionExpired(row["expires_at"]) {
		return "", ""
	}
	wsUserID, _ = row["workspace_user_id"].(string)
	wsID, _ = row["workspace_id"].(string)
	return
}

// ValidateSession checks if a session token is valid and returns the user ID.
// A session is valid if it exists, is active, and has not expired.
func (s *SessionService) ValidateSession(ctx context.Context, token string) (string, error) {
	if s.ops == nil {
		return "", fmt.Errorf("session service: database operations not initialised")
	}
	if token == "" {
		return "", fmt.Errorf("empty session token")
	}

	q := dbinterfaces.NewQueryBuilder().
		WhereEqualTo("token", token).
		WhereEqualTo("active", true)
	row, err := s.ops.QueryOne(ctx, "session", q)
	if err != nil {
		return "", fmt.Errorf("failed to validate session: %w", err)
	}
	if row == nil {
		return "", fmt.Errorf("invalid or expired session")
	}

	// expires_at check in Go — QueryBuilder has no WhereGreaterThan.
	if isSessionExpired(row["expires_at"]) {
		return "", fmt.Errorf("invalid or expired session")
	}

	userID, _ := row["user_id"].(string)
	if userID == "" {
		return "", fmt.Errorf("session has no associated user")
	}
	return userID, nil
}

// InvalidateSession marks a single session as inactive.
func (s *SessionService) InvalidateSession(ctx context.Context, token string) error {
	if s.ops == nil {
		return fmt.Errorf("session service: database operations not initialised")
	}
	if token == "" {
		return nil
	}

	// Look up session id by token so we can call Update(id).
	q := dbinterfaces.NewQueryBuilder().WhereEqualTo("token", token)
	row, err := s.ops.QueryOne(ctx, "session", q)
	if err != nil {
		return fmt.Errorf("failed to find session for invalidation: %w", err)
	}
	if row == nil {
		return nil // nothing to invalidate
	}
	id, _ := row["id"].(string)
	if id == "" {
		return nil
	}

	_, err = s.ops.Update(ctx, "session", id, map[string]any{"active": false})
	if err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}
	return nil
}

// InvalidateAllUserSessions marks all active sessions for a user as inactive.
// Used when password is changed or account is compromised.
// Note: iterates individually — acceptable for MVP; future optimisation can add a bulk-update path.
func (s *SessionService) InvalidateAllUserSessions(ctx context.Context, userID string) error {
	if s.ops == nil {
		return fmt.Errorf("session service: database operations not initialised")
	}

	q := dbinterfaces.NewQueryBuilder().
		WhereEqualTo("user_id", userID).
		WhereEqualTo("active", true)
	rows, err := s.ops.Query(ctx, "session", q)
	if err != nil {
		return fmt.Errorf("failed to list user sessions: %w", err)
	}

	for _, row := range rows {
		id, _ := row["id"].(string)
		if id == "" {
			continue
		}
		if _, err := s.ops.Update(ctx, "session", id, map[string]any{"active": false}); err != nil {
			return fmt.Errorf("failed to invalidate session %s: %w", id, err)
		}
	}
	return nil
}

// CleanupExpiredSessions hard-deletes expired or inactive sessions.
// Returns the count of sessions removed.
// Note: iterates individually — acceptable for MVP; future optimisation can add a bulk-delete path.
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	if s.ops == nil {
		return 0, fmt.Errorf("session service: database operations not initialised")
	}

	// Fetch inactive sessions first.
	qInactive := dbinterfaces.NewQueryBuilder().WhereEqualTo("active", false)
	inactiveRows, err := s.ops.Query(ctx, "session", qInactive)
	if err != nil {
		return 0, fmt.Errorf("failed to list inactive sessions: %w", err)
	}

	// Fetch all active sessions and filter expired ones in Go.
	// QueryBuilder has no WhereGreaterThan/WhereLessThan, so expiry is checked in Go.
	qActive := dbinterfaces.NewQueryBuilder().WhereEqualTo("active", true)
	activeRows, err := s.ops.Query(ctx, "session", qActive)
	if err != nil {
		return 0, fmt.Errorf("failed to list active sessions for expiry check: %w", err)
	}

	var toDelete []string
	for _, row := range inactiveRows {
		if id, _ := row["id"].(string); id != "" {
			toDelete = append(toDelete, id)
		}
	}
	for _, row := range activeRows {
		if isSessionExpired(row["expires_at"]) {
			if id, _ := row["id"].(string); id != "" {
				toDelete = append(toDelete, id)
			}
		}
	}

	var count int64
	for _, id := range toDelete {
		if err := s.ops.HardDelete(ctx, "session", id); err != nil {
			return count, fmt.Errorf("failed to hard-delete session %s: %w", id, err)
		}
		count++
	}
	return count, nil
}

// GetSessionExpiry returns the configured session duration.
func (s *SessionService) GetSessionExpiry() time.Duration {
	return s.expiry
}

// isSessionExpired checks whether a value read from the session.expires_at
// column represents a moment already in the past. The column is BIGINT (unix
// milliseconds), but different drivers can hand the value back as int64,
// float64, or time.Time depending on column-coercion paths — this helper
// normalises the comparison.
func isSessionExpired(raw any) bool {
	now := time.Now()
	switch v := raw.(type) {
	case int64:
		return now.After(time.UnixMilli(v))
	case int:
		return now.After(time.UnixMilli(int64(v)))
	case float64:
		return now.After(time.UnixMilli(int64(v)))
	case time.Time:
		return now.After(v)
	default:
		return false
	}
}
