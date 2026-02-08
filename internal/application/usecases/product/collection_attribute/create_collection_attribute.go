package collection_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
)

// CreateCollectionAttributeUseCase handles the business logic for creating collection attributes
// CreateCollectionAttributeRepositories groups all repository dependencies
type CreateCollectionAttributeRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer // Primary entity repository
	Collection          collectionpb.CollectionDomainServiceServer
	Attribute           attributepb.AttributeDomainServiceServer
}

// CreateCollectionAttributeServices groups all business service dependencies
type CreateCollectionAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateCollectionAttributeUseCase handles the business logic for creating collection attributes
type CreateCollectionAttributeUseCase struct {
	repositories CreateCollectionAttributeRepositories
	services     CreateCollectionAttributeServices
}

// NewCreateCollectionAttributeUseCase creates a new CreateCollectionAttributeUseCase
func NewCreateCollectionAttributeUseCase(
	repositories CreateCollectionAttributeRepositories,
	services CreateCollectionAttributeServices,
) *CreateCollectionAttributeUseCase {
	return &CreateCollectionAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create collection attribute operation
func (uc *CreateCollectionAttributeUseCase) Execute(ctx context.Context, req *collectionattributepb.CreateCollectionAttributeRequest) (*collectionattributepb.CreateCollectionAttributeResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection attribute creation within a transaction
func (uc *CreateCollectionAttributeUseCase) executeWithTransaction(ctx context.Context, req *collectionattributepb.CreateCollectionAttributeRequest) (*collectionattributepb.CreateCollectionAttributeResponse, error) {
	var result *collectionattributepb.CreateCollectionAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "collection_attribute.errors.creation_failed", "Collection attribute creation failed [DEFAULT]"), err)
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
func (uc *CreateCollectionAttributeUseCase) executeCore(ctx context.Context, req *collectionattributepb.CreateCollectionAttributeRequest) (*collectionattributepb.CreateCollectionAttributeResponse, error) {
	// TODO: Re-enable workspace-scoped authorization check once WorkspaceId is available
	// userID, err := contextutil.RequireUserIDFromContext(ctx)
	// if err != nil {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for collection attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }
	// permission := ports.EntityPermission(ports.EntityCollectionAttribute, ports.ActionCreate)
	// hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	// if err != nil {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for collection attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }
	// if !hasPerm {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for collection attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.validation_failed", "Input validation failed [DEFAULT]"), err)
	}

	// Business logic and enrichment
	if err := uc.enrichCollectionAttributeData(req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]"), err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]"), err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]"), err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionAttribute.CreateCollectionAttribute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.creation_failed", "Collection attribute creation failed [DEFAULT]"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreateCollectionAttributeUseCase) validateInput(ctx context.Context, req *collectionattributepb.CreateCollectionAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.data_required", "Collection attribute data is required [DEFAULT]"))
	}
	if req.Data.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.collection_id_required", "Collection ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.value_required", "Attribute value is required [DEFAULT]"))
	}
	return nil
}

// enrichCollectionAttributeData adds generated fields and audit information
func (uc *CreateCollectionAttributeUseCase) enrichCollectionAttributeData(collectionAttribute *collectionattributepb.CollectionAttribute) error {
	now := time.Now()

	// Generate CollectionAttribute ID if not provided
	if collectionAttribute.Id == "" {
		if uc.services.IDService != nil {
			collectionAttribute.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback ID generation when service is not available
			collectionAttribute.Id = fmt.Sprintf("collection-attr-%d", now.UnixNano())
		}
	}

	// Set audit fields
	collectionAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	collectionAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	collectionAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	collectionAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for collection attributes
func (uc *CreateCollectionAttributeUseCase) validateBusinessRules(ctx context.Context, collectionAttribute *collectionattributepb.CollectionAttribute) error {
	// Validate collection ID format
	if len(collectionAttribute.CollectionId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.collection_id_min_length", "Collection ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate attribute ID format
	if len(collectionAttribute.AttributeId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.attribute_id_min_length", "Attribute ID must be at least 2 characters long [DEFAULT]"))
	}

	// Validate attribute value length
	value := strings.TrimSpace(collectionAttribute.Value)
	if len(value) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.value_not_empty", "Attribute value must not be empty [DEFAULT]"))
	}

	if len(value) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.value_max_length", "Attribute value cannot exceed 500 characters [DEFAULT]"))
	}

	// Normalize value (trim spaces)
	collectionAttribute.Value = strings.TrimSpace(collectionAttribute.Value)

	// Business constraint: Collection attribute must be associated with a valid collection
	if collectionAttribute.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.collection_association_required", "Collection attribute must be associated with a collection [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateCollectionAttributeUseCase) validateEntityReferences(ctx context.Context, collectionAttribute *collectionattributepb.CollectionAttribute) error {
	// Validate Collection entity reference
	if collectionAttribute.CollectionId != "" {
		collection, err := uc.repositories.Collection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{
			Data: &collectionpb.Collection{Id: collectionAttribute.CollectionId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.collection_reference_validation_failed", "Failed to validate collection entity reference [DEFAULT]"), err)
		}
		if collection == nil || collection.Data == nil || len(collection.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.collection_not_found", "Referenced collection with ID '%s' does not exist [DEFAULT]"), collectionAttribute.CollectionId)
		}
		if !collection.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.collection_not_active", "Referenced collection with ID '%s' is not active [DEFAULT]"), collectionAttribute.CollectionId)
		}
	}

	// Validate Attribute entity reference
	if collectionAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: collectionAttribute.AttributeId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]"), err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.attribute_not_found", "Referenced attribute with ID '%s' does not exist [DEFAULT]"), collectionAttribute.AttributeId)
		}
		if !attribute.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.attribute_not_active", "Referenced attribute with ID '%s' is not active [DEFAULT]"), collectionAttribute.AttributeId)
		}
	}

	return nil
}
