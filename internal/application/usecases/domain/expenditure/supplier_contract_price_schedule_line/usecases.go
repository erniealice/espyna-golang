package suppliercontractpriceschedulesline

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

// SupplierContractPriceScheduleLineRepositories groups all repository dependencies.
type SupplierContractPriceScheduleLineRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// SupplierContractPriceScheduleLineServices groups all service dependencies.
type SupplierContractPriceScheduleLineServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all schedule-line use cases.
type UseCases struct {
	CreateSupplierContractPriceScheduleLine *CreateSupplierContractPriceScheduleLineUseCase
	ReadSupplierContractPriceScheduleLine   *ReadSupplierContractPriceScheduleLineUseCase
	UpdateSupplierContractPriceScheduleLine *UpdateSupplierContractPriceScheduleLineUseCase
	DeleteSupplierContractPriceScheduleLine *DeleteSupplierContractPriceScheduleLineUseCase
	ListSupplierContractPriceScheduleLines  *ListSupplierContractPriceScheduleLinesUseCase
	ResolveActiveScheduleLine               *ResolveActiveScheduleLineUseCase
}

// NewUseCases creates a new collection of supplier contract price schedule line use cases.
func NewUseCases(
	repositories SupplierContractPriceScheduleLineRepositories,
	services SupplierContractPriceScheduleLineServices,
) *UseCases {
	return &UseCases{
		CreateSupplierContractPriceScheduleLine: NewCreateSupplierContractPriceScheduleLineUseCase(
			CreateSupplierContractPriceScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			CreateSupplierContractPriceScheduleLineServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadSupplierContractPriceScheduleLine: NewReadSupplierContractPriceScheduleLineUseCase(
			ReadSupplierContractPriceScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			ReadSupplierContractPriceScheduleLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		UpdateSupplierContractPriceScheduleLine: NewUpdateSupplierContractPriceScheduleLineUseCase(
			UpdateSupplierContractPriceScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			UpdateSupplierContractPriceScheduleLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		DeleteSupplierContractPriceScheduleLine: NewDeleteSupplierContractPriceScheduleLineUseCase(
			DeleteSupplierContractPriceScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			DeleteSupplierContractPriceScheduleLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListSupplierContractPriceScheduleLines: NewListSupplierContractPriceScheduleLinesUseCase(
			ListSupplierContractPriceScheduleLinesRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			ListSupplierContractPriceScheduleLinesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ResolveActiveScheduleLine: NewResolveActiveScheduleLineUseCase(
			ResolveActiveScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			ResolveActiveScheduleLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
