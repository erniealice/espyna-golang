//go:build !asiapay && !paypal && !maya

package integration

import (
	"leapfor.xyz/espyna/internal/application/ports"
	integrationuc "leapfor.xyz/espyna/internal/application/usecases/integration"
	"leapfor.xyz/espyna/internal/composition/contracts"
)

// Ensure ports is used (for interface compatibility)
var _ ports.PaymentProvider = nil

// ConfigurePaymentIntegration stub for when asiapay build tag is not present
func ConfigurePaymentIntegration(
	_ ports.PaymentProvider,
	_ *integrationuc.IntegrationUseCases,
) contracts.DomainRouteConfiguration {
	return contracts.DomainRouteConfiguration{
		Domain:  "payment_integration",
		Prefix:  "/integration/payment",
		Enabled: false,
		Routes:  []contracts.RouteConfiguration{},
	}
}
