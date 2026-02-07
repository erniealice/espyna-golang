package scheduler

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	schedulerpb "leapfor.xyz/esqyma/golang/v1/integration/scheduler"
)

// GetEventTypeRepositories groups all repository dependencies
type GetEventTypeRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// GetEventTypeServices groups all service dependencies
type GetEventTypeServices struct {
	Provider ports.SchedulerProvider
}

// GetEventTypeUseCase handles retrieving event type details
type GetEventTypeUseCase struct {
	repositories GetEventTypeRepositories
	services     GetEventTypeServices
}

// NewGetEventTypeUseCase creates a new GetEventTypeUseCase
func NewGetEventTypeUseCase(
	repositories GetEventTypeRepositories,
	services GetEventTypeServices,
) *GetEventTypeUseCase {
	return &GetEventTypeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute retrieves event type details from the provider
func (uc *GetEventTypeUseCase) Execute(ctx context.Context, req *schedulerpb.GetEventTypeRequest) (*schedulerpb.GetEventTypeResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📋 Getting event type: %s", req.Data.EventTypeId)

	response, err := uc.services.Provider.GetEventType(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to get event type: %v", err)
		return &schedulerpb.GetEventTypeResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "GET_EVENT_TYPE_FAILED",
				Message: fmt.Sprintf("Failed to get event type: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("✅ Retrieved event type: %s", response.Data[0].Name)
		log.Printf("   Duration: %d minutes", response.Data[0].DurationMinutes)
		log.Printf("   Slug: %s", response.Data[0].Slug)
	}

	return response, nil
}
