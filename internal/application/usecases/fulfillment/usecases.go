package fulfillment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	fulfillmentdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/fulfillment/dashboard"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// Repositories groups all repository dependencies for fulfillment use cases.
type Repositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}

// Services groups all business service dependencies for fulfillment use cases.
type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all fulfillment-related use cases.
type UseCases struct {
	CreateFulfillment          *CreateFulfillmentUseCase
	GetFulfillment             *GetFulfillmentUseCase
	UpdateFulfillment          *UpdateFulfillmentUseCase
	DeleteFulfillment          *DeleteFulfillmentUseCase
	ListFulfillments           *ListFulfillmentsUseCase
	GetFulfillmentListPageData *GetFulfillmentListPageDataUseCase
	GetFulfillmentItemPageData *GetFulfillmentItemPageDataUseCase
	TransitionStatus           *TransitionStatusUseCase
	ListStatusEvents           *ListStatusEventsUseCase

	// Dashboard use case (nil when postgres build tag is inactive).
	Dashboard *fulfillmentdashboard.GetFulfillmentDashboardPageDataUseCase
}

// NewUseCases creates a new collection of fulfillment use cases.
func NewUseCases(
	repositories Repositories,
	services Services,
) *UseCases {
	// Wire fulfillment dashboard via type assertion on the fulfillment repo.
	var fulfillDash *fulfillmentdashboard.GetFulfillmentDashboardPageDataUseCase
	if repositories.Fulfillment != nil {
		if fq, ok := repositories.Fulfillment.(fulfillmentdashboard.FulfillmentDashboardQueries); ok {
			fulfillDash = fulfillmentdashboard.NewGetFulfillmentDashboardPageDataUseCase(fq)
		}
	}

	return &UseCases{
		CreateFulfillment: &CreateFulfillmentUseCase{
			repositories: CreateFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: CreateFulfillmentServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		},
		GetFulfillment: &GetFulfillmentUseCase{
			repositories: GetFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: GetFulfillmentServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		UpdateFulfillment: &UpdateFulfillmentUseCase{
			repositories: UpdateFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: UpdateFulfillmentServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		DeleteFulfillment: &DeleteFulfillmentUseCase{
			repositories: DeleteFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: DeleteFulfillmentServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		ListFulfillments: &ListFulfillmentsUseCase{
			repositories: ListFulfillmentsRepositories{Fulfillment: repositories.Fulfillment},
			services: ListFulfillmentsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetFulfillmentListPageData: &GetFulfillmentListPageDataUseCase{
			repositories: GetFulfillmentListPageDataRepositories{Fulfillment: repositories.Fulfillment},
			services: GetFulfillmentListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetFulfillmentItemPageData: &GetFulfillmentItemPageDataUseCase{
			repositories: GetFulfillmentItemPageDataRepositories{Fulfillment: repositories.Fulfillment},
			services: GetFulfillmentItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		TransitionStatus: &TransitionStatusUseCase{
			repositories: TransitionStatusRepositories{Fulfillment: repositories.Fulfillment},
			services: TransitionStatusServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
		ListStatusEvents: &ListStatusEventsUseCase{
			repositories: ListStatusEventsRepositories{Fulfillment: repositories.Fulfillment},
			services: ListStatusEventsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		Dashboard: fulfillDash,
	}
}
