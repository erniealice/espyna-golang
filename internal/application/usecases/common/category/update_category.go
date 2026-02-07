package category

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	categorypb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// UpdateCategoryRepositories groups all repository dependencies
type UpdateCategoryRepositories struct {
	Category categorypb.CategoryDomainServiceServer
}

// UpdateCategoryServices groups all business service dependencies
type UpdateCategoryServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateCategoryUseCase handles the business logic for updating categories
type UpdateCategoryUseCase struct {
	repositories UpdateCategoryRepositories
	services     UpdateCategoryServices
}

// NewUpdateCategoryUseCase creates use case with grouped dependencies
func NewUpdateCategoryUseCase(
	repositories UpdateCategoryRepositories,
	services UpdateCategoryServices,
) *UpdateCategoryUseCase {
	return &UpdateCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateCategoryUseCaseUngrouped creates a new UpdateCategoryUseCase
// Deprecated: Use NewUpdateCategoryUseCase with grouped parameters instead
func NewUpdateCategoryUseCaseUngrouped(categoryRepo categorypb.CategoryDomainServiceServer) *UpdateCategoryUseCase {
	repositories := UpdateCategoryRepositories{
		Category: categoryRepo,
	}

	services := UpdateCategoryServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUpdateCategoryUseCase(repositories, services)
}

// Execute performs the update category operation
func (uc *UpdateCategoryUseCase) Execute(ctx context.Context, req *categorypb.UpdateCategoryRequest) (*categorypb.UpdateCategoryResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes category update within a transaction
func (uc *UpdateCategoryUseCase) executeWithTransaction(ctx context.Context, req *categorypb.UpdateCategoryRequest) (*categorypb.UpdateCategoryResponse, error) {
	var result *categorypb.UpdateCategoryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("category update failed: %w", err)
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
func (uc *UpdateCategoryUseCase) executeCore(ctx context.Context, req *categorypb.UpdateCategoryRequest) (*categorypb.UpdateCategoryResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Category.UpdateCategory(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateCategoryUseCase) validateInput(req *categorypb.UpdateCategoryRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("category data is required")
	}
	if req.Data.Id == "" {
		return errors.New("category ID is required")
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateCategoryUseCase) validateBusinessRules(category *categorypb.Category) error {
	// Validate name length if provided
	if category.Name != "" {
		if len(strings.TrimSpace(category.Name)) == 0 {
			return errors.New("category name cannot be empty")
		}

		if len(category.Name) < 2 {
			return errors.New("category name must be at least 2 characters long")
		}

		if len(category.Name) > 100 {
			return errors.New("category name cannot exceed 100 characters")
		}
	}

	// Validate code length and format if provided
	if category.Code != "" {
		if len(category.Code) < 2 {
			return errors.New("category code must be at least 2 characters long")
		}

		if len(category.Code) > 50 {
			return errors.New("category code cannot exceed 50 characters")
		}
	}

	// Validate description length if provided
	if category.Description != "" && len(category.Description) > 500 {
		return errors.New("category description cannot exceed 500 characters")
	}

	// Validate module if provided
	if category.Module != "" {
		allowedModules := map[string]bool{
			"client":       true,
			"product":      true,
			"location":     true,
			"event":        true,
			"payment":      true,
			"subscription": true,
			"workflow":     true,
			"staff":        true,
			"delegate":     true,
		}
		if !allowedModules[category.Module] {
			return fmt.Errorf("category module must be one of: client, product, location, event, payment, subscription, workflow, staff, delegate")
		}
	}

	return nil
}
