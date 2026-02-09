package activity_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
)

// DeleteActivityTemplateRepositories groups all repository dependencies
type DeleteActivityTemplateRepositories struct {
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Primary entity repository
}

// DeleteActivityTemplateServices groups all business service dependencies
type DeleteActivityTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteActivityTemplateUseCase handles the business logic for deleting activity templates
type DeleteActivityTemplateUseCase struct {
	repositories DeleteActivityTemplateRepositories
	services     DeleteActivityTemplateServices
}

// NewDeleteActivityTemplateUseCase creates use case with grouped dependencies
func NewDeleteActivityTemplateUseCase(
	repositories DeleteActivityTemplateRepositories,
	services DeleteActivityTemplateServices,
) *DeleteActivityTemplateUseCase {
	return &DeleteActivityTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteActivityTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteActivityTemplateUseCase with grouped parameters instead
func NewDeleteActivityTemplateUseCaseUngrouped(activityTemplateRepo activityTemplatepb.ActivityTemplateDomainServiceServer) *DeleteActivityTemplateUseCase {
	repositories := DeleteActivityTemplateRepositories{
		ActivityTemplate: activityTemplateRepo,
	}

	services := DeleteActivityTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteActivityTemplateUseCase(repositories, services)
}

// Execute performs the delete activity template operation
func (uc *DeleteActivityTemplateUseCase) Execute(ctx context.Context, req *activityTemplatepb.DeleteActivityTemplateRequest) (*activityTemplatepb.DeleteActivityTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"activity_template", ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.request_required", "Request is required for activity templates [DEFAULT]"))
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

// executeWithTransaction executes activity template deletion within a transaction
func (uc *DeleteActivityTemplateUseCase) executeWithTransaction(ctx context.Context, activityTemplate *activityTemplatepb.ActivityTemplate) (*activityTemplatepb.DeleteActivityTemplateResponse, error) {
	var result *activityTemplatepb.DeleteActivityTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, activityTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity_template.errors.deletion_failed", "Activity template deletion failed [DEFAULT]")
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

// executeCore contains the core business logic for deleting an activity template
func (uc *DeleteActivityTemplateUseCase) executeCore(ctx context.Context, activityTemplate *activityTemplatepb.ActivityTemplate) (*activityTemplatepb.DeleteActivityTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.ActivityTemplate.DeleteActivityTemplate(ctx, &activityTemplatepb.DeleteActivityTemplateRequest{
		Data: activityTemplate,
	})
}

// validateBusinessRules enforces business constraints
func (uc *DeleteActivityTemplateUseCase) validateBusinessRules(ctx context.Context, activityTemplate *activityTemplatepb.ActivityTemplate) error {
	// Business rule: Required data validation
	if activityTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.data_required", "Activity template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for deleting
	if activityTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.id_required", "Activity template ID is required for delete operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateActivityTemplateID(activityTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.id_invalid", "Activity template ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateActivityTemplateID validates activity template ID format
func (uc *DeleteActivityTemplateUseCase) validateActivityTemplateID(id string) error {
	// Basic validation: non-empty, reasonable length, valid characters
	if id == "" {
		return errors.New("activity template ID cannot be empty")
	}

	if len(id) < 3 {
		return errors.New("activity template ID must be at least 3 characters long")
	}

	if len(id) > 100 {
		return errors.New("activity template ID cannot exceed 100 characters")
	}

	// Allow alphanumeric characters, hyphens, and underscores
	idRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !idRegex.MatchString(id) {
		return errors.New("activity template ID contains invalid characters")
	}

	return nil
}
