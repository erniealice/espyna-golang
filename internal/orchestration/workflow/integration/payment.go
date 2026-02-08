package integration

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/executor"
)

// RegisterPaymentIntegrationUseCases registers all payment integration use cases with the registry.
// Payment integration includes: CreateCheckout, ProcessWebhook, GetPaymentStatus,
// CheckHealth, GetCapabilities.
func RegisterPaymentIntegrationUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Integration == nil || useCases.Integration.Payment == nil {
		return
	}

	// Create checkout session use case
	if useCases.Integration.Payment.CreateCheckout != nil {
		register("integration.payment.create_checkout", executor.New(useCases.Integration.Payment.CreateCheckout.Execute))
	}

	// Process webhook use case
	if useCases.Integration.Payment.ProcessWebhook != nil {
		register("integration.payment.process_webhook", executor.New(useCases.Integration.Payment.ProcessWebhook.Execute))
	}

	// Log webhook use case - saves parsed webhook data to integration_payment collection
	if useCases.Integration.Payment.LogWebhook != nil {
		register("integration.payment.log_webhook", executor.New(useCases.Integration.Payment.LogWebhook.Execute))
	}

	// Get payment status use case
	if useCases.Integration.Payment.GetPaymentStatus != nil {
		register("integration.payment.get_status", executor.New(useCases.Integration.Payment.GetPaymentStatus.Execute))
	}

	// Check health use case
	if useCases.Integration.Payment.CheckHealth != nil {
		register("integration.payment.check_health", executor.New(useCases.Integration.Payment.CheckHealth.Execute))
	}

	// Get capabilities use case
	if useCases.Integration.Payment.GetCapabilities != nil {
		register("integration.payment.get_capabilities", executor.New(useCases.Integration.Payment.GetCapabilities.Execute))
	}
}
