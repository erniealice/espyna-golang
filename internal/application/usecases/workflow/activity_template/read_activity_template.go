package activity_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
)

// ReadActivityTemplateRepositories groups all repository dependencies
type ReadActivityTemplateRepositories struct {
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Primary entity repository
}

// ReadActivityTemplateServices groups all business service dependencies
type ReadActivityTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadActivityTemplateUseCase handles the business logic for reading activity templates
type ReadActivityTemplateUseCase struct {
	repositories ReadActivityTemplateRepositories
	services     ReadActivityTemplateServices
}

// NewReadActivityTemplateUseCase creates use case with grouped dependencies
func NewReadActivityTemplateUseCase(
	repositories ReadActivityTemplateRepositories,
	services ReadActivityTemplateServices,
) *ReadActivityTemplateUseCase {
	return &ReadActivityTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadActivityTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadActivityTemplateUseCase with grouped parameters instead
func NewReadActivityTemplateUseCaseUngrouped(activityTemplateRepo activityTemplatepb.ActivityTemplateDomainServiceServer) *ReadActivityTemplateUseCase {
	repositories := ReadActivityTemplateRepositories{
		ActivityTemplate: activityTemplateRepo,
	}

	services := ReadActivityTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadActivityTemplateUseCase(repositories, services)
}

// Execute performs the read activity template operation
func (uc *ReadActivityTemplateUseCase) Execute(ctx context.Context, req *activityTemplatepb.ReadActivityTemplateRequest) (*activityTemplatepb.ReadActivityTemplateResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.request_required", "Request is required for activity templates [DEFAULT]"))
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

// executeWithTransaction executes activity template read within a transaction
func (uc *ReadActivityTemplateUseCase) executeWithTransaction(ctx context.Context, activityTemplate *activityTemplatepb.ActivityTemplate) (*activityTemplatepb.ReadActivityTemplateResponse, error) {
	var result *activityTemplatepb.ReadActivityTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, activityTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity_template.errors.read_failed", "Activity template read failed [DEFAULT]")
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

// executeCore contains the core business logic for reading an activity template
func (uc *ReadActivityTemplateUseCase) executeCore(ctx context.Context, activityTemplate *activityTemplatepb.ActivityTemplate) (*activityTemplatepb.ReadActivityTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.ActivityTemplate.ReadActivityTemplate(ctx, &activityTemplatepb.ReadActivityTemplateRequest{
		Data: activityTemplate,
	})
}

// validateBusinessRules enforces business constraints
func (uc *ReadActivityTemplateUseCase) validateBusinessRules(ctx context.Context, activityTemplate *activityTemplatepb.ActivityTemplate) error {
	// Business rule: Required data validation
	if activityTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.data_required", "Activity template data is required [DEFAULT]"))
	}

	// Business rule: ID is required for reading
	if activityTemplate.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.id_required", "Activity template ID is required for read operations [DEFAULT]"))
	}

	// Business rule: ID format validation
	if err := uc.validateActivityTemplateID(activityTemplate.Id); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.id_invalid", "Activity template ID format is invalid [DEFAULT]"))
	}

	return nil
}

// validateActivityTemplateID validates activity template ID format
func (uc *ReadActivityTemplateUseCase) validateActivityTemplateID(id string) error {
	// Basic validation: non-empty, reasonable length, valid characters
	if strings.TrimSpace(id) == "" {
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
