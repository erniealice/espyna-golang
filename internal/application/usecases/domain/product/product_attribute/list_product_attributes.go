package product_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// ListProductAttributesUseCase handles the business logic for listing product attributes
// ListProductAttributesRepositories groups all repository dependencies
type ListProductAttributesRepositories struct {
	ProductAttribute productattributepb.ProductAttributeDomainServiceServer // Primary entity repository
}

// ListProductAttributesServices groups all business service dependencies
type ListProductAttributesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListProductAttributesUseCase handles the business logic for listing product attributes
type ListProductAttributesUseCase struct {
	repositories ListProductAttributesRepositories
	services     ListProductAttributesServices
}

// NewListProductAttributesUseCase creates a new ListProductAttributesUseCase
func NewListProductAttributesUseCase(
	repositories ListProductAttributesRepositories,
	services ListProductAttributesServices,
) *ListProductAttributesUseCase {
	return &ListProductAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product attributes operation
func (uc *ListProductAttributesUseCase) Execute(ctx context.Context, req *productattributepb.ListProductAttributesRequest) (*productattributepb.ListProductAttributesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ProductAttribute,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.ProductAttribute, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductAttribute.ListProductAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_attribute.errors.list_failed", "Failed to retrieve product attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductAttributesUseCase) validateInput(ctx context.Context, req *productattributepb.ListProductAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
