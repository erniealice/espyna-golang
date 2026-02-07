package payment

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	paymentpb "leapfor.xyz/esqyma/golang/v1/integration/payment"
)

// ProcessWebhookRepositories groups all repository dependencies
type ProcessWebhookRepositories struct {
	// No repositories needed for external payment provider integration
}

// ProcessWebhookServices groups all service dependencies
type ProcessWebhookServices struct {
	Provider ports.PaymentProvider
}

// ProcessWebhookUseCase handles processing payment webhooks
type ProcessWebhookUseCase struct {
	repositories ProcessWebhookRepositories
	services     ProcessWebhookServices
}

// NewProcessWebhookUseCase creates a new ProcessWebhookUseCase
func NewProcessWebhookUseCase(
	repositories ProcessWebhookRepositories,
	services ProcessWebhookServices,
) *ProcessWebhookUseCase {
	return &ProcessWebhookUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute processes an incoming webhook from the payment provider
func (uc *ProcessWebhookUseCase) Execute(ctx context.Context, req *paymentpb.ProcessWebhookRequest) (*paymentpb.ProcessWebhookResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Payment provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("🔔 Processing payment webhook from provider: %s", req.Data.ProviderId)

	response, err := uc.services.Provider.ProcessWebhook(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to process webhook: %v", err)
		return &paymentpb.ProcessWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "WEBHOOK_PROCESSING_FAILED",
				Message: fmt.Sprintf("Failed to process webhook: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("✅ Webhook processed successfully")
		log.Printf("   Payment ID: %s", response.Data[0].PaymentId)
		log.Printf("   Status: %s", response.Data[0].Status.String())
		log.Printf("   Action: %s", response.Data[0].Action)
	}

	return response, nil
}
