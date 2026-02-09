package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// UpdateResourceUseCase handles the business logic for updating resources
// UpdateResourceRepositories groups all repository dependencies
type UpdateResourceRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
	Product  productpb.ProductDomainServiceServer
}

// UpdateResourceServices groups all business service dependencies
type UpdateResourceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateResourceUseCase handles the business logic for updating resources
type UpdateResourceUseCase struct {
	repositories UpdateResourceRepositories
	services     UpdateResourceServices
}

// NewUpdateResourceUseCase creates a new UpdateResourceUseCase
func NewUpdateResourceUseCase(
	repositories UpdateResourceRepositories,
	services UpdateResourceServices,
) *UpdateResourceUseCase {
	return &UpdateResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update resource operation
func (uc *UpdateResourceUseCase) Execute(ctx context.Context, req *resourcepb.UpdateResourceRequest) (*resourcepb.UpdateResourceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityResource, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes resource update within a transaction
func (uc *UpdateResourceUseCase) executeWithTransaction(ctx context.Context, req *resourcepb.UpdateResourceRequest) (*resourcepb.UpdateResourceResponse, error) {
	var result *resourcepb.UpdateResourceResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *UpdateResourceUseCase) executeCore(ctx context.Context, req *resourcepb.UpdateResourceRequest) (*resourcepb.UpdateResourceResponse, error) {

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichResourceData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		// The validation function now returns the specific translated error, so we don't need to wrap it again.
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Resource.UpdateResource(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateResourceUseCase) validateInput(ctx context.Context, req *resourcepb.UpdateResourceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.data_required", "Resource data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.id_required", "Resource ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.name_required", "Resource name is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.product_id_required", "Product ID is required for the resource [DEFAULT]"))
	}
	return nil
}

// enrichResourceData adds generated fields and audit information
func (uc *UpdateResourceUseCase) enrichResourceData(resource *resourcepb.Resource) error {
	now := time.Now()

	// Update audit fields
	resource.DateModified = &[]int64{now.UnixMilli()}[0]
	resource.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for resources
func (uc *UpdateResourceUseCase) validateBusinessRules(ctx context.Context, resource *resourcepb.Resource) error {
	// Validate resource name length
	name := strings.TrimSpace(resource.Name)
	if len(name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.name_min_length", "Resource name must be at least 3 characters long [DEFAULT]"))
	}

	if len(name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.name_max_length", "Resource name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if resource.Description != nil && *resource.Description != "" {
		description := strings.TrimSpace(*resource.Description)
		if len(description) > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.description_max_length", "Resource description cannot exceed 1000 characters [DEFAULT]"))
		}
	}

	// Normalize name (trim spaces, proper capitalization)
	resource.Name = cases.Title(language.English).String(strings.ToLower(name))

	// Business rule: Product ID format validation
	if len(resource.ProductId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.product_id_min_length", "Product ID must be at least 3 characters long [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateResourceUseCase) validateEntityReferences(ctx context.Context, resource *resourcepb.Resource) error {
	// Validate Product entity reference
	if resource.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: resource.ProductId},
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.product_not_found", "Referenced product with ID '{productId}' does not exist [DEFAULT]")
				translatedError = strings.ReplaceAll(translatedError, "{productId}", resource.ProductId)
				return errors.New(translatedError)
			}
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.product_reference_validation_failed", "Failed to validate product entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			// This case handles when the repository returns no error but an empty response, which is a possible state.
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.product_not_found", "Referenced product with ID '{productId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", resource.ProductId)
			return errors.New(translatedError)
		}
		if !product.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.product_not_active", "Referenced product with ID '{productId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", resource.ProductId)
			return errors.New(translatedError)
		}
	}

	return nil
}
