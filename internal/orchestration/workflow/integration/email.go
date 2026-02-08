package integration

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/executor"
)

// RegisterEmailIntegrationUseCases registers all email integration use cases with the registry.
// Email integration includes: SendEmail, CheckHealth, GetCapabilities.
func RegisterEmailIntegrationUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Integration == nil || useCases.Integration.Email == nil {
		return
	}

	// Send email use case
	if useCases.Integration.Email.SendEmail != nil {
		register("integration.email.send", executor.New(useCases.Integration.Email.SendEmail.Execute))
	}

	// Check health use case
	if useCases.Integration.Email.CheckHealth != nil {
		register("integration.email.check_health", executor.New(useCases.Integration.Email.CheckHealth.Execute))
	}

	// Get capabilities use case
	if useCases.Integration.Email.GetCapabilities != nil {
		register("integration.email.get_capabilities", executor.New(useCases.Integration.Email.GetCapabilities.Execute))
	}
}
