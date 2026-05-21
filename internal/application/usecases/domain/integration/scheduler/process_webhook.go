package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	schedulerpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/scheduler"
)

// ProcessWebhookRepositories groups all repository dependencies
type ProcessWebhookRepositories struct {
	// No repositories needed for external scheduler provider integration
}

// ProcessWebhookServices groups all service dependencies
type ProcessWebhookServices struct {
	Provider ports.SchedulerProvider
}

// ProcessWebhookUseCase handles processing scheduler webhooks
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

// Execute processes an incoming scheduler webhook
func (uc *ProcessWebhookUseCase) Execute(ctx context.Context, req *schedulerpb.ProcessSchedulerWebhookRequest) (*schedulerpb.ProcessSchedulerWebhookResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &schedulerpb.ProcessSchedulerWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Scheduler provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &schedulerpb.ProcessSchedulerWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	log.Printf("🔔 Processing scheduler webhook from provider: %s", req.Data.ProviderId)
	log.Printf("   Content-Type: %s", req.Data.ContentType)
	log.Printf("   Payload size: %d bytes", len(req.Data.Payload))

	response, err := uc.services.Provider.ProcessWebhook(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to process webhook: %v", err)
		return &schedulerpb.ProcessSchedulerWebhookResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "WEBHOOK_PROCESSING_FAILED",
				Message: fmt.Sprintf("Failed to process webhook: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("✅ Webhook processed successfully")
		log.Printf("   Event type: %s", response.Data[0].EventType)
		log.Printf("   Action: %s", response.Data[0].Action)
		if response.Data[0].Schedule != nil {
			log.Printf("   Schedule ID: %s", response.Data[0].Schedule.ProviderScheduleId)
		}
		if response.Data[0].IsReschedule {
			log.Printf("   Reschedule from: %s", response.Data[0].OldScheduleId)
		}
	}

	return response, nil
}

// ToWebhookResult converts the protobuf response to a convenience type
func ToWebhookResult(response *schedulerpb.ProcessSchedulerWebhookResponse, err error) *ports.ScheduleWebhookResult {
	if err != nil {
		return &ports.ScheduleWebhookResult{
			Success: false,
			Error:   err,
		}
	}

	if len(response.Data) == 0 {
		return &ports.ScheduleWebhookResult{
			Success: response.Success,
			Error:   nil,
		}
	}

	return &ports.ScheduleWebhookResult{
		Success:       response.Success,
		EventType:     response.Data[0].EventType,
		Schedule:      response.Data[0].Schedule,
		Action:        response.Data[0].Action,
		IsReschedule:  response.Data[0].IsReschedule,
		OldScheduleID: response.Data[0].OldScheduleId,
		Error:         nil,
	}
}
