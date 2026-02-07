package infrastructure

import (
	"fmt"
	"os"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// CreateTranslationService creates a translation service using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_TRANSLATION_PROVIDER environment variable to select which provider to use:
//   - "lyngua" → Lyngua translation provider (default)
//   - "file" → File-based translation provider
//   - "noop" → NoOp translation provider (returns keys as-is)
func CreateTranslationService() (ports.TranslationService, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_TRANSLATION_PROVIDER"))

	// Default to lyngua if not specified
	if providerName == "" {
		providerName = "lyngua"
	}

	// Normalize provider names
	switch providerName {
	case "none", "disabled":
		providerName = "noop"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildTranslationProviderFromEnv(providerName)
	if err != nil {
		// Fall back to noop if the specified provider fails
		fmt.Printf("⚠️ Translation provider '%s' failed: %v, falling back to noop\n", providerName, err)
		return registry.BuildTranslationProviderFromEnv("noop")
	}

	return providerInstance, nil
}
