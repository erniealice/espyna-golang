package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

// CreateScheduleRepositories groups all repository dependencies
type CreateScheduleRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// CreateScheduleServices groups all service dependencies
type CreateScheduleServices struct {
	Provider ports.SchedulerProvider
}

// CreateScheduleUseCase handles creating scheduled events
type CreateScheduleUseCase struct {
	repositories CreateScheduleRepositories
	services     CreateScheduleServices
}

// NewCreateScheduleUseCase creates a new CreateScheduleUseCase
func NewCreateScheduleUseCase(
	repositories CreateScheduleRepositories,
	services CreateScheduleServices,
) *CreateScheduleUseCase {
	return &CreateScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute creates a new scheduled event with the scheduler provider
func (uc *CreateScheduleUseCase) Execute(ctx context.Context, req *schedulerpb.CreateScheduleRequest) (*schedulerpb.CreateScheduleResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.CreateScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.CreateScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📅 Creating schedule for event type: %s", req.Data.EventTypeId)
	if req.Data.Invitee != nil {
		log.Printf("   Invitee: %s (%s)", req.Data.Invitee.Name, req.Data.Invitee.Email)
	}
	log.Printf("   Date/Time: %s %s - %s %s", req.Data.StartDate, req.Data.StartTime, req.Data.EndDate, req.Data.EndTime)

	response, err := uc.services.Provider.CreateSchedule(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to create schedule: %v", err)
		return &schedulerpb.CreateScheduleResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "CREATE_FAILED",
				Message: fmt.Sprintf("Failed to create schedule: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("✅ Schedule created: %s", response.Data[0].ProviderScheduleId)
		log.Printf("   Name: %s", response.Data[0].Name)
	}

	return response, nil
}
