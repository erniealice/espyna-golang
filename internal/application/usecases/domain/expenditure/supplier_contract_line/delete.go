package suppliercontractline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// DeleteSupplierContractLineRepositories groups repository dependencies.
type DeleteSupplierContractLineRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// DeleteSupplierContractLineServices groups service dependencies.
type DeleteSupplierContractLineServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// DeleteSupplierContractLineUseCase handles deleting a supplier contract line.
type DeleteSupplierContractLineUseCase struct {
	repositories DeleteSupplierContractLineRepositories
	services     DeleteSupplierContractLineServices
}

// NewDeleteSupplierContractLineUseCase creates a use case with grouped dependencies.
func NewDeleteSupplierContractLineUseCase(
	repositories DeleteSupplierContractLineRepositories,
	services DeleteSupplierContractLineServices,
) *DeleteSupplierContractLineUseCase {
	return &DeleteSupplierContractLineUseCase{repositories: repositories, services: services}
}

// Execute performs the delete supplier contract line operation.
func (uc *DeleteSupplierContractLineUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.DeleteSupplierContractLineRequest) (*suppliercontractlinepb.DeleteSupplierContractLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContractLine, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract_line.validation.id_required", "Supplier contract line ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractLine.DeleteSupplierContractLine(ctx, req)
}
