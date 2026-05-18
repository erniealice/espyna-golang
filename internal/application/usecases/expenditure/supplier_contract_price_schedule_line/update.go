package suppliercontractpriceschedulesline

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

// UpdateSupplierContractPriceScheduleLineRepositories groups repository dependencies.
type UpdateSupplierContractPriceScheduleLineRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// UpdateSupplierContractPriceScheduleLineServices groups service dependencies.
type UpdateSupplierContractPriceScheduleLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UpdateSupplierContractPriceScheduleLineUseCase handles updating a schedule-line.
type UpdateSupplierContractPriceScheduleLineUseCase struct {
	repositories UpdateSupplierContractPriceScheduleLineRepositories
	services     UpdateSupplierContractPriceScheduleLineServices
}

// NewUpdateSupplierContractPriceScheduleLineUseCase creates a use case with grouped dependencies.
func NewUpdateSupplierContractPriceScheduleLineUseCase(
	repositories UpdateSupplierContractPriceScheduleLineRepositories,
	services UpdateSupplierContractPriceScheduleLineServices,
) *UpdateSupplierContractPriceScheduleLineUseCase {
	return &UpdateSupplierContractPriceScheduleLineUseCase{repositories: repositories, services: services}
}

// Execute performs the update operation.
func (uc *UpdateSupplierContractPriceScheduleLineUseCase) Execute(ctx context.Context, req *scpslpb.UpdateSupplierContractPriceScheduleLineRequest) (*scpslpb.UpdateSupplierContractPriceScheduleLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceScheduleLine, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule_line.validation.id_required", "Schedule line ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.SupplierContractPriceScheduleLine.UpdateSupplierContractPriceScheduleLine(ctx, req)
}
