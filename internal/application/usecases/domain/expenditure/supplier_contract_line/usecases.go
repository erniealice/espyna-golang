package suppliercontractline

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// SupplierContractLineRepositories groups all repository dependencies.
type SupplierContractLineRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// SupplierContractLineServices groups all business service dependencies.
type SupplierContractLineServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all supplier contract line use cases.
type UseCases struct {
	CreateSupplierContractLine          *CreateSupplierContractLineUseCase
	ReadSupplierContractLine            *ReadSupplierContractLineUseCase
	UpdateSupplierContractLine          *UpdateSupplierContractLineUseCase
	DeleteSupplierContractLine          *DeleteSupplierContractLineUseCase
	ListSupplierContractLines           *ListSupplierContractLinesUseCase
	GetSupplierContractLineListPageData *GetSupplierContractLineListPageDataUseCase
	GetSupplierContractLineItemPageData *GetSupplierContractLineItemPageDataUseCase
}

// NewUseCases creates a new collection of supplier contract line use cases.
func NewUseCases(
	repositories SupplierContractLineRepositories,
	services SupplierContractLineServices,
) *UseCases {
	return &UseCases{
		CreateSupplierContractLine: NewCreateSupplierContractLineUseCase(
			CreateSupplierContractLineRepositories{SupplierContractLine: repositories.SupplierContractLine},
			CreateSupplierContractLineServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadSupplierContractLine: NewReadSupplierContractLineUseCase(
			ReadSupplierContractLineRepositories{SupplierContractLine: repositories.SupplierContractLine},
			ReadSupplierContractLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateSupplierContractLine: NewUpdateSupplierContractLineUseCase(
			UpdateSupplierContractLineRepositories{SupplierContractLine: repositories.SupplierContractLine},
			UpdateSupplierContractLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteSupplierContractLine: NewDeleteSupplierContractLineUseCase(
			DeleteSupplierContractLineRepositories{SupplierContractLine: repositories.SupplierContractLine},
			DeleteSupplierContractLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListSupplierContractLines: NewListSupplierContractLinesUseCase(
			ListSupplierContractLinesRepositories{SupplierContractLine: repositories.SupplierContractLine},
			ListSupplierContractLinesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		GetSupplierContractLineListPageData: NewGetSupplierContractLineListPageDataUseCase(
			GetSupplierContractLineListPageDataRepositories{SupplierContractLine: repositories.SupplierContractLine},
			GetSupplierContractLineListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		GetSupplierContractLineItemPageData: NewGetSupplierContractLineItemPageDataUseCase(
			GetSupplierContractLineItemPageDataRepositories{SupplierContractLine: repositories.SupplierContractLine},
			GetSupplierContractLineItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
