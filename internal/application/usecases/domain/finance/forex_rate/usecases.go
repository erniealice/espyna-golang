package forex_rate

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

const entityForexRate = "forex_rate"

// ForexRateRepositories groups all repository dependencies for forex_rate use cases.
type ForexRateRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
}

// ForexRateServices groups all business service dependencies.
type ForexRateServices struct {
	Authorizer  ports.Authorizer
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all forex_rate use cases.
type UseCases struct {
	ReadForexRate      *ReadForexRateUseCase
	ListForexRates     *ListForexRatesUseCase
	RecordOperatorRate *RecordOperatorRateUseCase
	FindMostRecent     *FindMostRecentForexRateUseCase
}

// NewUseCases creates a new collection of forex_rate use cases.
func NewUseCases(repositories ForexRateRepositories, services ForexRateServices) *UseCases {
	return &UseCases{
		ReadForexRate: NewReadForexRateUseCase(
			ReadForexRateRepositories{ForexRate: repositories.ForexRate},
			ReadForexRateServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListForexRates: NewListForexRatesUseCase(
			ListForexRatesRepositories{ForexRate: repositories.ForexRate},
			ListForexRatesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		RecordOperatorRate: NewRecordOperatorRateUseCase(
			RecordOperatorRateRepositories{ForexRate: repositories.ForexRate},
			RecordOperatorRateServices{
				Authorizer:  services.Authorizer,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
	}
}
