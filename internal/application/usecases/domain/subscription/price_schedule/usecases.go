package price_schedule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// PriceScheduleRepositories groups all repository dependencies for price schedule use cases
type PriceScheduleRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// PriceScheduleServices groups all business service dependencies for price schedule use cases
type PriceScheduleServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator // Only for CreatePriceSchedule
}

// UseCases contains all price_schedule-related use cases
type UseCases struct {
	CreatePriceSchedule          *CreatePriceScheduleUseCase
	ReadPriceSchedule            *ReadPriceScheduleUseCase
	UpdatePriceSchedule          *UpdatePriceScheduleUseCase
	DeletePriceSchedule          *DeletePriceScheduleUseCase
	ListPriceSchedules           *ListPriceSchedulesUseCase
	GetPriceScheduleListPageData *GetPriceScheduleListPageDataUseCase
	GetPriceScheduleItemPageData *GetPriceScheduleItemPageDataUseCase
	FindApplicablePriceSchedule  *FindApplicablePriceScheduleUseCase
}

// NewUseCases creates a new collection of price_schedule use cases
func NewUseCases(
	repositories PriceScheduleRepositories,
	services PriceScheduleServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	createServices := CreatePriceScheduleServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	readServices := ReadPriceScheduleServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	updateServices := UpdatePriceScheduleServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeletePriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	deleteServices := DeletePriceScheduleServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPriceSchedulesRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	listServices := ListPriceSchedulesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetPriceScheduleListPageDataRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	listPageDataServices := GetPriceScheduleListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetPriceScheduleItemPageDataRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	itemPageDataServices := GetPriceScheduleItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	findApplicableRepos := FindApplicablePriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	findApplicableServices := FindApplicablePriceScheduleServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePriceSchedule:          NewCreatePriceScheduleUseCase(createRepos, createServices),
		ReadPriceSchedule:            NewReadPriceScheduleUseCase(readRepos, readServices),
		UpdatePriceSchedule:          NewUpdatePriceScheduleUseCase(updateRepos, updateServices),
		DeletePriceSchedule:          NewDeletePriceScheduleUseCase(deleteRepos, deleteServices),
		ListPriceSchedules:           NewListPriceSchedulesUseCase(listRepos, listServices),
		GetPriceScheduleListPageData: NewGetPriceScheduleListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPriceScheduleItemPageData: NewGetPriceScheduleItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		FindApplicablePriceSchedule:  NewFindApplicablePriceScheduleUseCase(findApplicableRepos, findApplicableServices),
	}
}
