package suppliercontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// ApproveSupplierContractRepositories groups repository dependencies.
type ApproveSupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// ApproveSupplierContractServices groups service dependencies.
type ApproveSupplierContractServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ApproveSupplierContractUseCase transitions a contract from PENDING_APPROVAL → APPROVED.
type ApproveSupplierContractUseCase struct {
	repositories ApproveSupplierContractRepositories
	services     ApproveSupplierContractServices
}

// NewApproveSupplierContractUseCase creates a use case with grouped dependencies.
func NewApproveSupplierContractUseCase(
	repositories ApproveSupplierContractRepositories,
	services ApproveSupplierContractServices,
) *ApproveSupplierContractUseCase {
	return &ApproveSupplierContractUseCase{repositories: repositories, services: services}
}

// Execute performs the approve supplier contract operation.
func (uc *ApproveSupplierContractUseCase) Execute(ctx context.Context, req *suppliercontractpb.ApproveSupplierContractRequest) (*suppliercontractpb.ApproveSupplierContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContract, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if req.ApprovedBy == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.approved_by_required", "Approver ID is required [DEFAULT]"))
	}

	resp, err := uc.repositories.SupplierContract.ApproveSupplierContract(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.errors.approve_failed", "[ERR-DEFAULT] Failed to approve supplier contract")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
