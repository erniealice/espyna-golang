package stage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
)

// UpdateStageRepositories groups all repository dependencies
type UpdateStageRepositories struct {
	Stage stagepb.StageDomainServiceServer // Primary entity repository
}

// UpdateStageServices groups all business service dependencies
type UpdateStageServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateStageUseCase handles the business logic for updating stages
type UpdateStageUseCase struct {
	repositories UpdateStageRepositories
	services     UpdateStageServices
}

// NewUpdateStageUseCase creates use case with grouped dependencies
func NewUpdateStageUseCase(
	repositories UpdateStageRepositories,
	services UpdateStageServices,
) *UpdateStageUseCase {
	return &UpdateStageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateStageUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateStageUseCase with grouped parameters instead
func NewUpdateStageUseCaseUngrouped(stageRepo stagepb.StageDomainServiceServer) *UpdateStageUseCase {
	repositories := UpdateStageRepositories{
		Stage: stageRepo,
	}

	services := UpdateStageServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateStageUseCase(repositories, services)
}

// Execute performs the update stage operation
func (uc *UpdateStageUseCase) Execute(ctx context.Context, req *stagepb.UpdateStageRequest) (*stagepb.UpdateStageResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"stage", ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.request_required", "Request is required for stages [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Existence validation
	if err := uc.validateStageExists(ctx, req.Data.Id); err != nil {
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

// executeWithTransaction executes stage update within a transaction
func (uc *UpdateStageUseCase) executeWithTransaction(ctx context.Context, enrichedStage *stagepb.Stage) (*stagepb.UpdateStageResponse, error) {
	var result *stagepb.UpdateStageResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedStage)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "stage.errors.update_failed", "Stage update failed [DEFAULT]")
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

// executeCore contains the core business logic for updating a stage
func (uc *UpdateStageUseCase) executeCore(ctx context.Context, enrichedStage *stagepb.Stage) (*stagepb.UpdateStageResponse, error) {
	// Delegate to repository
	return uc.repositories.Stage.UpdateStage(ctx, &stagepb.UpdateStageRequest{
		Data: enrichedStage,
	})
}

// applyBusinessLogic applies business rules and returns enriched stage
func (uc *UpdateStageUseCase) applyBusinessLogic(stage *stagepb.Stage) *stagepb.Stage {
	now := time.Now()

	// Business logic: Update modification audit fields
	stage.DateModified = &[]int64{now.UnixMilli()}[0]
	stage.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return stage
}

// validateStageExists validates that the stage exists and is active
func (uc *UpdateStageUseCase) validateStageExists(ctx context.Context, stageID string) error {
	// Check stage exists
	stageReadReq := &stagepb.ReadStageRequest{
		Data: &stagepb.Stage{
			Id: stageID,
		},
	}
	stageRes, err := uc.repositories.Stage.ReadStage(ctx, stageReadReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_not_found", "Stage not found [DEFAULT]"))
	}
	if stageRes == nil || len(stageRes.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_not_found", "Stage not found [DEFAULT]"))
	}

	// Business rule: Cannot update inactive stage
	if !stageRes.Data[0].Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.errors.stage_inactive", "Cannot update inactive stage [DEFAULT]"))
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateStageUseCase) validateBusinessRules(ctx context.Context, stage *stagepb.Stage) error {
	// Business rule: Required data validation
	if stage == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.data_required", "Stage data is required [DEFAULT]"))
	}

	// Business rule: Stage ID is required
	if stage.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.id_required", "Stage ID is required for update operations [DEFAULT]"))
	}

	// Business rule: Name validation if provided
	if stage.Name != "" {
		if len(stage.Name) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.name_too_short", "Stage name must be at least 2 characters long [DEFAULT]"))
		}
		if len(stage.Name) > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.name_too_long", "Stage name cannot exceed 100 characters [DEFAULT]"))
		}
		// Business rule: Name format validation
		if err := uc.validateStageName(stage.Name); err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.name_invalid", "Stage name contains invalid characters [DEFAULT]"))
		}
	}

	// Business rule: Description length constraints if provided
	if stage.Description != nil && len(*stage.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.description_too_long", "Stage description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Assigned to format validation if provided
	if stage.AssignedTo != nil && *stage.AssignedTo != "" {
		if len(*stage.AssignedTo) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.assigned_to_invalid", "Assigned to field is invalid [DEFAULT]"))
		}
	}

	// Business rule: Completed by format validation if provided
	if stage.CompletedBy != nil && *stage.CompletedBy != "" {
		if len(*stage.CompletedBy) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.completed_by_invalid", "Completed by field is invalid [DEFAULT]"))
		}
	}

	// Business rule: Due date validation if provided
	if stage.DateDue != nil && *stage.DateDue > 0 {
		// Allow past due dates for updates (might be delayed due to legitimate reasons)
		// But prevent obviously invalid dates
		if *stage.DateDue < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.due_date_invalid", "Due date is invalid [DEFAULT]"))
		}
	}

	// Business rule: Date validation if provided
	if stage.DateStarted != nil && *stage.DateStarted > 0 {
		if *stage.DateStarted < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.date_started_invalid", "Date started is invalid [DEFAULT]"))
		}
	}

	if stage.DateCompleted != nil && *stage.DateCompleted > 0 {
		if *stage.DateCompleted < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.date_completed_invalid", "Date completed is invalid [DEFAULT]"))
		}

		// Business rule: Completion date cannot be before start date
		if stage.DateStarted != nil && *stage.DateStarted > 0 && *stage.DateCompleted < *stage.DateStarted {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.completion_before_start", "Completion date cannot be before start date [DEFAULT]"))
		}
	}

	// Business rule: Completion percentage validation if provided
	if stage.CompletionPercentage != nil {
		if *stage.CompletionPercentage < 0 || *stage.CompletionPercentage > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.completion_percentage_invalid", "Completion percentage must be between 0 and 100 [DEFAULT]"))
		}
	}

	// Business rule: Error message length validation if provided
	if stage.ErrorMessage != nil && len(*stage.ErrorMessage) > 2000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.error_message_too_long", "Error message cannot exceed 2000 characters [DEFAULT]"))
	}

	// Business rule: Result JSON length validation if provided
	if stage.ResultJson != nil && len(*stage.ResultJson) > 10000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "stage.validation.result_json_too_long", "Result JSON cannot exceed 10000 characters [DEFAULT]"))
	}

	return nil
}

// validateStageName validates stage name format
func (uc *UpdateStageUseCase) validateStageName(name string) error {
	// Block only control chars and security-risky chars: < > \ | ;
	nameRegex := regexp.MustCompile(`^[^\x00-\x1f<>\\|;]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("invalid stage name format")
	}
	return nil
}
