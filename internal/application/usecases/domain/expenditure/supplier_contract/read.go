package suppliercontract

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// ReadSupplierContractRepositories groups repository dependencies.
type ReadSupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// ReadSupplierContractServices groups service dependencies.
type ReadSupplierContractServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadSupplierContractUseCase handles reading a supplier contract.
type ReadSupplierContractUseCase struct {
	repositories ReadSupplierContractRepositories
	services     ReadSupplierContractServices
}

// NewReadSupplierContractUseCase creates a use case with grouped dependencies.
func NewReadSupplierContractUseCase(
	repositories ReadSupplierContractRepositories,
	services ReadSupplierContractServices,
) *ReadSupplierContractUseCase {
	return &ReadSupplierContractUseCase{repositories: repositories, services: services}
}

// Execute performs the read supplier contract operation.
func (uc *ReadSupplierContractUseCase) Execute(ctx context.Context, req *suppliercontractpb.ReadSupplierContractRequest) (*suppliercontractpb.ReadSupplierContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContract, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContract.ReadSupplierContract(ctx, req)
}
