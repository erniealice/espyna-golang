package config

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases"
	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/composition/routing/config/domain"
	"leapfor.xyz/espyna/internal/composition/routing/config/integration"
	"leapfor.xyz/espyna/internal/composition/routing/config/orchestration"
)

// GetAllDomainConfigurations returns all domain route configurations with use cases injected.
// The engineService parameter is optional - if nil, orchestration routes will be disabled.
func GetAllDomainConfigurations(useCases *usecases.Aggregate, engineService ports.WorkflowEngineService) []contracts.DomainRouteConfiguration {
	configs := []contracts.DomainRouteConfiguration{
		domain.ConfigureCommonDomain(useCases.Common),
		domain.ConfigureEntityDomain(useCases.Entity),
		domain.ConfigureEventDomain(useCases.Event),
		domain.ConfigurePaymentDomain(useCases.Payment),
		domain.ConfigureProductDomain(useCases.Product),
		domain.ConfigureSubscriptionDomain(useCases.Subscription),
		domain.ConfigureWorkflowDomain(useCases.Workflow),
	}

	// Add integration routes if integration use cases are available
	if useCases.Integration != nil {
		// Add email integration routes
		emailConfig := integration.ConfigureEmailIntegration(nil, useCases.Integration)
		if emailConfig.Enabled {
			configs = append(configs, emailConfig)
		}

		// Add payment integration routes
		paymentConfig := integration.ConfigurePaymentIntegration(nil, useCases.Integration)
		if paymentConfig.Enabled {
			configs = append(configs, paymentConfig)
		}

		// Add tabular integration routes (Google Sheets, etc.)
		tabularConfig := integration.ConfigureTabularIntegration(nil, useCases.Integration)
		if tabularConfig.Enabled {
			configs = append(configs, tabularConfig)
		}
	}

	// Add orchestration routes if engine service is available
	if engineService != nil {
		engineConfig := orchestration.ConfigureWorkflowEngine(engineService)
		if engineConfig.Enabled {
			configs = append(configs, engineConfig)
		}
	}

	return configs
}
