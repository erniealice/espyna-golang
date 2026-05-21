package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// ListProductsRepositories groups all repository dependencies
type ListProductsRepositories struct {
	Product productpb.ProductDomainServiceServer // Primary entity repository
}

// ListProductsServices groups all business service dependencies
type ListProductsServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListProductsUseCase handles the business logic for listing products
type ListProductsUseCase struct {
	repositories ListProductsRepositories
	services     ListProductsServices
}

// NewListProductsUseCase creates a new ListProductsUseCase
func NewListProductsUseCase(
	repositories ListProductsRepositories,
	services ListProductsServices,
) *ListProductsUseCase {
	return &ListProductsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list products operation
func (uc *ListProductsUseCase) Execute(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityProduct, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProduct, ports.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Product.ListProducts(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.list_failed", "Failed to retrieve products [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductsUseCase) validateInput(ctx context.Context, req *productpb.ListProductsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
