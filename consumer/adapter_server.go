package consumer

import (
	"context"
	"fmt"
	"net/http"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/composition/core"
	"leapfor.xyz/espyna/internal/composition/providers/infrastructure"
)

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Server Adapter

Provides direct access to HTTP server operations without requiring
framework-specific code in consumer applications.

This adapter works with ANY HTTP server framework (Gin, Fiber, Vanilla, etc.)
based on:
  1. Build tags (compile-time selection)
  2. CONFIG_SERVER_PROVIDER environment variable (runtime hint)

Build Tags (one required):
  - gin: Use Gin framework
  - fiber: Use Fiber v2 framework
  - fiber_v3: Use Fiber v3 framework
  - vanilla: Use vanilla net/http

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewServerAdapterFromContainer(container)

	// Start server
	if err := adapter.Start(":8080"); err != nil {
	    log.Fatal(err)
	}

	// Access server name
	fmt.Println("Running:", adapter.Name())

Environment Variables:
  - CONFIG_SERVER_PROVIDER: Server framework to use (gin, fiber, fiber_v3, vanilla)
    Defaults to the first available compiled-in provider.
*/

// ServerAdapter provides technology-agnostic access to HTTP server implementations.
// It wraps the ServerProvider interface and works with Gin, Fiber, Vanilla HTTP, etc.
type ServerAdapter struct {
	provider  ports.ServerProvider
	container *core.Container
}

// NewServerAdapterFromContainer creates a ServerAdapter from an existing container.
// The server provider is selected based on CONFIG_SERVER_PROVIDER environment variable
// and what was compiled in via build tags.
func NewServerAdapterFromContainer(container *core.Container) *ServerAdapter {
	if container == nil {
		return nil
	}

	// Create server provider using the factory
	provider, err := infrastructure.CreateServerProvider()
	if err != nil {
		// Log the error but don't fail - consumer can check IsEnabled()
		fmt.Printf("WARNING: Failed to create server provider: %v\n", err)
		fmt.Printf("   Available providers: %v\n", infrastructure.ListAvailableServerProviders())
		return nil
	}

	// Initialize the provider with the container
	if err := provider.Initialize(container); err != nil {
		fmt.Printf("WARNING: Failed to initialize server provider: %v\n", err)
		return nil
	}

	return &ServerAdapter{
		provider:  provider,
		container: container,
	}
}

// Start starts the HTTP server on the specified address.
func (a *ServerAdapter) Start(addr string) error {
	if a.provider == nil {
		return fmt.Errorf("server provider not initialized")
	}
	return a.provider.Start(addr)
}

// Name returns the name of the server framework (e.g., "gin", "fiber", "vanilla")
func (a *ServerAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the server provider is enabled
func (a *ServerAdapter) IsEnabled() bool {
	return a.provider != nil && a.provider.IsEnabled()
}

// IsHealthy checks if the server is healthy.
func (a *ServerAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("server provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// Close shuts down the server gracefully.
func (a *ServerAdapter) Close() error {
	if a.provider == nil {
		return nil
	}
	return a.provider.Close()
}

// GetProvider returns the underlying ServerProvider for advanced operations.
func (a *ServerAdapter) GetProvider() ports.ServerProvider {
	return a.provider
}

// GetContainer returns the underlying container for advanced operations.
func (a *ServerAdapter) GetContainer() *core.Container {
	return a.container
}

// RegisterCustomHandler registers a custom HTTP handler for the given method and path.
// This allows consumer applications to add custom routes beyond the espyna-generated routes.
// The handler is a standard http.HandlerFunc, which all frameworks can wrap.
//
// Parameters:
//   - method: HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
//   - path: URL path (e.g., "/api/v1/custom", "/health")
//   - handler: Standard http.HandlerFunc to register
//
// Returns an error if the registration fails or the provider is not initialized.
func (a *ServerAdapter) RegisterCustomHandler(method, path string, handler http.HandlerFunc) error {
	if a.provider == nil {
		return fmt.Errorf("server provider not initialized")
	}
	return a.provider.RegisterCustomHandler(method, path, handler)
}

// --- Convenience Functions ---

// ListAvailableServerProviders returns the names of all compiled-in server providers.
func ListAvailableServerProviders() []string {
	return infrastructure.ListAvailableServerProviders()
}

// GetConfiguredServerProvider returns the name of the configured server provider.
func GetConfiguredServerProvider() string {
	return infrastructure.GetServerProviderName()
}
