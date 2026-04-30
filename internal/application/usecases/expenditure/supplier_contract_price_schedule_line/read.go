package suppliercontractpriceschedulesline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

// ReadSupplierContractPriceScheduleLineRepositories groups repository dependencies.
type ReadSupplierContractPriceScheduleLineRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// ReadSupplierContractPriceScheduleLineServices groups service dependencies.
type ReadSupplierContractPriceScheduleLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadSupplierContractPriceScheduleLineUseCase handles reading a schedule-line.
type ReadSupplierContractPriceScheduleLineUseCase struct {
	repositories ReadSupplierContractPriceScheduleLineRepositories
	services     ReadSupplierContractPriceScheduleLineServices
}

// NewReadSupplierContractPriceScheduleLineUseCase creates a use case with grouped dependencies.
func NewReadSupplierContractPriceScheduleLineUseCase(
	repositories ReadSupplierContractPriceScheduleLineRepositories,
	services ReadSupplierContractPriceScheduleLineServices,
) *ReadSupplierContractPriceScheduleLineUseCase {
	return &ReadSupplierContractPriceScheduleLineUseCase{repositories: repositories, services: services}
}

// Execute performs the read schedule-line operation.
func (uc *ReadSupplierContractPriceScheduleLineUseCase) Execute(ctx context.Context, req *scpslpb.ReadSupplierContractPriceScheduleLineRequest) (*scpslpb.ReadSupplierContractPriceScheduleLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceScheduleLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule_line.validation.id_required", "Schedule line ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractPriceScheduleLine.ReadSupplierContractPriceScheduleLine(ctx, req)
}
