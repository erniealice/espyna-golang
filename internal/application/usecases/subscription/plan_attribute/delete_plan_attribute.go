package plan_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

// DeletePlanAttributeRepositories groups all repository dependencies
type DeletePlanAttributeRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
}

// DeletePlanAttributeServices groups all business service dependencies
type DeletePlanAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeletePlanAttributeUseCase handles the business logic for deleting plan attributes
type DeletePlanAttributeUseCase struct {
	repositories DeletePlanAttributeRepositories
	services     DeletePlanAttributeServices
}

// NewDeletePlanAttributeUseCase creates a new DeletePlanAttributeUseCase
func NewDeletePlanAttributeUseCase(
	repositories DeletePlanAttributeRepositories,
	services DeletePlanAttributeServices,
) *DeletePlanAttributeUseCase {
	return &DeletePlanAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete plan attribute operation
func (uc *DeletePlanAttributeUseCase) Execute(ctx context.Context, req *planattributepb.DeletePlanAttributeRequest) (*planattributepb.DeletePlanAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.PlanAttribute.DeletePlanAttribute(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("plan_attribute with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"plan_attribute.errors.not_found",
				map[string]interface{}{"planAttributeId": req.Data.Id},
				"Plan attribute not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.deletion_failed", "Plan attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeletePlanAttributeUseCase) validateInput(ctx context.Context, req *planattributepb.DeletePlanAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.id_required", "Plan attribute ID is required [DEFAULT]"))
	}
	return nil
}
