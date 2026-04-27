package suppliercontractline

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// ListSupplierContractLinesRepositories groups repository dependencies.
type ListSupplierContractLinesRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// ListSupplierContractLinesServices groups service dependencies.
type ListSupplierContractLinesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListSupplierContractLinesUseCase handles listing supplier contract lines.
type ListSupplierContractLinesUseCase struct {
	repositories ListSupplierContractLinesRepositories
	services     ListSupplierContractLinesServices
}

// NewListSupplierContractLinesUseCase creates a use case with grouped dependencies.
func NewListSupplierContractLinesUseCase(
	repositories ListSupplierContractLinesRepositories,
	services ListSupplierContractLinesServices,
) *ListSupplierContractLinesUseCase {
	return &ListSupplierContractLinesUseCase{repositories: repositories, services: services}
}

// Execute performs the list supplier contract lines operation.
func (uc *ListSupplierContractLinesUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.ListSupplierContractLinesRequest) (*suppliercontractlinepb.ListSupplierContractLinesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractLine, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.SupplierContractLine.ListSupplierContractLines(ctx, req)
}
