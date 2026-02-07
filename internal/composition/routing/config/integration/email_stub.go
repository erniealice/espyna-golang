//go:build (!google || !gmail) && (!microsoft || !microsoftgraph)

package integration

import (
	"leapfor.xyz/espyna/internal/application/ports"
	integrationuc "leapfor.xyz/espyna/internal/application/usecases/integration"
	"leapfor.xyz/espyna/internal/composition/contracts"
)

// Ensure ports is used (for interface compatibility)
var _ ports.EmailProvider = nil

// ConfigureEmailIntegration stub for when gmail build tag is not present
func ConfigureEmailIntegration(
	_ ports.EmailProvider,
	_ *integrationuc.IntegrationUseCases,
) contracts.DomainRouteConfiguration {
	return contracts.DomainRouteConfiguration{
		Domain:  "email_integration",
		Prefix:  "/integration/email",
		Enabled: false,
		Routes:  []contracts.RouteConfiguration{},
	}
}
