package payment

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

// CreateCheckoutRepositories groups all repository dependencies
type CreateCheckoutRepositories struct {
	// No repositories needed for external payment provider integration
}

// CreateCheckoutServices groups all service dependencies
type CreateCheckoutServices struct {
	Provider ports.PaymentProvider
}

// CreateCheckoutUseCase handles creating checkout sessions
type CreateCheckoutUseCase struct {
	repositories CreateCheckoutRepositories
	services     CreateCheckoutServices
}

// NewCreateCheckoutUseCase creates a new CreateCheckoutUseCase
func NewCreateCheckoutUseCase(
	repositories CreateCheckoutRepositories,
	services CreateCheckoutServices,
) *CreateCheckoutUseCase {
	return &CreateCheckoutUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute creates a new checkout session with the payment provider
func (uc *CreateCheckoutUseCase) Execute(ctx context.Context, req *paymentpb.CreateCheckoutSessionRequest) (*paymentpb.CreateCheckoutSessionResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Payment provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📦 Creating checkout session for payment: %s", req.Data.PaymentId)

	response, err := uc.services.Provider.CreateCheckoutSession(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to create checkout session: %v", err)
		return &paymentpb.CreateCheckoutSessionResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "CHECKOUT_FAILED",
				Message: fmt.Sprintf("Failed to create checkout: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("✅ Checkout session created: %s", response.Data[0].Id)
		log.Printf("   Checkout URL: %s", response.Data[0].CheckoutUrl)
	}

	return response, nil
}
