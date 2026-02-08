package activity_template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
)

// CreateActivityTemplateRepositories groups all repository dependencies
type CreateActivityTemplateRepositories struct {
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Primary entity repository
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Foreign key reference
}

// CreateActivityTemplateServices groups all business service dependencies
type CreateActivityTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateActivityTemplateUseCase handles the business logic for creating activity templates
type CreateActivityTemplateUseCase struct {
	repositories CreateActivityTemplateRepositories
	services     CreateActivityTemplateServices
}

// NewCreateActivityTemplateUseCase creates use case with grouped dependencies
func NewCreateActivityTemplateUseCase(
	repositories CreateActivityTemplateRepositories,
	services CreateActivityTemplateServices,
) *CreateActivityTemplateUseCase {
	return &CreateActivityTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateActivityTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateActivityTemplateUseCase with grouped parameters instead
func NewCreateActivityTemplateUseCaseUngrouped(activityTemplateRepo activityTemplatepb.ActivityTemplateDomainServiceServer, stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer) *CreateActivityTemplateUseCase {
	repositories := CreateActivityTemplateRepositories{
		ActivityTemplate: activityTemplateRepo,
		StageTemplate:    stageTemplateRepo,
	}

	services := CreateActivityTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateActivityTemplateUseCase(repositories, services)
}

// Execute performs the create activity template operation
func (uc *CreateActivityTemplateUseCase) Execute(ctx context.Context, req *activityTemplatepb.CreateActivityTemplateRequest) (*activityTemplatepb.CreateActivityTemplateResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.request_required", "Request is required for activity templates [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedActivityTemplate := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedActivityTemplate)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedActivityTemplate)
}

