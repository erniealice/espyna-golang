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

// GetSupplierCategoryItemPageDataRepositories groups all repository dependencies
type GetSupplierCategoryItemPageDataRepositories struct {
	SupplierCategory suppliercategorypb.SupplierCategoryDomainServiceServer
}

// GetSupplierCategoryItemPageDataServices groups all business service dependencies
type GetSupplierCategoryItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetSupplierCategoryItemPageDataUseCase handles the business logic for getting supplier category item page data
type GetSupplierCategoryItemPageDataUseCase struct {
	repositories GetSupplierCategoryItemPageDataRepositories
	services     GetSupplierCategoryItemPageDataServices
}

// NewGetSupplierCategoryItemPageDataUseCase creates use case with grouped dependencies
func NewGetSupplierCategoryItemPageDataUseCase(
	repositories GetSupplierCategoryItemPageDataRepositories,
	services GetSupplierCategoryItemPageDataServices,
) *GetSupplierCategoryItemPageDataUseCase {
	return &GetSupplierCategoryItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetSupplierCategoryItemPageDataUseCaseUngrouped creates a new GetSupplierCategoryItemPageDataUseCase
// Deprecated: Use NewGetSupplierCategoryItemPageDataUseCase with grouped parameters instead
func NewGetSupplierCategoryItemPageDataUseCaseUngrouped(supplierCategoryRepo suppliercategorypb.SupplierCategoryDomainServiceServer) *GetSupplierCategoryItemPageDataUseCase {
	repositories := GetSupplierCategoryItemPageDataRepositories{
		SupplierCategory: supplierCategoryRepo,
	}

	services := GetSupplierCategoryItemPageDataServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewGetSupplierCategoryItemPageDataUseCase(repositories, services)
}

func (uc *GetSupplierCategoryItemPageDataUseCase) Execute(ctx context.Context, req *suppliercategorypb.GetSupplierCategoryItemPageDataRequest) (*suppliercategorypb.GetSupplierCategoryItemPageDataResponse, error) {
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

	resp, err := uc.repositories.SupplierCategory.GetSupplierCategoryItemPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (uc *GetSupplierCategoryItemPageDataUseCase) validateInput(ctx context.Context, req *suppliercategorypb.GetSupplierCategoryItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.request_required", "Request is required for supplier categories [DEFAULT]"))
	}
	if req.SupplierCategoryId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "supplier_category.validation.id_required", "Supplier category ID is required [DEFAULT]"))
	}
	return nil
}
