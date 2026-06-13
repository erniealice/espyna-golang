//go:build !google || !google_sheets

package integration

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	integrationuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/integration"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
)

// Ensure ports is used (for interface compatibility)
var _ ports.TabularSourceProvider = nil

// ConfigureTabularIntegration stub for when google_sheets build tag is not present
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
