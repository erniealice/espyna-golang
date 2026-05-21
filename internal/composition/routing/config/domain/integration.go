package domain

import (
	"fmt"

	integrationuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/integration"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// ConfigureIntegrationDomain configures routes for the Integration domain.
// Note: Individual integration provider routes (payment, email, tabular) are
// configured separately in the routing/config/integration package using build tags.
// This file handles any domain-level integration entity routes.
func ConfigureIntegrationDomain(integrationUseCases *integrationuc.IntegrationUseCases) contracts.DomainRouteConfiguration {
	if integrationUseCases == nil {
		fmt.Printf("Integration use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "integration",
			Prefix:  "/integration",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	// Integration domain routes are handled by the integration sub-package
	// (routing/config/integration/) with build-tag-selected providers.
	// This config exists for structural completeness and future domain entity routes.
	return contracts.DomainRouteConfiguration{
		Domain:  "integration",
		Prefix:  "/integration",
		Enabled: false,
		Routes:  []contracts.RouteConfiguration{},
	}
}
