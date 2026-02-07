//go:build lyngua

package lyngua

import (
	"context"
	"fmt"
	"log"
	"os"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	lynguaV1 "leapfor.xyz/lyngua/golang/v1"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterTranslationProvider(
		"lyngua",
		func() ports.TranslationService {
			return NewLynguaTranslationAdapter()
		},
		transformConfig,
	)
	registry.RegisterTranslationBuildFromEnv("lyngua", buildFromEnv)
}

// buildFromEnv creates and initializes a lyngua translation service from environment variables.
func buildFromEnv() (ports.TranslationService, error) {
	// Lyngua uses workspace-based path resolution by default
	translationsPath := os.Getenv("LEAPFOR_TRANSLATION_PATH")
	if translationsPath == "" {
		translationsPath = os.Getenv("TRANSLATIONS_PATH")
	}
	// If no path is set, lyngua will use its default workspace resolution

	adapter := NewLynguaTranslationAdapter()
	config := &registry.TranslationProviderConfig{
		Provider:         "lyngua",
		Enabled:          true,
		TranslationsPath: translationsPath,
	}

	if err := adapter.Initialize(config); err != nil {
		return nil, fmt.Errorf("lyngua translation: failed to initialize: %w", err)
	}
	return adapter, nil
}

// transformConfig converts raw config map to TranslationProviderConfig.
func transformConfig(rawConfig map[string]any) (*registry.TranslationProviderConfig, error) {
	config := &registry.TranslationProviderConfig{
		Provider: "lyngua",
		Enabled:  true,
	}

	if path, ok := rawConfig["translations_path"].(string); ok {
		config.TranslationsPath = path
	}

	return config, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// LynguaTranslationAdapter adapts the lyngua TranslationProvider to implement ports.TranslationService.
// This avoids circular dependencies by keeping the adaptation in espyna.
type LynguaTranslationAdapter struct {
	provider *lynguaV1.TranslationProvider
	enabled  bool
}

// NewLynguaTranslationAdapter creates a new lyngua translation adapter.
func NewLynguaTranslationAdapter() *LynguaTranslationAdapter {
	return &LynguaTranslationAdapter{
		enabled: false,
	}
}

// Initialize sets up the lyngua translation adapter with configuration.
func (a *LynguaTranslationAdapter) Initialize(config *registry.TranslationProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	// Use lyngua's factory method to get the provider with proper path resolution
	if config.TranslationsPath != "" {
		a.provider = lynguaV1.NewTranslationProvider(config.TranslationsPath)
	} else {
		// Use lyngua's workspace-aware default
		a.provider = lynguaV1.NewDefaultTranslationProviderWithWorkspace()
	}

	a.enabled = config.Enabled
	log.Printf("✅ Lyngua translation provider initialized")
	return nil
}

// Get implements ports.TranslationService.
func (a *LynguaTranslationAdapter) Get(ctx context.Context, businessType, key string, params ...any) string {
	if !a.enabled || a.provider == nil {
		return key
	}

	// Load messages using lyngua's provider
	messages, err := a.provider.LoadMessages("en", businessType)
	if err != nil {
		return key // Return key if translation loading fails
	}

	translated, ok := messages[key]
	if !ok {
		// If not found in business-specific, try general fallback
		if businessType != "general" {
			generalMessages, err := a.provider.LoadMessages("en", "general")
			if err != nil {
				return key
			}
			translated, ok = generalMessages[key]
		}
		if !ok {
			return key // Return the key itself if no translation is found
		}
	}

	return a.formatMessage(translated, params...)
}

// GetWithDefault implements ports.TranslationService.
func (a *LynguaTranslationAdapter) GetWithDefault(ctx context.Context, businessType, key, defaultMessage string, params ...any) string {
	translated := a.Get(ctx, businessType, key, params...)
	if translated == key {
		return a.formatMessage(defaultMessage, params...)
	}
	return translated
}

// formatMessage performs simple parameter substitution.
func (a *LynguaTranslationAdapter) formatMessage(message string, params ...any) string {
	// For now, simple implementation - can be enhanced later
	// Lyngua may provide its own formatting in the future
	return message
}

// Name returns the adapter name.
func (a *LynguaTranslationAdapter) Name() string {
	return "lyngua"
}

// IsEnabled returns whether the adapter is enabled.
func (a *LynguaTranslationAdapter) IsEnabled() bool {
	return a.enabled
}

var _ ports.TranslationService = (*LynguaTranslationAdapter)(nil)
