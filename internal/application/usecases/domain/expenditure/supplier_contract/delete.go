package suppliercontract

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// DeleteSupplierContractRepositories groups repository dependencies.
type DeleteSupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// DeleteSupplierContractServices groups service dependencies.
type DeleteSupplierContractServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteSupplierContractUseCase handles deleting a supplier contract.
type DeleteSupplierContractUseCase struct {
	repositories DeleteSupplierContractRepositories
	services     DeleteSupplierContractServices
}

// NewDeleteSupplierContractUseCase creates a use case with grouped dependencies.
func NewDeleteSupplierContractUseCase(
	repositories DeleteSupplierContractRepositories,
	services DeleteSupplierContractServices,
) *DeleteSupplierContractUseCase {
	return &DeleteSupplierContractUseCase{repositories: repositories, services: services}
}

// Execute performs the delete supplier contract operation.
func (uc *DeleteSupplierContractUseCase) Execute(ctx context.Context, req *suppliercontractpb.DeleteSupplierContractRequest) (*suppliercontractpb.DeleteSupplierContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContract, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContract.DeleteSupplierContract(ctx, req)
}
