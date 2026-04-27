package suppliercontractline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// ReadSupplierContractLineRepositories groups repository dependencies.
type ReadSupplierContractLineRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// ReadSupplierContractLineServices groups service dependencies.
type ReadSupplierContractLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadSupplierContractLineUseCase handles reading a supplier contract line.
type ReadSupplierContractLineUseCase struct {
	repositories ReadSupplierContractLineRepositories
	services     ReadSupplierContractLineServices
}

// NewReadSupplierContractLineUseCase creates a use case with grouped dependencies.
func NewReadSupplierContractLineUseCase(
	repositories ReadSupplierContractLineRepositories,
	services ReadSupplierContractLineServices,
) *ReadSupplierContractLineUseCase {
	return &ReadSupplierContractLineUseCase{repositories: repositories, services: services}
}

// Execute performs the read supplier contract line operation.
func (uc *ReadSupplierContractLineUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.ReadSupplierContractLineRequest) (*suppliercontractlinepb.ReadSupplierContractLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_line.validation.id_required", "Supplier contract line ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractLine.ReadSupplierContractLine(ctx, req)
}
