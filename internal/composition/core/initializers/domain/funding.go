package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/funding"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeFunding creates all funding use cases from provider repositories.
func InitializeFunding(
	repos *domain.FundingRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
) (*funding.FundingUseCases, error) {
	return funding.NewUseCases(
		funding.FundingRepositories{
			Fund:            repos.Fund,
			FundAllocation:  repos.FundAllocation,
			FundTransaction: repos.FundTransaction,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
