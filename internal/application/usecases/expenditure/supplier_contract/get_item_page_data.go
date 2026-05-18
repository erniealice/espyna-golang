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

// GetSupplierContractItemPageDataRepositories groups repository dependencies.
type GetSupplierContractItemPageDataRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// GetSupplierContractItemPageDataServices groups service dependencies.
type GetSupplierContractItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetSupplierContractItemPageDataUseCase handles fetching single supplier contract detail data.
type GetSupplierContractItemPageDataUseCase struct {
	repositories GetSupplierContractItemPageDataRepositories
	services     GetSupplierContractItemPageDataServices
}

// NewGetSupplierContractItemPageDataUseCase creates a use case with grouped dependencies.
func NewGetSupplierContractItemPageDataUseCase(
	repositories GetSupplierContractItemPageDataRepositories,
	services GetSupplierContractItemPageDataServices,
) *GetSupplierContractItemPageDataUseCase {
	return &GetSupplierContractItemPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get supplier contract item page data operation.
func (uc *GetSupplierContractItemPageDataUseCase) Execute(ctx context.Context, req *suppliercontractpb.GetSupplierContractItemPageDataRequest) (*suppliercontractpb.GetSupplierContractItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContract, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if uc.repositories.SupplierContract == nil {
		return nil, errors.New("supplier contract repository is not available")
	}
	resp, err := uc.repositories.SupplierContract.GetSupplierContractItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load supplier contract")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
