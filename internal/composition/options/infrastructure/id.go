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
	Provider string // "uuidv7" or "noop"
}

// =============================================================================
// ID PROVIDER OPTIONS
// =============================================================================

// WithIDFromEnv dynamically selects ID provider based on CONFIG_ID_PROVIDER.
// Accepts only canonical tokens: "uuidv7" or "noop".
// Retired aliases ("mock", "") fail at startup with a clear message; all other
// names fail as unsupported rather than being translated.
func WithIDFromEnv() ContainerOption {
	return func(c Container) error {
		idProvider := strings.ToLower(GetEnv("CONFIG_ID_PROVIDER", ""))

		switch idProvider {
		case "uuidv7":
			return WithUUIDv7()(c)
		case "noop":
			return WithNoOpID()(c)
		case "mock":
			return fmt.Errorf("CONFIG_ID_PROVIDER=%q is a retired alias — use \"noop\" instead", idProvider)
		case "":
			return fmt.Errorf("CONFIG_ID_PROVIDER is empty — set it explicitly to \"uuidv7\" or \"noop\"")
		default:
			return fmt.Errorf("unsupported ID provider: %s (valid: uuidv7, noop)", idProvider)
		}
	}
}

// WithUUIDv7 configures the Go standard library UUID v7 ID provider.
func WithUUIDv7() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(IDConfigSetter); ok {
			setter.SetIDConfig(IDConfig{Provider: "uuidv7"})
		}

		fmt.Printf("🆔 Configured standard UUID v7 ID provider\n")
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
