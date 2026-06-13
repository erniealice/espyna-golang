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

// ReadSupplierCategoryRepositories groups all repository dependencies
type ReadSupplierCategoryRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// ReadSupplierCategoryServices groups all business service dependencies
type ReadSupplierCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadSupplierCategoryUseCase handles the business logic for reading supplier categories
type ReadSupplierCategoryUseCase struct {
	repositories ReadSupplierCategoryRepositories
	services     ReadSupplierCategoryServices
}

// NewReadSupplierCategoryUseCase creates use case with grouped dependencies
func NewReadSupplierCategoryUseCase(
	repositories ReadSupplierCategoryRepositories,
	services ReadSupplierCategoryServices,
) *ReadSupplierCategoryUseCase {
	return &ReadSupplierCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadSupplierCategoryUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadSupplierCategoryUseCase with grouped parameters instead
func NewReadSupplierCategoryUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *ReadSupplierCategoryUseCase {
	repositories := ReadSupplierCategoryRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := ReadSupplierCategoryServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewReadSupplierCategoryUseCase(repositories, services)
}

func (uc *ReadSupplierCategoryUseCase) Execute(ctx context.Context, req *suppliercategorypb.ReadSupplierCategoryRequest) (*suppliercategorypb.ReadSupplierCategoryResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "supplier_category",
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.SupplierCategory.ReadSupplierCategory(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *ReadSupplierCategoryUseCase) validateInput(ctx context.Context, req *suppliercategorypb.ReadSupplierCategoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.data_required", "Supplier category data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.id_required", "Supplier category ID is required [DEFAULT]"))
	}
	return nil
}
