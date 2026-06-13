package suppliercontractpriceschedule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	scpspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule"
)

// SupplierContractPriceScheduleRepositories groups all repository dependencies.
type SupplierContractPriceScheduleRepositories struct {
	SupplierContractPriceSchedule scpspb.SupplierContractPriceScheduleDomainServiceServer
}

// SupplierContractPriceScheduleServices groups all service dependencies.
type SupplierContractPriceScheduleServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadSupplierContractPriceSchedule: NewReadSupplierContractPriceScheduleUseCase(
			ReadSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			ReadSupplierContractPriceScheduleServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		UpdateSupplierContractPriceSchedule: NewUpdateSupplierContractPriceScheduleUseCase(
			UpdateSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			UpdateSupplierContractPriceScheduleServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		DeleteSupplierContractPriceSchedule: NewDeleteSupplierContractPriceScheduleUseCase(
			DeleteSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			DeleteSupplierContractPriceScheduleServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListSupplierContractPriceSchedules: NewListSupplierContractPriceSchedulesUseCase(
			ListSupplierContractPriceSchedulesRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			ListSupplierContractPriceSchedulesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ActivateSupplierContractPriceSchedule: NewActivateSupplierContractPriceScheduleUseCase(
			ActivateSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			ActivateSupplierContractPriceScheduleServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		SupersedeSupplierContractPriceSchedule: NewSupersedeSupplierContractPriceScheduleUseCase(
			SupersedeSupplierContractPriceScheduleRepositories{SupplierContractPriceSchedule: repositories.SupplierContractPriceSchedule},
			SupersedeSupplierContractPriceScheduleServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
