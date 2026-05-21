package suppliercontractpriceschedulesline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

const entitySupplierContractPriceScheduleLine = "supplier_contract_price_schedule_line"

// CreateSupplierContractPriceScheduleLineRepositories groups repository dependencies.
type CreateSupplierContractPriceScheduleLineRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// CreateSupplierContractPriceScheduleLineServices groups service dependencies.
type CreateSupplierContractPriceScheduleLineServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateSupplierContractPriceScheduleLineUseCase handles creating a new schedule line.
type CreateSupplierContractPriceScheduleLineUseCase struct {
	repositories CreateSupplierContractPriceScheduleLineRepositories
	services     CreateSupplierContractPriceScheduleLineServices
}

// NewCreateSupplierContractPriceScheduleLineUseCase creates a use case with grouped dependencies.
func NewCreateSupplierContractPriceScheduleLineUseCase(
	repositories CreateSupplierContractPriceScheduleLineRepositories,
	services CreateSupplierContractPriceScheduleLineServices,
) *CreateSupplierContractPriceScheduleLineUseCase {
	return &CreateSupplierContractPriceScheduleLineUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create operation.
func (uc *CreateSupplierContractPriceScheduleLineUseCase) Execute(ctx context.Context, req *scpslpb.CreateSupplierContractPriceScheduleLineRequest) (*scpslpb.CreateSupplierContractPriceScheduleLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractPriceScheduleLine, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *scpslpb.CreateSupplierContractPriceScheduleLineResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("supplier contract price schedule line creation failed: %w", err)
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

func (uc *CreateSupplierContractPriceScheduleLineUseCase) executeCore(ctx context.Context, req *scpslpb.CreateSupplierContractPriceScheduleLineRequest) (*scpslpb.CreateSupplierContractPriceScheduleLineResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule_line.validation.data_required", "Schedule line data is required [DEFAULT]"))
	}
	if req.Data.SupplierContractPriceScheduleId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule_line.validation.schedule_id_required", "Schedule ID is required [DEFAULT]"))
	}
	if req.Data.SupplierContractLineId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_price_schedule_line.validation.contract_line_id_required", "Supplier contract line ID is required [DEFAULT]"))
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

	return uc.repositories.SupplierContractPriceScheduleLine.CreateSupplierContractPriceScheduleLine(ctx, req)
}
