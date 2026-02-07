package category

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	categorypb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// DeleteCategoryRepositories groups all repository dependencies
type DeleteCategoryRepositories struct {
	Category categorypb.CategoryDomainServiceServer
}

// DeleteCategoryServices groups all business service dependencies
type DeleteCategoryServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteCategoryUseCase handles the business logic for deleting categories
type DeleteCategoryUseCase struct {
	repositories DeleteCategoryRepositories
	services     DeleteCategoryServices
}

// NewDeleteCategoryUseCase creates use case with grouped dependencies
func NewDeleteCategoryUseCase(
	repositories DeleteCategoryRepositories,
	services DeleteCategoryServices,
) *DeleteCategoryUseCase {
	return &DeleteCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteCategoryUseCaseUngrouped creates a new DeleteCategoryUseCase
// Deprecated: Use NewDeleteCategoryUseCase with grouped parameters instead
func NewDeleteCategoryUseCaseUngrouped(categoryRepo categorypb.CategoryDomainServiceServer) *DeleteCategoryUseCase {
	repositories := DeleteCategoryRepositories{
		Category: categoryRepo,
	}

	services := DeleteCategoryServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewDeleteCategoryUseCase(repositories, services)
}

// Execute performs the delete category operation
func (uc *DeleteCategoryUseCase) Execute(ctx context.Context, req *categorypb.DeleteCategoryRequest) (*categorypb.DeleteCategoryResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes category deletion within a transaction
func (uc *DeleteCategoryUseCase) executeWithTransaction(ctx context.Context, req *categorypb.DeleteCategoryRequest) (*categorypb.DeleteCategoryResponse, error) {
	var result *categorypb.DeleteCategoryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("category deletion failed: %w", err)
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
func (uc *DeleteCategoryUseCase) executeCore(ctx context.Context, req *categorypb.DeleteCategoryRequest) (*categorypb.DeleteCategoryResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Category.DeleteCategory(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteCategoryUseCase) validateInput(req *categorypb.DeleteCategoryRequest) error {
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
