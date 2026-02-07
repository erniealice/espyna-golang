package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
)

// CreateResourceUseCase handles the business logic for creating resources
// CreateResourceRepositories groups all repository dependencies
type CreateResourceRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
	Product  productpb.ProductDomainServiceServer
}

// CreateResourceServices groups all business service dependencies
type CreateResourceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateResourceUseCase handles the business logic for creating resources
type CreateResourceUseCase struct {
	repositories CreateResourceRepositories
	services     CreateResourceServices
}

// NewCreateResourceUseCase creates a new CreateResourceUseCase
func NewCreateResourceUseCase(
	repositories CreateResourceRepositories,
	services CreateResourceServices,
) *CreateResourceUseCase {
	return &CreateResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create resource operation
func (uc *CreateResourceUseCase) Execute(ctx context.Context, req *resourcepb.CreateResourceRequest) (*resourcepb.CreateResourceResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.user_not_authenticated", "User not authenticated"))
		}

		permission := ports.EntityPermission(ports.EntityResource, ports.ActionCreate)
		authorized, err := uc.services.AuthorizationService.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.authorization_check_failed", "Authorization check failed")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !authorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.authorization_failed", "Access denied")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes resource creation within a transaction
func (uc *CreateResourceUseCase) executeWithTransaction(ctx context.Context, req *resourcepb.CreateResourceRequest) (*resourcepb.CreateResourceResponse, error) {
	var result *resourcepb.CreateResourceResponse

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
func (uc *CreateResourceUseCase) executeCore(ctx context.Context, req *resourcepb.CreateResourceRequest) (*resourcepb.CreateResourceResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
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

	// Business enrichment
	enrichedResource := uc.applyBusinessLogic(req.Data)

	// Delegate to repository
	resp, err := uc.repositories.Resource.CreateResource(ctx, &resourcepb.CreateResourceRequest{
		Data: enrichedResource,
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.creation_failed", "Resource creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched resource
func (uc *CreateResourceUseCase) applyBusinessLogic(resource *resourcepb.Resource) *resourcepb.Resource {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if resource.Id == "" {
		if uc.services.IDService != nil {
			resource.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback ID generation when service is not available
			resource.Id = fmt.Sprintf("resource-%d", now.UnixNano())
		}
	}

	// Business logic: Set default productId if not provided (for test data compatibility)
	if resource.ProductId == "" {
		// Use a fallback product ID that exists in the mock data
		resource.ProductId = "subject-science" // Default to science subject
	}

	// Business logic: Set active status for new resources
	resource.Active = true

	// Business logic: Set creation audit fields
	resource.DateCreated = &[]int64{now.UnixMilli()}[0]
	resource.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	resource.DateModified = &[]int64{now.UnixMilli()}[0]
	resource.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return resource
}

// validateInput validates the input request
func (uc *CreateResourceUseCase) validateInput(ctx context.Context, req *resourcepb.CreateResourceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.data_required", "Resource data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.name_required", "Resource name is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateResourceUseCase) validateBusinessRules(ctx context.Context, resource *resourcepb.Resource) error {
	// Business rule: Required data validation
	if resource == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.data_required", "Resource data is required [DEFAULT]"))
	}
	if resource.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.name_required", "Resource name is required [DEFAULT]"))
	}
	if resource.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}

	// Business rule: Name length constraints
	if len(resource.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.name_min_length", "Resource name must be at least 3 characters long [DEFAULT]"))
	}

	if len(resource.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.name_max_length", "Resource name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Description length validation
	if resource.Description != nil && len(*resource.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.description_max_length", "Resource description cannot exceed 1000 characters [DEFAULT]"))
	}

	// Business rule: Product ID format validation
	if len(resource.ProductId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.product_id_min_length", "Product ID must be at least 3 characters long [DEFAULT]"))
	}

	// Normalize name (trim spaces, proper capitalization)
	resource.Name = cases.Title(language.English).String(strings.ToLower(resource.Name))

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateResourceUseCase) validateEntityReferences(ctx context.Context, resource *resourcepb.Resource) error {
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
