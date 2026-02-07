package payment

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	paymentpb "leapfor.xyz/esqyma/golang/v1/integration/payment"
)

// CheckHealthRepositories groups all repository dependencies
type CheckHealthRepositories struct {
	// No repositories needed for health checks
}

// CheckHealthServices groups all service dependencies
type CheckHealthServices struct {
	Provider ports.PaymentProvider
}

// CheckHealthUseCase handles payment provider health checks
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

// Execute checks the health of the payment provider
func (uc *CheckHealthUseCase) Execute(ctx context.Context, req *paymentpb.CheckHealthRequest) (*paymentpb.CheckHealthResponse, error) {
	if uc.services.Provider == nil {
		return &paymentpb.CheckHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_NOT_CONFIGURED",
				Message: "Payment provider is not configured",
			},
		}, nil
	}

	err := uc.services.Provider.IsHealthy(ctx)
	isHealthy := err == nil

	if err != nil {
		return &paymentpb.CheckHealthResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNHEALTHY",
				Message: fmt.Sprintf("Provider unhealthy: %v", err),
			},
		}, nil
	}

	return &paymentpb.CheckHealthResponse{
		Success: true,
		Data: &paymentpb.HealthStatus{
			IsHealthy: isHealthy,
			HealthStatus: &paymentpb.ProviderHealthStatus{
				ProviderId:    uc.services.Provider.Name(),
				IsHealthy:     isHealthy,
				StatusMessage: "Provider is healthy",
			},
		},
	}, nil
}
