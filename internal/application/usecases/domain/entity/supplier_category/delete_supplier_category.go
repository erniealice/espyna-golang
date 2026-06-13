package supplier_category

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
)

// DeleteSupplierCategoryRepositories groups all repository dependencies
type DeleteSupplierCategoryRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// DeleteSupplierCategoryServices groups all business service dependencies
type DeleteSupplierCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteSupplierCategoryUseCase handles the business logic for deleting supplier categories
type DeleteSupplierCategoryUseCase struct {
	repositories DeleteSupplierCategoryRepositories
	services     DeleteSupplierCategoryServices
}

// NewDeleteSupplierCategoryUseCase creates use case with grouped dependencies
func NewDeleteSupplierCategoryUseCase(
	repositories DeleteSupplierCategoryRepositories,
	services DeleteSupplierCategoryServices,
) *DeleteSupplierCategoryUseCase {
	return &DeleteSupplierCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteSupplierCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteSupplierCategoryUseCase with grouped parameters instead
func NewDeleteSupplierCategoryUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *DeleteSupplierCategoryUseCase {
	repositories := DeleteSupplierCategoryRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := DeleteSupplierCategoryServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewDeleteSupplierCategoryUseCase(repositories, services)
}

func (uc *DeleteSupplierCategoryUseCase) Execute(ctx context.Context, req *suppliercategorypb.DeleteSupplierCategoryRequest) (*suppliercategorypb.DeleteSupplierCategoryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "supplier_category",
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

func (uc *DeleteSupplierCategoryUseCase) executeWithTransaction(ctx context.Context, req *suppliercategorypb.DeleteSupplierCategoryRequest) (*suppliercategorypb.DeleteSupplierCategoryResponse, error) {
	var result *suppliercategorypb.DeleteSupplierCategoryResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "supplier_category.errors.deletion_failed", "Supplier category deletion failed [DEFAULT]")
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

func (uc *DeleteSupplierCategoryUseCase) executeCore(ctx context.Context, req *suppliercategorypb.DeleteSupplierCategoryRequest) (*suppliercategorypb.DeleteSupplierCategoryResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	return uc.repositories.SupplierCategory.DeleteSupplierCategory(ctx, req)
}

func (uc *DeleteSupplierCategoryUseCase) validateInput(ctx context.Context, req *suppliercategorypb.DeleteSupplierCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.data_required", "Supplier category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.id_required", "Supplier category ID is required for deletion [DEFAULT]"))
	}
	return nil
}
