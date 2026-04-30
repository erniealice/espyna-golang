package suppliercontractpriceschedulesline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

// DeleteSupplierContractPriceScheduleLineRepositories groups repository dependencies.
type DeleteSupplierContractPriceScheduleLineRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// DeleteSupplierContractPriceScheduleLineServices groups service dependencies.
type DeleteSupplierContractPriceScheduleLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteSupplierContractPriceScheduleLineUseCase handles deleting a schedule-line.
type DeleteSupplierContractPriceScheduleLineUseCase struct {
	repositories DeleteSupplierContractPriceScheduleLineRepositories
	services     DeleteSupplierContractPriceScheduleLineServices
}

// NewDeleteSupplierContractPriceScheduleLineUseCase creates a use case with grouped dependencies.
func NewDeleteSupplierContractPriceScheduleLineUseCase(
	repositories DeleteSupplierContractPriceScheduleLineRepositories,
	services DeleteSupplierContractPriceScheduleLineServices,
) *DeleteSupplierContractPriceScheduleLineUseCase {
	return &DeleteSupplierContractPriceScheduleLineUseCase{repositories: repositories, services: services}
}

// Execute performs the delete operation.
func (uc *DeleteSupplierContractPriceScheduleLineUseCase) Execute(ctx context.Context, req *scpslpb.DeleteSupplierContractPriceScheduleLineRequest) (*scpslpb.DeleteSupplierContractPriceScheduleLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceScheduleLine, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule_line.validation.id_required", "Schedule line ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractPriceScheduleLine.DeleteSupplierContractPriceScheduleLine(ctx, req)
}
