// Package billing_event contains Layer-7 use case wrappers for the
// BillingEvent proto domain service. 20260518-hexagonal-strict-adherence
// Phase 3 F7 closure — replaces the raw
// billingeventpb.BillingEventDomainServiceServer leak that was previously
// exposed as a flat field on SubscriptionUseCases.
package billing_event

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
)

// BillingEventRepositories groups all repository dependencies for billing
// event use cases.
type BillingEventRepositories struct {
	BillingEvent billingeventpb.BillingEventDomainServiceServer
}

// BillingEventServices groups all business service dependencies for billing
// event use cases.
type BillingEventServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UseCases contains all billing-event use cases.
type UseCases struct {
	ListBySubscription *ListBillingEventsBySubscriptionUseCase
	SetStatus          *SetBillingEventStatusUseCase
}

// NewUseCases creates the billing-event use case sub-aggregate.
func NewUseCases(
	repositories BillingEventRepositories,
	services BillingEventServices,
) *UseCases {
	if repositories.BillingEvent == nil {
		return &UseCases{}
	}
	return &UseCases{
		ListBySubscription: NewListBillingEventsBySubscriptionUseCase(
			ListBillingEventsBySubscriptionRepositories{BillingEvent: repositories.BillingEvent},
			ListBillingEventsBySubscriptionServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		SetStatus: NewSetBillingEventStatusUseCase(
			SetBillingEventStatusRepositories{BillingEvent: repositories.BillingEvent},
			SetBillingEventStatusServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
	}
}
