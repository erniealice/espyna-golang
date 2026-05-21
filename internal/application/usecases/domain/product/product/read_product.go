package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// ReadProductRepositories groups all repository dependencies
type ReadProductRepositories struct {
	Product productpb.ProductDomainServiceServer // Primary entity repository
}

// ReadProductServices groups all business service dependencies
type ReadProductServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Transaction management
	Translator ports.Translator
}

// ReadProductUseCase handles the business logic for reading a product
type ReadProductUseCase struct {
	repositories ReadProductRepositories
	services     ReadProductServices
}

// NewReadProductUseCase creates use case with grouped dependencies
func NewReadProductUseCase(
	repositories ReadProductRepositories,
	services ReadProductServices,
) *ReadProductUseCase {
	return &ReadProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product operation
func (uc *ReadProductUseCase) Execute(ctx context.Context, req *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityProduct, ports.ActionRead); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProduct, ports.ActionRead)
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
	resp, err := uc.repositories.Product.ReadProduct(ctx, req)
	if err != nil {
		// Handle not found errors by checking for specific patterns in error message
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.not_found", "Product with ID \"{productId}\" not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", req.Data.Id)
			return nil, errors.New(translatedError)
		}

		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.read_failed", "Failed to retrieve product [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductUseCase) validateInput(ctx context.Context, req *productpb.ReadProductRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.data_required", "Product data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.id_required", "Product ID is required [DEFAULT]"))
	}
	return nil
}
