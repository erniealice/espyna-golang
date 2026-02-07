package tabular

import (
	"context"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// GetCapabilitiesRepositories groups all repository dependencies
type GetCapabilitiesRepositories struct {
	// No repositories needed for getting capabilities
}

// GetCapabilitiesServices groups all service dependencies
type GetCapabilitiesServices struct {
	Provider integration.TabularSourceProvider
}

// GetCapabilitiesUseCase handles retrieving tabular provider capabilities
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

// Execute retrieves the capabilities of the tabular provider
func (uc *GetCapabilitiesUseCase) Execute(ctx context.Context, req *tabularpb.GetCapabilitiesRequest) (*tabularpb.GetCapabilitiesResponse, error) {
	if uc.services.Provider == nil {
		return &tabularpb.GetCapabilitiesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not configured",
			},
		}, nil
	}

	log.Printf("Getting capabilities for tabular provider: %s", uc.services.Provider.Name())

	// Try to use the structured capabilities method first
	if req != nil {
		response, err := uc.services.Provider.GetCapabilitiesInfo(ctx, req)
		if err == nil && response != nil {
			return response, nil
		}
		// Fall back to simple capabilities if structured method fails
		log.Printf("Falling back to simple capabilities: %v", err)
	}

	// Use simple capabilities
	capabilities := uc.services.Provider.GetCapabilities()
	providerType := uc.services.Provider.GetProviderType()

	return &tabularpb.GetCapabilitiesResponse{
		Success: true,
		Data: []*tabularpb.ProviderCapabilities{
			{
				ProviderId:   uc.services.Provider.Name(),
				ProviderType: providerType,
				Capabilities: capabilities,
			},
		},
	}, nil
}
