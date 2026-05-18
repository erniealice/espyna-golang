package suppliercontract

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

const entitySupplierContract = "supplier_contract"

// CreateSupplierContractRepositories groups repository dependencies.
type CreateSupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// CreateSupplierContractServices groups service dependencies.
type CreateSupplierContractServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateSupplierContractUseCase handles the business logic for creating supplier contracts.
type CreateSupplierContractUseCase struct {
	repositories CreateSupplierContractRepositories
	services     CreateSupplierContractServices
}

// NewCreateSupplierContractUseCase creates a use case with grouped dependencies.
func NewCreateSupplierContractUseCase(
	repositories CreateSupplierContractRepositories,
	services CreateSupplierContractServices,
) *CreateSupplierContractUseCase {
	return &CreateSupplierContractUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create supplier contract operation.
func (uc *CreateSupplierContractUseCase) Execute(ctx context.Context, req *suppliercontractpb.CreateSupplierContractRequest) (*suppliercontractpb.CreateSupplierContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContract, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *suppliercontractpb.CreateSupplierContractResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("supplier contract creation failed: %w", err)
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

func (uc *CreateSupplierContractUseCase) executeCore(ctx context.Context, req *suppliercontractpb.CreateSupplierContractRequest) (*suppliercontractpb.CreateSupplierContractResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.data_required", "Supplier contract data is required [DEFAULT]"))
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

	// Default status: DRAFT (per entity-status-conventions, create defaults to a valid status).
	if req.Data.Status == suppliercontractpb.SupplierContractStatus_SUPPLIER_CONTRACT_STATUS_UNSPECIFIED {
		req.Data.Status = suppliercontractpb.SupplierContractStatus_SUPPLIER_CONTRACT_STATUS_DRAFT
	}

	// Initialise balance fields to zero if not set.
	zero := int64(0)
	if req.Data.ReleasedAmount == nil {
		req.Data.ReleasedAmount = &zero
	}
	if req.Data.BilledAmount == nil {
		req.Data.BilledAmount = &zero
	}
	if req.Data.CommittedAmount != nil && req.Data.RemainingAmount == nil {
		remaining := *req.Data.CommittedAmount
		req.Data.RemainingAmount = &remaining
	} else if req.Data.RemainingAmount == nil {
		req.Data.RemainingAmount = &zero
	}

	return uc.repositories.SupplierContract.CreateSupplierContract(ctx, req)
}
