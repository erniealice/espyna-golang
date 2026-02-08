package collection_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// CreateCollectionPlanRepositories groups all repository dependencies
type CreateCollectionPlanRepositories struct {
	CollectionPlan collectionplanpb.CollectionPlanDomainServiceServer // Primary entity repository
	Collection     collectionpb.CollectionDomainServiceServer         // Entity reference dependency
	Plan           planpb.PlanDomainServiceServer                     // Entity reference dependency
}

// CreateCollectionPlanServices groups all business service dependencies
type CreateCollectionPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateCollectionPlanUseCase handles the business logic for creating collection plans
type CreateCollectionPlanUseCase struct {
	repositories CreateCollectionPlanRepositories
	services     CreateCollectionPlanServices
}

// NewCreateCollectionPlanUseCase creates a new CreateCollectionPlanUseCase
func NewCreateCollectionPlanUseCase(
	repositories CreateCollectionPlanRepositories,
	services CreateCollectionPlanServices,
) *CreateCollectionPlanUseCase {
	return &CreateCollectionPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create collection plan operation
func (uc *CreateCollectionPlanUseCase) Execute(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) (*collectionplanpb.CreateCollectionPlanResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.authorization_failed", "Authorization failed for collection plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollectionPlan, ports.ActionCreate)
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

	// Business logic and enrichment
	if err := uc.enrichCollectionPlanData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection plan creation within a transaction
func (uc *CreateCollectionPlanUseCase) executeWithTransaction(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) (*collectionplanpb.CreateCollectionPlanResponse, error) {
	var result *collectionplanpb.CreateCollectionPlanResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("collection plan creation failed: %w", err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateCollectionPlanUseCase) executeCore(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) (*collectionplanpb.CreateCollectionPlanResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichCollectionPlanData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionPlan.CreateCollectionPlan(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.creation_failed", "Collection Plan creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateCollectionPlanUseCase) validateInput(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.request_required", "Request is required for collection plans [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.data_required", "Collection Plan data is required [DEFAULT]"))
	}
	if req.Data.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.collection_id_required", "Collection ID is required [DEFAULT]"))
	}
	if req.Data.PlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.plan_id_required", "Plan ID is required [DEFAULT]"))
	}
	return nil
}

// enrichCollectionPlanData adds generated fields and audit information
func (uc *CreateCollectionPlanUseCase) enrichCollectionPlanData(collectionPlan *collectionplanpb.CollectionPlan) error {
	now := time.Now()

	// Generate CollectionPlan ID if not provided
	if collectionPlan.Id == "" {
		collectionPlan.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	collectionPlan.DateCreated = &[]int64{now.UnixMilli()}[0]
	collectionPlan.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	collectionPlan.DateModified = &[]int64{now.UnixMilli()}[0]
	collectionPlan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	collectionPlan.Active = true

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateCollectionPlanUseCase) validateEntityReferences(ctx context.Context, collectionPlan *collectionplanpb.CollectionPlan) error {
	// Validate Collection entity reference
	if collectionPlan.CollectionId != "" {
		collection, err := uc.repositories.Collection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{
			Data: &collectionpb.Collection{Id: collectionPlan.CollectionId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.collection_reference_validation_failed", "Failed to validate collection entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if collection == nil || collection.Data == nil || len(collection.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.collection_not_found", "Referenced collection with ID '{collectionId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{collectionId}", collectionPlan.CollectionId)
			return errors.New(translatedError)
		}
		if !collection.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.collection_not_active", "Referenced collection with ID '{collectionId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{collectionId}", collectionPlan.CollectionId)
			return errors.New(translatedError)
		}
	}

	// Validate Plan entity reference
	if collectionPlan.PlanId != "" {
		planId := collectionPlan.PlanId
		plan, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
			Data: &planpb.Plan{Id: &planId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.plan_reference_validation_failed", "Failed to validate plan entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if plan == nil || plan.Data == nil || len(plan.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.plan_not_found", "Referenced plan with ID '{planId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{planId}", collectionPlan.PlanId)
			return errors.New(translatedError)
		}
		if !plan.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.errors.plan_not_active", "Referenced plan with ID '{planId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{planId}", collectionPlan.PlanId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateCollectionPlanUseCase) validateBusinessRules(ctx context.Context, collectionPlan *collectionplanpb.CollectionPlan) error {
	// Check for duplicate collection-plan association
	if err := uc.validateUniqueAssociation(ctx, collectionPlan.CollectionId, collectionPlan.PlanId); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.duplicate_association", "Duplicate collection-plan association [DEFAULT]")
		return errors.New(translatedError)
	}

	return nil
}

// validateCollectionExists checks if the collection exists
func (uc *CreateCollectionPlanUseCase) validateCollectionExists(ctx context.Context, collectionID string) error {
	// This would typically query the collection repository
	// For now, we'll implement a placeholder
	// TODO: Implement actual collection existence check
	if collectionID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.collection_id_empty", "Collection ID cannot be empty [DEFAULT]"))
	}
	return nil
}

// validatePlanExists checks if the plan exists
func (uc *CreateCollectionPlanUseCase) validatePlanExists(ctx context.Context, planID string) error {
	// This would typically query the plan repository
	// For now, we'll implement a placeholder
	// TODO: Implement actual plan existence check
	if planID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_plan.validation.plan_id_empty", "Plan ID cannot be empty [DEFAULT]"))
	}
	return nil
}

// validateUniqueAssociation ensures no duplicate collection-plan associations
func (uc *CreateCollectionPlanUseCase) validateUniqueAssociation(ctx context.Context, collectionID, planID string) error {
	// businessType := uc.getBusinessTypeFromContext(ctx)

	// This would typically query the collection plan repository to check for duplicates
	// For now, we'll implement a placeholder
	// TODO: Implement actual duplicate check
	return nil
}
