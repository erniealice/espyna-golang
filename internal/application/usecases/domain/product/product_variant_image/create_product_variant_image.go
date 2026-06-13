package product_variant_image

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

// CreateProductVariantImageRepositories groups all repository dependencies
type CreateProductVariantImageRepositories struct {
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer // Primary entity repository
}

// CreateProductVariantImageServices groups all business service dependencies
type CreateProductVariantImageServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateProductVariantImageUseCase handles the business logic for creating product variant images
type CreateProductVariantImageUseCase struct {
	repositories CreateProductVariantImageRepositories
	services     CreateProductVariantImageServices
}

// NewCreateProductVariantImageUseCase creates use case with grouped dependencies
func NewCreateProductVariantImageUseCase(
	repositories CreateProductVariantImageRepositories,
	services CreateProductVariantImageServices,
) *CreateProductVariantImageUseCase {
	return &CreateProductVariantImageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product variant image operation
func (uc *CreateProductVariantImageUseCase) Execute(ctx context.Context, req *productvariantimagepb.CreateProductVariantImageRequest) (*productvariantimagepb.CreateProductVariantImageResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ProductVariantImage,
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

// executeWithTransaction executes product variant image creation within a transaction
func (uc *CreateProductVariantImageUseCase) executeWithTransaction(ctx context.Context, req *productvariantimagepb.CreateProductVariantImageRequest) (*productvariantimagepb.CreateProductVariantImageResponse, error) {
	var result *productvariantimagepb.CreateProductVariantImageResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("product variant image creation failed: %w", err)
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
func (uc *CreateProductVariantImageUseCase) executeCore(ctx context.Context, req *productvariantimagepb.CreateProductVariantImageRequest) (*productvariantimagepb.CreateProductVariantImageResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.ProductVariantImage, entityid.ActionCreate)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	return uc.repositories.ProductVariantImage.CreateProductVariantImage(ctx, req)
}

// validateInput validates the input request
func (uc *CreateProductVariantImageUseCase) validateInput(ctx context.Context, req *productvariantimagepb.CreateProductVariantImageRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.validation.data_required", "Product variant image data is required [DEFAULT]"))
	}
	if req.Data.ProductVariantId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.validation.product_variant_id_required", "Product variant ID is required [DEFAULT]"))
	}
	if req.Data.ImageUrl == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_variant_image.validation.image_url_required", "Image URL is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *CreateProductVariantImageUseCase) enrichData(data *productvariantimagepb.ProductVariantImage) error {
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
