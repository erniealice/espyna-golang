package suppliercontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// TerminateSupplierContractRepositories groups repository dependencies.
type TerminateSupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// TerminateSupplierContractServices groups service dependencies.
type TerminateSupplierContractServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// TerminateSupplierContractUseCase transitions a contract to TERMINATED (terminal state).
type TerminateSupplierContractUseCase struct {
	repositories TerminateSupplierContractRepositories
	services     TerminateSupplierContractServices
}

// NewTerminateSupplierContractUseCase creates a use case with grouped dependencies.
func NewTerminateSupplierContractUseCase(
	repositories TerminateSupplierContractRepositories,
	services TerminateSupplierContractServices,
) *TerminateSupplierContractUseCase {
	return &TerminateSupplierContractUseCase{repositories: repositories, services: services}
}

// Execute performs the terminate supplier contract operation.
func (uc *TerminateSupplierContractUseCase) Execute(ctx context.Context, req *suppliercontractpb.TerminateSupplierContractRequest) (*suppliercontractpb.TerminateSupplierContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContract, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}

	resp, err := uc.repositories.SupplierContract.TerminateSupplierContract(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.errors.terminate_failed", "[ERR-DEFAULT] Failed to terminate supplier contract")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
