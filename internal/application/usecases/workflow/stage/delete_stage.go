package stage

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
)

// DeleteStageRepositories groups all repository dependencies
type DeleteStageRepositories struct {
	Stage stagepb.StageDomainServiceServer // Primary entity repository
}

// DeleteStageServices groups all business service dependencies
type DeleteStageServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteStageUseCase handles the business logic for deleting stages
type DeleteStageUseCase struct {
	repositories DeleteStageRepositories
	services     DeleteStageServices
}

// NewDeleteStageUseCase creates use case with grouped dependencies
func NewDeleteStageUseCase(
	repositories DeleteStageRepositories,
	services DeleteStageServices,
) *DeleteStageUseCase {
	return &DeleteStageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteStageUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteStageUseCase with grouped parameters instead
func NewDeleteStageUseCaseUngrouped(stageRepo stagepb.StageDomainServiceServer) *DeleteStageUseCase {
	repositories := DeleteStageRepositories{
		Stage: stageRepo,
	}

	services := DeleteStageServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteStageUseCase(repositories, services)
}

// Execute performs the delete stage operation
func (uc *DeleteStageUseCase) Execute(ctx context.Context, req *stagepb.DeleteStageRequest) (*stagepb.DeleteStageResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.request_required", "Request is required for stages [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Existence validation
	if err := uc.validateStageExists(ctx, req.Data.Id); err != nil {
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

// executeWithTransaction executes stage deletion within a transaction
func (uc *DeleteStageUseCase) executeWithTransaction(ctx context.Context, req *stagepb.DeleteStageRequest) (*stagepb.DeleteStageResponse, error) {
	var result *stagepb.DeleteStageResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage.errors.delete_failed", "Stage deletion failed [DEFAULT]")
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

// executeCore contains the core business logic for deleting a stage
func (uc *DeleteStageUseCase) executeCore(ctx context.Context, req *stagepb.DeleteStageRequest) (*stagepb.DeleteStageResponse, error) {
	// Delegate to repository
	return uc.repositories.Stage.DeleteStage(ctx, req)
}

// applyBusinessLogic applies business rules and returns enriched request
func (uc *DeleteStageUseCase) applyBusinessLogic(req *stagepb.DeleteStageRequest) *stagepb.DeleteStageRequest {
	// Create a copy to avoid modifying the original request
	enrichedReq := &stagepb.DeleteStageRequest{}

	if req.Data != nil {
		// Use pointer assignment to avoid copying mutexes
		enrichedReq.Data = req.Data
	} else {
		enrichedReq.Data = &stagepb.Stage{}
	}

	// Business logic: Ensure stage is marked as inactive before deletion
	// This is typically handled at the repository level, but we enforce it here
	enrichedReq.Data.Active = false

	return enrichedReq
}

// validateStageExists validates that the stage exists and can be deleted
func (uc *DeleteStageUseCase) validateStageExists(ctx context.Context, stageID string) error {
	// Check stage exists
	stageReadReq := &stagepb.ReadStageRequest{
		Data: &stagepb.Stage{
			Id: stageID,
		},
	}
	stageRes, err := uc.repositories.Stage.ReadStage(ctx, stageReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_not_found", "Stage not found [DEFAULT]"))
	}
	if stageRes == nil || len(stageRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_not_found", "Stage not found [DEFAULT]"))
	}

	existingStage := stageRes.Data[0]

	// Business rule: Cannot delete inactive stage
	if !existingStage.Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_already_inactive", "Stage is already inactive [DEFAULT]"))
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *DeleteStageUseCase) validateBusinessRules(ctx context.Context, req *stagepb.DeleteStageRequest) error {
	// Business rule: Request data validation
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.data_required", "Stage data is required for delete operations [DEFAULT]"))
	}

	// Business rule: Stage ID is required
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.id_required", "Stage ID is required for delete operations [DEFAULT]"))
	}

	// Business rule: Stage ID format validation
	if err := uc.validateStageID(req.Data.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.id_invalid", "Stage ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateStageID validates stage ID format
func (uc *DeleteStageUseCase) validateStageID(id string) error {
	// Basic validation: reasonable length
	if len(id) < 3 {
		return errors.New("stage ID too short")
	}

	if len(id) > 100 {
		return errors.New("stage ID too long")
	}

	// Additional validation can be added based on ID format
	// For now, we'll keep it simple

	return nil
}
