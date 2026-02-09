package plan

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// UpdatePlanRepositories groups all repository dependencies
type UpdatePlanRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// UpdatePlanServices groups all business service dependencies
type UpdatePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdatePlanUseCase handles the business logic for updating plans
type UpdatePlanUseCase struct {
	repositories UpdatePlanRepositories
	services     UpdatePlanServices
}

// NewUpdatePlanUseCase creates a new UpdatePlanUseCase
func NewUpdatePlanUseCase(
	repositories UpdatePlanRepositories,
	services UpdatePlanServices,
) *UpdatePlanUseCase {
	return &UpdatePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update plan operation
func (uc *UpdatePlanUseCase) Execute(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPlan, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPlanData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	_, err := uc.repositories.Plan.UpdatePlan(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.errors.update_failed", "plan update failed [DEFAULT]"))
	}

	return &planpb.UpdatePlanResponse{
		Success: true,
		Data:    []*planpb.Plan{req.Data},
	}, nil
}

// validateInput validates the input request
func (uc *UpdatePlanUseCase) validateInput(ctx context.Context, req *planpb.UpdatePlanRequest) error {
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

// enrichPlanData adds audit information for updates
func (uc *UpdatePlanUseCase) enrichPlanData(plan *planpb.Plan) error {
	now := time.Now()

	// Update modification timestamp
	plan.DateModified = &[]int64{now.UnixMilli()}[0]
	plan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdatePlanUseCase) validateBusinessRules(ctx context.Context, plan *planpb.Plan) error {
	// Validate plan ID format
	if plan.Id == nil || len(*plan.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.id_too_short", "plan ID must be at least 3 characters long"))
	}

	// Validate name is required
	if strings.TrimSpace(plan.Name) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.name_required", "plan name is required"))
	}

	if len(plan.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.name_too_long", "plan name cannot exceed 100 characters"))
	}

	// Validate description length (only if provided)
	if plan.Description != nil && len(*plan.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.description_too_long", "plan description cannot exceed 500 characters"))
	}

	return nil
}
