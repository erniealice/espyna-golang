package stage_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
)

// DeleteStageTemplateRepositories groups all repository dependencies
type DeleteStageTemplateRepositories struct {
	StageTemplate stageTemplatepb.StageTemplateDomainServiceServer // Primary entity repository
}

// DeleteStageTemplateServices groups all business service dependencies
type DeleteStageTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteStageTemplateUseCase handles the business logic for deleting stage templates
type DeleteStageTemplateUseCase struct {
	repositories DeleteStageTemplateRepositories
	services     DeleteStageTemplateServices
}

// NewDeleteStageTemplateUseCase creates use case with grouped dependencies
func NewDeleteStageTemplateUseCase(
	repositories DeleteStageTemplateRepositories,
	services DeleteStageTemplateServices,
) *DeleteStageTemplateUseCase {
	return &DeleteStageTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteStageTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteStageTemplateUseCase with grouped parameters instead
func NewDeleteStageTemplateUseCaseUngrouped(stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer) *DeleteStageTemplateUseCase {
	repositories := DeleteStageTemplateRepositories{
		StageTemplate: stageTemplateRepo,
	}

	services := DeleteStageTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteStageTemplateUseCase(repositories, services)
}

// Execute performs the delete stage template operation
func (uc *DeleteStageTemplateUseCase) Execute(ctx context.Context, req *stageTemplatepb.DeleteStageTemplateRequest) (*stageTemplatepb.DeleteStageTemplateResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.request_required", "Request is required for stage templates [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req.Data)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req.Data)
}

// executeWithTransaction executes stage template deletion within a transaction
func (uc *DeleteStageTemplateUseCase) executeWithTransaction(ctx context.Context, stageTemplate *stageTemplatepb.StageTemplate) (*stageTemplatepb.DeleteStageTemplateResponse, error) {
	var result *stageTemplatepb.DeleteStageTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, stageTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage_template.errors.deletion_failed", "Stage template deletion failed [DEFAULT]")
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

// executeCore contains the core business logic for deleting a stage template
func (uc *DeleteStageTemplateUseCase) executeCore(ctx context.Context, stageTemplate *stageTemplatepb.StageTemplate) (*stageTemplatepb.DeleteStageTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.StageTemplate.DeleteStageTemplate(ctx, &stageTemplatepb.DeleteStageTemplateRequest{
		Data: stageTemplate,
	})
}

// validateBusinessRules enforces business constraints
func (uc *DeleteStageTemplateUseCase) validateBusinessRules(ctx context.Context, stageTemplate *stageTemplatepb.StageTemplate) error {
	// Business rule: Required data validation
	if stageTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.data_required", "Stage template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for deleting
	if stageTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.id_required", "Stage template ID is required for delete operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateStageTemplateID(stageTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.id_invalid", "Stage template ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateStageTemplateID validates stage template ID format
func (uc *DeleteStageTemplateUseCase) validateStageTemplateID(id string) error {
	// Basic validation: non-empty, reasonable length, valid characters
	if id == "" {
		return errors.New("stage template ID cannot be empty")
	}

	if len(id) < 3 {
		return errors.New("stage template ID must be at least 3 characters long")
	}

	if len(id) > 100 {
		return errors.New("stage template ID cannot exceed 100 characters")
	}

	// Allow alphanumeric characters, hyphens, and underscores
	idRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !idRegex.MatchString(id) {
		return errors.New("stage template ID contains invalid characters")
	}

	return nil
}
