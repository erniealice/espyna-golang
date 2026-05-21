package payment

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

// GetPaymentStatusRepositories groups all repository dependencies
type GetPaymentStatusRepositories struct {
	// No repositories needed for external payment provider integration
}

// GetPaymentStatusServices groups all service dependencies
type GetPaymentStatusServices struct {
	Provider ports.PaymentProvider
}

// GetPaymentStatusUseCase handles retrieving payment status
type GetPaymentStatusUseCase struct {
	repositories GetPaymentStatusRepositories
	services     GetPaymentStatusServices
}

// NewGetPaymentStatusUseCase creates a new GetPaymentStatusUseCase
func NewGetPaymentStatusUseCase(
	repositories GetPaymentStatusRepositories,
	services GetPaymentStatusServices,
) *GetPaymentStatusUseCase {
	return &GetPaymentStatusUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute retrieves the status of a payment
func (uc *GetPaymentStatusUseCase) Execute(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &paymentpb.GetPaymentStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Payment provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &paymentpb.GetPaymentStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📊 Getting payment status for: %s", req.Data.PaymentId)

	response, err := uc.services.Provider.GetPaymentStatus(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to get payment status: %v", err)
		return &paymentpb.GetPaymentStatusResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "STATUS_CHECK_FAILED",
				Message: fmt.Sprintf("Failed to get status: %v", err),
			},
		}, nil
	}

	return response, nil
}
