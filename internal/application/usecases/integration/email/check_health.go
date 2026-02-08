package email

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// CheckHealthRepositories groups all repository dependencies
type CheckHealthRepositories struct {
	// No repositories needed for health checks
}

// CheckHealthServices groups all service dependencies
type CheckHealthServices struct {
	Provider ports.EmailProvider
}

// CheckHealthUseCase handles email provider health checks
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

// Execute checks the health of the email provider
func (uc *CheckHealthUseCase) Execute(ctx context.Context, req *emailpb.CheckHealthRequest) (*emailpb.CheckHealthResponse, error) {
	if uc.services.Provider == nil {
		return &emailpb.CheckHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_NOT_CONFIGURED",
				Message: "Email provider is not configured",
			},
		}, nil
	}

	err := uc.services.Provider.IsHealthy(ctx)
	if err != nil {
		return &emailpb.CheckHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNHEALTHY",
				Message: fmt.Sprintf("Provider unhealthy: %v", err),
			},
		}, nil
	}

	return &emailpb.CheckHealthResponse{
		Success: true,
		Data: []*emailpb.EmailHealthStatus{
			{
				IsHealthy:     true,
				StatusMessage: "Provider is healthy",
			},
		},
	}, nil
}
