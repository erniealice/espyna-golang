package forex_rate

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

const entityForexRate = "forex_rate"

// ForexRateRepositories groups all repository dependencies for forex_rate use cases.
type ForexRateRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
}

// ForexRateServices groups all business service dependencies.
type ForexRateServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListForexRates: NewListForexRatesUseCase(
			ListForexRatesRepositories{ForexRate: repositories.ForexRate},
			ListForexRatesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		RecordOperatorRate: NewRecordOperatorRateUseCase(
			RecordOperatorRateRepositories{ForexRate: repositories.ForexRate},
			RecordOperatorRateServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
	}
}
