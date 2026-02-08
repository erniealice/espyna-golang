package payment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	integrationPorts "github.com/erniealice/espyna-golang/internal/application/ports/integration"
)

// PaymentRepositories groups all repository dependencies for payment use cases
type PaymentRepositories struct {
	IntegrationPayment integrationPorts.IntegrationPaymentRepository
}

// PaymentServices groups all business service dependencies for payment use cases
type PaymentServices struct {
	Provider ports.PaymentProvider
}

// UseCases contains all payment integration use cases
type UseCases struct {
	CreateCheckout   *CreateCheckoutUseCase
	ProcessWebhook   *ProcessWebhookUseCase
	LogWebhook       *LogWebhookUseCase
	GetPaymentStatus *GetPaymentStatusUseCase
	CheckHealth      *CheckHealthUseCase
	GetCapabilities  *GetCapabilitiesUseCase
}

// NewUseCases creates a new collection of payment integration use cases
func NewUseCases(
	repositories PaymentRepositories,
	services PaymentServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createCheckoutRepos := CreateCheckoutRepositories{}
	createCheckoutServices := CreateCheckoutServices{
		Provider: services.Provider,
	}

	processWebhookRepos := ProcessWebhookRepositories{}
	processWebhookServices := ProcessWebhookServices{
		Provider: services.Provider,
	}

	logWebhookRepos := LogWebhookRepositories{
		IntegrationPayment: repositories.IntegrationPayment,
	}
	logWebhookServices := LogWebhookServices{}

	getPaymentStatusRepos := GetPaymentStatusRepositories{}
	getPaymentStatusServices := GetPaymentStatusServices{
		Provider: services.Provider,
	}

	checkHealthRepos := CheckHealthRepositories{}
	checkHealthServices := CheckHealthServices{
		Provider: services.Provider,
	}

	getCapabilitiesRepos := GetCapabilitiesRepositories{}
	getCapabilitiesServices := GetCapabilitiesServices{
		Provider: services.Provider,
	}

	return &UseCases{
		CreateCheckout:   NewCreateCheckoutUseCase(createCheckoutRepos, createCheckoutServices),
		ProcessWebhook:   NewProcessWebhookUseCase(processWebhookRepos, processWebhookServices),
		LogWebhook:       NewLogWebhookUseCase(logWebhookRepos, logWebhookServices),
		GetPaymentStatus: NewGetPaymentStatusUseCase(getPaymentStatusRepos, getPaymentStatusServices),
		CheckHealth:      NewCheckHealthUseCase(checkHealthRepos, checkHealthServices),
		GetCapabilities:  NewGetCapabilitiesUseCase(getCapabilitiesRepos, getCapabilitiesServices),
	}
}

// NewUseCasesFromProvider creates use cases directly from a payment provider
// This is a convenience function for simple setups
func NewUseCasesFromProvider(provider ports.PaymentProvider) *UseCases {
	if provider == nil {
		return nil
	}

	repositories := PaymentRepositories{}
	services := PaymentServices{
		Provider: provider,
	}

	return NewUseCases(repositories, services)
}
