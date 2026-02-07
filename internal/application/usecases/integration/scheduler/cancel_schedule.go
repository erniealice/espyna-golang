package scheduler

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	schedulerpb "leapfor.xyz/esqyma/golang/v1/integration/scheduler"
)

// CancelScheduleRepositories groups all repository dependencies
type CancelScheduleRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// CancelScheduleServices groups all service dependencies
type CancelScheduleServices struct {
	Provider ports.SchedulerProvider
}

// CancelScheduleUseCase handles cancelling scheduled events
type CancelScheduleUseCase struct {
	repositories CancelScheduleRepositories
	services     CancelScheduleServices
}

// NewCancelScheduleUseCase creates a new CancelScheduleUseCase
func NewCancelScheduleUseCase(
	repositories CancelScheduleRepositories,
	services CancelScheduleServices,
) *CancelScheduleUseCase {
	return &CancelScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute cancels a scheduled event
func (uc *CancelScheduleUseCase) Execute(ctx context.Context, req *schedulerpb.CancelScheduleRequest) (*schedulerpb.CancelScheduleResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	scheduleID := req.Data.ScheduleId
	if scheduleID == "" {
		scheduleID = req.Data.ProviderScheduleId
	}

	log.Printf("🚫 Cancelling schedule: %s", scheduleID)
	if req.Data.Reason != "" {
		log.Printf("   Reason: %s", req.Data.Reason)
	}

	response, err := uc.services.Provider.CancelSchedule(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to cancel schedule: %v", err)
		return &schedulerpb.CancelScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "CANCEL_FAILED",
				Message: fmt.Sprintf("Failed to cancel schedule: %v", err),
			},
		}, nil
	}

	if response.Success {
		log.Printf("✅ Schedule cancelled successfully")
	}

	return response, nil
}
