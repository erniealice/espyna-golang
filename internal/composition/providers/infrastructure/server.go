package infrastructure

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateServerProvider creates a server provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_SERVER_PROVIDER environment variable to select which provider to use:
//   - "http" -> stdlib net/http
//   - "gin" -> Gin HTTP framework
//   - "fiber" -> Fiber v2 HTTP framework
//   - "fiber_v3" -> Fiber v3 HTTP framework
//   - "grpc" -> gRPC server
//
// Retired aliases (vanilla, net/http, fiberv3, grpc_vanilla, empty) are rejected
// with a clear error directing the user to the canonical token.
//
// Note: The actual availability of providers depends on build tags.
// If a provider is requested but not compiled in, an error is returned.
func CreateServerProvider() (ports.ServerProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_SERVER_PROVIDER"))

	// Reject retired aliases with a clear migration message.
	switch providerName {
	case "vanilla":
		return nil, fmt.Errorf("server provider 'vanilla' is retired - use CONFIG_SERVER_PROVIDER=http")
	case "net/http":
		return nil, fmt.Errorf("server provider 'net/http' is retired - use CONFIG_SERVER_PROVIDER=http")
	case "fiberv3":
		return nil, fmt.Errorf("server provider 'fiberv3' is retired - use CONFIG_SERVER_PROVIDER=fiber_v3")
	case "grpc_vanilla":
		return nil, fmt.Errorf("server provider 'grpc_vanilla' is retired - use CONFIG_SERVER_PROVIDER=grpc")
	case "":
		return nil, fmt.Errorf("CONFIG_SERVER_PROVIDER is empty - set it explicitly (http, gin, fiber, fiber_v3, grpc)")
	}

	// Only canonical tokens reach the registry.
	canonical := map[string]bool{
		"http": true, "gin": true, "fiber": true, "fiber_v3": true, "grpc": true,
	}
	if !canonical[providerName] {
		available := registry.ListAvailableServerBuildFromEnv()
		return nil, fmt.Errorf("unknown server provider '%s' (canonical: http, gin, fiber, fiber_v3, grpc; available: %v)", providerName, available)
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

	canonical := map[string]bool{
		"http": true, "gin": true, "fiber": true, "fiber_v3": true, "grpc": true,
	}
	if canonical[providerName] {
		return providerName
	}
	// Return the raw value so the caller can report the error.
	return providerName
}

// ListAvailableServerProviders returns the names of all compiled-in server providers.
// This helps users understand what providers are available with the current build tags.
func ListAvailableServerProviders() []string {
	return registry.ListAvailableServerBuildFromEnv()
}
