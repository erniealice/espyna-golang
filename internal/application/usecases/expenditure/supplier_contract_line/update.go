package suppliercontractline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// UpdateSupplierContractLineRepositories groups repository dependencies.
type UpdateSupplierContractLineRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// UpdateSupplierContractLineServices groups service dependencies.
type UpdateSupplierContractLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UpdateSupplierContractLineUseCase handles updating a supplier contract line.
type UpdateSupplierContractLineUseCase struct {
	repositories UpdateSupplierContractLineRepositories
	services     UpdateSupplierContractLineServices
}

// NewUpdateSupplierContractLineUseCase creates a use case with grouped dependencies.
func NewUpdateSupplierContractLineUseCase(
	repositories UpdateSupplierContractLineRepositories,
	services UpdateSupplierContractLineServices,
) *UpdateSupplierContractLineUseCase {
	return &UpdateSupplierContractLineUseCase{repositories: repositories, services: services}
}

// Execute performs the update supplier contract line operation.
func (uc *UpdateSupplierContractLineUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.UpdateSupplierContractLineRequest) (*suppliercontractlinepb.UpdateSupplierContractLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractLine, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_line.validation.id_required", "Supplier contract line ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractLine.UpdateSupplierContractLine(ctx, req)
}
