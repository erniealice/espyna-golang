package payment

import (
	"context"

	"leapfor.xyz/espyna/internal/application/ports"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	paymentpb "leapfor.xyz/esqyma/golang/v1/integration/payment"
)

// GetCapabilitiesRepositories groups all repository dependencies
type GetCapabilitiesRepositories struct {
	// No repositories needed for capabilities query
}

// GetCapabilitiesServices groups all service dependencies
type GetCapabilitiesServices struct {
	Provider ports.PaymentProvider
}

// GetCapabilitiesUseCase handles retrieving provider capabilities
type GetCapabilitiesUseCase struct {
	repositories GetCapabilitiesRepositories
	services     GetCapabilitiesServices
}

// NewGetCapabilitiesUseCase creates a new GetCapabilitiesUseCase
func NewGetCapabilitiesUseCase(
	repositories GetCapabilitiesRepositories,
	services GetCapabilitiesServices,
) *GetCapabilitiesUseCase {
	return &GetCapabilitiesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute retrieves the capabilities of the payment provider
func (uc *GetCapabilitiesUseCase) Execute(ctx context.Context, req *paymentpb.GetCapabilitiesRequest) (*paymentpb.GetCapabilitiesResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &paymentpb.GetCapabilitiesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Payment provider is not available",
			},
		}, nil
	}

	return &paymentpb.GetCapabilitiesResponse{
		Success: true,
		Data: []*paymentpb.ProviderCapabilities{
			{
				ProviderId:          uc.services.Provider.Name(),
				ProviderType:        paymentpb.PaymentProviderType_PAYMENT_PROVIDER_TYPE_GATEWAY,
				Capabilities:        uc.services.Provider.GetCapabilities(),
				SupportedCurrencies: uc.services.Provider.GetSupportedCurrencies(),
			},
		},
	}, nil
}
