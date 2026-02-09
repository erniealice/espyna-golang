package activity

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
)

// CreateActivityRepositories groups all repository dependencies
type CreateActivityRepositories struct {
	Activity         activitypb.ActivityDomainServiceServer                 // Primary entity repository
	Stage            stagepb.StageDomainServiceServer                       // Foreign key reference
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer // Foreign key reference
}

// CreateActivityServices groups all business service dependencies
type CreateActivityServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateActivityUseCase handles the business logic for creating activities
type CreateActivityUseCase struct {
	repositories CreateActivityRepositories
	services     CreateActivityServices
}

// NewCreateActivityUseCase creates use case with grouped dependencies
func NewCreateActivityUseCase(
	repositories CreateActivityRepositories,
	services CreateActivityServices,
) *CreateActivityUseCase {
	return &CreateActivityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateActivityUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateActivityUseCase with grouped parameters instead
func NewCreateActivityUseCaseUngrouped(activityRepo activitypb.ActivityDomainServiceServer, stageRepo stagepb.StageDomainServiceServer, activityTemplateRepo activityTemplatepb.ActivityTemplateDomainServiceServer) *CreateActivityUseCase {
	repositories := CreateActivityRepositories{
		Activity:         activityRepo,
		Stage:            stageRepo,
		ActivityTemplate: activityTemplateRepo,
	}

	services := CreateActivityServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateActivityUseCase(repositories, services)
}

// Execute performs the create activity operation
func (uc *CreateActivityUseCase) Execute(ctx context.Context, req *activitypb.CreateActivityRequest) (*activitypb.CreateActivityResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"activity", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.request_required", "Request is required for activities [DEFAULT]"))
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
	enrichedActivity := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedActivity)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedActivity)
}

// executeWithTransaction executes activity creation within a transaction
func (uc *CreateActivityUseCase) executeWithTransaction(ctx context.Context, enrichedActivity *activitypb.Activity) (*activitypb.CreateActivityResponse, error) {
	var result *activitypb.CreateActivityResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedActivity)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity.errors.creation_failed", "Activity creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating an activity
func (uc *CreateActivityUseCase) executeCore(ctx context.Context, enrichedActivity *activitypb.Activity) (*activitypb.CreateActivityResponse, error) {
	// Delegate to repository
	return uc.repositories.Activity.CreateActivity(ctx, &activitypb.CreateActivityRequest{
		Data: enrichedActivity,
	})
}

