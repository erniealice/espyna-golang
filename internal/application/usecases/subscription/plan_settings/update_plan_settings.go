package plan_settings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
)

// UpdatePlanSettingsRepositories groups all repository dependencies
type UpdatePlanSettingsRepositories struct {
	PlanSettings plansettingspb.PlanSettingsDomainServiceServer // Primary entity repository
	Plan         planpb.PlanDomainServiceServer                 // Entity reference dependency
}

// UpdatePlanSettingsServices groups all business service dependencies
type UpdatePlanSettingsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdatePlanSettingsUseCase handles the business logic for updating plan_settings
type UpdatePlanSettingsUseCase struct {
	repositories UpdatePlanSettingsRepositories
	services     UpdatePlanSettingsServices
}

// NewUpdatePlanSettingsUseCase creates a new UpdatePlanSettingsUseCase
func NewUpdatePlanSettingsUseCase(
	repositories UpdatePlanSettingsRepositories,
	services UpdatePlanSettingsServices,
) *UpdatePlanSettingsUseCase {
	return &UpdatePlanSettingsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update plan_settings operation
func (uc *UpdatePlanSettingsUseCase) Execute(ctx context.Context, req *plansettingspb.UpdatePlanSettingsRequest) (*plansettingspb.UpdatePlanSettingsResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedPlanSettings := uc.applyBusinessLogic(req.Data)

	// Delegate to repository
	response, err := uc.repositories.PlanSettings.UpdatePlanSettings(ctx, &plansettingspb.UpdatePlanSettingsRequest{
		Data: enrichedPlanSettings,
	})
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.update_failed", "plan settings update failed: %s"), err.Error())
	}

	return response, nil
}

// applyBusinessLogic applies business rules and returns enriched plan settings
func (uc *UpdatePlanSettingsUseCase) applyBusinessLogic(planSettings *plansettingspb.PlanSettings) *plansettingspb.PlanSettings {
	now := time.Now()

	// Business logic: Update modification audit fields
	planSettings.DateModified = &[]int64{now.Unix()}[0]
	planSettings.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return planSettings
}

// validateInput validates the input request
func (uc *UpdatePlanSettingsUseCase) validateInput(ctx context.Context, req *plansettingspb.UpdatePlanSettingsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.data_required", "plan settings data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.id_required", "plan settings ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdatePlanSettingsUseCase) validateBusinessRules(ctx context.Context, planSettings *plansettingspb.PlanSettings) error {
	// Business rule: Required data validation
	if planSettings == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.data_required", "plan settings data is required"))
	}
	if planSettings.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.id_required", "plan settings ID is required"))
	}
	if planSettings.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.name_required", "plan settings name is required"))
	}
	if planSettings.PlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.plan_id_required", "plan ID is required"))
	}

	// Business rule: ID length constraints
	if len(planSettings.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.id_too_short", "plan settings ID must be at least 3 characters long"))
	}

	// Business rule: Name length constraints
	if len(planSettings.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.name_too_short", "plan settings name must be at least 3 characters long"))
	}

	if len(planSettings.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.name_too_long", "plan settings name cannot exceed 100 characters"))
	}

	// Business rule: Plan ID format validation
	if len(planSettings.PlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.plan_id_too_short", "plan ID must be at least 3 characters long"))
	}

	// Business rule: Description length validation
	if len(planSettings.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.description_too_long", "plan settings description cannot exceed 500 characters"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdatePlanSettingsUseCase) validateEntityReferences(ctx context.Context, planSettings *plansettingspb.PlanSettings) error {
	// Validate Plan entity reference
	if planSettings.PlanId != "" {
		planId := planSettings.PlanId
		plan, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
			Data: &planpb.Plan{Id: &planId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.plan_validation_failed", "failed to validate plan entity reference: %s"), err.Error())
		}
		if plan == nil || plan.Data == nil || len(plan.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.plan_not_found", "referenced plan with ID '%s' does not exist"), planSettings.PlanId)
		}
		if !plan.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.plan_inactive", "referenced plan with ID '%s' is not active"), planSettings.PlanId)
		}
	}

	return nil
}
