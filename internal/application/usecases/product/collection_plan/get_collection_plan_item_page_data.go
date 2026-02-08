package collection_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
)

type GetCollectionPlanItemPageDataRepositories struct {
	CollectionPlan collectionplanpb.CollectionPlanDomainServiceServer
}

type GetCollectionPlanItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetCollectionPlanItemPageDataUseCase handles the business logic for getting collection plan item page data
type GetCollectionPlanItemPageDataUseCase struct {
	repositories GetCollectionPlanItemPageDataRepositories
	services     GetCollectionPlanItemPageDataServices
}

// NewGetCollectionPlanItemPageDataUseCase creates a new GetCollectionPlanItemPageDataUseCase
func NewGetCollectionPlanItemPageDataUseCase(
	repositories GetCollectionPlanItemPageDataRepositories,
	services GetCollectionPlanItemPageDataServices,
) *GetCollectionPlanItemPageDataUseCase {
	return &GetCollectionPlanItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get collection plan item page data operation
func (uc *GetCollectionPlanItemPageDataUseCase) Execute(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanItemPageDataRequest,
) (*collectionplanpb.GetCollectionPlanItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.CollectionPlanId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection plan item page data retrieval within a transaction
func (uc *GetCollectionPlanItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanItemPageDataRequest,
) (*collectionplanpb.GetCollectionPlanItemPageDataResponse, error) {
	var result *collectionplanpb.GetCollectionPlanItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"collection_plan.errors.item_page_data_failed",
				"collection plan item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting collection plan item page data
func (uc *GetCollectionPlanItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanItemPageDataRequest,
) (*collectionplanpb.GetCollectionPlanItemPageDataResponse, error) {
	// Create read request for the collection plan
	readReq := &collectionplanpb.ReadCollectionPlanRequest{
		Data: &collectionplanpb.CollectionPlan{
			Id: req.CollectionPlanId,
		},
	}

	// Retrieve the collection plan
	readResp, err := uc.repositories.CollectionPlan.ReadCollectionPlan(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.errors.read_failed",
			"failed to retrieve collection plan: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.errors.not_found",
			"collection plan not found",
		))
	}

	// Get the collection plan (should be only one)
	collectionPlan := readResp.Data[0]

	// Validate that we got the expected collection plan
	if collectionPlan.Id != req.CollectionPlanId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.errors.id_mismatch",
			"retrieved collection plan ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (collection details, plan details, etc.) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access

	// For now, return the collection plan as-is
	return &collectionplanpb.GetCollectionPlanItemPageDataResponse{
		CollectionPlan: collectionPlan,
		Success:        true,
	}, nil
}

// validateInput validates the input request
func (uc *GetCollectionPlanItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.request_required",
			"request is required",
		))
	}

	if req.CollectionPlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.id_required",
			"collection plan ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading collection plan item page data
func (uc *GetCollectionPlanItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	collectionPlanId string,
) error {
	// Validate collection plan ID format
	if len(collectionPlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_plan.validation.id_too_short",
			"collection plan ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this collection plan
	// - Validate collection plan belongs to the current user's organization
	// - Check if collection plan is in a state that allows viewing
	// - Rate limiting for collection plan access
	// - Audit logging requirements

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like collection and plan details
// This would be called from executeCore if needed
func (uc *GetCollectionPlanItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	collectionPlan *collectionplanpb.CollectionPlan,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to collection and plan repositories
	// to populate the nested objects if they're not already loaded

	// Example implementation would be:
	// if collectionPlan.Collection == nil && collectionPlan.CollectionId != "" {
	//     // Load collection data
	// }
	// if collectionPlan.Plan == nil && collectionPlan.PlanId != "" {
	//     // Load plan data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetCollectionPlanItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	collectionPlan *collectionplanpb.CollectionPlan,
) *collectionplanpb.CollectionPlan {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields
	// - Applying localization
	// - Sanitizing sensitive data

	return collectionPlan
}

// checkAccessPermissions validates user has permission to access this collection plan
func (uc *GetCollectionPlanItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	collectionPlanId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating collection plan belongs to user's organization
	// - Applying multi-tenant access controls

	return nil
}
