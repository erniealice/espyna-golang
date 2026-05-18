package suppliercontract

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// DeleteSupplierContractRepositories groups repository dependencies.
type DeleteSupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// DeleteSupplierContractServices groups service dependencies.
type DeleteSupplierContractServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContract, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContract.DeleteSupplierContract(ctx, req)
}
