package product_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// ReadProductAttributeUseCase handles the business logic for reading a product attribute
// ReadProductAttributeRepositories groups all repository dependencies
type ReadProductAttributeRepositories struct {
	ProductAttribute productattributepb.ProductAttributeDomainServiceServer // Primary entity repository
}

// ReadProductAttributeServices groups all business service dependencies
type ReadProductAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadProductAttributeUseCase handles the business logic for reading a product attribute
type ReadProductAttributeUseCase struct {
	repositories ReadProductAttributeRepositories
	services     ReadProductAttributeServices
}

// NewReadProductAttributeUseCase creates a new ReadProductAttributeUseCase
func NewReadProductAttributeUseCase(
	repositories ReadProductAttributeRepositories,
	services ReadProductAttributeServices,
) *ReadProductAttributeUseCase {
	return &ReadProductAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product attribute operation
func (uc *ReadProductAttributeUseCase) Execute(ctx context.Context, req *productattributepb.ReadProductAttributeRequest) (*productattributepb.ReadProductAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductAttribute, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.ProductAttribute.ReadProductAttribute(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_attribute.errors.not_found", map[string]interface{}{"productAttributeId": req.Data.Id}, "Product attribute not found")
			return nil, errors.New(translatedError)
		}
		// Other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.read_failed", "Failed to read product attribute")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductAttributeUseCase) validateInput(ctx context.Context, req *productattributepb.ReadProductAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.data_required", "product attribute data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.id_required", "product attribute ID is required"))
	}
	return nil
}
