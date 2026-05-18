package suppliercontractline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

const entitySupplierContractLine = "supplier_contract_line"

// CreateSupplierContractLineRepositories groups repository dependencies.
type CreateSupplierContractLineRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// CreateSupplierContractLineServices groups service dependencies.
type CreateSupplierContractLineServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateSupplierContractLineUseCase handles creating a supplier contract line.
type CreateSupplierContractLineUseCase struct {
	repositories CreateSupplierContractLineRepositories
	services     CreateSupplierContractLineServices
}

// NewCreateSupplierContractLineUseCase creates a use case with grouped dependencies.
func NewCreateSupplierContractLineUseCase(
	repositories CreateSupplierContractLineRepositories,
	services CreateSupplierContractLineServices,
) *CreateSupplierContractLineUseCase {
	return &CreateSupplierContractLineUseCase{repositories: repositories, services: services}
}

// Execute performs the create supplier contract line operation.
func (uc *CreateSupplierContractLineUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.CreateSupplierContractLineRequest) (*suppliercontractlinepb.CreateSupplierContractLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractLine, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_line.validation.data_required", "Supplier contract line data is required [DEFAULT]"))
	}
	if req.Data.SupplierContractId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_line.validation.contract_id_required", "Supplier contract ID is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *suppliercontractlinepb.CreateSupplierContractLineResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierContractLine.CreateSupplierContractLine(txCtx, req)
			if err != nil {
				return fmt.Errorf("supplier contract line creation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierContractLine.CreateSupplierContractLine(ctx, req)
}
