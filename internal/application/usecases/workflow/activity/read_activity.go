package activity

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
)

// ReadActivityRepositories groups all repository dependencies
type ReadActivityRepositories struct {
	Activity activitypb.ActivityDomainServiceServer // Primary entity repository
}

// ReadActivityServices groups all business service dependencies
type ReadActivityServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadActivityUseCase handles the business logic for reading activities
type ReadActivityUseCase struct {
	repositories ReadActivityRepositories
	services     ReadActivityServices
}

// NewReadActivityUseCase creates use case with grouped dependencies
func NewReadActivityUseCase(
	repositories ReadActivityRepositories,
	services ReadActivityServices,
) *ReadActivityUseCase {
	return &ReadActivityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadActivityUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadActivityUseCase with grouped parameters instead
func NewReadActivityUseCaseUngrouped(activityRepo activitypb.ActivityDomainServiceServer) *ReadActivityUseCase {
	repositories := ReadActivityRepositories{
		Activity: activityRepo,
	}

	services := ReadActivityServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadActivityUseCase(repositories, services)
}

// Execute performs the read activity operation
func (uc *ReadActivityUseCase) Execute(ctx context.Context, req *activitypb.ReadActivityRequest) (*activitypb.ReadActivityResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.request_required", "Request is required for activities [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Apply business logic defaults
	enrichedRequest := uc.applyBusinessLogic(req)

	// Use transaction service if available (for consistent reads)
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedRequest)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedRequest)
}

// executeWithTransaction executes activity reading within a transaction
func (uc *ReadActivityUseCase) executeWithTransaction(ctx context.Context, req *activitypb.ReadActivityRequest) (*activitypb.ReadActivityResponse, error) {
	var result *activitypb.ReadActivityResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity.errors.read_failed", "Activity read failed [DEFAULT]")
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

// executeCore contains the core business logic for reading an activity
func (uc *ReadActivityUseCase) executeCore(ctx context.Context, req *activitypb.ReadActivityRequest) (*activitypb.ReadActivityResponse, error) {
	// Delegate to repository
	return uc.repositories.Activity.ReadActivity(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *ReadActivityUseCase) applyBusinessLogic(req *activitypb.ReadActivityRequest) *activitypb.ReadActivityRequest {
	// Create a copy to avoid modifying the original request
	enrichedReq := &activitypb.ReadActivityRequest{}

	if req.Data != nil {
		// Use pointer assignment to avoid copying mutexes
		enrichedReq.Data = req.Data
	} else {
		enrichedReq.Data = &activitypb.Activity{}
	}

	// Business logic: Ensure ID is provided if activity data is provided
	if enrichedReq.Data.Id == "" {
		// No enrichment needed for read - ID must be provided by caller
	}

	return enrichedReq
}

// validateBusinessRules enforces business constraints
func (uc *ReadActivityUseCase) validateBusinessRules(ctx context.Context, req *activitypb.ReadActivityRequest) error {
	// Business rule: Request data validation
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.data_required", "Activity data is required for read operations [DEFAULT]"))
	}

	// Business rule: Activity ID is required
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.id_required", "Activity ID is required for read operations [DEFAULT]"))
	}

	// Business rule: Activity ID format validation
	if err := uc.validateActivityID(req.Data.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.id_invalid", "Activity ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateActivityID validates activity ID format
func (uc *ReadActivityUseCase) validateActivityID(id string) error {
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
