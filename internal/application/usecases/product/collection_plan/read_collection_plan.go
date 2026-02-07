package collection_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	collectionplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_plan"
)

// ReadCollectionPlanRepositories groups all repository dependencies
type ReadCollectionPlanRepositories struct {
	CollectionPlan collectionplanpb.CollectionPlanDomainServiceServer // Primary entity repository
}

// ReadCollectionPlanServices groups all business service dependencies
type ReadCollectionPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadCollectionPlanUseCase handles the business logic for reading collection plans
type ReadCollectionPlanUseCase struct {
	repositories ReadCollectionPlanRepositories
	services     ReadCollectionPlanServices
}

// NewReadCollectionPlanUseCase creates a new ReadCollectionPlanUseCase
func NewReadCollectionPlanUseCase(
	repositories ReadCollectionPlanRepositories,
	services ReadCollectionPlanServices,
) *ReadCollectionPlanUseCase {
	return &ReadCollectionPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read collection plan operation
func (uc *ReadCollectionPlanUseCase) Execute(ctx context.Context, req *collectionplanpb.ReadCollectionPlanRequest) (*collectionplanpb.ReadCollectionPlanResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.authorization_failed", "Authorization failed for collection plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollectionPlan, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.authorization_failed", "Authorization failed for collection plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.authorization_failed", "Authorization failed for collection plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionPlan.ReadCollectionPlan(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.read_failed", "Failed to retrieve collection plan [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Not found error
	if resp == nil || resp.Data == nil || len(resp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.not_found", "Collection Plan with ID \"{collectionPlanId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{collectionPlanId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadCollectionPlanUseCase) validateInput(ctx context.Context, req *collectionplanpb.ReadCollectionPlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.request_required", "Request is required for collection plans [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.data_required", "Collection Plan data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.id_required", "Collection Plan ID is required [DEFAULT]"))
	}
	return nil
}
