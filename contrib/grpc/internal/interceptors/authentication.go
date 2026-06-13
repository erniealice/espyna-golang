//go:build grpc

package interceptors

import (
	"context"
	"os"
	"slices"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/shared/identity"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
)

// AuthenticationInterceptor provides authentication interceptor for gRPC requests
type AuthenticationInterceptor struct {
	authService   ports.AuthService
	publicMethods []string
}

// NewAuthenticationInterceptor creates a new authentication interceptor instance
func NewAuthenticationInterceptor(authService ports.AuthService) *AuthenticationInterceptor {
	return &AuthenticationInterceptor{
		authService: authService,
		publicMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		},
	}
}

// UnaryInterceptor returns a unary server interceptor for authentication
func (i *AuthenticationInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth if disabled or service unavailable
		if i.authService == nil || !i.authService.IsEnabled() {
			return handler(ctx, req)
		}

		// Skip authentication for public methods
		if i.isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Check for API key authentication
		if i.isAuthorizedAPIKey(ctx) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		token, err := i.extractToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "Missing or invalid authorization token")
		}

		// Verify the authentication token using proto types
		authReq := &authpb.ValidateJwtTokenRequest{
			Token:    token,
			Provider: authpb.Provider_PROVIDER_GCP, // Default provider, could be configured
		}

		resp, err := i.authService.VerifyToken(ctx, authReq)
		if err != nil {
			return nil, status.Error(codes.Internal, "Authentication failed")
		}

		if !resp.IsValid {
			return nil, status.Error(codes.Unauthenticated, resp.ErrorMessage)
		}

		// Add user information to context.
		//
		// SECURITY: Do NOT write identity.RequestIdentity here. This JWT-based
		// auth interceptor only knows UserID/Email — it has no workspace context.
		// Writing a RequestIdentity with empty WorkspaceID would cause
		// identity.Must(ctx).WorkspaceID to return "" instead of panicking,
		// which disables tenant filtering on fail-open SQL predicates.
		// The session middleware resolves the full identity and writes
		// RequestIdentity with workspace context populated.
		ctx = context.WithValue(ctx, "identity", resp.Identity)
		if resp.Token != nil && resp.Token.ExpiresAt != nil {
			ctx = context.WithValue(ctx, "expires", resp.Token.ExpiresAt.AsTime().Unix())
		}

		// Continue with authenticated request
		return handler(ctx, req)
	}
}

// extractToken extracts the token from gRPC metadata
func (i *AuthenticationInterceptor) extractToken(ctx context.Context) (string, error) {
	// gRPC metadata is accessed via the metadata.FromIncomingContext function
	// But to avoid import cycles, we'll use a simpler approach here
	// The adapter will extract metadata and pass it via context

	// Try to get authorization from context (set by adapter)
	if authVal := ctx.Value("authorization"); authVal != nil {
		if authStr, ok := authVal.(string); ok {
			if strings.HasPrefix(authStr, "Bearer ") {
				return strings.TrimPrefix(authStr, "Bearer "), nil
			}
			return authStr, nil
		}
	}

	return "", nil
}

// isPublicMethod checks if the method is public (no auth required)
func (i *AuthenticationInterceptor) isPublicMethod(fullMethod string) bool {
	return slices.Contains(i.publicMethods, fullMethod)
}

// isAuthorizedAPIKey checks for valid API keys
func (i *AuthenticationInterceptor) isAuthorizedAPIKey(ctx context.Context) bool {
	// Check X-API-Key header
	apiKey := os.Getenv("X_API_KEY")
	if apiKey != "" {
		if keyVal := ctx.Value("x-api-key"); keyVal != nil {
			if key, ok := keyVal.(string); ok && key == apiKey {
				return true
			}
		}
	}

	// Check X-API-Key-Scheduler header
	schedulerKey := os.Getenv("X_API_KEY_SCHEDULER")
	if schedulerKey != "" {
		if keyVal := ctx.Value("x-api-key-scheduler"); keyVal != nil {
			if key, ok := keyVal.(string); ok && key == schedulerKey {
				return true
			}
		}
	}

	return false
}

// GetUserFromContext extracts user information from context
func GetUserFromContext(ctx context.Context) (uid string, email string, ok bool) {
	id, found := identity.FromContext(ctx)
	if !found || id.UserID == "" {
		return "", "", false
	}
	return id.UserID, id.Email, true
}

// GetIdentityFromContext extracts the full identity from context
func GetIdentityFromContext(ctx context.Context) (*authpb.Identity, bool) {
	identityVal := ctx.Value("identity")
	if identityVal == nil {
		return nil, false
	}

	authID, ok := identityVal.(*authpb.Identity)
	return authID, ok
}
