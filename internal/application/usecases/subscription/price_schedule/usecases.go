package price_schedule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// PriceScheduleRepositories groups all repository dependencies for price schedule use cases
type PriceScheduleRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// PriceScheduleServices groups all business service dependencies for price schedule use cases
type PriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Only for CreatePriceSchedule
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	readServices := ReadPriceScheduleServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	updateServices := UpdatePriceScheduleServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePriceScheduleRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	deleteServices := DeletePriceScheduleServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPriceSchedulesRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	listServices := ListPriceSchedulesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetPriceScheduleListPageDataRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	listPageDataServices := GetPriceScheduleListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetPriceScheduleItemPageDataRepositories{
		PriceSchedule: repositories.PriceSchedule,
	}
	itemPageDataServices := GetPriceScheduleItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePriceSchedule:          NewCreatePriceScheduleUseCase(createRepos, createServices),
		ReadPriceSchedule:            NewReadPriceScheduleUseCase(readRepos, readServices),
		UpdatePriceSchedule:          NewUpdatePriceScheduleUseCase(updateRepos, updateServices),
		DeletePriceSchedule:          NewDeletePriceScheduleUseCase(deleteRepos, deleteServices),
		ListPriceSchedules:           NewListPriceSchedulesUseCase(listRepos, listServices),
		GetPriceScheduleListPageData: NewGetPriceScheduleListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPriceScheduleItemPageData: NewGetPriceScheduleItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
