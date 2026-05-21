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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadSupplierContractLine: NewReadSupplierContractLineUseCase(
			ReadSupplierContractLineRepositories{SupplierContractLine: repositories.SupplierContractLine},
			ReadSupplierContractLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		UpdateSupplierContractLine: NewUpdateSupplierContractLineUseCase(
			UpdateSupplierContractLineRepositories{SupplierContractLine: repositories.SupplierContractLine},
			UpdateSupplierContractLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		DeleteSupplierContractLine: NewDeleteSupplierContractLineUseCase(
			DeleteSupplierContractLineRepositories{SupplierContractLine: repositories.SupplierContractLine},
			DeleteSupplierContractLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListSupplierContractLines: NewListSupplierContractLinesUseCase(
			ListSupplierContractLinesRepositories{SupplierContractLine: repositories.SupplierContractLine},
			ListSupplierContractLinesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		GetSupplierContractLineListPageData: NewGetSupplierContractLineListPageDataUseCase(
			GetSupplierContractLineListPageDataRepositories{SupplierContractLine: repositories.SupplierContractLine},
			GetSupplierContractLineListPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		GetSupplierContractLineItemPageData: NewGetSupplierContractLineItemPageDataUseCase(
			GetSupplierContractLineItemPageDataRepositories{SupplierContractLine: repositories.SupplierContractLine},
			GetSupplierContractLineItemPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
