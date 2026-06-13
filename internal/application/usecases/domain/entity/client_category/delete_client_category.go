package client_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// DeleteClientCategoryRepositories groups all repository dependencies
type DeleteClientCategoryRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// DeleteClientCategoryServices groups all business service dependencies
type DeleteClientCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteClientCategoryUseCase handles the business logic for deleting client categories
type DeleteClientCategoryUseCase struct {
	repositories DeleteClientCategoryRepositories
	services     DeleteClientCategoryServices
}

// NewDeleteClientCategoryUseCase creates use case with grouped dependencies
func NewDeleteClientCategoryUseCase(
	repositories DeleteClientCategoryRepositories,
	services DeleteClientCategoryServices,
) *DeleteClientCategoryUseCase {
	return &DeleteClientCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteClientCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteClientCategoryUseCase with grouped parameters instead
func NewDeleteClientCategoryUseCaseUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *DeleteClientCategoryUseCase {
	repositories := DeleteClientCategoryRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := DeleteClientCategoryServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewDeleteClientCategoryUseCase(repositories, services)
}

func (uc *DeleteClientCategoryUseCase) Execute(ctx context.Context, req *clientcategorypb.DeleteClientCategoryRequest) (*clientcategorypb.DeleteClientCategoryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "client_category",
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *DeleteClientCategoryUseCase) executeWithTransaction(ctx context.Context, req *clientcategorypb.DeleteClientCategoryRequest) (*clientcategorypb.DeleteClientCategoryResponse, error) {
	var result *clientcategorypb.DeleteClientCategoryResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "client_category.errors.deletion_failed", "Client category deletion failed [DEFAULT]")
			return errors.New(translatedError + ": " + err.Error())
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *DeleteClientCategoryUseCase) executeCore(ctx context.Context, req *clientcategorypb.DeleteClientCategoryRequest) (*clientcategorypb.DeleteClientCategoryResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	return uc.repositories.ClientCategory.DeleteClientCategory(ctx, req)
}

func (uc *DeleteClientCategoryUseCase) validateInput(ctx context.Context, req *clientcategorypb.DeleteClientCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.data_required", "Client category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client_category.validation.id_required", "Client category ID is required for deletion [DEFAULT]"))
	}
	return nil
}
