package product_collection

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
)

// ListProductCollectionsUseCase handles the business logic for listing product collections
// ListProductCollectionsRepositories groups all repository dependencies
type ListProductCollectionsRepositories struct {
	ProductCollection productcollectionpb.ProductCollectionDomainServiceServer // Primary entity repository
}

// ListProductCollectionsServices groups all business service dependencies
type ListProductCollectionsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListProductCollectionsUseCase handles the business logic for listing product collections
type ListProductCollectionsUseCase struct {
	repositories ListProductCollectionsRepositories
	services     ListProductCollectionsServices
}

// NewListProductCollectionsUseCase creates a new ListProductCollectionsUseCase
func NewListProductCollectionsUseCase(
	repositories ListProductCollectionsRepositories,
	services ListProductCollectionsServices,
) *ListProductCollectionsUseCase {
	return &ListProductCollectionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product collections operation
func (uc *ListProductCollectionsUseCase) Execute(ctx context.Context, req *productcollectionpb.ListProductCollectionsRequest) (*productcollectionpb.ListProductCollectionsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductCollection, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductCollection, ports.ActionList)
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
	resp, err := uc.repositories.ProductCollection.ListProductCollections(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.list_failed", "Failed to retrieve product collections [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductCollectionsUseCase) validateInput(ctx context.Context, req *productcollectionpb.ListProductCollectionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
