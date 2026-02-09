package collection_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
)

// DeleteCollectionPlanRepositories groups all repository dependencies
type DeleteCollectionPlanRepositories struct {
	CollectionPlan collectionplanpb.CollectionPlanDomainServiceServer // Primary entity repository
}

// DeleteCollectionPlanServices groups all business service dependencies
type DeleteCollectionPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteCollectionPlanUseCase handles the business logic for deleting collection plans
type DeleteCollectionPlanUseCase struct {
	repositories DeleteCollectionPlanRepositories
	services     DeleteCollectionPlanServices
}

// NewDeleteCollectionPlanUseCase creates a new DeleteCollectionPlanUseCase
func NewDeleteCollectionPlanUseCase(
	repositories DeleteCollectionPlanRepositories,
	services DeleteCollectionPlanServices,
) *DeleteCollectionPlanUseCase {
	return &DeleteCollectionPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete collection plan operation
func (uc *DeleteCollectionPlanUseCase) Execute(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) (*collectionplanpb.DeleteCollectionPlanResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCollectionPlan, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection plan deletion within a transaction
func (uc *DeleteCollectionPlanUseCase) executeWithTransaction(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) (*collectionplanpb.DeleteCollectionPlanResponse, error) {
	var result *collectionplanpb.DeleteCollectionPlanResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a collection plan
func (uc *DeleteCollectionPlanUseCase) executeCore(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) (*collectionplanpb.DeleteCollectionPlanResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.authorization_failed", "Authorization failed for collection plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollectionPlan, ports.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionPlan.DeleteCollectionPlan(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.deletion_failed", "Collection Plan deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteCollectionPlanUseCase) validateInput(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.data_required", "Collection Plan data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.id_required", "Collection Plan ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints before deletion
func (uc *DeleteCollectionPlanUseCase) validateBusinessRules(ctx context.Context, collectionPlan *collectionplanpb.CollectionPlan) error {
	// Check if there are active subscriptions using this collection plan
	if hasActiveSubscriptions, err := uc.hasActiveSubscriptions(ctx, collectionPlan.Id); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.active_subscriptions_check_failed", "Failed to check for active subscriptions [DEFAULT]")
		return fmt.Errorf("%s: %w", translatedError, err)
	} else if hasActiveSubscriptions {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.cannot_delete_active_subscriptions", "Cannot delete collection plan with active subscriptions [DEFAULT]"))
	}

	return nil
}

// hasActiveSubscriptions checks if there are active subscriptions using this collection plan
func (uc *DeleteCollectionPlanUseCase) hasActiveSubscriptions(ctx context.Context, collectionPlanID string) (bool, error) {
	// This would typically query the subscription repository
	// For now, we'll return false as a placeholder
	// TODO: Implement actual check for active subscriptions
	return false, nil
}
