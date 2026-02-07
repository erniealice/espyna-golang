package activity

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
)

// UpdateActivityRepositories groups all repository dependencies
type UpdateActivityRepositories struct {
	Activity activitypb.ActivityDomainServiceServer // Primary entity repository
}

// UpdateActivityServices groups all business service dependencies
type UpdateActivityServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateActivityUseCase handles the business logic for updating activities
type UpdateActivityUseCase struct {
	repositories UpdateActivityRepositories
	services     UpdateActivityServices
}

// NewUpdateActivityUseCase creates use case with grouped dependencies
func NewUpdateActivityUseCase(
	repositories UpdateActivityRepositories,
	services UpdateActivityServices,
) *UpdateActivityUseCase {
	return &UpdateActivityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateActivityUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateActivityUseCase with grouped parameters instead
func NewUpdateActivityUseCaseUngrouped(activityRepo activitypb.ActivityDomainServiceServer) *UpdateActivityUseCase {
	repositories := UpdateActivityRepositories{
		Activity: activityRepo,
	}

	services := UpdateActivityServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateActivityUseCase(repositories, services)
}

// Execute performs the update activity operation
func (uc *UpdateActivityUseCase) Execute(ctx context.Context, req *activitypb.UpdateActivityRequest) (*activitypb.UpdateActivityResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.request_required", "Request is required for activities [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Existence validation
	if err := uc.validateActivityExists(ctx, req.Data.Id); err != nil {
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

// executeWithTransaction executes activity update within a transaction
func (uc *UpdateActivityUseCase) executeWithTransaction(ctx context.Context, enrichedActivity *activitypb.Activity) (*activitypb.UpdateActivityResponse, error) {
	var result *activitypb.UpdateActivityResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedActivity)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "activity.errors.update_failed", "Activity update failed [DEFAULT]")
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

// executeCore contains the core business logic for updating an activity
func (uc *UpdateActivityUseCase) executeCore(ctx context.Context, enrichedActivity *activitypb.Activity) (*activitypb.UpdateActivityResponse, error) {
	// Delegate to repository
	return uc.repositories.Activity.UpdateActivity(ctx, &activitypb.UpdateActivityRequest{
		Data: enrichedActivity,
	})
}

// applyBusinessLogic applies business rules and returns enriched activity
func (uc *UpdateActivityUseCase) applyBusinessLogic(activity *activitypb.Activity) *activitypb.Activity {
	now := time.Now()

	// Business logic: Update modification audit fields
	activity.DateModified = &[]int64{now.UnixMilli()}[0]
	activity.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return activity
}

// validateActivityExists validates that the activity exists and is active
func (uc *UpdateActivityUseCase) validateActivityExists(ctx context.Context, activityID string) error {
	// Check activity exists
	activityReadReq := &activitypb.ReadActivityRequest{
		Data: &activitypb.Activity{
			Id: activityID,
		},
	}
	activityRes, err := uc.repositories.Activity.ReadActivity(ctx, activityReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_not_found", "Activity not found [DEFAULT]"))
	}
	if activityRes == nil || len(activityRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_not_found", "Activity not found [DEFAULT]"))
	}

	// Business rule: Cannot update inactive activity
	if !activityRes.Data[0].Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.errors.activity_inactive", "Cannot update inactive activity [DEFAULT]"))
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateActivityUseCase) validateBusinessRules(ctx context.Context, activity *activitypb.Activity) error {
	// Business rule: Required data validation
	if activity == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.data_required", "Activity data is required [DEFAULT]"))
	}

	// Business rule: Activity ID is required
	if activity.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.id_required", "Activity ID is required for update operations [DEFAULT]"))
	}

	// Business rule: Name validation if provided
	if activity.Name != "" {
		if len(activity.Name) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.name_too_short", "Activity name must be at least 2 characters long [DEFAULT]"))
		}
		if len(activity.Name) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.name_too_long", "Activity name cannot exceed 100 characters [DEFAULT]"))
		}
		// Business rule: Name format validation
		if err := uc.validateActivityName(activity.Name); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.name_invalid", "Activity name contains invalid characters [DEFAULT]"))
		}
	}

	// Business rule: Description length constraints if provided
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
		// Allow past due dates for updates (might be delayed due to legitimate reasons)
		// But prevent obviously invalid dates
		if *activity.DateDue < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.due_date_invalid", "Due date is invalid [DEFAULT]"))
		}
	}

	// Business rule: Date validation if provided
	if activity.DateAssigned != nil && *activity.DateAssigned > 0 {
		if *activity.DateAssigned < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.date_assigned_invalid", "Date assigned is invalid [DEFAULT]"))
		}
	}

	if activity.DateStarted != nil && *activity.DateStarted > 0 {
		if *activity.DateStarted < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.date_started_invalid", "Date started is invalid [DEFAULT]"))
		}
	}

	if activity.DateCompleted != nil && *activity.DateCompleted > 0 {
		if *activity.DateCompleted < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.date_completed_invalid", "Date completed is invalid [DEFAULT]"))
		}

		// Business rule: Completion date cannot be before start date
		if activity.DateStarted != nil && *activity.DateStarted > 0 && *activity.DateCompleted < *activity.DateStarted {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.completion_before_start", "Completion date cannot be before start date [DEFAULT]"))
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

	if activity.ActualDurationMinutes != nil && *activity.ActualDurationMinutes >= 0 {
		if *activity.ActualDurationMinutes > 10080 { // Max 1 week in minutes
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

	// Business rule: Attachment IDs validation if provided
	if activity.AttachmentIds != nil && len(activity.AttachmentIds) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "activity.validation.too_many_attachments", "Cannot have more than 50 attachments [DEFAULT]"))
	}

	return nil
}

// validateActivityName validates activity name format
func (uc *UpdateActivityUseCase) validateActivityName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid activity name format")
	}
	return nil
}
