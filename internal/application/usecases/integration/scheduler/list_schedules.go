package scheduler

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	schedulerpb "leapfor.xyz/esqyma/golang/v1/integration/scheduler"
)

// ListSchedulesRepositories groups all repository dependencies
type ListSchedulesRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// ListSchedulesServices groups all service dependencies
type ListSchedulesServices struct {
	Provider ports.SchedulerProvider
}

// ListSchedulesUseCase handles listing scheduled events
type ListSchedulesUseCase struct {
	repositories ListSchedulesRepositories
	services     ListSchedulesServices
}

// NewListSchedulesUseCase creates a new ListSchedulesUseCase
func NewListSchedulesUseCase(
	repositories ListSchedulesRepositories,
	services ListSchedulesServices,
) *ListSchedulesUseCase {
	return &ListSchedulesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute lists scheduled events from the provider
func (uc *ListSchedulesUseCase) Execute(ctx context.Context, req *schedulerpb.ListSchedulesRequest) (*schedulerpb.ListSchedulesResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📃 Listing schedules")
	if req.Data.FromDate != "" && req.Data.ToDate != "" {
		log.Printf("   Date range: %s to %s", req.Data.FromDate, req.Data.ToDate)
	}
	if req.Data.Status != "" {
		log.Printf("   Status filter: %s", req.Data.Status)
	}

	response, err := uc.services.Provider.ListSchedules(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to list schedules: %v", err)
		return &schedulerpb.ListSchedulesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "LIST_SCHEDULES_FAILED",
				Message: fmt.Sprintf("Failed to list schedules: %v", err),
			},
		}, nil
	}

	log.Printf("✅ Found %d schedules", len(response.Data))

	return response, nil
}
