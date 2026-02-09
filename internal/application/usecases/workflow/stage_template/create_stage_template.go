package stage_template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// CreateStageTemplateRepositories groups all repository dependencies
type CreateStageTemplateRepositories struct {
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Primary entity repository
	WorkflowTemplate workflowTemplatepb.WorkflowTemplateDomainServiceServer // Foreign key reference
}

// CreateStageTemplateServices groups all business service dependencies
type CreateStageTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateStageTemplateUseCase handles the business logic for creating stage templates
type CreateStageTemplateUseCase struct {
	repositories CreateStageTemplateRepositories
	services     CreateStageTemplateServices
}

// NewCreateStageTemplateUseCase creates use case with grouped dependencies
func NewCreateStageTemplateUseCase(
	repositories CreateStageTemplateRepositories,
	services CreateStageTemplateServices,
) *CreateStageTemplateUseCase {
	return &CreateStageTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateStageTemplateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateStageTemplateUseCase with grouped parameters instead
func NewCreateStageTemplateUseCaseUngrouped(stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer, workflowTemplateRepo workflowTemplatepb.WorkflowTemplateDomainServiceServer) *CreateStageTemplateUseCase {
	repositories := CreateStageTemplateRepositories{
		StageTemplate:    stageTemplateRepo,
		WorkflowTemplate: workflowTemplateRepo,
	}

	services := CreateStageTemplateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateStageTemplateUseCase(repositories, services)
}

// Execute performs the create stage template operation
func (uc *CreateStageTemplateUseCase) Execute(ctx context.Context, req *stageTemplatepb.CreateStageTemplateRequest) (*stageTemplatepb.CreateStageTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"stage_template", ports.ActionCreate); err != nil {
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

	// Business enrichment
	enrichedStageTemplate := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedStageTemplate)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedStageTemplate)
}

// executeWithTransaction executes stage template creation within a transaction
func (uc *CreateStageTemplateUseCase) executeWithTransaction(ctx context.Context, enrichedStageTemplate *stageTemplatepb.StageTemplate) (*stageTemplatepb.CreateStageTemplateResponse, error) {
	var result *stageTemplatepb.CreateStageTemplateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedStageTemplate)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage_template.errors.creation_failed", "Stage template creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating a stage template
func (uc *CreateStageTemplateUseCase) executeCore(ctx context.Context, enrichedStageTemplate *stageTemplatepb.StageTemplate) (*stageTemplatepb.CreateStageTemplateResponse, error) {
	// Delegate to repository
	return uc.repositories.StageTemplate.CreateStageTemplate(ctx, &stageTemplatepb.CreateStageTemplateRequest{
		Data: enrichedStageTemplate,
	})
}

// applyBusinessLogic applies business rules and returns enriched stage template
func (uc *CreateStageTemplateUseCase) applyBusinessLogic(stageTemplate *stageTemplatepb.StageTemplate) *stageTemplatepb.StageTemplate {
	now := time.Now()

	// Business logic: Generate Stage Template ID if not provided
	if stageTemplate.Id == "" {
		if uc.services.IDService != nil {
			stageTemplate.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			stageTemplate.Id = fmt.Sprintf("stage-template-%d", now.UnixNano())
		}
	}

	// Business logic: Set default order index if not provided
	if stageTemplate.OrderIndex == nil {
		stageTemplate.OrderIndex = &[]int32{1}[0]
	}

	// Business logic: Set default is_required if not provided
	if stageTemplate.IsRequired == nil {
		stageTemplate.IsRequired = &[]bool{true}[0]
	}

	// Business logic: Set active status for new stage templates
	stageTemplate.Active = true

	// Business logic: Set creation audit fields
	stageTemplate.DateCreated = &[]int64{now.UnixMilli()}[0]
	stageTemplate.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	stageTemplate.DateModified = &[]int64{now.UnixMilli()}[0]
	stageTemplate.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return stageTemplate
}

// validateBusinessRules enforces business constraints
func (uc *CreateStageTemplateUseCase) validateBusinessRules(ctx context.Context, stageTemplate *stageTemplatepb.StageTemplate) error {
	// Business rule: Required data validation
	if stageTemplate == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.data_required", "Stage template data is required [DEFAULT]"))
	}

	// Business rule: Name is required
	if stageTemplate.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.name_required", "Stage template name is required [DEFAULT]"))
	}

	// Business rule: Workflow Template ID is required (foreign key)
	if stageTemplate.WorkflowTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.workflow_template_id_required", "Workflow Template ID is required [DEFAULT]"))
	}

	// Business rule: Validate foreign key - workflow template must exist
	if err := uc.validateWorkflowTemplateExists(ctx, stageTemplate.WorkflowTemplateId); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.workflow_template_not_found", "Referenced workflow template does not exist [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(stageTemplate.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.name_too_short", "Stage template name must be at least 2 characters long [DEFAULT]"))
	}

	if len(stageTemplate.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.name_too_long", "Stage template name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Name format validation
	if err := uc.validateStageTemplateName(stageTemplate.Name); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.name_invalid", "Stage template name contains invalid characters [DEFAULT]"))
	}

	// Business rule: Description length constraints
	if stageTemplate.Description != nil && len(*stageTemplate.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.description_too_long", "Stage template description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Order index validation if provided
	if stageTemplate.OrderIndex != nil && *stageTemplate.OrderIndex < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.order_index_negative", "Order index cannot be negative [DEFAULT]"))
	}

	// Business rule: Condition expression length constraints if provided
	if stageTemplate.ConditionExpression != nil && len(*stageTemplate.ConditionExpression) > 2000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage_template.validation.condition_expression_too_long", "Condition expression cannot exceed 2000 characters [DEFAULT]"))
	}

	return nil
}

// validateWorkflowTemplateExists validates that the referenced workflow template exists
func (uc *CreateStageTemplateUseCase) validateWorkflowTemplateExists(ctx context.Context, workflowTemplateID string) error {
	// Check if workflow template exists by attempting to read it
	workflowTemplate := &workflowTemplatepb.WorkflowTemplate{Id: workflowTemplateID}
	_, err := uc.repositories.WorkflowTemplate.ReadWorkflowTemplate(ctx, &workflowTemplatepb.ReadWorkflowTemplateRequest{
		Data: workflowTemplate,
	})

	if err != nil {
		return fmt.Errorf("workflow template not found: %w", err)
	}

	return nil
}

// validateStageTemplateName validates stage template name format
func (uc *CreateStageTemplateUseCase) validateStageTemplateName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid stage template name format")
	}
	return nil
}
