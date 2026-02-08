package plan_settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
)

// DeletePlanSettingsRepositories groups all repository dependencies
type DeletePlanSettingsRepositories struct {
	PlanSettings plansettingspb.PlanSettingsDomainServiceServer // Primary entity repository
}

// DeletePlanSettingsServices groups all business service dependencies
type DeletePlanSettingsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeletePlanSettingsUseCase handles the business logic for deleting plan_settings
type DeletePlanSettingsUseCase struct {
	repositories DeletePlanSettingsRepositories
	services     DeletePlanSettingsServices
}

// NewDeletePlanSettingsUseCase creates a new DeletePlanSettingsUseCase
func NewDeletePlanSettingsUseCase(
	repositories DeletePlanSettingsRepositories,
	services DeletePlanSettingsServices,
) *DeletePlanSettingsUseCase {
	return &DeletePlanSettingsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete plan_settings operation
func (uc *DeletePlanSettingsUseCase) Execute(ctx context.Context, req *plansettingspb.DeletePlanSettingsRequest) (*plansettingspb.DeletePlanSettingsResponse, error) {
	businessType := uc.getBusinessTypeFromContext(ctx)

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	response, err := uc.repositories.PlanSettings.DeletePlanSettings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(uc.getTranslatedMessage(ctx, businessType, "plan_settings.errors.deletion_failed", "plan settings deletion failed: %s"), err.Error())
	}

	return response, nil
}

// getBusinessTypeFromContext extracts business type from context
func (uc *DeletePlanSettingsUseCase) getBusinessTypeFromContext(ctx context.Context) string {
	if businessType, ok := ctx.Value("businessType").(string); ok {
		return businessType
	}
	return "education"
}

// getTranslatedMessage gets a translated message with fallback
func (uc *DeletePlanSettingsUseCase) getTranslatedMessage(ctx context.Context, businessType, key, fallback string) string {
	if uc.services.TranslationService != nil {
		return uc.services.TranslationService.GetWithDefault(ctx, businessType, key, fallback)
	}
	return fallback
}

// validateInput validates the input request
func (uc *DeletePlanSettingsUseCase) validateInput(ctx context.Context, req *plansettingspb.DeletePlanSettingsRequest) error {
	businessType := uc.getBusinessTypeFromContext(ctx)

	if req == nil {
		return errors.New(uc.getTranslatedMessage(ctx, businessType, "plan_settings.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(uc.getTranslatedMessage(ctx, businessType, "plan_settings.validation.data_required", "plan settings data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(uc.getTranslatedMessage(ctx, businessType, "plan_settings.validation.id_required", "plan settings ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for plan_settings deletion
func (uc *DeletePlanSettingsUseCase) validateBusinessRules(ctx context.Context, req *plansettingspb.DeletePlanSettingsRequest) error {
	businessType := uc.getBusinessTypeFromContext(ctx)

	// Validate plan settings ID format
	if len(req.Data.Id) < 3 {
		return errors.New(uc.getTranslatedMessage(ctx, businessType, "plan_settings.validation.id_too_short", "plan settings ID must be at least 3 characters long"))
	}

	// Additional business rules for deletion can be added here
	// For example, preventing deletion of critical plan settings

	return nil
}
