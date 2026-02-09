package client_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// UpdateClientCategoryRepositories groups all repository dependencies
type UpdateClientCategoryRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// UpdateClientCategoryServices groups all business service dependencies
type UpdateClientCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UpdateClientCategoryUseCase handles the business logic for updating client categories
type UpdateClientCategoryUseCase struct {
	repositories UpdateClientCategoryRepositories
	services     UpdateClientCategoryServices
}

// NewUpdateClientCategoryUseCase creates use case with grouped dependencies
func NewUpdateClientCategoryUseCase(
	repositories UpdateClientCategoryRepositories,
	services UpdateClientCategoryServices,
) *UpdateClientCategoryUseCase {
	return &UpdateClientCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateClientCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateClientCategoryUseCase with grouped parameters instead
func NewUpdateClientCategoryUseCaseUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *UpdateClientCategoryUseCase {
	repositories := UpdateClientCategoryRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := UpdateClientCategoryServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUpdateClientCategoryUseCase(repositories, services)
}

func (uc *UpdateClientCategoryUseCase) Execute(ctx context.Context, req *clientcategorypb.UpdateClientCategoryRequest) (*clientcategorypb.UpdateClientCategoryResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"client_category", ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *UpdateClientCategoryUseCase) executeWithTransaction(ctx context.Context, req *clientcategorypb.UpdateClientCategoryRequest) (*clientcategorypb.UpdateClientCategoryResponse, error) {
	var result *clientcategorypb.UpdateClientCategoryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "client_category.errors.update_failed", "Client category update failed [DEFAULT]")
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

func (uc *UpdateClientCategoryUseCase) executeCore(ctx context.Context, req *clientcategorypb.UpdateClientCategoryRequest) (*clientcategorypb.UpdateClientCategoryResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichClientCategoryData(req.Data); err != nil {
		return nil, err
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	return uc.repositories.ClientCategory.UpdateClientCategory(ctx, req)
}

func (uc *UpdateClientCategoryUseCase) validateInput(ctx context.Context, req *clientcategorypb.UpdateClientCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.request_required", "Request is required for client categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.data_required", "Client category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_category.validation.id_required", "Client category ID is required for update [DEFAULT]"))
	}
	return nil
}

func (uc *UpdateClientCategoryUseCase) enrichClientCategoryData(category *clientcategorypb.ClientCategory) error {
	// For junction table, we mainly update the active status
	// The client_id and category_id should not be changed after creation
	// The nested category object can be updated to reflect changes in the common category
	return nil
}

func (uc *UpdateClientCategoryUseCase) validateBusinessRules(ctx context.Context, category *clientcategorypb.ClientCategory) error {
	// For a junction table, we primarily validate that:
	// 1. The ID is present (already validated)
	// 2. The active status is valid if provided
	// 3. The client_id and category_id should not be modified (business rule)

	// Note: The actual validation of client_id and category_id modification
	// should be handled at the repository level or through additional business logic

	return nil
}
