package infrastructure

import (
	"fmt"
	"strings"
)

// =============================================================================
// ID PROVIDER CONFIGURATION
// =============================================================================

// IDConfig holds ID provider configuration
type IDConfig struct {
	Provider string // "google_uuidv7", "noop", "mock"
}

// =============================================================================
// ID PROVIDER OPTIONS
// =============================================================================

// WithIDFromEnv dynamically selects ID provider based on CONFIG_ID_PROVIDER
func WithIDFromEnv() ContainerOption {
	return func(c Container) error {
		idProvider := strings.ToLower(GetEnv("CONFIG_ID_PROVIDER", "noop"))

		switch idProvider {
		case "google_uuidv7", "uuidv7":
			return WithGoogleUUIDv7()(c)
		case "noop", "mock", "":
			return WithNoOpID()(c)
		default:
			return fmt.Errorf("unsupported ID provider: %s", idProvider)
		}
	}
}

// WithGoogleUUIDv7 configures Google UUID v7 as ID provider
func WithGoogleUUIDv7() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(IDConfigSetter); ok {
			setter.SetIDConfig(IDConfig{Provider: "google_uuidv7"})
		}

		fmt.Printf("🆔 Configured Google UUID v7 ID provider\n")
		return nil
	}
}

// WithNoOpID configures NoOp ID provider (fallback)
func WithNoOpID() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(IDConfigSetter); ok {
			setter.SetIDConfig(IDConfig{Provider: "noop"})
		}

		fmt.Printf("🆔 Configured NoOp ID provider\n")
		return nil
	}
}
