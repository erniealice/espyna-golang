package suppliercontractline

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// GetSupplierContractLineItemPageDataRepositories groups repository dependencies.
type GetSupplierContractLineItemPageDataRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// GetSupplierContractLineItemPageDataServices groups service dependencies.
type GetSupplierContractLineItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetSupplierContractLineItemPageDataUseCase fetches a single contract line detail.
type GetSupplierContractLineItemPageDataUseCase struct {
	repositories GetSupplierContractLineItemPageDataRepositories
	services     GetSupplierContractLineItemPageDataServices
}

// NewGetSupplierContractLineItemPageDataUseCase creates a use case with grouped dependencies.
func NewGetSupplierContractLineItemPageDataUseCase(
	repositories GetSupplierContractLineItemPageDataRepositories,
	services GetSupplierContractLineItemPageDataServices,
) *GetSupplierContractLineItemPageDataUseCase {
	return &GetSupplierContractLineItemPageDataUseCase{repositories: repositories, services: services}
}

// Execute performs the get supplier contract line item page data operation.
func (uc *GetSupplierContractLineItemPageDataUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.GetSupplierContractLineItemPageDataRequest) (*suppliercontractlinepb.GetSupplierContractLineItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entitySupplierContractLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractLineId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_line.validation.id_required", "Supplier contract line ID is required [DEFAULT]"))
	}
	resp, err := uc.repositories.SupplierContractLine.GetSupplierContractLineItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"supplier_contract_line.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load supplier contract line")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
