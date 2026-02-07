package plan_settings

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	plansettingspb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_settings"
)

// ReadPlanSettingsRepositories groups all repository dependencies
type ReadPlanSettingsRepositories struct {
	PlanSettings plansettingspb.PlanSettingsDomainServiceServer // Primary entity repository
}

// ReadPlanSettingsServices groups all business service dependencies
type ReadPlanSettingsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPlanSettingsUseCase handles the business logic for reading plan_settings
type ReadPlanSettingsUseCase struct {
	repositories ReadPlanSettingsRepositories
	services     ReadPlanSettingsServices
}

// NewReadPlanSettingsUseCase creates a new ReadPlanSettingsUseCase
func NewReadPlanSettingsUseCase(
	repositories ReadPlanSettingsRepositories,
	services ReadPlanSettingsServices,
) *ReadPlanSettingsUseCase {
	return &ReadPlanSettingsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read plan_settings operation
func (uc *ReadPlanSettingsUseCase) Execute(ctx context.Context, req *plansettingspb.ReadPlanSettingsRequest) (*plansettingspb.ReadPlanSettingsResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.PlanSettings.ReadPlanSettings(ctx, req)
}

// getBusinessTypeFromContext extracts business type from context
func (uc *ReadPlanSettingsUseCase) getBusinessTypeFromContext(ctx context.Context) string {
	if businessType, ok := ctx.Value("businessType").(string); ok {
		return businessType
	}
	return "education"
}

// getTranslatedMessage gets a translated message with fallback
func (uc *ReadPlanSettingsUseCase) getTranslatedMessage(ctx context.Context, businessType, key, fallback string) string {
	if uc.services.TranslationService != nil {
		return uc.services.TranslationService.GetWithDefault(ctx, businessType, key, fallback)
	}
	return fallback
}

// validateInput validates the input request
func (uc *ReadPlanSettingsUseCase) validateInput(ctx context.Context, req *plansettingspb.ReadPlanSettingsRequest) error {
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

// validateBusinessRules enforces business constraints for plan_settings reading
func (uc *ReadPlanSettingsUseCase) validateBusinessRules(ctx context.Context, req *plansettingspb.ReadPlanSettingsRequest) error {
	businessType := uc.getBusinessTypeFromContext(ctx)

	// Validate plan settings ID format
	if len(req.Data.Id) < 3 {
		return errors.New(uc.getTranslatedMessage(ctx, businessType, "plan_settings.validation.id_too_short", "plan settings ID must be at least 3 characters long"))
	}

	return nil
}
