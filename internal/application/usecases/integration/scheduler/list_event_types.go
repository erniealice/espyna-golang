package scheduler

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	schedulerpb "leapfor.xyz/esqyma/golang/v1/integration/scheduler"
)

// ListEventTypesRepositories groups all repository dependencies
type ListEventTypesRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// ListEventTypesServices groups all service dependencies
type ListEventTypesServices struct {
	Provider ports.SchedulerProvider
}

// ListEventTypesUseCase handles listing available event types
type ListEventTypesUseCase struct {
	repositories ListEventTypesRepositories
	services     ListEventTypesServices
}

// NewListEventTypesUseCase creates a new ListEventTypesUseCase
func NewListEventTypesUseCase(
	repositories ListEventTypesRepositories,
	services ListEventTypesServices,
) *ListEventTypesUseCase {
	return &ListEventTypesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute lists available event types from the provider
func (uc *ListEventTypesUseCase) Execute(ctx context.Context, req *schedulerpb.ListEventTypesRequest) (*schedulerpb.ListEventTypesResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📋 Listing event types")
	if req.Data.ActiveOnly {
		log.Printf("   Filter: active only")
	}

	response, err := uc.services.Provider.ListEventTypes(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to list event types: %v", err)
		return &schedulerpb.ListEventTypesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "LIST_EVENT_TYPES_FAILED",
				Message: fmt.Sprintf("Failed to list event types: %v", err),
			},
		}, nil
	}

	if response.Success {
		log.Printf("✅ Found %d event types", len(response.Data))
		for _, et := range response.Data {
			log.Printf("   - %s (%d min)", et.Name, et.DurationMinutes)
		}
	}

	return response, nil
}
