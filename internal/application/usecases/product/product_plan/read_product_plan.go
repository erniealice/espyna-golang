package product_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
)

// ReadProductPlanUseCase handles the business logic for reading a product plan
type ReadProductPlanUseCase struct {
	repositories ReadProductPlanRepositories
	services     ReadProductPlanServices
}

// NewReadProductPlanUseCase creates a new ReadProductPlanUseCase
func NewReadProductPlanUseCase(
	repositories ReadProductPlanRepositories,
	services ReadProductPlanServices,
) *ReadProductPlanUseCase {
	return &ReadProductPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product plan operation
func (uc *ReadProductPlanUseCase) Execute(ctx context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductPlan, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInputWithTranslation(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductPlan.ReadProductPlan(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_plan.errors.not_found", map[string]interface{}{"productPlanId": req.Data.Id}, "Course plan not found")
			return nil, errors.New(translatedError)
		}
		// Other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.read_failed", "Failed to read course plan")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Not found error
	if resp == nil || resp.Data == nil || len(resp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.not_found", "Product Plan with ID \"{id}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{id}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductPlanUseCase) validateInput(req *productplanpb.ReadProductPlanRequest) error {
	// This function is deprecated and should not be used directly. Use validateInputWithTranslation instead.
	return errors.New("deprecated: use validateInputWithTranslation")
}

// validateInputWithTranslation validates the input request with translated messages
func (uc *ReadProductPlanUseCase) validateInputWithTranslation(ctx context.Context, req *productplanpb.ReadProductPlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.request_required", "Request is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.data_required", "Product plan data is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.id_required", "Product plan ID is required [DEFAULT]")
		return errors.New(msg)
	}
	return nil
}
