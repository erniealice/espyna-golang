package subscription_seat

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// UseCases contains all subscription seat-related use cases
type UseCases struct {
	CreateSubscriptionSeat          *CreateSubscriptionSeatUseCase
	ReadSubscriptionSeat            *ReadSubscriptionSeatUseCase
	UpdateSubscriptionSeat          *UpdateSubscriptionSeatUseCase
	DeleteSubscriptionSeat          *DeleteSubscriptionSeatUseCase
	ListSubscriptionSeats           *ListSubscriptionSeatsUseCase
	GetSubscriptionSeatListPageData *GetSubscriptionSeatListPageDataUseCase
	GetSubscriptionSeatItemPageData *GetSubscriptionSeatItemPageDataUseCase
	// SR-2 lifecycle operations
	ReplaceSubscriptionSeat   *ReplaceSubscriptionSeatUseCase
	SetSubscriptionSeatStatus *SetSubscriptionSeatStatusUseCase
}

// SubscriptionSeatRepositories groups all repository dependencies for subscription seat use cases
type SubscriptionSeatRepositories struct {
	SubscriptionSeat subscriptionseatpb.SubscriptionSeatDomainServiceServer // Primary entity repository
	Subscription     subscriptionpb.SubscriptionDomainServiceServer         // client_id stamping + FK validation
}

// SubscriptionSeatServices groups all business service dependencies for subscription seat use cases
type SubscriptionSeatServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of subscription seat use cases
func NewUseCases(
	repositories SubscriptionSeatRepositories,
	services SubscriptionSeatServices,
) *UseCases {
	return &UseCases{
		CreateSubscriptionSeat: NewCreateSubscriptionSeatUseCase(
			CreateSubscriptionSeatRepositories{
				SubscriptionSeat: repositories.SubscriptionSeat,
				Subscription:     repositories.Subscription,
			},
			CreateSubscriptionSeatServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadSubscriptionSeat: NewReadSubscriptionSeatUseCase(
			ReadSubscriptionSeatRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			ReadSubscriptionSeatServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		UpdateSubscriptionSeat: NewUpdateSubscriptionSeatUseCase(
			UpdateSubscriptionSeatRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			UpdateSubscriptionSeatServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		DeleteSubscriptionSeat: NewDeleteSubscriptionSeatUseCase(
			DeleteSubscriptionSeatRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			DeleteSubscriptionSeatServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		ListSubscriptionSeats: NewListSubscriptionSeatsUseCase(
			ListSubscriptionSeatsRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			ListSubscriptionSeatsServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		GetSubscriptionSeatListPageData: NewGetSubscriptionSeatListPageDataUseCase(
			GetSubscriptionSeatListPageDataRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			GetSubscriptionSeatListPageDataServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		GetSubscriptionSeatItemPageData: NewGetSubscriptionSeatItemPageDataUseCase(
			GetSubscriptionSeatItemPageDataRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			GetSubscriptionSeatItemPageDataServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		ReplaceSubscriptionSeat: NewReplaceSubscriptionSeatUseCase(
			ReplaceSubscriptionSeatRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			ReplaceSubscriptionSeatServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		SetSubscriptionSeatStatus: NewSetSubscriptionSeatStatusUseCase(
			SetSubscriptionSeatStatusRepositories{SubscriptionSeat: repositories.SubscriptionSeat},
			SetSubscriptionSeatStatusServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
	}
}
