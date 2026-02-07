package plan_settings

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	plansettingspb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_settings"
)

// ListPlanSettingsRepositories groups all repository dependencies
type ListPlanSettingsRepositories struct {
	PlanSettings plansettingspb.PlanSettingsDomainServiceServer // Primary entity repository
}

// ListPlanSettingsServices groups all business service dependencies
type ListPlanSettingsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService   // Current: Text translation and localization
}

// ListPlanSettingsUseCase handles the business logic for listing plan_settings
type ListPlanSettingsUseCase struct {
	repositories ListPlanSettingsRepositories
	services     ListPlanSettingsServices
}

// NewListPlanSettingsUseCase creates a new ListPlanSettingsUseCase
func NewListPlanSettingsUseCase(
	repositories ListPlanSettingsRepositories,
	services ListPlanSettingsServices,
) *ListPlanSettingsUseCase {
	return &ListPlanSettingsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list plan_settings operation
func (uc *ListPlanSettingsUseCase) Execute(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) (*plansettingspb.ListPlanSettingsResponse, error) {
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
	response, err := uc.repositories.PlanSettings.ListPlanSettings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(uc.getTranslatedMessage(ctx, businessType, "plan_settings.errors.list_failed", "plan settings list failed: %s"), err.Error())
	}

	return response, nil
}

// getBusinessTypeFromContext extracts business type from context
func (uc *ListPlanSettingsUseCase) getBusinessTypeFromContext(ctx context.Context) string {
	if businessType, ok := ctx.Value("businessType").(string); ok {
		return businessType
	}
	return "education"
}

// getTranslatedMessage gets a translated message with fallback
func (uc *ListPlanSettingsUseCase) getTranslatedMessage(ctx context.Context, businessType, key, fallback string) string {
	if uc.services.TranslationService != nil {
		return uc.services.TranslationService.GetWithDefault(ctx, businessType, key, fallback)
	}
	return fallback
}

// validateInput validates the input request
func (uc *ListPlanSettingsUseCase) validateInput(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) error {
	businessType := uc.getBusinessTypeFromContext(ctx)

	if req == nil {
		return errors.New(uc.getTranslatedMessage(ctx, businessType, "plan_settings.validation.request_required", "request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for plan_settings listing
func (uc *ListPlanSettingsUseCase) validateBusinessRules(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) error {
	// No specific business rules for listing plan settings
	return nil
}
