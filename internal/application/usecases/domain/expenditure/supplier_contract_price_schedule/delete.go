package suppliercontractpriceschedule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// DeleteSupplierContractPriceScheduleRepositories groups repository dependencies.
type DeleteSupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// DeleteSupplierContractPriceScheduleServices groups service dependencies.
type DeleteSupplierContractPriceScheduleServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// DeleteSupplierContractPriceScheduleUseCase handles deleting a schedule.
type DeleteSupplierContractPriceScheduleUseCase struct {
	repositories DeleteSupplierContractPriceScheduleRepositories
	services     DeleteSupplierContractPriceScheduleServices
}

// NewDeleteSupplierContractPriceScheduleUseCase creates a use case with grouped dependencies.
func NewDeleteSupplierContractPriceScheduleUseCase(
	repositories DeleteSupplierContractPriceScheduleRepositories,
	services DeleteSupplierContractPriceScheduleServices,
) *DeleteSupplierContractPriceScheduleUseCase {
	return &DeleteSupplierContractPriceScheduleUseCase{repositories: repositories, services: services}
}

// Execute performs the delete operation.
func (uc *DeleteSupplierContractPriceScheduleUseCase) Execute(ctx context.Context, req *scpspb.DeleteSupplierContractPriceScheduleRequest) (*scpspb.DeleteSupplierContractPriceScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContractPriceSchedule, entityid.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract_price_schedule.validation.id_required", "Supplier contract price schedule ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractPriceSchedule.DeleteSupplierContractPriceSchedule(ctx, req)
}
