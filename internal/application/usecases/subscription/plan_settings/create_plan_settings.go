package plan_settings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
)

// CreatePlanSettingsRepositories groups all repository dependencies
type CreatePlanSettingsRepositories struct {
	PlanSettings plansettingspb.PlanSettingsDomainServiceServer // Primary entity repository
	Plan         planpb.PlanDomainServiceServer                 // Entity reference dependency
}

// CreatePlanSettingsServices groups all business service dependencies
type CreatePlanSettingsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePlanSettingsUseCase handles the business logic for creating plan_settings
type CreatePlanSettingsUseCase struct {
	repositories CreatePlanSettingsRepositories
	services     CreatePlanSettingsServices
}

// NewCreatePlanSettingsUseCase creates a new CreatePlanSettingsUseCase
func NewCreatePlanSettingsUseCase(
	repositories CreatePlanSettingsRepositories,
	services CreatePlanSettingsServices,
) *CreatePlanSettingsUseCase {
	return &CreatePlanSettingsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create plan_settings operation
func (uc *CreatePlanSettingsUseCase) Execute(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) (*plansettingspb.CreatePlanSettingsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPlanSettings, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.authorization_failed", "Authorization failed for plan settings [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPlanSettings, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.authorization_failed", "Authorization failed for plan settings [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.authorization_failed", "Authorization failed for plan settings [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// validateInput validates the input request
func (uc *CreatePlanSettingsUseCase) validateInput(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.data_required", "plan settings data is required")
		return errors.New(msg)
	}
	return nil
}

// applyBusinessLogic applies business rules and returns enriched plan settings
func (uc *CreatePlanSettingsUseCase) applyBusinessLogic(planSettings *plansettingspb.PlanSettings) *plansettingspb.PlanSettings {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if planSettings.Id == "" {
		planSettings.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Set active status for new plan settings
	planSettings.Active = true

	// Business logic: Set creation audit fields
	planSettings.DateCreated = &[]int64{now.Unix()}[0]
	planSettings.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	planSettings.DateModified = &[]int64{now.Unix()}[0]
	planSettings.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return planSettings
}

// validateBusinessRules enforces business constraints
func (uc *CreatePlanSettingsUseCase) validateBusinessRules(ctx context.Context, planSettings *plansettingspb.PlanSettings) error {
	// Business rule: Required data validation
	if planSettings == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.data_required", "plan settings data is required"))
	}
	if planSettings.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.name_required", "plan settings name is required"))
	}
	if planSettings.PlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.validation.plan_id_required", "plan ID is required"))
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
func (uc *CreatePlanSettingsUseCase) validateEntityReferences(ctx context.Context, planSettings *plansettingspb.PlanSettings) error {
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

// executeWithTransaction executes plan settings creation within a transaction
func (uc *CreatePlanSettingsUseCase) executeWithTransaction(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) (*plansettingspb.CreatePlanSettingsResponse, error) {
	var result *plansettingspb.CreatePlanSettingsResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_settings.errors.creation_failed", "plan settings creation failed: %s"), err.Error())
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreatePlanSettingsUseCase) executeCore(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) (*plansettingspb.CreatePlanSettingsResponse, error) {
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
	return uc.repositories.PlanSettings.CreatePlanSettings(ctx, &plansettingspb.CreatePlanSettingsRequest{
		Data: enrichedPlanSettings,
	})
}
