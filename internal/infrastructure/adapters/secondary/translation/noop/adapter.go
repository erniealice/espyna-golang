package noop

import (
	"context"
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterTranslationProvider(
		"noop",
		func() ports.TranslationService {
			return NewNoOpTranslationAdapter()
		},
		nil, // No config transformer needed
	)
	registry.RegisterTranslationBuildFromEnv("noop", buildFromEnv)
}

// buildFromEnv creates a noop translation service.
func buildFromEnv() (ports.TranslationService, error) {
	adapter := NewNoOpTranslationAdapter()
	log.Printf("✅ NoOp translation provider initialized (returns keys as-is)")
	return adapter, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// NoOpTranslationAdapter provides a non-operational fallback.
// It simply returns keys or default messages without translation.
type NoOpTranslationAdapter struct{}

// NewNoOpTranslationAdapter creates a new noop translation adapter.
func NewNoOpTranslationAdapter() *NoOpTranslationAdapter {
	return &NoOpTranslationAdapter{}
}

// Get returns the key for debugging (no translation).
func (a *NoOpTranslationAdapter) Get(ctx context.Context, businessType, key string, params ...any) string {
	return key
}

// GetWithDefault returns the default message (no translation).
func (a *NoOpTranslationAdapter) GetWithDefault(ctx context.Context, businessType, key, defaultMessage string, params ...any) string {
	return defaultMessage
}

// Name returns the adapter name.
func (a *NoOpTranslationAdapter) Name() string {
	return "noop"
}

// IsEnabled returns whether the adapter is enabled.
func (a *NoOpTranslationAdapter) IsEnabled() bool {
	return true // NoOp is always "enabled" - it just doesn't do anything
}

var _ ports.TranslationService = (*NoOpTranslationAdapter)(nil)
