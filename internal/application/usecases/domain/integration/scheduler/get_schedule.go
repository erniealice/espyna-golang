package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

// GetScheduleRepositories groups all repository dependencies
type GetScheduleRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// GetScheduleServices groups all service dependencies
type GetScheduleServices struct {
	Provider ports.SchedulerProvider
}

// GetScheduleUseCase handles retrieving schedule details
type GetScheduleUseCase struct {
	repositories GetScheduleRepositories
	services     GetScheduleServices
}

// NewGetScheduleUseCase creates a new GetScheduleUseCase
func NewGetScheduleUseCase(
	repositories GetScheduleRepositories,
	services GetScheduleServices,
) *GetScheduleUseCase {
	return &GetScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute retrieves schedule details from the provider
func (uc *GetScheduleUseCase) Execute(ctx context.Context, req *schedulerpb.GetScheduleRequest) (*schedulerpb.GetScheduleResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.GetScheduleResponse{
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

	log.Printf("📋 Getting schedule details: %s", scheduleID)

	response, err := uc.services.Provider.GetSchedule(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to get schedule: %v", err)
		return &schedulerpb.GetScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "GET_SCHEDULE_FAILED",
				Message: fmt.Sprintf("Failed to get schedule: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("✅ Retrieved schedule: %s (%s)", response.Data[0].Name, response.Data[0].Status.String())
	}

	return response, nil
}