// applyBusinessLogic applies business rules and returns enriched activity
func (uc *CreateActivityUseCase) applyBusinessLogic(activity *activitypb.Activity) *activitypb.Activity {
	now := time.Now()

	// Business logic: Generate Activity ID if not provided
	if activity.Id == "" {
		if uc.services.IDService != nil {
			activity.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			activity.Id = fmt.Sprintf("activity-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new activities
	activity.Active = true

	// Business logic: Set default completion percentage for new activities
	if activity.CompletionPercentage == nil {
		defaultCompletion := int32(0)
		activity.CompletionPercentage = &defaultCompletion
	}

	// Business logic: Set creation audit fields
	activity.DateCreated = &[]int64{now.UnixMilli()}[0]
	activity.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	activity.DateModified = &[]int64{now.UnixMilli()}[0]
	activity.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return activity
}

// validateForeignKeys validates that all foreign key references exist and are valid
func (uc *CreateActivityUseCase) validateForeignKeys(ctx context.Context, activity *activitypb.Activity) error {
	// Foreign key validation: Stage must exist and be active
	if activity.StageId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.stage_id_required", "Stage ID is required for activities [DEFAULT]"))
	}

	// Check stage exists
	stageReadReq := &stagepb.ReadStageRequest{
		Data: &stagepb.Stage{
			Id: activity.StageId,
		},
	}
	stageRes, err := uc.repositories.Stage.ReadStage(ctx, stageReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.stage_not_found", "Stage not found [DEFAULT]"))
	}
	if stageRes == nil || len(stageRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.stage_not_found", "Stage not found [DEFAULT]"))
	}
	if !stageRes.Data[0].Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.stage_inactive", "Stage is inactive [DEFAULT]"))
	}

	// Foreign key validation: Activity template must exist and be active
	if activity.ActivityTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.activity_template_id_required", "Activity template ID is required for activities [DEFAULT]"))
	}

	// Check activity template exists
	activityTemplateReadReq := &activityTemplatepb.ReadActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Id: activity.ActivityTemplateId,
		},
	}
	activityTemplateRes, err := uc.repositories.ActivityTemplate.ReadActivityTemplate(ctx, activityTemplateReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_template_not_found", "Activity template not found [DEFAULT]"))
	}
	if activityTemplateRes == nil || len(activityTemplateRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_template_not_found", "Activity template not found [DEFAULT]"))
	}
	if !activityTemplateRes.Data[0].Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_template_inactive", "Activity template is inactive [DEFAULT]"))
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateActivityUseCase) validateBusinessRules(ctx context.Context, activity *activitypb.Activity) error {
	// Business rule: Required data validation
	if activity == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.data_required", "Activity data is required [DEFAULT]"))
	}

	// Business rule: Name is required
	if activity.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.name_required", "Activity name is required [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(activity.Name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.name_too_short", "Activity name must be at least 2 characters long [DEFAULT]"))
	}

	if len(activity.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.name_too_long", "Activity name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Name format validation (alphanumeric, spaces, hyphens, underscores)
	if err := uc.validateActivityName(activity.Name); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.name_invalid", "Activity name contains invalid characters [DEFAULT]"))
	}

	// Business rule: Description length constraints
	if activity.Description != nil && len(*activity.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.description_too_long", "Activity description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Assigned to format validation if provided
	if activity.AssignedTo != nil && *activity.AssignedTo != "" {
		if len(*activity.AssignedTo) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.assigned_to_invalid", "Assigned to field is invalid [DEFAULT]"))
		}
	}

	// Business rule: Completed by format validation if provided
	if activity.CompletedBy != nil && *activity.CompletedBy != "" {
		if len(*activity.CompletedBy) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.completed_by_invalid", "Completed by field is invalid [DEFAULT]"))
		}
	}

	// Business rule: Due date validation if provided
	if activity.DateDue != nil && *activity.DateDue > 0 {
		if *activity.DateDue < time.Now().UnixMilli() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.due_date_past", "Due date cannot be in the past [DEFAULT]"))
		}
	}

	// Business rule: Completion percentage validation if provided
	if activity.CompletionPercentage != nil {
		if *activity.CompletionPercentage < 0 || *activity.CompletionPercentage > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.completion_percentage_invalid", "Completion percentage must be between 0 and 100 [DEFAULT]"))
		}
	}

	// Business rule: Duration validation if provided
	if activity.EstimatedDurationMinutes != nil && *activity.EstimatedDurationMinutes > 0 {
		if *activity.EstimatedDurationMinutes < 1 || *activity.EstimatedDurationMinutes > 10080 { // Max 1 week in minutes
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.estimated_duration_invalid", "Estimated duration must be between 1 and 10080 minutes [DEFAULT]"))
		}
	}

	if activity.ActualDurationMinutes != nil && *activity.ActualDurationMinutes > 0 {
		if *activity.ActualDurationMinutes < 0 || *activity.ActualDurationMinutes > 10080 { // Max 1 week in minutes
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.actual_duration_invalid", "Actual duration must be between 0 and 10080 minutes [DEFAULT]"))
		}
	}

	// Business rule: JSON field length validation if provided
	if activity.InputDataJson != nil && len(*activity.InputDataJson) > 10000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.input_data_too_long", "Input data JSON cannot exceed 10000 characters [DEFAULT]"))
	}

	if activity.OutputDataJson != nil && len(*activity.OutputDataJson) > 10000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.output_data_too_long", "Output data JSON cannot exceed 10000 characters [DEFAULT]"))
	}

	if activity.ResultJson != nil && len(*activity.ResultJson) > 10000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.result_json_too_long", "Result JSON cannot exceed 10000 characters [DEFAULT]"))
	}

	// Business rule: Text field length validation if provided
	if activity.ErrorMessage != nil && len(*activity.ErrorMessage) > 2000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.error_message_too_long", "Error message cannot exceed 2000 characters [DEFAULT]"))
	}

	if activity.ApprovalComments != nil && len(*activity.ApprovalComments) > 2000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.approval_comments_too_long", "Approval comments cannot exceed 2000 characters [DEFAULT]"))
	}

	if activity.RejectionReason != nil && len(*activity.RejectionReason) > 2000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.rejection_reason_too_long", "Rejection reason cannot exceed 2000 characters [DEFAULT]"))
	}

	return nil
}

// validateActivityName validates activity name format
func (uc *CreateActivityUseCase) validateActivityName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid activity name format")
	}
	return nil
}
