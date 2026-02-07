package category

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	categorypb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// ReadCategoryRepositories groups all repository dependencies
type ReadCategoryRepositories struct {
	Category categorypb.CategoryDomainServiceServer
}

// ReadCategoryServices groups all business service dependencies
type ReadCategoryServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadCategoryUseCase handles the business logic for reading categories
type ReadCategoryUseCase struct {
	repositories ReadCategoryRepositories
	services     ReadCategoryServices
}

// NewReadCategoryUseCase creates use case with grouped dependencies
func NewReadCategoryUseCase(
	repositories ReadCategoryRepositories,
	services ReadCategoryServices,
) *ReadCategoryUseCase {
	return &ReadCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadCategoryUseCaseUngrouped creates a new ReadCategoryUseCase
// Deprecated: Use NewReadCategoryUseCase with grouped parameters instead
func NewReadCategoryUseCaseUngrouped(categoryRepo categorypb.CategoryDomainServiceServer) *ReadCategoryUseCase {
	repositories := ReadCategoryRepositories{
		Category: categoryRepo,
	}

	services := ReadCategoryServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewReadCategoryUseCase(repositories, services)
}

// Execute performs the read category operation
func (uc *ReadCategoryUseCase) Execute(ctx context.Context, req *categorypb.ReadCategoryRequest) (*categorypb.ReadCategoryResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Category.ReadCategory(ctx, req)
}

// validateInput validates the input request
func (uc *ReadCategoryUseCase) validateInput(req *categorypb.ReadCategoryRequest) error {
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
