package suppliercontractpriceschedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// SupersedeSupplierContractPriceScheduleRepositories groups repository dependencies.
type SupersedeSupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// SupersedeSupplierContractPriceScheduleServices groups service dependencies.
type SupersedeSupplierContractPriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// SupersedeSupplierContractPriceScheduleUseCase transitions ACTIVE -> SUPERSEDED.
//
// Operator-driven (or scheduled-transition-job-driven) flip when a window's
// date_time_end has passed. Idempotent on `(schedule_id, target_status)`.
type SupersedeSupplierContractPriceScheduleUseCase struct {
	repositories SupersedeSupplierContractPriceScheduleRepositories
	services     SupersedeSupplierContractPriceScheduleServices
}

// NewSupersedeSupplierContractPriceScheduleUseCase creates a use case with grouped dependencies.
func NewSupersedeSupplierContractPriceScheduleUseCase(
	repositories SupersedeSupplierContractPriceScheduleRepositories,
	services SupersedeSupplierContractPriceScheduleServices,
) *SupersedeSupplierContractPriceScheduleUseCase {
	return &SupersedeSupplierContractPriceScheduleUseCase{repositories: repositories, services: services}
}

// Execute performs the supersede operation.
func (uc *SupersedeSupplierContractPriceScheduleUseCase) Execute(ctx context.Context, req *scpspb.SupersedeSupplierContractPriceScheduleRequest) (*scpspb.SupersedeSupplierContractPriceScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceSchedule, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.validation.id_required", "Supplier contract price schedule ID is required [DEFAULT]"))
	}

	resp, err := uc.repositories.SupplierContractPriceSchedule.SupersedeSupplierContractPriceSchedule(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.errors.supersede_failed", "[ERR-DEFAULT] Failed to supersede schedule")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
