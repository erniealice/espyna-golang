package suppliercontractline

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// GetSupplierContractLineListPageDataRepositories groups repository dependencies.
type GetSupplierContractLineListPageDataRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// GetSupplierContractLineListPageDataServices groups service dependencies.
type GetSupplierContractLineListPageDataServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// GetSupplierContractLineListPageDataUseCase fetches paginated contract line list data.
type GetSupplierContractLineListPageDataUseCase struct {
	repositories GetSupplierContractLineListPageDataRepositories
	services     GetSupplierContractLineListPageDataServices
}

// NewGetSupplierContractLineListPageDataUseCase creates a use case with grouped dependencies.
func NewGetSupplierContractLineListPageDataUseCase(
	repositories GetSupplierContractLineListPageDataRepositories,
	services GetSupplierContractLineListPageDataServices,
) *GetSupplierContractLineListPageDataUseCase {
	return &GetSupplierContractLineListPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get supplier contract line list page data operation.
func (uc *GetSupplierContractLineListPageDataUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.GetSupplierContractLineListPageDataRequest) (*suppliercontractlinepb.GetSupplierContractLineListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContractLine, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract_line.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	resp, err := uc.repositories.SupplierContractLine.GetSupplierContractLineListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract_line.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load supplier contract line list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
