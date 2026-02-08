package product_collection

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
)

// ReadProductCollectionUseCase handles the business logic for reading a product collection
// ReadProductCollectionRepositories groups all repository dependencies
type ReadProductCollectionRepositories struct {
	ProductCollection productcollectionpb.ProductCollectionDomainServiceServer // Primary entity repository
}

// ReadProductCollectionServices groups all business service dependencies
type ReadProductCollectionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadProductCollectionUseCase handles the business logic for reading a product collection
type ReadProductCollectionUseCase struct {
	repositories ReadProductCollectionRepositories
	services     ReadProductCollectionServices
}

// NewReadProductCollectionUseCase creates a new ReadProductCollectionUseCase
func NewReadProductCollectionUseCase(
	repositories ReadProductCollectionRepositories,
	services ReadProductCollectionServices,
) *ReadProductCollectionUseCase {
	return &ReadProductCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product collection operation
func (uc *ReadProductCollectionUseCase) Execute(ctx context.Context, req *productcollectionpb.ReadProductCollectionRequest) (*productcollectionpb.ReadProductCollectionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductCollection, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductCollection.ReadProductCollection(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if resp == nil || resp.Data == nil || len(resp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_collection.errors.not_found", map[string]interface{}{"productCollectionId": req.Data.Id}, "Course-collection mapping not found")
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductCollectionUseCase) validateInput(ctx context.Context, req *productcollectionpb.ReadProductCollectionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.data_required", "Product collection data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.id_required", "Product collection ID is required [DEFAULT]"))
	}
	return nil
}
