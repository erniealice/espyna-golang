package suppliercontractpriceschedule

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// UpdateSupplierContractPriceScheduleRepositories groups repository dependencies.
type UpdateSupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// UpdateSupplierContractPriceScheduleServices groups service dependencies.
type UpdateSupplierContractPriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateSupplierContractPriceScheduleUseCase handles updating a schedule.
type UpdateSupplierContractPriceScheduleUseCase struct {
	repositories UpdateSupplierContractPriceScheduleRepositories
	services     UpdateSupplierContractPriceScheduleServices
}

// NewUpdateSupplierContractPriceScheduleUseCase creates a use case with grouped dependencies.
func NewUpdateSupplierContractPriceScheduleUseCase(
	repositories UpdateSupplierContractPriceScheduleRepositories,
	services UpdateSupplierContractPriceScheduleServices,
) *UpdateSupplierContractPriceScheduleUseCase {
	return &UpdateSupplierContractPriceScheduleUseCase{repositories: repositories, services: services}
}

// Execute performs the update operation.
func (uc *UpdateSupplierContractPriceScheduleUseCase) Execute(ctx context.Context, req *scpspb.UpdateSupplierContractPriceScheduleRequest) (*scpspb.UpdateSupplierContractPriceScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceSchedule, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.validation.id_required", "Supplier contract price schedule ID is required [DEFAULT]"))
	}

	// CRIT-1: re-validate overlap on update (date window or contract may have moved).
	if err := validateNoOverlap(ctx, uc.repositories.SupplierContractPriceSchedule, req.Data); err != nil {
		return nil, err
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.SupplierContractPriceSchedule.UpdateSupplierContractPriceSchedule(ctx, req)
}
