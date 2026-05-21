//go:build !google_uuidv7

package noop

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterIDProvider(
		"noop",
		func() ports.IDGenerator {
			return NewNoOpIDGenerator()
		},
		transformConfig,
	)
	registry.RegisterIDBuildFromEnv("noop", buildFromEnv)
}

// buildFromEnv creates and returns a NoOp ID service.
// No environment variables required - uses timestamp-based IDs.
func buildFromEnv() (ports.IDGenerator, error) {
	return NewNoOpIDGenerator(), nil
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

// NoOpIDGenerator wraps the ports.NoOpIDGenerator for adapter pattern consistency.
// This adapter is used when no specific ID provider is configured or as a fallback.
//
// The NoOp service generates timestamp-based IDs in the format: noop_{unix_nano}
// These IDs are:
//   - Unique within a single process
//   - Chronologically sortable
//   - Suitable for testing and development
//   - NOT globally unique across distributed systems
type NoOpIDGenerator struct {
	delegate ports.IDGenerator
}

// NewNoOpIDGenerator creates a new NoOp ID service
func NewNoOpIDGenerator() ports.IDGenerator {
	return &NoOpIDGenerator{
		delegate: ports.NewNoOpIDGenerator(),
	}
}

// Name returns the name of this ID service
func (s *NoOpIDGenerator) Name() string {
	return "noop"
}

// GenerateID creates a new timestamp-based identifier
func (s *NoOpIDGenerator) GenerateID() string {
	return s.delegate.GenerateID()
}

// GenerateIDWithPrefix creates an ID with specified prefix
func (s *NoOpIDGenerator) GenerateIDWithPrefix(prefix string) string {
	return s.delegate.GenerateIDWithPrefix(prefix)
}

// IsEnabled returns whether the service is enabled (always false for noop)
func (s *NoOpIDGenerator) IsEnabled() bool {
	return s.delegate.IsEnabled()
}

// GetProviderInfo returns provider information
func (s *NoOpIDGenerator) GetProviderInfo() string {
	return s.delegate.GetProviderInfo()
}

// Compile-time interface check
var _ ports.IDGenerator = (*NoOpIDGenerator)(nil)
