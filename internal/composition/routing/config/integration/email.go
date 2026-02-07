//go:build (google && gmail) || (microsoft && microsoftgraph)

// Package integration provides HTTP routing configuration for integration use cases.
//
// # Email Integration Routes
//
// This file configures HTTP endpoints for email operations (Gmail, Microsoft Graph, etc.)
// All use cases are exposed via both HTTP routing and workflow activities.
//
// # Available Endpoints
//
//   - SendEmail: Sends an email through the configured provider
//   - CheckHealth: Verifies the email provider connection
//   - GetCapabilities: Returns supported features of the email provider
//
// # Keeping in Sync
//
// When adding new email use cases, update:
//   - This file (for HTTP routing)
//   - packages/espyna/internal/orchestration/workflow/integration/email.go (for workflows)
//
// When adding a NEW integration type, also update:
//   - packages/espyna/internal/composition/routing/config/config.go (to register the integration)
package integration

import (
	"leapfor.xyz/espyna/internal/application/ports"
	integrationuc "leapfor.xyz/espyna/internal/application/usecases/integration"
	"leapfor.xyz/espyna/internal/composition/contracts"
	emailpb "leapfor.xyz/esqyma/golang/v1/integration/email"
)

// Ensure ports is used (for interface compatibility)
var _ ports.EmailProvider = nil

// ConfigureEmailIntegration configures routes for email integration
// This is only compiled when both 'google' and 'gmail' build tags are present
func ConfigureEmailIntegration(
	_ ports.EmailProvider, // Kept for backward compatibility
	integration *integrationuc.IntegrationUseCases,
) contracts.DomainRouteConfiguration {
	// Check if email use cases are available (not just the provider)
	if integration == nil || integration.Email == nil {
		return contracts.DomainRouteConfiguration{
			Domain:  "email_integration",
			Prefix:  "/integration/email",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// Send email endpoint
	if integration.Email.SendEmail != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/integration/email/send",
			Handler: contracts.NewGenericHandler(integration.Email.SendEmail, &emailpb.SendEmailRequest{}),
		})
	}

	// Health check endpoint
	if integration.Email.CheckHealth != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "GET",
			Path:    "/integration/email/health",
			Handler: contracts.NewGenericHandler(integration.Email.CheckHealth, &emailpb.CheckHealthRequest{}),
		})
	}

	// Get provider capabilities endpoint
	if integration.Email.GetCapabilities != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "GET",
			Path:    "/integration/email/capabilities",
			Handler: contracts.NewGenericHandler(integration.Email.GetCapabilities, &emailpb.GetCapabilitiesRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "email_integration",
		Prefix:  "/integration/email",
		Enabled: true,
		Routes:  routes,
	}
}
