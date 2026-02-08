//go:build firebase

package firebase

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	firebaseCommon "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/common/firebase"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
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
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
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
	clientManager *firebaseCommon.FirebaseClientManager
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
	manager, err := firebaseCommon.NewFirebaseClientManager(ctx, "")
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
