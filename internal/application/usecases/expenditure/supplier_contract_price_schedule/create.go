package suppliercontractpriceschedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

const entitySupplierContractPriceSchedule = "supplier_contract_price_schedule"

// CreateSupplierContractPriceScheduleRepositories groups repository dependencies.
type CreateSupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// CreateSupplierContractPriceScheduleServices groups service dependencies.
type CreateSupplierContractPriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateSupplierContractPriceScheduleUseCase handles creating a new schedule.
//
// CRIT-1 contract: this use case calls validateNoOverlap before persisting.
// The validator lives in `validate_no_overlap.go` (Opus scope). Until that
// file lands, build will fail with "undefined: validateNoOverlap" — that's
// the agreed-upon synchronization point.
type CreateSupplierContractPriceScheduleUseCase struct {
	repositories CreateSupplierContractPriceScheduleRepositories
	services     CreateSupplierContractPriceScheduleServices
}

// NewCreateSupplierContractPriceScheduleUseCase creates a use case with grouped dependencies.
func NewCreateSupplierContractPriceScheduleUseCase(
	repositories CreateSupplierContractPriceScheduleRepositories,
	services CreateSupplierContractPriceScheduleServices,
) *CreateSupplierContractPriceScheduleUseCase {
	return &CreateSupplierContractPriceScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create supplier contract price schedule operation.
func (uc *CreateSupplierContractPriceScheduleUseCase) Execute(ctx context.Context, req *scpspb.CreateSupplierContractPriceScheduleRequest) (*scpspb.CreateSupplierContractPriceScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceSchedule, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *scpspb.CreateSupplierContractPriceScheduleResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("supplier contract price schedule creation failed: %w", err)
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

func (uc *CreateSupplierContractPriceScheduleUseCase) executeCore(ctx context.Context, req *scpspb.CreateSupplierContractPriceScheduleRequest) (*scpspb.CreateSupplierContractPriceScheduleResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.validation.data_required", "Supplier contract price schedule data is required [DEFAULT]"))
	}
	if req.Data.SupplierContractId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.validation.supplier_contract_id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if req.Data.DateTimeStart == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule.validation.date_time_start_required", "Schedule start date is required [DEFAULT]"))
	}

	// CRIT-1: defense-in-depth overlap check at the use-case layer (postgres GIST
	// exclusion constraint is the DB-side enforcement).
	if err := validateNoOverlap(ctx, uc.repositories.SupplierContractPriceSchedule, req.Data); err != nil {
		return nil, err
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	// Default status: SCHEDULED — caller activates explicitly via the
	// activate use case (which auto-supersedes any prior ACTIVE for the contract).
	if req.Data.Status == scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_UNSPECIFIED {
		req.Data.Status = scpspb.SupplierContractPriceScheduleStatus_SUPPLIER_CONTRACT_PRICE_SCHEDULE_STATUS_SCHEDULED
	}

	return uc.repositories.SupplierContractPriceSchedule.CreateSupplierContractPriceSchedule(ctx, req)
}
