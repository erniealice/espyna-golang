package scheduler

import (
	"context"
	"log"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

// CheckHealthRepositories groups all repository dependencies
type CheckHealthRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// CheckHealthServices groups all service dependencies
type CheckHealthServices struct {
	Provider ports.SchedulerProvider
}

// CheckHealthUseCase handles checking scheduler provider health
type CheckHealthUseCase struct {
	repositories CheckHealthRepositories
	services     CheckHealthServices
}

// NewCheckHealthUseCase creates a new CheckHealthUseCase
func NewCheckHealthUseCase(
	repositories CheckHealthRepositories,
	services CheckHealthServices,
) *CheckHealthUseCase {
	return &CheckHealthUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute checks the health of the scheduler provider
func (uc *CheckHealthUseCase) Execute(ctx context.Context, req *schedulerpb.CheckSchedulerHealthRequest) (*schedulerpb.CheckSchedulerHealthResponse, error) {
	startTime := time.Now()

	if uc.services.Provider == nil {
		return &schedulerpb.CheckSchedulerHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_NOT_CONFIGURED",
				Message: "Scheduler provider is not configured",
			},
		}, nil
	}

	if !uc.services.Provider.IsEnabled() {
		return &schedulerpb.CheckSchedulerHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_DISABLED",
				Message: "Scheduler provider is disabled",
			},
		}, nil
	}

	log.Printf("🏥 Checking scheduler provider health: %s", uc.services.Provider.Name())

	err := uc.services.Provider.IsHealthy(ctx)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Printf("❌ Scheduler provider unhealthy: %v", err)
		return &schedulerpb.CheckSchedulerHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNHEALTHY",
				Message: err.Error(),
			},
		}, nil
	}

	log.Printf("✅ Scheduler provider healthy (latency: %dms)", latencyMs)

	return &schedulerpb.CheckSchedulerHealthResponse{
		Success: true,
		Data: []*schedulerpb.SchedulerHealthStatus{
			{
				IsHealthy: true,
				HealthStatus: &schedulerpb.SchedulerProviderHealthStatus{
					IsHealthy: true,
					Message:   "Scheduler provider is healthy",
					LatencyMs: latencyMs,
					LastCheck: timestamppb.Now(),
				},
			},
		},
	}, nil
}
