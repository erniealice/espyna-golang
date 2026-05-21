package finance

import (
	// Finance use cases
	forexRateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/finance/forex_rate"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

// FinanceRepositories contains all finance domain repositories.
type FinanceRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
}

// FinanceUseCases contains all finance-related use cases.
type FinanceUseCases struct {
	ForexRate *forexRateUseCases.UseCases
}

// NewUseCases creates all finance use cases with proper constructor injection.
func NewUseCases(
	repos FinanceRepositories,
	authSvc ports.AuthorizationService,
	_ ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *FinanceUseCases {
	// Wire the ForexRateMutator via type assertion.
	// The postgres adapter implements ForexRateMutator; when nil, RecordOperatorRate
	// is still instantiated but will fail at runtime with a nil-dereference guard.
	var mutator forexRateUseCases.ForexRateMutator
	if m, ok := repos.ForexRate.(forexRateUseCases.ForexRateMutator); ok {
		mutator = m
	}

	forexRateUC := forexRateUseCases.NewUseCases(
		forexRateUseCases.ForexRateRepositories{
			ForexRate: repos.ForexRate,
		},
		forexRateUseCases.ForexRateServices{
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Override RecordOperatorRate with the mutator-aware variant.
	forexRateUC.RecordOperatorRate = forexRateUseCases.NewRecordOperatorRateUseCase(
		forexRateUseCases.RecordOperatorRateRepositories{
			ForexRate: repos.ForexRate,
			Mutator:   mutator,
		},
		forexRateUseCases.RecordOperatorRateServices{
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Wire FindMostRecent with the mutator-aware variant.
	forexRateUC.FindMostRecent = forexRateUseCases.NewFindMostRecentForexRateUseCase(
		forexRateUseCases.FindMostRecentForexRateRepositories{
			ForexRate: repos.ForexRate,
			Mutator:   mutator,
		},
		forexRateUseCases.FindMostRecentForexRateServices{
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
		},
	)

	return &FinanceUseCases{
		ForexRate: forexRateUC,
	}
}
