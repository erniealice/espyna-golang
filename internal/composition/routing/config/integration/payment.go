//go:build asiapay || paypal || maya

// Package integration provides HTTP routing configuration for integration use cases.
//
// # Payment Integration Routes
//
// This file configures HTTP endpoints for payment provider operations (AsiaPay, PayPal, Maya, etc.)
// All use cases are exposed via both HTTP routing and workflow activities.
//
// # Available Endpoints
//
//   - ProcessWebhook: Handles incoming payment provider callbacks
//   - LogWebhook: Saves parsed webhook data to integration_payment collection
//   - CreateCheckout: Creates a new checkout session with the payment provider
//   - GetPaymentStatus: Retrieves the status of a payment transaction
//   - CheckHealth: Verifies the payment provider connection
//   - GetCapabilities: Returns supported features of the payment provider
//
// # Keeping in Sync
//
// When adding new payment use cases, update:
//   - This file (for HTTP routing)
//   - packages/espyna/internal/orchestration/workflow/integration/payment.go (for workflows)
//
// When adding a NEW integration type, also update:
//   - packages/espyna/internal/composition/routing/config/config.go (to register the integration)
package integration

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	integrationuc "github.com/erniealice/espyna-golang/internal/application/usecases/integration"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

// Ensure ports is used (for interface compatibility)
var _ ports.PaymentProvider = nil

// ConfigurePaymentIntegration configures routes for payment provider integration
// This is only compiled when the 'asiapay' build tag is present
func ConfigurePaymentIntegration(
	_ ports.PaymentProvider, // Kept for backward compatibility
	integration *integrationuc.IntegrationUseCases,
) contracts.DomainRouteConfiguration {
	// Check if payment use cases are available (not just the provider)
	if integration == nil || integration.Payment == nil {
		return contracts.DomainRouteConfiguration{
			Domain:  "payment_integration",
			Prefix:  "/integration/payment",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// Webhook endpoint - processes incoming payment provider callbacks
	if integration.Payment.ProcessWebhook != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/payment/webhook",
			Handler: contracts.NewGenericHandler(integration.Payment.ProcessWebhook, &paymentpb.ProcessWebhookRequest{}),
		})
	}

	// Log webhook endpoint - saves parsed webhook data to integration_payment collection
	if integration.Payment.LogWebhook != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/payment/log",
			Handler: contracts.NewGenericHandler(integration.Payment.LogWebhook, &paymentpb.LogWebhookRequest{}),
		})
	}

	// Create checkout session endpoint
	if integration.Payment.CreateCheckout != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/payment/checkout",
			Handler: contracts.NewGenericHandler(integration.Payment.CreateCheckout, &paymentpb.CreateCheckoutSessionRequest{}),
		})
	}

	// Get payment status endpoint
	if integration.Payment.GetPaymentStatus != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/payment/status",
			Handler: contracts.NewGenericHandler(integration.Payment.GetPaymentStatus, &paymentpb.GetPaymentStatusRequest{}),
		})
	}

	// Health check endpoint
	if integration.Payment.CheckHealth != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "GET",
			Path:    "/integration/payment/health",
			Handler: contracts.NewGenericHandler(integration.Payment.CheckHealth, &paymentpb.CheckHealthRequest{}),
		})
	}

	// Get provider capabilities endpoint
	if integration.Payment.GetCapabilities != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "GET",
			Path:    "/integration/payment/capabilities",
			Handler: contracts.NewGenericHandler(integration.Payment.GetCapabilities, &paymentpb.GetCapabilitiesRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "payment_integration",
		Prefix:  "/integration/payment",
		Enabled: true,
		Routes:  routes,
	}
}
