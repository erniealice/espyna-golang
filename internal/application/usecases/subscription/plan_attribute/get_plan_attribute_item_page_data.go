package plan_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

// GetPlanAttributeItemPageDataRepositories groups all repository dependencies
type GetPlanAttributeItemPageDataRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
}

// GetPlanAttributeItemPageDataServices groups all business service dependencies
type GetPlanAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetPlanAttributeItemPageDataUseCase handles the business logic for getting plan attribute item page data
type GetPlanAttributeItemPageDataUseCase struct {
	repositories GetPlanAttributeItemPageDataRepositories
	services     GetPlanAttributeItemPageDataServices
}

// NewGetPlanAttributeItemPageDataUseCase creates a new GetPlanAttributeItemPageDataUseCase
func NewGetPlanAttributeItemPageDataUseCase(
	repositories GetPlanAttributeItemPageDataRepositories,
	services GetPlanAttributeItemPageDataServices,
) *GetPlanAttributeItemPageDataUseCase {
	return &GetPlanAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get plan attribute item page data operation
func (uc *GetPlanAttributeItemPageDataUseCase) Execute(ctx context.Context, req *planattributepb.GetPlanAttributeItemPageDataRequest) (*planattributepb.GetPlanAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PlanAttribute.GetPlanAttributeItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.item_page_data_failed", "Failed to retrieve plan attribute item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetPlanAttributeItemPageDataUseCase) validateInput(ctx context.Context, req *planattributepb.GetPlanAttributeItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.request_required", "Request is required for plan attributes [DEFAULT]"))
	}

	// Validate plan attribute ID - uses direct field req.PlanAttributeId
	if strings.TrimSpace(req.PlanAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.id_required", "Plan attribute ID is required [DEFAULT]"))
	}

	// Basic ID format validation
	if len(req.PlanAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.id_too_short", "Plan attribute ID must be at least 3 characters [DEFAULT]"))
	}

	return nil
}
