package scheduler

import (
	"context"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

// GetCapabilitiesRepositories groups all repository dependencies
type GetCapabilitiesRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// GetCapabilitiesServices groups all service dependencies
type GetCapabilitiesServices struct {
	Provider ports.SchedulerProvider
}

// GetCapabilitiesUseCase handles retrieving scheduler provider capabilities
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

// Execute retrieves the capabilities of the scheduler provider
func (uc *GetCapabilitiesUseCase) Execute(ctx context.Context, req *schedulerpb.GetSchedulerCapabilitiesRequest) (*schedulerpb.GetSchedulerCapabilitiesResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.GetSchedulerCapabilitiesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	log.Printf("📋 Getting scheduler provider capabilities: %s", uc.services.Provider.Name())

	capabilities := uc.services.Provider.GetCapabilities()

	log.Printf("✅ Provider capabilities retrieved: %d capabilities", len(capabilities))
	for _, cap := range capabilities {
		log.Printf("   - %s", cap.String())
	}

	// Determine provider type from name
	providerType := schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_UNSPECIFIED
	switch uc.services.Provider.Name() {
	case "calendly":
		providerType = schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_CALENDLY
	case "google_calendar":
		providerType = schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_GOOGLE_CALENDAR
	case "mock", "mock_scheduler":
		providerType = schedulerpb.SchedulerProviderType_SCHEDULER_PROVIDER_TYPE_MOCK
	}

	return &schedulerpb.GetSchedulerCapabilitiesResponse{
		Success: true,
		Data: []*schedulerpb.SchedulerProviderCapabilities{
			{
				ProviderId:   uc.services.Provider.Name(),
				ProviderType: providerType,
				Capabilities: capabilities,
				Limits:       map[string]string{},
			},
		},
	}, nil
}
