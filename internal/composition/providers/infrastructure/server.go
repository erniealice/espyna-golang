package infrastructure

import (
	"fmt"
	"os"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// CreateServerProvider creates a server provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_SERVER_PROVIDER environment variable to select which provider to use:
//   - "gin" → Gin HTTP framework
//   - "fiber" → Fiber v2 HTTP framework
//   - "fiber_v3" → Fiber v3 HTTP framework
//   - "vanilla" or "" → Vanilla net/http (default)
//   - "grpc_vanilla" or "grpc" → gRPC server
//
// Note: The actual availability of providers depends on build tags.
// If a provider is requested but not compiled in, an error is returned.
func CreateServerProvider() (ports.ServerProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_SERVER_PROVIDER"))

	// Normalize provider names and set default
	switch providerName {
	case "gin":
		providerName = "gin"
	case "fiber":
		providerName = "fiber"
	case "fiber_v3", "fiberv3":
		providerName = "fiber_v3"
	case "vanilla", "http", "net/http", "":
		providerName = "vanilla"
	case "grpc_vanilla", "grpc":
		providerName = "grpc_vanilla"
	default:
		// Check if it's a valid registered provider
		available := registry.ListAvailableServerBuildFromEnv()
		found := false
		for _, name := range available {
			if name == providerName {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown server provider '%s' (available: %v)", providerName, available)
		}
	}

	// Let the provider build itself from environment
	providerInstance, err := registry.BuildServerProviderFromEnv(providerName)
	if err != nil {
		available := registry.ListAvailableServerBuildFromEnv()
		return nil, fmt.Errorf("failed to create server provider '%s': %w (available providers: %v)", providerName, err, available)
	}

	return providerInstance, nil
}

// GetServerProviderName returns the configured server provider name.
// This is useful for logging and validation before CreateServerProvider is called.
func GetServerProviderName() string {
	providerName := strings.ToLower(os.Getenv("CONFIG_SERVER_PROVIDER"))

	switch providerName {
	case "gin":
		return "gin"
	case "fiber":
		return "fiber"
	case "fiber_v3", "fiberv3":
		return "fiber_v3"
	case "vanilla", "http", "net/http", "":
		return "vanilla"
	case "grpc_vanilla", "grpc":
		return "grpc_vanilla"
	default:
		return providerName
	}
}

// ListAvailableServerProviders returns the names of all compiled-in server providers.
// This helps users understand what providers are available with the current build tags.
func ListAvailableServerProviders() []string {
	return registry.ListAvailableServerBuildFromEnv()
}
