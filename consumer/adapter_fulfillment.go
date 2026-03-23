package consumer

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	fulfillment "github.com/erniealice/espyna-golang/internal/application/ports/integration"
)

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Fulfillment Adapter

Provides direct access to delivery/logistics operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY fulfillment provider (Lalamove, GrabExpress, etc.)
based on your CONFIG_FULFILLMENT_PROVIDER environment variable.

Supports multiple simultaneous providers (e.g., Lalamove + GrabExpress).

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewFulfillmentAdapterFromContainer(container)

	// Get a delivery quote
	quote, err := adapter.GetQuote(ctx, &fulfillment.FulfillmentQuoteRequest{
	    PickupAddress:   fulfillment.Address{Line1: "123 Sender St", City: "Manila", Country: "PH"},
	    DeliveryAddress: fulfillment.Address{Line1: "456 Recipient Ave", City: "Quezon City", Country: "PH"},
	    ItemDescription: "Electronics",
	    ItemWeight:      1.5,
	    ServiceType:     "standard",
	})

	// Create a delivery
	delivery, err := adapter.CreateDelivery(ctx, &fulfillment.CreateDeliveryRequest{...})

	// Track a delivery
	status, err := adapter.TrackDelivery(ctx, &fulfillment.TrackDeliveryRequest{DeliveryID: "del_123"})

	// Use a specific provider by name
	provider := adapter.ForProvider("lalamove")
	if provider != nil {
	    quote, err := provider.GetQuote(ctx, req)
	}
*/

// FulfillmentAdapter provides technology-agnostic access to delivery/logistics services.
// It wraps the FulfillmentProvider interface and supports multiple simultaneous providers
// (e.g., Lalamove + GrabExpress).
type FulfillmentAdapter struct {
	provider  ports.FulfillmentProvider            // primary (first provider, for convenience)
	providers map[string]ports.FulfillmentProvider // all registered providers
	container *Container
}

// NewFulfillmentAdapterFromContainer creates a FulfillmentAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's providers.
// Returns nil if no fulfillment providers are configured.
func NewFulfillmentAdapterFromContainer(container *Container) *FulfillmentAdapter {
	if container == nil {
		return nil
	}

	providers := container.GetFulfillmentProviders()
	if len(providers) == 0 {
		return nil
	}

	// Set primary to first provider
	var primary ports.FulfillmentProvider
	for _, p := range providers {
		primary = p
		break
	}

	return &FulfillmentAdapter{
		provider:  primary,
		providers: providers,
		container: container,
	}
}

// Close closes the fulfillment adapter.
// Note: If created from container, this does NOT close the container.
func (a *FulfillmentAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetProvider returns the underlying primary FulfillmentProvider for advanced operations.
func (a *FulfillmentAdapter) GetProvider() ports.FulfillmentProvider {
	return a.provider
}

// Name returns the name of the primary fulfillment provider (e.g., "lalamove", "grabexpress", "mock_fulfillment")
func (a *FulfillmentAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the primary fulfillment provider is enabled
func (a *FulfillmentAdapter) IsEnabled() bool {
	return a.provider != nil && a.provider.IsEnabled()
}

// --- Fulfillment Operations (primary provider) ---

// GetQuote requests a delivery price estimate from the primary provider.
func (a *FulfillmentAdapter) GetQuote(ctx context.Context, req *fulfillment.FulfillmentQuoteRequest) (*fulfillment.FulfillmentQuoteResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("fulfillment provider not initialized")
	}
	return a.provider.GetQuote(ctx, req)
}

// CreateDelivery creates a new delivery order via the primary provider.
func (a *FulfillmentAdapter) CreateDelivery(ctx context.Context, req *fulfillment.CreateDeliveryRequest) (*fulfillment.CreateDeliveryResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("fulfillment provider not initialized")
	}
	return a.provider.CreateDelivery(ctx, req)
}

// CancelDelivery cancels an existing delivery via the primary provider.
func (a *FulfillmentAdapter) CancelDelivery(ctx context.Context, req *fulfillment.CancelDeliveryRequest) (*fulfillment.CancelDeliveryResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("fulfillment provider not initialized")
	}
	return a.provider.CancelDelivery(ctx, req)
}

// TrackDelivery gets the current status of a delivery via the primary provider.
func (a *FulfillmentAdapter) TrackDelivery(ctx context.Context, req *fulfillment.TrackDeliveryRequest) (*fulfillment.TrackDeliveryResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("fulfillment provider not initialized")
	}
	return a.provider.TrackDelivery(ctx, req)
}

// ProcessWebhook handles an incoming webhook event via the primary provider.
func (a *FulfillmentAdapter) ProcessWebhook(ctx context.Context, req *fulfillment.FulfillmentWebhookRequest) (*fulfillment.FulfillmentWebhookResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("fulfillment provider not initialized")
	}
	return a.provider.ProcessWebhook(ctx, req)
}

// IsHealthy checks if the primary fulfillment provider is healthy and available.
func (a *FulfillmentAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("fulfillment provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// GetCapabilities returns the capabilities supported by the primary fulfillment provider.
func (a *FulfillmentAdapter) GetCapabilities() []string {
	if a.provider == nil {
		return nil
	}
	return a.provider.GetCapabilities()
}

// --- Multi-provider methods ---

// ForProvider returns a specific fulfillment provider by name.
// Returns nil if the provider is not registered.
// Usage: adapter.ForProvider("lalamove").CreateDelivery(ctx, req)
func (a *FulfillmentAdapter) ForProvider(name string) ports.FulfillmentProvider {
	if a.providers == nil {
		return nil
	}
	return a.providers[name]
}

// ProviderNames returns the names of all registered fulfillment providers.
func (a *FulfillmentAdapter) ProviderNames() []string {
	if a.providers == nil {
		return nil
	}
	names := make([]string, 0, len(a.providers))
	for name := range a.providers {
		names = append(names, name)
	}
	return names
}

// AllHealthy checks health of all registered fulfillment providers.
// Returns a map of provider name to error (nil = healthy).
func (a *FulfillmentAdapter) AllHealthy(ctx context.Context) map[string]error {
	result := make(map[string]error)
	for name, p := range a.providers {
		result[name] = p.IsHealthy(ctx)
	}
	return result
}

// ProviderCount returns the number of registered fulfillment providers.
func (a *FulfillmentAdapter) ProviderCount() int {
	return len(a.providers)
}
