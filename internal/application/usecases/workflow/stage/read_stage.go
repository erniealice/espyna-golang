package stage

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
)

// ReadStageRepositories groups all repository dependencies
type ReadStageRepositories struct {
	Stage stagepb.StageDomainServiceServer // Primary entity repository
}

// ReadStageServices groups all business service dependencies
type ReadStageServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadStageUseCase handles the business logic for reading stages
type ReadStageUseCase struct {
	repositories ReadStageRepositories
	services     ReadStageServices
}

// NewReadStageUseCase creates use case with grouped dependencies
func NewReadStageUseCase(
	repositories ReadStageRepositories,
	services ReadStageServices,
) *ReadStageUseCase {
	return &ReadStageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadStageUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadStageUseCase with grouped parameters instead
func NewReadStageUseCaseUngrouped(stageRepo stagepb.StageDomainServiceServer) *ReadStageUseCase {
	repositories := ReadStageRepositories{
		Stage: stageRepo,
	}

	services := ReadStageServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadStageUseCase(repositories, services)
}

// Execute performs the read stage operation
func (uc *ReadStageUseCase) Execute(ctx context.Context, req *stagepb.ReadStageRequest) (*stagepb.ReadStageResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"stage", ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.request_required", "Request is required for stages [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Apply business logic defaults (no enrichment needed for read operations)
	// enrichedRequest := uc.applyBusinessLogic(req)
	enrichedRequest := req

	// Use transaction service if available (for consistent reads)
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedRequest)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedRequest)
}

// executeWithTransaction executes stage reading within a transaction
func (uc *ReadStageUseCase) executeWithTransaction(ctx context.Context, req *stagepb.ReadStageRequest) (*stagepb.ReadStageResponse, error) {
	var result *stagepb.ReadStageResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage.errors.read_failed", "Stage read failed [DEFAULT]")
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

// executeCore contains the core business logic for reading a stage
func (uc *ReadStageUseCase) executeCore(ctx context.Context, req *stagepb.ReadStageRequest) (*stagepb.ReadStageResponse, error) {
	// Delegate to repository
	return uc.repositories.Stage.ReadStage(ctx, req)
}

// validateBusinessRules enforces business constraints
func (uc *ReadStageUseCase) validateBusinessRules(ctx context.Context, req *stagepb.ReadStageRequest) error {
	// Business rule: Request data validation
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.data_required", "Stage data is required for read operations [DEFAULT]"))
	}

	// Business rule: Stage ID is required
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.id_required", "Stage ID is required for read operations [DEFAULT]"))
	}

	// Business rule: Stage ID format validation
	if err := uc.validateStageID(req.Data.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.id_invalid", "Stage ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateStageID validates stage ID format
func (uc *ReadStageUseCase) validateStageID(id string) error {
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
