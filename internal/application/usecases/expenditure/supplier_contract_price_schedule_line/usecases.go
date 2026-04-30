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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadSupplierContractPriceScheduleLine: NewReadSupplierContractPriceScheduleLineUseCase(
			ReadSupplierContractPriceScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			ReadSupplierContractPriceScheduleLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateSupplierContractPriceScheduleLine: NewUpdateSupplierContractPriceScheduleLineUseCase(
			UpdateSupplierContractPriceScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			UpdateSupplierContractPriceScheduleLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteSupplierContractPriceScheduleLine: NewDeleteSupplierContractPriceScheduleLineUseCase(
			DeleteSupplierContractPriceScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			DeleteSupplierContractPriceScheduleLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListSupplierContractPriceScheduleLines: NewListSupplierContractPriceScheduleLinesUseCase(
			ListSupplierContractPriceScheduleLinesRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			ListSupplierContractPriceScheduleLinesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ResolveActiveScheduleLine: NewResolveActiveScheduleLineUseCase(
			ResolveActiveScheduleLineRepositories{SupplierContractPriceScheduleLine: repositories.SupplierContractPriceScheduleLine},
			ResolveActiveScheduleLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
