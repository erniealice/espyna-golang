//go:build (!google || !gmail) && (!microsoft || !microsoftgraph)

package integration

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	integrationuc "github.com/erniealice/espyna-golang/internal/application/usecases/integration"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
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
