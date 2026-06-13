package suppliercontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// GetSupplierContractListPageDataRepositories groups repository dependencies.
type GetSupplierContractListPageDataRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// GetSupplierContractListPageDataServices groups service dependencies.
type GetSupplierContractListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entitySupplierContract,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if uc.repositories.SupplierContract == nil {
		return nil, errors.New("supplier contract repository is not available")
	}
	resp, err := uc.repositories.SupplierContract.GetSupplierContractListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier contract list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
