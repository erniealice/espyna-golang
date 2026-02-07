package consumer

import (
	"leapfor.xyz/espyna/internal/application/ports"
)

/*
 ESPYNA CONSUMER APP - Technology-Agnostic ID Adapter

Provides direct access to ID generation operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY ID provider (UUIDv7, ULID, Snowflake, NoOp)
based on your CONFIG_ID_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewIDAdapterFromContainer(container)

	// Generate ID
	id := adapter.GenerateID()

	// Generate ID with prefix (e.g., "client_01HXYZ...")
	clientID := adapter.GenerateIDWithPrefix("client")
*/

// IDAdapter provides technology-agnostic access to ID generation services.
// It wraps the IDService interface and works with UUIDv7, ULID, Snowflake, etc.
type IDAdapter struct {
	service   ports.IDService
	container *Container
}

// NewIDAdapterFromContainer creates an IDAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's provider.
func NewIDAdapterFromContainer(container *Container) *IDAdapter {
	if container == nil {
		return nil
	}

	// Get ID provider from container
	idProvider := container.GetIDProvider()
	if idProvider == nil {
		// Fallback to NoOp service if no provider
		return &IDAdapter{
			service:   ports.NewNoOpIDService(),
			container: container,
		}
	}

	// Try to extract IDService from provider
	// ID providers typically implement GetIDService() method
	if wrapper, ok := idProvider.(interface{ GetIDService() ports.IDService }); ok {
		service := wrapper.GetIDService()
		if service != nil {
			return &IDAdapter{
				service:   service,
				container: container,
			}
		}
	}

	// Fallback to NoOp service
	return &IDAdapter{
		service:   ports.NewNoOpIDService(),
		container: container,
	}
}

// Close closes the ID adapter.
// Note: If created from container, this does NOT close the container.
func (a *IDAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetService returns the underlying IDService for advanced operations.
func (a *IDAdapter) GetService() ports.IDService {
	return a.service
}

// IsEnabled returns whether the ID service is enabled
func (a *IDAdapter) IsEnabled() bool {
	if a.service == nil {
		return false
	}
	return a.service.IsEnabled()
}

// GetProviderInfo returns information about the underlying ID provider.
func (a *IDAdapter) GetProviderInfo() string {
	if a.service == nil {
		return ""
	}
	return a.service.GetProviderInfo()
}

// --- ID Generation Operations ---

// GenerateID creates a new unique identifier.
// The format depends on the configured provider (e.g., UUIDv7, ULID).
func (a *IDAdapter) GenerateID() string {
	if a.service == nil {
		return ""
	}
	return a.service.GenerateID()
}

// GenerateIDWithPrefix creates a unique identifier with a specified prefix.
// Useful for maintaining readable ID formats (e.g., "client_uuid", "admin_uuid").
//
// Example:
//
//	adapter.GenerateIDWithPrefix("client") // Returns "client_01HXYZ..."
//	adapter.GenerateIDWithPrefix("order")  // Returns "order_01HXYZ..."
func (a *IDAdapter) GenerateIDWithPrefix(prefix string) string {
	if a.service == nil {
		return ""
	}
	return a.service.GenerateIDWithPrefix(prefix)
}

// --- Convenience Methods ---

// GenerateClientID generates an ID with "client" prefix.
func (a *IDAdapter) GenerateClientID() string {
	return a.GenerateIDWithPrefix("client")
}

// GenerateAdminID generates an ID with "admin" prefix.
func (a *IDAdapter) GenerateAdminID() string {
	return a.GenerateIDWithPrefix("admin")
}

// GenerateManagerID generates an ID with "manager" prefix.
func (a *IDAdapter) GenerateManagerID() string {
	return a.GenerateIDWithPrefix("manager")
}

// GenerateDelegateID generates an ID with "delegate" prefix.
func (a *IDAdapter) GenerateDelegateID() string {
	return a.GenerateIDWithPrefix("delegate")
}

// GenerateOrderID generates an ID with "order" prefix.
func (a *IDAdapter) GenerateOrderID() string {
	return a.GenerateIDWithPrefix("order")
}

// GeneratePaymentID generates an ID with "payment" prefix.
func (a *IDAdapter) GeneratePaymentID() string {
	return a.GenerateIDWithPrefix("payment")
}

// GenerateSubscriptionID generates an ID with "subscription" prefix.
func (a *IDAdapter) GenerateSubscriptionID() string {
	return a.GenerateIDWithPrefix("subscription")
}

// GenerateInvoiceID generates an ID with "invoice" prefix.
func (a *IDAdapter) GenerateInvoiceID() string {
	return a.GenerateIDWithPrefix("invoice")
}

// GenerateEventID generates an ID with "event" prefix.
func (a *IDAdapter) GenerateEventID() string {
	return a.GenerateIDWithPrefix("event")
}

// GenerateWorkflowID generates an ID with "workflow" prefix.
func (a *IDAdapter) GenerateWorkflowID() string {
	return a.GenerateIDWithPrefix("workflow")
}
