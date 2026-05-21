package fulfillment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// Repositories groups all repository dependencies for fulfillment use cases.
type Repositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}

// Services groups all business service dependencies for fulfillment use cases.
type Services struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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

	// Dashboard field retired 2026-05-21 (Wave C P1.C.12 Fulfillment) — the
	// dashboard now lives under `service.Dashboard.Fulfillment` per Q-SDM-
	// DASHBOARD-DOWNSTREAM. The `usecases/fulfillment/dashboard/` package is
	// retired in the same commit; the repository composition relocated to
	// `usecases/service/dashboard/fulfillment/`.
}

// NewUseCases creates a new collection of fulfillment use cases.
func NewUseCases(
	repositories Repositories,
	services Services,
) *UseCases {
	// Fulfillment dashboard wiring retired 2026-05-21 (Wave C P1.C.12) —
	// type-assertion + factory wiring now lives in the service-layer
	// initializer at `internal/composition/core/initializers/service.go`
	// (search "Wave C P1.C.12 Fulfillment").

	return &UseCases{
		CreateFulfillment: &CreateFulfillmentUseCase{
			repositories: CreateFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: CreateFulfillmentServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		},
		GetFulfillment: &GetFulfillmentUseCase{
			repositories: GetFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: GetFulfillmentServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		UpdateFulfillment: &UpdateFulfillmentUseCase{
			repositories: UpdateFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: UpdateFulfillmentServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		DeleteFulfillment: &DeleteFulfillmentUseCase{
			repositories: DeleteFulfillmentRepositories{Fulfillment: repositories.Fulfillment},
			services: DeleteFulfillmentServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		ListFulfillments: &ListFulfillmentsUseCase{
			repositories: ListFulfillmentsRepositories{Fulfillment: repositories.Fulfillment},
			services: ListFulfillmentsServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		GetFulfillmentListPageData: &GetFulfillmentListPageDataUseCase{
			repositories: GetFulfillmentListPageDataRepositories{Fulfillment: repositories.Fulfillment},
			services: GetFulfillmentListPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		GetFulfillmentItemPageData: &GetFulfillmentItemPageDataUseCase{
			repositories: GetFulfillmentItemPageDataRepositories{Fulfillment: repositories.Fulfillment},
			services: GetFulfillmentItemPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		TransitionStatus: &TransitionStatusUseCase{
			repositories: TransitionStatusRepositories{Fulfillment: repositories.Fulfillment},
			services: TransitionStatusServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		},
		ListStatusEvents: &ListStatusEventsUseCase{
			repositories: ListStatusEventsRepositories{Fulfillment: repositories.Fulfillment},
			services: ListStatusEventsServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
	}
}
