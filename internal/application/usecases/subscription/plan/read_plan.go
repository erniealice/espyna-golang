package plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// ReadPlanRepositories groups all repository dependencies
type ReadPlanRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// ReadPlanServices groups all business service dependencies
type ReadPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadPlanUseCase handles the business logic for reading plans
type ReadPlanUseCase struct {
	repositories ReadPlanRepositories
	services     ReadPlanServices
}

// NewReadPlanUseCase creates a new ReadPlanUseCase
func NewReadPlanUseCase(
	repositories ReadPlanRepositories,
	services ReadPlanServices,
) *ReadPlanUseCase {
	return &ReadPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read plan operation
func (uc *ReadPlanUseCase) Execute(ctx context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.Plan.ReadPlan(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.errors.not_found", "plan not found [DEFAULT]"))
	}

	return result, nil
}

// validateInput validates the input request
func (uc *ReadPlanUseCase) validateInput(ctx context.Context, req *planpb.ReadPlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.data_required", "plan data is required"))
	}
	if req.Data.Id == nil || *req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.id_required", "plan ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for plan reading
func (uc *ReadPlanUseCase) validateBusinessRules(ctx context.Context, req *planpb.ReadPlanRequest) error {
	// Validate plan ID format
	if req.Data.Id == nil || len(*req.Data.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.id_too_short", "plan ID must be at least 3 characters long"))
	}

	return nil
}
