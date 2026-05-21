package cost_schedule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

// Repositories groups all repository dependencies for cost_schedule use cases
type Repositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

// Services groups all business service dependencies for cost_schedule use cases
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all cost_schedule-related use cases
type UseCases struct {
	CreateCostSchedule          *CreateCostScheduleUseCase
	ReadCostSchedule            *ReadCostScheduleUseCase
	UpdateCostSchedule          *UpdateCostScheduleUseCase
	DeleteCostSchedule          *DeleteCostScheduleUseCase
	ListCostSchedules           *ListCostSchedulesUseCase
	GetCostScheduleListPageData *GetCostScheduleListPageDataUseCase
	GetCostScheduleItemPageData *GetCostScheduleItemPageDataUseCase
	FindApplicableCostSchedule  *FindApplicableCostScheduleUseCase
}

// NewUseCases creates a new collection of cost_schedule use cases
func NewUseCases(repos Repositories, svcs Services) *UseCases {
	return &UseCases{
		CreateCostSchedule: NewCreateCostScheduleUseCase(
			CreateCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			CreateCostScheduleServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator, IDGenerator: svcs.IDGenerator},
		),
		ReadCostSchedule: NewReadCostScheduleUseCase(
			ReadCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			ReadCostScheduleServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		UpdateCostSchedule: NewUpdateCostScheduleUseCase(
			UpdateCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			UpdateCostScheduleServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		DeleteCostSchedule: NewDeleteCostScheduleUseCase(
			DeleteCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			DeleteCostScheduleServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListCostSchedules: NewListCostSchedulesUseCase(
			ListCostSchedulesRepositories{CostSchedule: repos.CostSchedule},
			ListCostSchedulesServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetCostScheduleListPageData: NewGetCostScheduleListPageDataUseCase(
			GetCostScheduleListPageDataRepositories{CostSchedule: repos.CostSchedule},
			GetCostScheduleListPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetCostScheduleItemPageData: NewGetCostScheduleItemPageDataUseCase(
			GetCostScheduleItemPageDataRepositories{CostSchedule: repos.CostSchedule},
			GetCostScheduleItemPageDataServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		FindApplicableCostSchedule: NewFindApplicableCostScheduleUseCase(
			FindApplicableCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			FindApplicableCostScheduleServices{Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
	}
}
