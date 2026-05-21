package client_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// ListClientCategoriesRepositories groups all repository dependencies
type ListClientCategoriesRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// ListClientCategoriesServices groups all business service dependencies
type ListClientCategoriesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListClientCategoriesUseCase handles the business logic for listing client categories
type ListClientCategoriesUseCase struct {
	repositories ListClientCategoriesRepositories
	services     ListClientCategoriesServices
}

// NewListClientCategoriesUseCase creates use case with grouped dependencies
func NewListClientCategoriesUseCase(
	repositories ListClientCategoriesRepositories,
	services ListClientCategoriesServices,
) *ListClientCategoriesUseCase {
	return &ListClientCategoriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListClientCategoriesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListClientCategoriesUseCase with grouped parameters instead
func NewListClientCategoriesUseCaseUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *ListClientCategoriesUseCase {
	repositories := ListClientCategoriesRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := ListClientCategoriesServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListClientCategoriesUseCase(repositories, services)
}

func (uc *ListClientCategoriesUseCase) Execute(ctx context.Context, req *clientcategorypb.ListClientCategoriesRequest) (*clientcategorypb.ListClientCategoriesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"client_category", ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ClientCategory.ListClientCategories(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ListClientCategoriesUseCase) validateInput(ctx context.Context, req *clientcategorypb.ListClientCategoriesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}
	return nil
}
