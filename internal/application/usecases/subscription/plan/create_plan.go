package plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// CreatePlanRepositories groups all repository dependencies
type CreatePlanRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// CreatePlanServices groups all business service dependencies
type CreatePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePlanUseCase handles the business logic for creating plans
type CreatePlanUseCase struct {
	repositories CreatePlanRepositories
	services     CreatePlanServices
}

// NewCreatePlanUseCase creates use case with grouped dependencies
func NewCreatePlanUseCase(
	repositories CreatePlanRepositories,
	services CreatePlanServices,
) *CreatePlanUseCase {
	return &CreatePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create plan operation
func (uc *CreatePlanUseCase) Execute(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the create plan operation within a transaction
func (uc *CreatePlanUseCase) executeWithTransaction(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	var result *planpb.CreatePlanResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core create plan operation
func (uc *CreatePlanUseCase) executeCore(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.errors.authorization_failed", "Authorization failed for academic year plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPlan, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.errors.authorization_failed", "Authorization failed for academic year plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.errors.authorization_failed", "Authorization failed for academic year plans [DEFAULT]")
		return nil, errors.New(translatedError)
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
	response, err := uc.repositories.Plan.CreatePlan(ctx, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateInput validates the input request
func (uc *CreatePlanUseCase) validateInput(ctx context.Context, req *planpb.CreatePlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.data_required", "plan data is required [DEFAULT]"))
	}
	return nil
}

// enrichPlanData adds generated fields and audit information
func (uc *CreatePlanUseCase) enrichPlanData(plan *planpb.Plan) error {
	now := time.Now()

	// Always generate a new Plan ID, overriding any passed ID
	var newId string
	if uc.services.IDService != nil {
		newId = uc.services.IDService.GenerateID()
	} else {
		newId = fmt.Sprintf("plan-%d", now.UnixNano())
	}
	plan.Id = &newId

	// Set audit fields
	plan.DateCreated = &[]int64{now.UnixMilli()}[0]
	plan.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	plan.DateModified = &[]int64{now.UnixMilli()}[0]
	plan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	plan.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for plans
func (uc *CreatePlanUseCase) validateBusinessRules(ctx context.Context, plan *planpb.Plan) error {
	// Validate name is required
	if strings.TrimSpace(plan.Name) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.name_required", "plan name is required [DEFAULT]"))
	}

	// Validate name length
	if len(plan.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.name_too_short", "plan name must be at least 3 characters long [DEFAULT]"))
	}
	if len(plan.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.name_too_long", "plan name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length (only if provided)
	if plan.Description != nil && len(*plan.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.description_too_long", "plan description cannot exceed 500 characters [DEFAULT]"))
	}

	return nil
}
