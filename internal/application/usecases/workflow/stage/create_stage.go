package stage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
)

// CreateStageRepositories groups all repository dependencies
type CreateStageRepositories struct {
	Stage         stagepb.StageDomainServiceServer                 // Primary entity repository
	Workflow      workflowpb.WorkflowDomainServiceServer           // Foreign key reference
	StageTemplate stageTemplatepb.StageTemplateDomainServiceServer // Foreign key reference
}

// CreateStageServices groups all business service dependencies
type CreateStageServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateStageUseCase handles the business logic for creating stages
type CreateStageUseCase struct {
	repositories CreateStageRepositories
	services     CreateStageServices
}

// NewCreateStageUseCase creates use case with grouped dependencies
func NewCreateStageUseCase(
	repositories CreateStageRepositories,
	services CreateStageServices,
) *CreateStageUseCase {
	return &CreateStageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateStageUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateStageUseCase with grouped parameters instead
func NewCreateStageUseCaseUngrouped(stageRepo stagepb.StageDomainServiceServer, workflowRepo workflowpb.WorkflowDomainServiceServer, stageTemplateRepo stageTemplatepb.StageTemplateDomainServiceServer) *CreateStageUseCase {
	repositories := CreateStageRepositories{
		Stage:         stageRepo,
		Workflow:      workflowRepo,
		StageTemplate: stageTemplateRepo,
	}

	services := CreateStageServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateStageUseCase(repositories, services)
}

// Execute performs the create stage operation
func (uc *CreateStageUseCase) Execute(ctx context.Context, req *stagepb.CreateStageRequest) (*stagepb.CreateStageResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.request_required", "Request is required for stages [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Foreign key validation
	if err := uc.validateForeignKeys(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedStage := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedStage)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedStage)
}

// executeWithTransaction executes stage creation within a transaction
func (uc *CreateStageUseCase) executeWithTransaction(ctx context.Context, enrichedStage *stagepb.Stage) (*stagepb.CreateStageResponse, error) {
	var result *stagepb.CreateStageResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedStage)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage.errors.creation_failed", "Stage creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating a stage
func (uc *CreateStageUseCase) executeCore(ctx context.Context, enrichedStage *stagepb.Stage) (*stagepb.CreateStageResponse, error) {
	// Delegate to repository
	return uc.repositories.Stage.CreateStage(ctx, &stagepb.CreateStageRequest{
		Data: enrichedStage,
	})
}

// applyBusinessLogic applies business rules and returns enriched stage
func (uc *CreateStageUseCase) applyBusinessLogic(stage *stagepb.Stage) *stagepb.Stage {
	now := time.Now()

	// Business logic: Generate Stage ID if not provided
	if stage.Id == "" {
		if uc.services.IDService != nil {
			stage.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			stage.Id = fmt.Sprintf("stage-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new stages
	stage.Active = true

	// Business logic: Set default completion percentage for new stages
	if stage.CompletionPercentage == nil {
		defaultCompletion := int32(0)
		stage.CompletionPercentage = &defaultCompletion
	}

	// Business logic: Set creation audit fields
	stage.DateCreated = &[]int64{now.UnixMilli()}[0]
	stage.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	stage.DateModified = &[]int64{now.UnixMilli()}[0]
	stage.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return stage
}

// validateForeignKeys validates that all foreign key references exist and are valid
func (uc *CreateStageUseCase) validateForeignKeys(ctx context.Context, stage *stagepb.Stage) error {
	// Foreign key validation: Workflow instance must exist and be active
	if stage.WorkflowInstanceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.workflow_instance_id_required", "Workflow instance ID is required for stages [DEFAULT]"))
	}

	// Check workflow exists
	workflowReadReq := &workflowpb.ReadWorkflowRequest{
		Data: &workflowpb.Workflow{
			Id: stage.WorkflowInstanceId,
		},
	}
	workflowRes, err := uc.repositories.Workflow.ReadWorkflow(ctx, workflowReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.workflow_instance_not_found", "Workflow instance not found [DEFAULT]"))
	}
	if workflowRes == nil || len(workflowRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.workflow_instance_not_found", "Workflow instance not found [DEFAULT]"))
	}
	if !workflowRes.Data[0].Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.workflow_instance_inactive", "Workflow instance is inactive [DEFAULT]"))
	}

	// Foreign key validation: Stage template must exist and be active
	if stage.StageTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.stage_template_id_required", "Stage template ID is required for stages [DEFAULT]"))
	}

	// Check stage template exists
	stageTemplateReadReq := &stageTemplatepb.ReadStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Id: stage.StageTemplateId,
		},
	}
	stageTemplateRes, err := uc.repositories.StageTemplate.ReadStageTemplate(ctx, stageTemplateReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_template_not_found", "Stage template not found [DEFAULT]"))
	}
	if stageTemplateRes == nil || len(stageTemplateRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_template_not_found", "Stage template not found [DEFAULT]"))
	}
	if !stageTemplateRes.Data[0].Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_template_inactive", "Stage template is inactive [DEFAULT]"))
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateStageUseCase) validateBusinessRules(ctx context.Context, stage *stagepb.Stage) error {
	// Business rule: Required data validation
	if stage == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.data_required", "Stage data is required [DEFAULT]"))
	}

	// Business rule: Name is required
	if stage.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.name_required", "Stage name is required [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(stage.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.name_too_short", "Stage name must be at least 2 characters long [DEFAULT]"))
	}

	if len(stage.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.name_too_long", "Stage name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Name format validation (alphanumeric, spaces, hyphens, underscores)
	if err := uc.validateStageName(stage.Name); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.name_invalid", "Stage name contains invalid characters [DEFAULT]"))
	}

	// Business rule: Description length constraints
	if stage.Description != nil && len(*stage.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.description_too_long", "Stage description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Assigned to format validation if provided
	if stage.AssignedTo != nil && *stage.AssignedTo != "" {
		if len(*stage.AssignedTo) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.assigned_to_invalid", "Assigned to field is invalid [DEFAULT]"))
		}
	}

	// Business rule: Due date validation if provided
	if stage.DateDue != nil && *stage.DateDue > 0 {
		if *stage.DateDue < time.Now().UnixMilli() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.due_date_past", "Due date cannot be in the past [DEFAULT]"))
		}
	}

	// Business rule: Completion percentage validation if provided
	if stage.CompletionPercentage != nil {
		if *stage.CompletionPercentage < 0 || *stage.CompletionPercentage > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.completion_percentage_invalid", "Completion percentage must be between 0 and 100 [DEFAULT]"))
		}
	}

	return nil
}

// validateStageName validates stage name format
func (uc *CreateStageUseCase) validateStageName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid stage name format")
	}
	return nil
}
