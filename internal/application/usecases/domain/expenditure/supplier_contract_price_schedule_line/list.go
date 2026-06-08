package suppliercontractpriceschedulesline

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

// ListSupplierContractPriceScheduleLinesRepositories groups repository dependencies.
type ListSupplierContractPriceScheduleLinesRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// ListSupplierContractPriceScheduleLinesServices groups service dependencies.
type ListSupplierContractPriceScheduleLinesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListSupplierContractPriceScheduleLinesUseCase handles listing schedule lines.
type ListSupplierContractPriceScheduleLinesUseCase struct {
	repositories ListSupplierContractPriceScheduleLinesRepositories
	services     ListSupplierContractPriceScheduleLinesServices
}

// NewListSupplierContractPriceScheduleLinesUseCase creates a use case with grouped dependencies.
func NewListSupplierContractPriceScheduleLinesUseCase(
	repositories ListSupplierContractPriceScheduleLinesRepositories,
	services ListSupplierContractPriceScheduleLinesServices,
) *ListSupplierContractPriceScheduleLinesUseCase {
	return &ListSupplierContractPriceScheduleLinesUseCase{repositories: repositories, services: services}
}

// Execute performs the list operation.
func (uc *ListSupplierContractPriceScheduleLinesUseCase) Execute(ctx context.Context, req *scpslpb.ListSupplierContractPriceScheduleLinesRequest) (*scpslpb.ListSupplierContractPriceScheduleLinesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContractPriceScheduleLine, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.SupplierContractPriceScheduleLine.ListSupplierContractPriceScheduleLines(ctx, req)
}
