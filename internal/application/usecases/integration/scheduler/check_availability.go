package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

// CheckAvailabilityRepositories groups all repository dependencies
type CheckAvailabilityRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// CheckAvailabilityServices groups all service dependencies
type CheckAvailabilityServices struct {
	Provider ports.SchedulerProvider
}

// CheckAvailabilityUseCase handles checking available time slots
type CheckAvailabilityUseCase struct {
	repositories CheckAvailabilityRepositories
	services     CheckAvailabilityServices
}

// NewCheckAvailabilityUseCase creates a new CheckAvailabilityUseCase
func NewCheckAvailabilityUseCase(
	repositories CheckAvailabilityRepositories,
	services CheckAvailabilityServices,
) *CheckAvailabilityUseCase {
	return &CheckAvailabilityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute checks available time slots for the given event type and date range
func (uc *CheckAvailabilityUseCase) Execute(ctx context.Context, req *schedulerpb.CheckAvailabilityRequest) (*schedulerpb.CheckAvailabilityResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("🔍 Checking availability for event type: %s", req.Data.EventTypeId)
	log.Printf("   Date range: %s %s - %s %s", req.Data.StartDate, req.Data.StartTime, req.Data.EndDate, req.Data.EndTime)
	if req.Data.Timezone != "" {
		log.Printf("   Timezone: %s", req.Data.Timezone)
	}

	response, err := uc.services.Provider.CheckAvailability(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to check availability: %v", err)
		return &schedulerpb.CheckAvailabilityResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "AVAILABILITY_CHECK_FAILED",
				Message: fmt.Sprintf("Failed to check availability: %v", err),
			},
		}, nil
	}

	if response.Success {
		log.Printf("✅ Found %d available time slots", len(response.Data))
	}

	return response, nil
}
