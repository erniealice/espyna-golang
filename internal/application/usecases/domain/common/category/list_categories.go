package category

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListCategoriesRepositories groups all repository dependencies
type ListCategoriesRepositories struct {
	Category categorypb.CategoryDomainServiceServer
}

// ListCategoriesServices groups all business service dependencies
type ListCategoriesServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListCategoriesUseCase handles the business logic for listing categories
type ListCategoriesUseCase struct {
	repositories ListCategoriesRepositories
	services     ListCategoriesServices
}

// NewListCategoriesUseCase creates use case with grouped dependencies
func NewListCategoriesUseCase(
	repositories ListCategoriesRepositories,
	services ListCategoriesServices,
) *ListCategoriesUseCase {
	return &ListCategoriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListCategoriesUseCaseUngrouped creates a new ListCategoriesUseCase
// Deprecated: Use NewListCategoriesUseCase with grouped parameters instead
func NewListCategoriesUseCaseUngrouped(categoryRepo categorypb.CategoryDomainServiceServer) *ListCategoriesUseCase {
	repositories := ListCategoriesRepositories{
		Category: categoryRepo,
	}

	services := ListCategoriesServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewListCategoriesUseCase(repositories, services)
}

// Execute performs the list categories operation
func (uc *ListCategoriesUseCase) Execute(ctx context.Context, req *categorypb.ListCategoriesRequest) (*categorypb.ListCategoriesResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Category.ListCategories(ctx, req)
}

// validateInput validates the input request
func (uc *ListCategoriesUseCase) validateInput(req *categorypb.ListCategoriesRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	// Filters and pagination are optional, so no additional validation needed
	return nil
}
