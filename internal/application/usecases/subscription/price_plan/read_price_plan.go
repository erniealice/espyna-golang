package price_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// ReadPricePlanRepositories groups all repository dependencies
type ReadPricePlanRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
}

// ReadPricePlanServices groups all business service dependencies
type ReadPricePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadPricePlanUseCase handles the business logic for reading price_plans
type ReadPricePlanUseCase struct {
	repositories ReadPricePlanRepositories
	services     ReadPricePlanServices
}

// NewReadPricePlanUseCase creates use case with grouped dependencies
func NewReadPricePlanUseCase(
	repositories ReadPricePlanRepositories,
	services ReadPricePlanServices,
) *ReadPricePlanUseCase {
	return &ReadPricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read price_plan operation
func (uc *ReadPricePlanUseCase) Execute(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.PricePlan.ReadPricePlan(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// validateInput validates the input request
func (uc *ReadPricePlanUseCase) validateInput(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.data_required", "price plan data is required")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.id_required", "price plan ID is required")
		return errors.New(msg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for price_plan reading
func (uc *ReadPricePlanUseCase) validateBusinessRules(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) error {
	// Validate price plan ID format
	if req.Data != nil && len(req.Data.Id) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.id_min_length", "price plan ID must be at least 3 characters long")
		return errors.New(msg)
	}

	return nil
}
