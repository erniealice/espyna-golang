//go:build !google || !uuidv7

package noop

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterIDProvider(
		"noop",
		func() ports.IDService {
			return NewNoOpIDService()
		},
		transformConfig,
	)
	registry.RegisterIDBuildFromEnv("noop", buildFromEnv)
}

// buildFromEnv creates and returns a NoOp ID service.
// No environment variables required - uses timestamp-based IDs.
func buildFromEnv() (ports.IDService, error) {
	return NewNoOpIDService(), nil
}

// transformConfig converts raw config map to ID provider config.
func transformConfig(rawConfig map[string]any) (*registry.IDProviderConfig, error) {
	return &registry.IDProviderConfig{
		Provider: "noop",
		Enabled:  true,
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// NoOpIDService wraps the ports.NoOpIDService for adapter pattern consistency.
// This adapter is used when no specific ID provider is configured or as a fallback.
//
// The NoOp service generates timestamp-based IDs in the format: noop_{unix_nano}
// These IDs are:
//   - Unique within a single process
//   - Chronologically sortable
//   - Suitable for testing and development
//   - NOT globally unique across distributed systems
type NoOpIDService struct {
	delegate ports.IDService
}

// NewNoOpIDService creates a new NoOp ID service
func NewNoOpIDService() ports.IDService {
	return &NoOpIDService{
		delegate: ports.NewNoOpIDService(),
	}
}

// Name returns the name of this ID service
func (s *NoOpIDService) Name() string {
	return "noop"
}

// GenerateID creates a new timestamp-based identifier
func (s *NoOpIDService) GenerateID() string {
	return s.delegate.GenerateID()
}

// GenerateIDWithPrefix creates an ID with specified prefix
func (s *NoOpIDService) GenerateIDWithPrefix(prefix string) string {
	return s.delegate.GenerateIDWithPrefix(prefix)
}

// IsEnabled returns whether the service is enabled (always false for noop)
func (s *NoOpIDService) IsEnabled() bool {
	return s.delegate.IsEnabled()
}

// GetProviderInfo returns provider information
func (s *NoOpIDService) GetProviderInfo() string {
	return s.delegate.GetProviderInfo()
}

// Compile-time interface check
var _ ports.IDService = (*NoOpIDService)(nil)
