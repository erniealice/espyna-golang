package email

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// GetCapabilitiesRepositories groups all repository dependencies
type GetCapabilitiesRepositories struct {
	// No repositories needed for capabilities query
}

// GetCapabilitiesServices groups all service dependencies
type GetCapabilitiesServices struct {
	Provider ports.EmailProvider
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

// Execute retrieves the capabilities of the email provider
func (uc *GetCapabilitiesUseCase) Execute(ctx context.Context, req *emailpb.GetCapabilitiesRequest) (*emailpb.GetCapabilitiesResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &emailpb.GetCapabilitiesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Email provider is not available",
			},
		}, nil
	}

	return &emailpb.GetCapabilitiesResponse{
		Success: true,
		Data: []*emailpb.EmailProviderCapabilities{
			{
				ProviderId:   uc.services.Provider.Name(),
				ProviderType: uc.services.Provider.GetProviderType(),
				Capabilities: uc.services.Provider.GetCapabilities(),
			},
		},
	}, nil
}
