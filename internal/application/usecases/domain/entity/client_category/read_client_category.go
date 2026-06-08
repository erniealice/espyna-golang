package client_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// ReadClientCategoryRepositories groups all repository dependencies
type ReadClientCategoryRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// ReadClientCategoryServices groups all business service dependencies
type ReadClientCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadClientCategoryUseCase handles the business logic for reading client categories
type ReadClientCategoryUseCase struct {
	repositories ReadClientCategoryRepositories
	services     ReadClientCategoryServices
}

// NewReadClientCategoryUseCase creates use case with grouped dependencies
func NewReadClientCategoryUseCase(
	repositories ReadClientCategoryRepositories,
	services ReadClientCategoryServices,
) *ReadClientCategoryUseCase {
	return &ReadClientCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadClientCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadClientCategoryUseCase with grouped parameters instead
func NewReadClientCategoryUseCaseUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *ReadClientCategoryUseCase {
	repositories := ReadClientCategoryRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := ReadClientCategoryServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewReadClientCategoryUseCase(repositories, services)
}

func (uc *ReadClientCategoryUseCase) Execute(ctx context.Context, req *clientcategorypb.ReadClientCategoryRequest) (*clientcategorypb.ReadClientCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"client_category", entityid.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ClientCategory.ReadClientCategory(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ReadClientCategoryUseCase) validateInput(ctx context.Context, req *clientcategorypb.ReadClientCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.data_required", "Client category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.id_required", "Client category ID is required [DEFAULT]"))
	}
	return nil
}
