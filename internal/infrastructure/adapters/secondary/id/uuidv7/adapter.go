//go:build google_uuidv7

package uuidv7

import (
	"fmt"

	"github.com/google/uuid"
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterIDProvider(
		"google_uuidv7",
		func() ports.IDService {
			return NewGoogleUUIDv7Service()
		},
		transformConfig,
	)
	registry.RegisterIDBuildFromEnv("google_uuidv7", buildFromEnv)
}

// buildFromEnv creates and returns a Google UUID v7 service.
// No environment variables required - uses google/uuid library for UUID v7 generation.
func buildFromEnv() (ports.IDService, error) {
	return NewGoogleUUIDv7Service(), nil
}

// transformConfig converts raw config map to ID provider config.
func transformConfig(rawConfig map[string]any) (*registry.IDProviderConfig, error) {
	return &registry.IDProviderConfig{
		Provider: "google_uuidv7",
		Enabled:  true,
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// GoogleUUIDv7Service provides UUID v7 generation using google/uuid package.
// UUID v7 is a time-ordered UUID that provides:
//   - Chronological sortability (newer IDs sort after older ones)
//   - Global uniqueness across distributed systems
//   - Better database indexing performance than random UUIDs (UUID v4)
//   - 48-bit Unix timestamp in milliseconds + 74 bits of randomness
type GoogleUUIDv7Service struct {
	enabled bool
}

// NewGoogleUUIDv7Service creates a new Google UUID v7 service
func NewGoogleUUIDv7Service() ports.IDService {
	return &GoogleUUIDv7Service{
		enabled: true,
	}
}

// Name returns the name of this ID service
func (s *GoogleUUIDv7Service) Name() string {
	return "google_uuidv7"
}

// GenerateID creates a new UUID v7 identifier
func (s *GoogleUUIDv7Service) GenerateID() string {
	if !s.enabled {
		fallback := ports.NewNoOpIDService()
		return fallback.GenerateID()
	}

	uuidV7, err := uuid.NewV7()
	if err != nil {
		fallback := ports.NewNoOpIDService()
		return fallback.GenerateID()
	}

	return uuidV7.String()
}

// GenerateIDWithPrefix creates a UUID v7 with specified prefix
func (s *GoogleUUIDv7Service) GenerateIDWithPrefix(prefix string) string {
	baseID := s.GenerateID()
	return fmt.Sprintf("%s_%s", prefix, baseID)
}

// IsEnabled returns whether the service is enabled
func (s *GoogleUUIDv7Service) IsEnabled() bool {
	return s.enabled
}

// GetProviderInfo returns provider information
func (s *GoogleUUIDv7Service) GetProviderInfo() string {
	return "Google UUID v7 Service"
}

// Compile-time interface check
var _ ports.IDService = (*GoogleUUIDv7Service)(nil)