// executeWithTransaction executes activity template creation within a transaction
func (uc *CreateActivityTemplateUseCase) executeWithTransaction(ctx context.Context, enrichedActivityTemplate *activityTemplatepb.ActivityTemplate) (*activityTemplatepb.CreateActivityTemplateResponse, error) {
	var result *activityTemplatepb.CreateActivityTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedActivityTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity_template.errors.creation_failed", "Activity template creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating an activity template
func (uc *CreateActivityTemplateUseCase) executeCore(ctx context.Context, enrichedActivityTemplate *activityTemplatepb.ActivityTemplate) (*activityTemplatepb.CreateActivityTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.ActivityTemplate.CreateActivityTemplate(ctx, &activityTemplatepb.CreateActivityTemplateRequest{
		Data: enrichedActivityTemplate,
	})
}

// applyBusinessLogic applies business rules and returns enriched activity template
func (uc *CreateActivityTemplateUseCase) applyBusinessLogic(activityTemplate *activityTemplatepb.ActivityTemplate) *activityTemplatepb.ActivityTemplate {
	now := time.Now()

	// Business logic: Generate Activity Template ID if not provided
	if activityTemplate.Id == "" {
		if uc.services.IDService != nil {
			activityTemplate.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			activityTemplate.Id = fmt.Sprintf("activity-template-%d", now.UnixNano())
		}
	}

	// Business logic: Set default order index if not provided
	if activityTemplate.OrderIndex == nil {
		activityTemplate.OrderIndex = &[]int32{1}[0]
	}

	// Business logic: Set default is_required if not provided
	if activityTemplate.IsRequired == nil {
		activityTemplate.IsRequired = &[]bool{true}[0]
	}

	// Business logic: Set default estimated duration if not provided
	if activityTemplate.EstimatedDurationMinutes == nil {
		activityTemplate.EstimatedDurationMinutes = &[]int32{30}[0]
	}

	// Business logic: Set active status for new activity templates
	activityTemplate.Active = true

	// Business logic: Set creation audit fields
	activityTemplate.DateCreated = &[]int64{now.UnixMilli()}[0]
	activityTemplate.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	activityTemplate.DateModified = &[]int64{now.UnixMilli()}[0]
	activityTemplate.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return activityTemplate
}

// validateBusinessRules enforces business constraints
func (uc *CreateActivityTemplateUseCase) validateBusinessRules(ctx context.Context, activityTemplate *activityTemplatepb.ActivityTemplate) error {
	// Business rule: Required data validation
	if activityTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.data_required", "Activity template data is required [DEFAULT]"))
	}

	// Business rule: Name is required
	if activityTemplate.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.name_required", "Activity template name is required [DEFAULT]"))
	}

	// Business rule: Stage Template ID is required (foreign key)
	if activityTemplate.StageTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.stage_template_id_required", "Stage template ID is required [DEFAULT]"))
	}

	// Business rule: Validate foreign key - stage template must exist
	if err := uc.validateStageTemplateExists(ctx, activityTemplate.StageTemplateId); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.stage_template_not_found", "Referenced stage template does not exist [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(activityTemplate.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.name_too_short", "Activity template name must be at least 2 characters long [DEFAULT]"))
	}

	if len(activityTemplate.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.name_too_long", "Activity template name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Name format validation
	if err := uc.validateActivityTemplateName(activityTemplate.Name); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.name_invalid", "Activity template name contains invalid characters [DEFAULT]"))
	}

	// Business rule: Description length constraints
	if activityTemplate.Description != nil && len(*activityTemplate.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.description_too_long", "Activity template description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Order index validation if provided
	if activityTemplate.OrderIndex != nil && *activityTemplate.OrderIndex < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.order_index_negative", "Order index cannot be negative [DEFAULT]"))
	}

	// Business rule: Estimated duration validation if provided
	if activityTemplate.EstimatedDurationMinutes != nil {
		if *activityTemplate.EstimatedDurationMinutes < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.duration_negative", "Estimated duration cannot be negative [DEFAULT]"))
		}
		if *activityTemplate.EstimatedDurationMinutes > 8760 { // 1 year in minutes
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.duration_too_large", "Estimated duration cannot exceed 8760 minutes (1 year) [DEFAULT]"))
		}
	}

	// Business rule: Condition expression length constraints if provided
	if activityTemplate.ConditionExpression != nil && len(*activityTemplate.ConditionExpression) > 2000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.condition_expression_too_long", "Condition expression cannot exceed 2000 characters [DEFAULT]"))
	}

	// Business rule: Default assignee ID validation if provided
	if activityTemplate.DefaultAssigneeId != nil && *activityTemplate.DefaultAssigneeId != "" {
		if len(*activityTemplate.DefaultAssigneeId) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.assignee_id_invalid", "Default assignee ID is invalid [DEFAULT]"))
		}
	}

	// Business rule: Configuration JSON validation if provided
	if activityTemplate.ConfigurationJson != nil && *activityTemplate.ConfigurationJson != "" {
		if err := uc.validateJSON(*activityTemplate.ConfigurationJson); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.configuration_json_invalid", "Configuration JSON is invalid [DEFAULT]"))
		}
		if len(*activityTemplate.ConfigurationJson) > 10000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.configuration_json_too_long", "Configuration JSON cannot exceed 10000 characters [DEFAULT]"))
		}
	}

	// Business rule: Validation rules JSON validation if provided
	if activityTemplate.ValidationRulesJson != nil && *activityTemplate.ValidationRulesJson != "" {
		if err := uc.validateJSON(*activityTemplate.ValidationRulesJson); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.validation_rules_json_invalid", "Validation rules JSON is invalid [DEFAULT]"))
		}
		if len(*activityTemplate.ValidationRulesJson) > 5000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity_template.validation.validation_rules_json_too_long", "Validation rules JSON cannot exceed 5000 characters [DEFAULT]"))
		}
	}

	return nil
}

// validateStageTemplateExists validates that the referenced stage template exists
func (uc *CreateActivityTemplateUseCase) validateStageTemplateExists(ctx context.Context, stageTemplateID string) error {
	// Check if stage template exists by attempting to read it
	stageTemplate := &stageTemplatepb.StageTemplate{Id: stageTemplateID}
	_, err := uc.repositories.StageTemplate.ReadStageTemplate(ctx, &stageTemplatepb.ReadStageTemplateRequest{
		Data: stageTemplate,
	})

	if err != nil {
		return fmt.Errorf("stage template not found: %w", err)
	}

	return nil
}

// validateActivityTemplateName validates activity template name format
func (uc *CreateActivityTemplateUseCase) validateActivityTemplateName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid activity template name format")
	}
	return nil
}

// validateJSON validates that a string is valid JSON
func (uc *CreateActivityTemplateUseCase) validateJSON(jsonStr string) error {
	var js map[string]interface{}
	return json.Unmarshal([]byte(jsonStr), &js)
}
