package collection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

// UpdateCollectionRepositories groups all repository dependencies
type UpdateCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer // Primary entity repository
}

// UpdateCollectionServices groups all business service dependencies
type UpdateCollectionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateCollectionUseCase handles the business logic for updating collections
type UpdateCollectionUseCase struct {
	repositories UpdateCollectionRepositories
	services     UpdateCollectionServices
}

// NewUpdateCollectionUseCase creates use case with grouped dependencies
func NewUpdateCollectionUseCase(
	repositories UpdateCollectionRepositories,
	services UpdateCollectionServices,
) *UpdateCollectionUseCase {
	return &UpdateCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update collection operation
func (uc *UpdateCollectionUseCase) Execute(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCollection, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Use transaction service if available, otherwise execute directly.
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection update within a transaction
func (uc *UpdateCollectionUseCase) executeWithTransaction(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	var result *collectionpb.UpdateCollectionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			// Wrapping the error inside the transaction ensures it can be rolled back.
			return fmt.Errorf("collection update failed within transaction: %w", err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for updating a collection.
func (uc *UpdateCollectionUseCase) executeCore(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {

	// 1. First, perform basic input validation.
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// 2. Check if the collection exists before attempting to update it.
	if _, err := uc.repositories.Collection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{Data: &collectionpb.Collection{Id: req.Data.Id}}); err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("collection with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"collection.errors.not_found",
				map[string]interface{}{"collectionId": req.Data.Id},
				"Collection not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// For other errors during the read check, return the error directly
		return nil, err
	}

	// 3. Enrich and validate business rules for the new data.
	if err := uc.enrichCollectionData(req.Data); err != nil {
		return nil, err
	}
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// 4. Call repository to perform the update.
	resp, err := uc.repositories.Collection.UpdateCollection(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateCollectionUseCase) validateInput(ctx context.Context, req *collectionpb.UpdateCollectionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.request_required", "Request is required for course collections [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.data_required", "Course collection data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.id_required", "Course collection ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.name_required", "Course collection name is required [DEFAULT]"))
	}
	return nil
}

// enrichCollectionData adds generated fields and audit information
func (uc *UpdateCollectionUseCase) enrichCollectionData(collection *collectionpb.Collection) error {
	now := time.Now()

	// Update audit fields
	collection.DateModified = &[]int64{now.UnixMilli()}[0]
	collection.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateCollectionUseCase) validateBusinessRules(ctx context.Context, collection *collectionpb.Collection) error {
	// Validate collection name length
	name := strings.TrimSpace(collection.Name)
	if len(name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.name_too_short", "Collection name must be at least 2 characters long [DEFAULT]"))
	}

	if len(name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.name_too_long", "Collection name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if collection.Description != "" {
		description := strings.TrimSpace(collection.Description)
		if len(description) > 500 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.description_too_long", "Collection description cannot exceed 500 characters [DEFAULT]"))
		}
	}

	// Normalize name (trim spaces, proper capitalization)
	collection.Name = strings.Title(strings.ToLower(name))

	return nil
}
