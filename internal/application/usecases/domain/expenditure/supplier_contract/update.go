package suppliercontract

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// UpdateSupplierContractRepositories groups repository dependencies.
type UpdateSupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// UpdateSupplierContractServices groups service dependencies.
type UpdateSupplierContractServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateSupplierContractUseCase handles updating a supplier contract.
type UpdateSupplierContractUseCase struct {
	repositories UpdateSupplierContractRepositories
	services     UpdateSupplierContractServices
}

// NewUpdateSupplierContractUseCase creates a use case with grouped dependencies.
func NewUpdateSupplierContractUseCase(
	repositories UpdateSupplierContractRepositories,
	services UpdateSupplierContractServices,
) *UpdateSupplierContractUseCase {
	return &UpdateSupplierContractUseCase{repositories: repositories, services: services}
}

// Execute performs the update supplier contract operation.
func (uc *UpdateSupplierContractUseCase) Execute(ctx context.Context, req *suppliercontractpb.UpdateSupplierContractRequest) (*suppliercontractpb.UpdateSupplierContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContract, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.SupplierContract.UpdateSupplierContract(ctx, req)
}
