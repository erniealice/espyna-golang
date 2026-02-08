package payment

import (
	"context"
	"fmt"
	"log"

	integrationPorts "github.com/erniealice/espyna-golang/internal/application/ports/integration"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/payment"
)

// LogWebhookRepositories groups all repository dependencies
type LogWebhookRepositories struct {
	IntegrationPayment integrationPorts.IntegrationPaymentRepository
}

// LogWebhookServices groups all service dependencies
type LogWebhookServices struct {
	// No external services needed for logging
}

// LogWebhookUseCase handles logging parsed webhook data to the integration_payment collection
type LogWebhookUseCase struct {
	repositories LogWebhookRepositories
	services     LogWebhookServices
}

// NewLogWebhookUseCase creates a new LogWebhookUseCase
func NewLogWebhookUseCase(
	repositories LogWebhookRepositories,
	services LogWebhookServices,
) *LogWebhookUseCase {
	return &LogWebhookUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute logs parsed webhook data to the integration_payment collection
func (uc *LogWebhookUseCase) Execute(ctx context.Context, req *paymentpb.LogWebhookRequest) (*paymentpb.LogWebhookResponse, error) {
	if uc.repositories.IntegrationPayment == nil {
		return &paymentpb.LogWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "REPOSITORY_UNAVAILABLE",
				Message: "Integration payment repository is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &paymentpb.LogWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("📝 Logging payment webhook: provider=%s, payment_id=%s, status=%s",
		req.Data.ProviderId, req.Data.PaymentId, req.Data.PaymentStatus)

	response, err := uc.repositories.IntegrationPayment.LogWebhook(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to log webhook: %v", err)
		return &paymentpb.LogWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "LOG_WEBHOOK_FAILED",
				Message: fmt.Sprintf("Failed to log webhook: %v", err),
			},
		}, nil
	}

	log.Printf("✅ Webhook logged successfully: id=%s", response.Id)

	return response, nil
}
