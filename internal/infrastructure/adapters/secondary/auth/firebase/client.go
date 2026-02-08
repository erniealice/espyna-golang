//go:build firebase

package firebase

import (
	"context"
	"log"
	"time"

	"firebase.google.com/go/v4/auth"
	firebaseCommon "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/common/firebase"
)

// AuthService provides Firebase Authentication operations
type AuthService struct {
	client        *auth.Client
	clientManager *firebaseCommon.FirebaseClientManager
}

// AuthServiceInterface defines the authentication operations
type AuthServiceInterface interface {
	// User management
	GetUser(ctx context.Context, uid string) (*auth.UserRecord, error)
	GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error)
	CreateUser(ctx context.Context, params *auth.UserToCreate) (*auth.UserRecord, error)
	UpdateUser(ctx context.Context, uid string, params *auth.UserToUpdate) (*auth.UserRecord, error)
	DeleteUser(ctx context.Context, uid string) error

	// Token operations
	VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error)
	VerifyIDTokenAndCheckRevoked(ctx context.Context, idToken string) (*auth.Token, error)
	CreateCustomToken(ctx context.Context, uid string) (string, error)
	CreateCustomTokenWithClaims(ctx context.Context, uid string, claims map[string]any) (string, error)

	// Custom claims
	SetCustomUserClaims(ctx context.Context, uid string, claims map[string]any) error

	// Session management
	CreateSessionCookie(ctx context.Context, idToken string, expiresIn int64) (string, error)
	VerifySessionCookie(ctx context.Context, sessionCookie string) (*auth.Token, error)
	VerifySessionCookieAndCheckRevoked(ctx context.Context, sessionCookie string) (*auth.Token, error)

	// User management operations
	ListUsers(ctx context.Context, maxResults int, nextPageToken string) ([]*auth.UserRecord, string, error)
	GetUsers(ctx context.Context, identifiers []auth.UserIdentifier) ([]*auth.UserRecord, error)

	// Revocation
	RevokeRefreshTokens(ctx context.Context, uid string) error
}

// NewAuthService creates a new Firebase Auth service instance
func NewAuthService(ctx context.Context) (AuthServiceInterface, error) {
	// Create Firebase client manager
	manager, err := firebaseCommon.NewFirebaseClientManager(ctx, "")
	if err != nil {
		return nil, err
	}

	// Get auth client
	client, err := manager.GetAuthClient(ctx)
	if err != nil {
		manager.Close()
		return nil, err
	}

	return &AuthService{
		client:        client,
		clientManager: manager,
	}, nil
}

// Close closes the Firebase client manager
func (s *AuthService) Close() error {
	if s.clientManager != nil {
		return s.clientManager.Close()
	}
	return nil
}

// GetUser retrieves a user by UID
func (s *AuthService) GetUser(ctx context.Context, uid string) (*auth.UserRecord, error) {
	return s.client.GetUser(ctx, uid)
}

// GetUserByEmail retrieves a user by email
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	return s.client.GetUserByEmail(ctx, email)
}

// CreateUser creates a new user
func (s *AuthService) CreateUser(ctx context.Context, params *auth.UserToCreate) (*auth.UserRecord, error) {
	return s.client.CreateUser(ctx, params)
}

// UpdateUser updates an existing user
func (s *AuthService) UpdateUser(ctx context.Context, uid string, params *auth.UserToUpdate) (*auth.UserRecord, error) {
	return s.client.UpdateUser(ctx, uid, params)
}

// DeleteUser deletes a user
func (s *AuthService) DeleteUser(ctx context.Context, uid string) error {
	return s.client.DeleteUser(ctx, uid)
}

// VerifyIDToken verifies an ID token
func (s *AuthService) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	return s.client.VerifyIDToken(ctx, idToken)
}

// VerifyIDTokenAndCheckRevoked verifies an ID token and checks if it's revoked
func (s *AuthService) VerifyIDTokenAndCheckRevoked(ctx context.Context, idToken string) (*auth.Token, error) {
	return s.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
}

// CreateCustomToken creates a custom token for a user
func (s *AuthService) CreateCustomToken(ctx context.Context, uid string) (string, error) {
	return s.client.CustomToken(ctx, uid)
}

// CreateCustomTokenWithClaims creates a custom token with additional claims
func (s *AuthService) CreateCustomTokenWithClaims(ctx context.Context, uid string, claims map[string]any) (string, error) {
	return s.client.CustomTokenWithClaims(ctx, uid, claims)
}

// SetCustomUserClaims sets custom claims for a user
func (s *AuthService) SetCustomUserClaims(ctx context.Context, uid string, claims map[string]any) error {
	return s.client.SetCustomUserClaims(ctx, uid, claims)
}

// CreateSessionCookie creates a session cookie
func (s *AuthService) CreateSessionCookie(ctx context.Context, idToken string, expiresIn int64) (string, error) {
	return s.client.SessionCookie(ctx, idToken, time.Duration(expiresIn)*time.Second)
}

// VerifySessionCookie verifies a session cookie
func (s *AuthService) VerifySessionCookie(ctx context.Context, sessionCookie string) (*auth.Token, error) {
	return s.client.VerifySessionCookie(ctx, sessionCookie)
}

// VerifySessionCookieAndCheckRevoked verifies a session cookie and checks if it's revoked
func (s *AuthService) VerifySessionCookieAndCheckRevoked(ctx context.Context, sessionCookie string) (*auth.Token, error) {
	return s.client.VerifySessionCookieAndCheckRevoked(ctx, sessionCookie)
}

// ListUsers lists users with pagination
func (s *AuthService) ListUsers(ctx context.Context, maxResults int, nextPageToken string) ([]*auth.UserRecord, string, error) {
	iter := s.client.Users(ctx, nextPageToken)
	var users []*auth.UserRecord

	for i := 0; i < maxResults; i++ {
		exportedUser, err := iter.Next()
		if err != nil {
			// Check if it's an end-of-iterator error
			if err.Error() == "no more users" {
				break
			}
			return nil, "", err
		}
		// Convert ExportedUserRecord to UserRecord
		users = append(users, exportedUser.UserRecord)
	}

	// Get next page token
	pageToken := iter.PageInfo().Token

	return users, pageToken, nil
}

// GetUsers retrieves multiple users by identifiers
func (s *AuthService) GetUsers(ctx context.Context, identifiers []auth.UserIdentifier) ([]*auth.UserRecord, error) {
	result, err := s.client.GetUsers(ctx, identifiers)
	if err != nil {
		return nil, err
	}

	var users []*auth.UserRecord
	for _, userRecord := range result.Users {
		users = append(users, userRecord)
	}

	// Log any errors for missing users
	for _, notFound := range result.NotFound {
		log.Printf("⚠️ User not found: %v", notFound)
	}

	return users, nil
}

// RevokeRefreshTokens revokes all refresh tokens for a user
func (s *AuthService) RevokeRefreshTokens(ctx context.Context, uid string) error {
	return s.client.RevokeRefreshTokens(ctx, uid)
}
