package price_product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

// ReadPriceProductRepositories groups all repository dependencies
type ReadPriceProductRepositories struct {
	PriceProduct priceproductpb.PriceProductDomainServiceServer // Primary entity repository
}

// ReadPriceProductServices groups all business service dependencies
type ReadPriceProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadPriceProductUseCase handles the business logic for reading a price product
type ReadPriceProductUseCase struct {
	repositories ReadPriceProductRepositories
	services     ReadPriceProductServices
}

// NewReadPriceProductUseCase creates use case with grouped dependencies
func NewReadPriceProductUseCase(
	repositories ReadPriceProductRepositories,
	services ReadPriceProductServices,
) *ReadPriceProductUseCase {
	return &ReadPriceProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read price product operation
func (uc *ReadPriceProductUseCase) Execute(ctx context.Context, req *priceproductpb.ReadPriceProductRequest) (*priceproductpb.ReadPriceProductResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceProduct, ports.ActionRead); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceProduct, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceProduct.ReadPriceProduct(ctx, req)
	if err != nil {
		// Handle not found error specifically - repository should return proper not found error
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.not_found", "Product pricing with ID \"{id}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{id}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPriceProductUseCase) validateInput(ctx context.Context, req *priceproductpb.ReadPriceProductRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.data_required", "Price Product data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.id_required", "Price Product ID is required [DEFAULT]"))
	}
	return nil
}
