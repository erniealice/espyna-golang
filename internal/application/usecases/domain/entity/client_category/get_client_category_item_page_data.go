package client_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// GetClientCategoryItemPageDataRepositories groups all repository dependencies
type GetClientCategoryItemPageDataRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// GetClientCategoryItemPageDataServices groups all business service dependencies
type GetClientCategoryItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetClientCategoryItemPageDataUseCase handles the business logic for getting client category item page data
type GetClientCategoryItemPageDataUseCase struct {
	repositories GetClientCategoryItemPageDataRepositories
	services     GetClientCategoryItemPageDataServices
}

// NewGetClientCategoryItemPageDataUseCase creates use case with grouped dependencies
func NewGetClientCategoryItemPageDataUseCase(
	repositories GetClientCategoryItemPageDataRepositories,
	services GetClientCategoryItemPageDataServices,
) *GetClientCategoryItemPageDataUseCase {
	return &GetClientCategoryItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetClientCategoryItemPageDataUseCaseUngrouped creates a new GetClientCategoryItemPageDataUseCase
// Deprecated: Use NewGetClientCategoryItemPageDataUseCase with grouped parameters instead
func NewGetClientCategoryItemPageDataUseCaseUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *GetClientCategoryItemPageDataUseCase {
	repositories := GetClientCategoryItemPageDataRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := GetClientCategoryItemPageDataServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewGetClientCategoryItemPageDataUseCase(repositories, services)
}

func (uc *GetClientCategoryItemPageDataUseCase) Execute(ctx context.Context, req *clientcategorypb.GetClientCategoryItemPageDataRequest) (*clientcategorypb.GetClientCategoryItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"client_category", ports.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ClientCategory.GetClientCategoryItemPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *GetClientCategoryItemPageDataUseCase) validateInput(ctx context.Context, req *clientcategorypb.GetClientCategoryItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}
	if req.ClientCategoryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.id_required", "Client category ID is required [DEFAULT]"))
	}
	return nil
}
