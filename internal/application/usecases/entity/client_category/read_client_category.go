package client_category

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	clientcategorypb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_category"
)

// ReadClientCategoryRepositories groups all repository dependencies
type ReadClientCategoryRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// ReadClientCategoryServices groups all business service dependencies
type ReadClientCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadClientCategoryUseCase(repositories, services)
}

func (uc *ReadClientCategoryUseCase) Execute(ctx context.Context, req *clientcategorypb.ReadClientCategoryRequest) (*clientcategorypb.ReadClientCategoryResponse, error) {
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.data_required", "Client category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.id_required", "Client category ID is required [DEFAULT]"))
	}
	return nil
}
