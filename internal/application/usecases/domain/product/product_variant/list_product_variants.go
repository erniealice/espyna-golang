package product_variant

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

// ListProductVariantsRepositories groups all repository dependencies
type ListProductVariantsRepositories struct {
	ProductVariant productvariantpb.ProductVariantDomainServiceServer // Primary entity repository
}

// ListProductVariantsServices groups all business service dependencies
type ListProductVariantsServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListProductVariantsUseCase handles the business logic for listing product variants
type ListProductVariantsUseCase struct {
	repositories ListProductVariantsRepositories
	services     ListProductVariantsServices
}

// NewListProductVariantsUseCase creates a new ListProductVariantsUseCase
func NewListProductVariantsUseCase(
	repositories ListProductVariantsRepositories,
	services ListProductVariantsServices,
) *ListProductVariantsUseCase {
	return &ListProductVariantsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product variants operation
func (uc *ListProductVariantsUseCase) Execute(ctx context.Context, req *productvariantpb.ListProductVariantsRequest) (*productvariantpb.ListProductVariantsResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ProductVariant,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.ProductVariant, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductVariant.ListProductVariants(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.errors.list_failed", "Failed to retrieve product variants [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductVariantsUseCase) validateInput(ctx context.Context, req *productvariantpb.ListProductVariantsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
