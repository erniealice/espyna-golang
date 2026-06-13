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
	Provider string // "google_uuidv7" or "noop"
}

// =============================================================================
// ID PROVIDER OPTIONS
// =============================================================================

// WithIDFromEnv dynamically selects ID provider based on CONFIG_ID_PROVIDER.
// Accepts only canonical tokens: "google_uuidv7" or "noop".
// Retired aliases ("uuidv7", "mock", "") fail at startup with a clear message.
func WithIDFromEnv() ContainerOption {
	return func(c Container) error {
		idProvider := strings.ToLower(GetEnv("CONFIG_ID_PROVIDER", ""))

		switch idProvider {
		case "google_uuidv7":
			return WithGoogleUUIDv7()(c)
		case "noop":
			return WithNoOpID()(c)
		case "uuidv7":
			return fmt.Errorf("CONFIG_ID_PROVIDER=%q is a retired alias — use \"google_uuidv7\" instead", idProvider)
		case "mock":
			return fmt.Errorf("CONFIG_ID_PROVIDER=%q is a retired alias — use \"noop\" instead", idProvider)
		case "":
			return fmt.Errorf("CONFIG_ID_PROVIDER is empty — set it explicitly to \"google_uuidv7\" or \"noop\"")
		default:
			return fmt.Errorf("unsupported ID provider: %s (valid: google_uuidv7, noop)", idProvider)
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
