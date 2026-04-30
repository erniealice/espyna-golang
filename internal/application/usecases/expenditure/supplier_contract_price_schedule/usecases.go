package suppliercontractpriceschedule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// SupplierContractPriceScheduleRepositories groups all repository dependencies.
type SupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// SupplierContractPriceScheduleServices groups all service dependencies.
type SupplierContractPriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all supplier contract price schedule use cases.
//
// SPS Wave 2 Opus: ValidateNoOverlap is implemented as the package-private
// validateNoOverlap helper in `validate_no_overlap.go` and called from
// CreateSupplierContractPriceScheduleUseCase / UpdateSupplierContractPriceScheduleUseCase
// at executeCore time — it does not appear as a top-level UseCases field.
type UseCases struct {
	CreateSupplierContractPriceSchedule    *CreateSupplierContractPriceScheduleUseCase
	ReadSupplierContractPriceSchedule      *ReadSupplierContractPriceScheduleUseCase
	UpdateSupplierContractPriceSchedule    *UpdateSupplierContractPriceScheduleUseCase
	DeleteSupplierContractPriceSchedule    *DeleteSupplierContractPriceScheduleUseCase
	ListSupplierContractPriceSchedules     *ListSupplierContractPriceSchedulesUseCase
	ActivateSupplierContractPriceSchedule  *ActivateSupplierContractPriceScheduleUseCase
	SupersedeSupplierContractPriceSchedule *SupersedeSupplierContractPriceScheduleUseCase
	// SPS Wave 2 Opus: ValidateNoOverlap registered separately as the package-private
	// validateNoOverlap helper invoked from create.go + update.go executeCore path.
}

// NewUseCases creates a new collection of supplier contract price schedule use cases.
func NewUseCases(
	repositories SupplierContractPriceScheduleRepositories,
	services SupplierContractPriceScheduleServices,
) *UseCases {
	return &UseCases{
		CreateSupplierContractPriceSchedule: NewCreateSupplierContractPriceScheduleUseCase(
			CreateSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			CreateSupplierContractPriceScheduleServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadSupplierContractPriceSchedule: NewReadSupplierContractPriceScheduleUseCase(
			ReadSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			ReadSupplierContractPriceScheduleServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateSupplierContractPriceSchedule: NewUpdateSupplierContractPriceScheduleUseCase(
			UpdateSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			UpdateSupplierContractPriceScheduleServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteSupplierContractPriceSchedule: NewDeleteSupplierContractPriceScheduleUseCase(
			DeleteSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			DeleteSupplierContractPriceScheduleServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListSupplierContractPriceSchedules: NewListSupplierContractPriceSchedulesUseCase(
			ListSupplierContractPriceSchedulesRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			ListSupplierContractPriceSchedulesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ActivateSupplierContractPriceSchedule: NewActivateSupplierContractPriceScheduleUseCase(
			ActivateSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			ActivateSupplierContractPriceScheduleServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		SupersedeSupplierContractPriceSchedule: NewSupersedeSupplierContractPriceScheduleUseCase(
			SupersedeSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			SupersedeSupplierContractPriceScheduleServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
