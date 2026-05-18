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

// GetSupplierContractListPageDataRepositories groups repository dependencies.
type GetSupplierContractListPageDataRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// GetSupplierContractListPageDataServices groups service dependencies.
type GetSupplierContractListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetSupplierContractListPageDataUseCase handles fetching paginated supplier contract list data.
type GetSupplierContractListPageDataUseCase struct {
	repositories GetSupplierContractListPageDataRepositories
	services     GetSupplierContractListPageDataServices
}

// NewGetSupplierContractListPageDataUseCase creates a use case with grouped dependencies.
func NewGetSupplierContractListPageDataUseCase(
	repositories GetSupplierContractListPageDataRepositories,
	services GetSupplierContractListPageDataServices,
) *GetSupplierContractListPageDataUseCase {
	return &GetSupplierContractListPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get supplier contract list page data operation.
func (uc *GetSupplierContractListPageDataUseCase) Execute(ctx context.Context, req *suppliercontractpb.GetSupplierContractListPageDataRequest) (*suppliercontractpb.GetSupplierContractListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContract, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if uc.repositories.SupplierContract == nil {
		return nil, errors.New("supplier contract repository is not available")
	}
	resp, err := uc.repositories.SupplierContract.GetSupplierContractListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier contract list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
