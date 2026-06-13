package product_variant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

// CreateProductVariantRepositories groups all repository dependencies
type CreateProductVariantRepositories struct {
	ProductVariant productvariantpb.ProductVariantDomainServiceServer // Primary entity repository
}

// CreateProductVariantServices groups all business service dependencies
type CreateProductVariantServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateProductVariantUseCase handles the business logic for creating product variants
type CreateProductVariantUseCase struct {
	repositories CreateProductVariantRepositories
	services     CreateProductVariantServices
}

// NewCreateProductVariantUseCase creates use case with grouped dependencies
func NewCreateProductVariantUseCase(
	repositories CreateProductVariantRepositories,
	services CreateProductVariantServices,
) *CreateProductVariantUseCase {
	return &CreateProductVariantUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product variant operation
func (uc *CreateProductVariantUseCase) Execute(ctx context.Context, req *productvariantpb.CreateProductVariantRequest) (*productvariantpb.CreateProductVariantResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ProductVariant,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product variant creation within a transaction
func (uc *CreateProductVariantUseCase) executeWithTransaction(ctx context.Context, req *productvariantpb.CreateProductVariantRequest) (*productvariantpb.CreateProductVariantResponse, error) {
	var result *productvariantpb.CreateProductVariantResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("product variant creation failed: %w", err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *CreateProductVariantUseCase) executeCore(ctx context.Context, req *productvariantpb.CreateProductVariantRequest) (*productvariantpb.CreateProductVariantResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.ProductVariant, entityid.ActionCreate)
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

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	return uc.repositories.ProductVariant.CreateProductVariant(ctx, req)
}

// validateInput validates the input request
func (uc *CreateProductVariantUseCase) validateInput(ctx context.Context, req *productvariantpb.CreateProductVariantRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.validation.data_required", "Product variant data is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.Sku == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.validation.sku_required", "SKU is required [DEFAULT]"))
	}
	// PriceOverride is optional (nil = "no override"). Validate only when set.
	if req.Data.PriceOverride != nil && *req.Data.PriceOverride < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant.validation.price_override_invalid", "Price override must be >= 0 [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateProductVariantUseCase) enrichData(data *productvariantpb.ProductVariant) error {
	now := time.Now()

	// Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	data.Active = true

	return nil
}
