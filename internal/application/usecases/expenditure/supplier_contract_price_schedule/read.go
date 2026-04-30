package suppliercontractpriceschedule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// ReadSupplierContractPriceScheduleRepositories groups repository dependencies.
type ReadSupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// ReadSupplierContractPriceScheduleServices groups service dependencies.
type ReadSupplierContractPriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadSupplierContractPriceScheduleUseCase handles reading a schedule.
type ReadSupplierContractPriceScheduleUseCase struct {
	repositories ReadSupplierContractPriceScheduleRepositories
	services     ReadSupplierContractPriceScheduleServices
}

// NewReadSupplierContractPriceScheduleUseCase creates a use case with grouped dependencies.
func NewReadSupplierContractPriceScheduleUseCase(
	repositories ReadSupplierContractPriceScheduleRepositories,
	services ReadSupplierContractPriceScheduleServices,
) *ReadSupplierContractPriceScheduleUseCase {
	return &ReadSupplierContractPriceScheduleUseCase{repositories: repositories, services: services}
}

// Execute performs the read schedule operation.
func (uc *ReadSupplierContractPriceScheduleUseCase) Execute(ctx context.Context, req *scpspb.ReadSupplierContractPriceScheduleRequest) (*scpspb.ReadSupplierContractPriceScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceSchedule, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.validation.id_required", "Supplier contract price schedule ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractPriceSchedule.ReadSupplierContractPriceSchedule(ctx, req)
}
