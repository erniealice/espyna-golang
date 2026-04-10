package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/integration"
	integrationPorts "github.com/erniealice/espyna-golang/internal/application/ports/integration"
)

// InitializeIntegration creates all integration use cases from provider dependencies.
// Unlike other domain initializers, integration depends on provider interfaces rather
// than repository interfaces -- providers are selected at compile time via build tags.
func InitializeIntegration(
	paymentProvider ports.PaymentProvider,
	emailProvider ports.EmailProvider,
	schedulerProvider ports.SchedulerProvider,
	tabularProvider ports.TabularSourceProvider,
	integrationPaymentRepo integrationPorts.IntegrationPaymentRepository,
) *integration.IntegrationUseCases {
	return integration.NewIntegrationUseCases(
		paymentProvider,
		emailProvider,
		schedulerProvider,
		tabularProvider,
		integrationPaymentRepo,
	)
}
