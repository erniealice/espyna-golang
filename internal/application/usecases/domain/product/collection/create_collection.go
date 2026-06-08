package collection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

// CreateCollectionRepositories groups all repository dependencies
type CreateCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer // Primary entity repository
}

// CreateCollectionServices groups all business service dependencies
type CreateCollectionServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Transaction management
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateCollectionUseCase handles the business logic for creating collections
type CreateCollectionUseCase struct {
	repositories CreateCollectionRepositories
	services     CreateCollectionServices
}

// NewCreateCollectionUseCase creates use case with grouped dependencies
func NewCreateCollectionUseCase(
	repositories CreateCollectionRepositories,
	services CreateCollectionServices,
) *CreateCollectionUseCase {
	return &CreateCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create collection operation
func (uc *CreateCollectionUseCase) Execute(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Collection, entityid.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection creation within a transaction
func (uc *CreateCollectionUseCase) executeWithTransaction(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	var result *collectionpb.CreateCollectionResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "collection.errors.creation_failed", "Collection creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
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
func (uc *CreateCollectionUseCase) executeCore(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.Collection, entityid.ActionCreate)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichCollectionData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Collection.CreateCollection(ctx, req)
}

// validateInput validates the input request
func (uc *CreateCollectionUseCase) validateInput(ctx context.Context, req *collectionpb.CreateCollectionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.request_required", "Request is required for collections [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.data_required", "Collection data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.name_required", "Collection name is required [DEFAULT]"))
	}
	return nil
}

// enrichCollectionData adds generated fields and audit information
func (uc *CreateCollectionUseCase) enrichCollectionData(collection *collectionpb.Collection) error {
	now := time.Now()

	// Generate Collection ID if not provided
	if collection.Id == "" {
		collection.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	collection.DateCreated = &[]int64{now.UnixMilli()}[0]
	collection.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	collection.DateModified = &[]int64{now.UnixMilli()}[0]
	collection.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	collection.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateCollectionUseCase) validateBusinessRules(ctx context.Context, collection *collectionpb.Collection) error {
	// Validate collection name length
	name := strings.TrimSpace(collection.Name)
	if len(name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.name_too_short", "Collection name must be at least 2 characters long [DEFAULT]"))
	}

	if len(name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.name_too_long", "Collection name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if collection.Description != "" {
		description := strings.TrimSpace(collection.Description)
		if len(description) > 500 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.description_too_long", "Collection description cannot exceed 500 characters [DEFAULT]"))
		}
	}

	// Normalize name (trim spaces, proper capitalization)
	collection.Name = strings.TrimSpace(name)

	return nil
}
