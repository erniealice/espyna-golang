// Package integration aggregates all external provider integration use cases.
//
// # Adding a New Integration Type
//
// When adding a new integration type (e.g., SMS, Storage), update:
//
//  1. IntegrationUseCases struct - Add the new use case field
//  2. NewIntegrationUseCases() - Add provider parameter and initialization logic
//  3. packages/espyna/internal/composition/core/usecases.go - Update initializeIntegrationUseCases()
//     to get the new provider from container and pass it to NewIntegrationUseCases()
//
// # Current Integrations
//
//   - Payment: AsiaPay, PayPal, Maya payment providers
//   - Email: Gmail email provider
//   - Scheduler: Calendly scheduling provider
//   - Tabular: Google Sheets data provider
package integration

import (
	"leapfor.xyz/espyna/internal/application/ports"
	integrationPorts "leapfor.xyz/espyna/internal/application/ports/integration"

	// Email integration use cases
	emailUseCases "leapfor.xyz/espyna/internal/application/usecases/integration/email"
	// Payment integration use cases
	paymentUseCases "leapfor.xyz/espyna/internal/application/usecases/integration/payment"
	// Scheduler integration use cases
	schedulerUseCases "leapfor.xyz/espyna/internal/application/usecases/integration/scheduler"
	// Tabular integration use cases
	tabularUseCases "leapfor.xyz/espyna/internal/application/usecases/integration/tabular"
)

// IntegrationUseCases contains all integration domain use cases
type IntegrationUseCases struct {
	Payment   *paymentUseCases.UseCases
	Email     *emailUseCases.UseCases
	Scheduler *schedulerUseCases.UseCases
	Tabular   *tabularUseCases.UseCases
}

// NewIntegrationUseCases creates a new collection of integration use cases
func NewIntegrationUseCases(
	paymentProvider ports.PaymentProvider,
	emailProvider ports.EmailProvider,
	schedulerProvider ports.SchedulerProvider,
	tabularProvider ports.TabularSourceProvider,
	integrationPaymentRepo integrationPorts.IntegrationPaymentRepository,
) *IntegrationUseCases {
	var paymentUC *paymentUseCases.UseCases
	var emailUC *emailUseCases.UseCases
	var schedulerUC *schedulerUseCases.UseCases
	var tabularUC *tabularUseCases.UseCases

	// Initialize payment use cases if provider is available
	if paymentProvider != nil {
		paymentRepositories := paymentUseCases.PaymentRepositories{
			IntegrationPayment: integrationPaymentRepo,
		}
		paymentServices := paymentUseCases.PaymentServices{
			Provider: paymentProvider,
		}
		paymentUC = paymentUseCases.NewUseCases(paymentRepositories, paymentServices)
	}

	// Initialize email use cases if provider is available
	if emailProvider != nil {
		emailRepositories := emailUseCases.EmailRepositories{}
		emailServices := emailUseCases.EmailServices{
			Provider: emailProvider,
		}
		emailUC = emailUseCases.NewUseCases(emailRepositories, emailServices)
	}

	// Initialize scheduler use cases if provider is available
	if schedulerProvider != nil {
		schedulerRepositories := schedulerUseCases.SchedulerRepositories{}
		schedulerServices := schedulerUseCases.SchedulerServices{
			Provider: schedulerProvider,
		}
		schedulerUC = schedulerUseCases.NewUseCases(schedulerRepositories, schedulerServices)
	}

	// Initialize tabular use cases if provider is available
	if tabularProvider != nil {
		tabularRepositories := tabularUseCases.TabularRepositories{}
		tabularServices := tabularUseCases.TabularServices{
			Provider: tabularProvider,
		}
		tabularUC = tabularUseCases.NewUseCases(tabularRepositories, tabularServices)
	}

	return &IntegrationUseCases{
		Payment:   paymentUC,
		Email:     emailUC,
		Scheduler: schedulerUC,
		Tabular:   tabularUC,
	}
}
