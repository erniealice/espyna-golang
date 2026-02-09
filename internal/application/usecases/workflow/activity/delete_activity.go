package activity

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
)

// DeleteActivityRepositories groups all repository dependencies
type DeleteActivityRepositories struct {
	Activity activitypb.ActivityDomainServiceServer // Primary entity repository
}

// DeleteActivityServices groups all business service dependencies
type DeleteActivityServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteActivityUseCase handles the business logic for deleting activities
type DeleteActivityUseCase struct {
	repositories DeleteActivityRepositories
	services     DeleteActivityServices
}

// NewDeleteActivityUseCase creates use case with grouped dependencies
func NewDeleteActivityUseCase(
	repositories DeleteActivityRepositories,
	services DeleteActivityServices,
) *DeleteActivityUseCase {
	return &DeleteActivityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteActivityUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteActivityUseCase with grouped parameters instead
func NewDeleteActivityUseCaseUngrouped(activityRepo activitypb.ActivityDomainServiceServer) *DeleteActivityUseCase {
	repositories := DeleteActivityRepositories{
		Activity: activityRepo,
	}

	services := DeleteActivityServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteActivityUseCase(repositories, services)
}

// Execute performs the delete activity operation
func (uc *DeleteActivityUseCase) Execute(ctx context.Context, req *activitypb.DeleteActivityRequest) (*activitypb.DeleteActivityResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"activity", ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.request_required", "Request is required for activities [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Existence validation
	if err := uc.validateActivityExists(ctx, req.Data.Id); err != nil {
		return nil, err
	}

	// Apply business logic
	enrichedRequest := uc.applyBusinessLogic(req)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedRequest)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedRequest)
}

// executeWithTransaction executes activity deletion within a transaction
func (uc *DeleteActivityUseCase) executeWithTransaction(ctx context.Context, req *activitypb.DeleteActivityRequest) (*activitypb.DeleteActivityResponse, error) {
	var result *activitypb.DeleteActivityResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity.errors.delete_failed", "Activity deletion failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for deleting an activity
func (uc *DeleteActivityUseCase) executeCore(ctx context.Context, req *activitypb.DeleteActivityRequest) (*activitypb.DeleteActivityResponse, error) {
	// Delegate to repository
	return uc.repositories.Activity.DeleteActivity(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *DeleteActivityUseCase) applyBusinessLogic(req *activitypb.DeleteActivityRequest) *activitypb.DeleteActivityRequest {
	// Create a copy to avoid modifying the original request
	enrichedReq := &activitypb.DeleteActivityRequest{}

	if req.Data != nil {
		// Use proto.Clone to properly copy protobuf messages
		enrichedReq.Data = req.Data
	} else {
		enrichedReq.Data = &activitypb.Activity{}
	}

	// Business logic: Ensure activity is marked as inactive before deletion
	// This is typically handled at the repository level, but we enforce it here
	enrichedReq.Data.Active = false

	return enrichedReq
}

// validateActivityExists validates that the activity exists and can be deleted
func (uc *DeleteActivityUseCase) validateActivityExists(ctx context.Context, activityID string) error {
	// Check activity exists
	activityReadReq := &activitypb.ReadActivityRequest{
		Data: &activitypb.Activity{
			Id: activityID,
		},
	}
	activityRes, err := uc.repositories.Activity.ReadActivity(ctx, activityReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_not_found", "Activity not found [DEFAULT]"))
	}
	if activityRes == nil || len(activityRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_not_found", "Activity not found [DEFAULT]"))
	}

	existingActivity := activityRes.Data[0]

	// Business rule: Cannot delete inactive activity
	if !existingActivity.Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_already_inactive", "Activity is already inactive [DEFAULT]"))
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *DeleteActivityUseCase) validateBusinessRules(ctx context.Context, req *activitypb.DeleteActivityRequest) error {
	// Business rule: Request data validation
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.data_required", "Activity data is required for delete operations [DEFAULT]"))
	}

	// Business rule: Activity ID is required
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.id_required", "Activity ID is required for delete operations [DEFAULT]"))
	}

	// Business rule: Activity ID format validation
	if err := uc.validateActivityID(req.Data.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.id_invalid", "Activity ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateActivityID validates activity ID format
func (uc *DeleteActivityUseCase) validateActivityID(id string) error {
	// Basic validation: reasonable length
	if len(id) < 3 {
		return errors.New("activity ID too short")
	}

	if len(id) > 100 {
		return errors.New("activity ID too long")
	}

	// Additional validation can be added based on ID format
	// For now, we'll keep it simple

	return nil
}
