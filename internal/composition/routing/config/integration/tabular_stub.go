//go:build !google || !googlesheets

package integration

import (
	"leapfor.xyz/espyna/internal/application/ports"
	integrationuc "leapfor.xyz/espyna/internal/application/usecases/integration"
	"leapfor.xyz/espyna/internal/composition/contracts"
)

// Ensure ports is used (for interface compatibility)
var _ ports.TabularSourceProvider = nil

// ConfigureTabularIntegration stub for when googlesheets build tag is not present
func ConfigureTabularIntegration(
	_ ports.TabularSourceProvider,
	_ *integrationuc.IntegrationUseCases,
) contracts.DomainRouteConfiguration {
	return contracts.DomainRouteConfiguration{
		Domain:  "tabular_integration",
		Prefix:  "/integration/tabular",
		Enabled: false,
		Routes:  []contracts.RouteConfiguration{},
	}
}
