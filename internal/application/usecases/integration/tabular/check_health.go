package tabular

import (
	"context"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	tabularpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/tabular"
)

// CheckHealthRepositories groups all repository dependencies
type CheckHealthRepositories struct {
	// No repositories needed for health check
}

// CheckHealthServices groups all service dependencies
type CheckHealthServices struct {
	Provider integration.TabularSourceProvider
}

// CheckHealthUseCase handles tabular provider health checks
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

// Execute checks the health of the tabular provider
func (uc *CheckHealthUseCase) Execute(ctx context.Context, req *tabularpb.CheckHealthRequest) (*tabularpb.CheckHealthResponse, error) {
	if uc.services.Provider == nil {
		return &tabularpb.CheckHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not configured",
			},
		}, nil
	}

	if !uc.services.Provider.IsEnabled() {
		return &tabularpb.CheckHealthResponse{
			Success: true,
			Data: []*tabularpb.HealthStatus{
				{
					IsHealthy: false,
					Message:   "Provider is disabled",
				},
			},
		}, nil
	}

	log.Printf("Checking health for tabular provider: %s", uc.services.Provider.Name())

	// Use the structured health check if deep check is requested
	if req != nil && req.Data != nil && req.Data.DeepCheck {
		response, err := uc.services.Provider.CheckHealth(ctx, req)
		if err != nil {
			log.Printf("Tabular provider health check failed: %v", err)
			return &tabularpb.CheckHealthResponse{
				Success: true,
				Data: []*tabularpb.HealthStatus{
					{
						IsHealthy: false,
						Message:   err.Error(),
					},
				},
			}, nil
		}
		return response, nil
	}

	// Use the simple health check
	err := uc.services.Provider.IsHealthy(ctx)
	if err != nil {
		log.Printf("Tabular provider health check failed: %v", err)
		return &tabularpb.CheckHealthResponse{
			Success: true,
			Data: []*tabularpb.HealthStatus{
				{
					IsHealthy: false,
					Message:   err.Error(),
				},
			},
		}, nil
	}

	log.Printf("Tabular provider %s is healthy", uc.services.Provider.Name())

	return &tabularpb.CheckHealthResponse{
		Success: true,
		Data: []*tabularpb.HealthStatus{
			{
				IsHealthy: true,
				Message:   "Provider is healthy",
			},
		},
	}, nil
}
