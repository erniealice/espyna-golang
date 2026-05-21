package suppliercontract

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// ListSupplierContractsRepositories groups repository dependencies.
type ListSupplierContractsRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// ListSupplierContractsServices groups service dependencies.
type ListSupplierContractsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListSupplierContractsUseCase handles listing supplier contracts.
type ListSupplierContractsUseCase struct {
	repositories ListSupplierContractsRepositories
	services     ListSupplierContractsServices
}

// NewListSupplierContractsUseCase creates a use case with grouped dependencies.
func NewListSupplierContractsUseCase(
	repositories ListSupplierContractsRepositories,
	services ListSupplierContractsServices,
) *ListSupplierContractsUseCase {
	return &ListSupplierContractsUseCase{repositories: repositories, services: services}
}

// Execute performs the list supplier contracts operation.
func (uc *ListSupplierContractsUseCase) Execute(ctx context.Context, req *suppliercontractpb.ListSupplierContractsRequest) (*suppliercontractpb.ListSupplierContractsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContract, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.SupplierContract.ListSupplierContracts(ctx, req)
}
