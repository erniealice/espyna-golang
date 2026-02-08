package plan_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

// ReadPlanAttributeRepositories groups all repository dependencies
type ReadPlanAttributeRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
}

// ReadPlanAttributeServices groups all business service dependencies
type ReadPlanAttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadPlanAttributeUseCase handles the business logic for reading plan attributes
type ReadPlanAttributeUseCase struct {
	repositories ReadPlanAttributeRepositories
	services     ReadPlanAttributeServices
}

// NewReadPlanAttributeUseCase creates a new ReadPlanAttributeUseCase
func NewReadPlanAttributeUseCase(
	repositories ReadPlanAttributeRepositories,
	services ReadPlanAttributeServices,
) *ReadPlanAttributeUseCase {
	return &ReadPlanAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read plan attribute operation
func (uc *ReadPlanAttributeUseCase) Execute(ctx context.Context, req *planattributepb.ReadPlanAttributeRequest) (*planattributepb.ReadPlanAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.PlanAttribute.ReadPlanAttribute(ctx, req)
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
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPlanAttributeUseCase) validateInput(ctx context.Context, req *planattributepb.ReadPlanAttributeRequest) error {
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
