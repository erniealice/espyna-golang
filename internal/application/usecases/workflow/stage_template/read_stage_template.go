package stage_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
)

// ReadStageTemplateRepositories groups all repository dependencies
type ReadStageTemplateRepositories struct {
	StageTemplate stageTemplatepb.StageTemplateDomainServiceServer // Primary entity repository
}

// ReadStageTemplateServices groups all business service dependencies
type ReadStageTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadStageTemplateUseCase handles the business logic for reading stage templates
type ReadStageTemplateUseCase struct {
	repositories ReadStageTemplateRepositories
	services     ReadStageTemplateServices
}

// NewReadStageTemplateUseCase creates use case with grouped dependencies
func NewReadStageTemplateUseCase(
	repositories ReadStageTemplateRepositories,
	services ReadStageTemplateServices,
) *ReadStageTemplateUseCase {
	return &ReadStageTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadStageTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadStageTemplateUseCase with grouped parameters instead
func NewReadStageTemplateUseCaseUngrouped(stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer) *ReadStageTemplateUseCase {
	repositories := ReadStageTemplateRepositories{
		StageTemplate: stageTemplateRepo,
	}

	services := ReadStageTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadStageTemplateUseCase(repositories, services)
}

// Execute performs the read stage template operation
func (uc *ReadStageTemplateUseCase) Execute(ctx context.Context, req *stageTemplatepb.ReadStageTemplateRequest) (*stageTemplatepb.ReadStageTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"stage_template", ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.request_required", "Request is required for stage templates [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Use transaction service if available (for consistent reads)
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req.Data)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req.Data)
}

// executeWithTransaction executes stage template read within a transaction
func (uc *ReadStageTemplateUseCase) executeWithTransaction(ctx context.Context, stageTemplate *stageTemplatepb.StageTemplate) (*stageTemplatepb.ReadStageTemplateResponse, error) {
	var result *stageTemplatepb.ReadStageTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, stageTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage_template.errors.read_failed", "Stage template read failed [DEFAULT]")
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

// executeCore contains the core business logic for reading a stage template
func (uc *ReadStageTemplateUseCase) executeCore(ctx context.Context, stageTemplate *stageTemplatepb.StageTemplate) (*stageTemplatepb.ReadStageTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.StageTemplate.ReadStageTemplate(ctx, &stageTemplatepb.ReadStageTemplateRequest{
		Data: stageTemplate,
	})
}

// validateBusinessRules enforces business constraints
func (uc *ReadStageTemplateUseCase) validateBusinessRules(ctx context.Context, stageTemplate *stageTemplatepb.StageTemplate) error {
	// Business rule: Required data validation
	if stageTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.data_required", "Stage template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for reading
	if stageTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.id_required", "Stage template ID is required for read operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateStageTemplateID(stageTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.id_invalid", "Stage template ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateStageTemplateID validates stage template ID format
func (uc *ReadStageTemplateUseCase) validateStageTemplateID(id string) error {
	// Basic validation: non-empty, reasonable length, valid characters
	if strings.TrimSpace(id) == "" {
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
