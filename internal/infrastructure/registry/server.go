package registry

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// =============================================================================
// Server Factory Registry
// =============================================================================
//
// ServerFactory is a function that creates a new ServerProvider instance.
// The provider is uninitialized - call Initialize(container) before Start().
//
// Unlike other providers that use protobuf configs, server providers use
// a simpler pattern with just the Container for initialization.
//
// =============================================================================

// serverRegistry holds registered server provider factories and builders
// Uses 'any' for the config type since server providers use Container (avoiding import cycles)
var serverRegistry = NewFactoryRegistry[ports.ServerProvider, any]("server")

// =============================================================================
// Server Provider Factory Functions
// =============================================================================

// RegisterServerProviderFactory registers a factory function for creating server providers.
// The factory should return an uninitialized provider.
func RegisterServerProviderFactory(name string, factory func() ports.ServerProvider) {
	serverRegistry.RegisterFactory(name, factory)
}

// GetServerProviderFactory retrieves a registered server factory by name.
func GetServerProviderFactory(name string) (func() ports.ServerProvider, bool) {
	return serverRegistry.GetFactory(name)
}

// ListAvailableServerProviderFactories returns all registered server factory names.
func ListAvailableServerProviderFactories() []string {
	return serverRegistry.ListFactories()
}

// =============================================================================
// Server BuildFromEnv Functions
// =============================================================================

// RegisterServerBuildFromEnv registers a self-configuration builder for server providers.
// The builder should create, initialize, and return a ready-to-use provider.
func RegisterServerBuildFromEnv(name string, builder func() (ports.ServerProvider, error)) {
	serverRegistry.RegisterBuildFromEnv(name, builder)
}

// GetServerBuildFromEnv retrieves a registered BuildFromEnv function.
func GetServerBuildFromEnv(name string) (func() (ports.ServerProvider, error), bool) {
	return serverRegistry.GetBuildFromEnv(name)
}

// BuildServerProviderFromEnv creates a server provider using its registered BuildFromEnv function.
func BuildServerProviderFromEnv(name string) (ports.ServerProvider, error) {
	return serverRegistry.BuildFromEnv(name)
}

// ListAvailableServerBuildFromEnv returns all registered BuildFromEnv names.
func ListAvailableServerBuildFromEnv() []string {
	return serverRegistry.ListBuildFromEnv()
}

// =============================================================================
// Convenience Registration Function
// =============================================================================

// RegisterServerProvider registers a server provider with both factory and optional config transformer.
// This is a convenience function that combines RegisterServerProviderFactory and optionally
// RegisterServerBuildFromEnv in one call.
func RegisterServerProvider(name string, factory func() ports.ServerProvider, builder func() (ports.ServerProvider, error)) {
	RegisterServerProviderFactory(name, factory)
	if builder != nil {
		RegisterServerBuildFromEnv(name, builder)
	}
}
