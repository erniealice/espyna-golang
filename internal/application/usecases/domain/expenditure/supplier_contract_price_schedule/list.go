package suppliercontractpriceschedule

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// ListSupplierContractPriceSchedulesRepositories groups repository dependencies.
type ListSupplierContractPriceSchedulesRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// ListSupplierContractPriceSchedulesServices groups service dependencies.
type ListSupplierContractPriceSchedulesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListSupplierContractPriceSchedulesUseCase handles listing schedules.
type ListSupplierContractPriceSchedulesUseCase struct {
	repositories ListSupplierContractPriceSchedulesRepositories
	services     ListSupplierContractPriceSchedulesServices
}

// NewListSupplierContractPriceSchedulesUseCase creates a use case with grouped dependencies.
func NewListSupplierContractPriceSchedulesUseCase(
	repositories ListSupplierContractPriceSchedulesRepositories,
	services ListSupplierContractPriceSchedulesServices,
) *ListSupplierContractPriceSchedulesUseCase {
	return &ListSupplierContractPriceSchedulesUseCase{repositories: repositories, services: services}
}

// Execute performs the list schedules operation.
func (uc *ListSupplierContractPriceSchedulesUseCase) Execute(ctx context.Context, req *scpspb.ListSupplierContractPriceSchedulesRequest) (*scpspb.ListSupplierContractPriceSchedulesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContractPriceSchedule, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.SupplierContractPriceSchedule.ListSupplierContractPriceSchedules(ctx, req)
}
