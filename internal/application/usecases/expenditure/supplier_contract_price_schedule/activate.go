package suppliercontractpriceschedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// ActivateSupplierContractPriceScheduleRepositories groups repository dependencies.
type ActivateSupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// ActivateSupplierContractPriceScheduleServices groups service dependencies.
type ActivateSupplierContractPriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ActivateSupplierContractPriceScheduleUseCase transitions SCHEDULED -> ACTIVE.
//
// Auto-supersede policy: when activating a row, any existing row for the same
// supplier_contract_id with status=ACTIVE is auto-superseded in the same
// transaction. The DB-side partial unique index
// `supplier_contract_price_schedule_one_active_per_contract` is the
// defense-in-depth backstop.
type ActivateSupplierContractPriceScheduleUseCase struct {
	repositories ActivateSupplierContractPriceScheduleRepositories
	services     ActivateSupplierContractPriceScheduleServices
}

// NewActivateSupplierContractPriceScheduleUseCase creates a use case with grouped dependencies.
func NewActivateSupplierContractPriceScheduleUseCase(
	repositories ActivateSupplierContractPriceScheduleRepositories,
	services ActivateSupplierContractPriceScheduleServices,
) *ActivateSupplierContractPriceScheduleUseCase {
	return &ActivateSupplierContractPriceScheduleUseCase{repositories: repositories, services: services}
}

// Execute performs the activate operation.
func (uc *ActivateSupplierContractPriceScheduleUseCase) Execute(ctx context.Context, req *scpspb.ActivateSupplierContractPriceScheduleRequest) (*scpspb.ActivateSupplierContractPriceScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceSchedule, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractPriceScheduleId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.validation.id_required", "Supplier contract price schedule ID is required [DEFAULT]"))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *scpspb.ActivateSupplierContractPriceScheduleResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("supplier contract price schedule activation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.executeCore(ctx, req)
}

func (uc *ActivateSupplierContractPriceScheduleUseCase) executeCore(ctx context.Context, req *scpspb.ActivateSupplierContractPriceScheduleRequest) (*scpspb.ActivateSupplierContractPriceScheduleResponse, error) {
	// Look up the row being activated to get its supplier_contract_id.
	readResp, err := uc.repositories.SupplierContractPriceSchedule.ReadSupplierContractPriceSchedule(ctx, &scpspb.ReadSupplierContractPriceScheduleRequest{
		Data: &scpspb.SupplierContractPriceSchedule{Id: req.GetSupplierContractPriceScheduleId()},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read schedule: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.errors.not_found", "[ERR-DEFAULT] Schedule not found"))
	}
	target := readResp.Data[0]

	// Auto-supersede the prior ACTIVE row (if any) for the same contract.
	listResp, err := uc.repositories.SupplierContractPriceSchedule.ListSupplierContractPriceSchedules(ctx, &scpspb.ListSupplierContractPriceSchedulesRequest{
		SupplierContractId: ptr(target.GetSupplierContractId()),
		Status:             ptrStatus(scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_ACTIVE),
	})
	if err == nil && listResp != nil {
		for _, row := range listResp.Data {
			if row.Id == target.Id {
				continue
			}
			if _, err := uc.repositories.SupplierContractPriceSchedule.SupersedeSupplierContractPriceSchedule(ctx, &scpspb.SupersedeSupplierContractPriceScheduleRequest{
				SupplierContractPriceScheduleId: row.Id,
			}); err != nil {
				return nil, fmt.Errorf("failed to auto-supersede prior active schedule %s: %w", row.Id, err)
			}
		}
	}

	resp, err := uc.repositories.SupplierContractPriceSchedule.ActivateSupplierContractPriceSchedule(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.errors.activate_failed", "[ERR-DEFAULT] Failed to activate schedule")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// ptr is a small helper for taking the address of a string literal at the
// supplier contract id list filter.
func ptr(s string) *string { return &s }

// ptrStatus is a small helper for taking the address of a status enum.
func ptrStatus(s scpspb.SupplierContractPriceScheduleStatus) *scpspb.SupplierContractPriceScheduleStatus {
	return &s
}
