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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
			CreateCostScheduleServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService, IDService: svcs.IDService},
		),
		ReadCostSchedule: NewReadCostScheduleUseCase(
			ReadCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			ReadCostScheduleServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		UpdateCostSchedule: NewUpdateCostScheduleUseCase(
			UpdateCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			UpdateCostScheduleServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		DeleteCostSchedule: NewDeleteCostScheduleUseCase(
			DeleteCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			DeleteCostScheduleServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		ListCostSchedules: NewListCostSchedulesUseCase(
			ListCostSchedulesRepositories{CostSchedule: repos.CostSchedule},
			ListCostSchedulesServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetCostScheduleListPageData: NewGetCostScheduleListPageDataUseCase(
			GetCostScheduleListPageDataRepositories{CostSchedule: repos.CostSchedule},
			GetCostScheduleListPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetCostScheduleItemPageData: NewGetCostScheduleItemPageDataUseCase(
			GetCostScheduleItemPageDataRepositories{CostSchedule: repos.CostSchedule},
			GetCostScheduleItemPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		FindApplicableCostSchedule: NewFindApplicableCostScheduleUseCase(
			FindApplicableCostScheduleRepositories{CostSchedule: repos.CostSchedule},
			FindApplicableCostScheduleServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
	}
}
