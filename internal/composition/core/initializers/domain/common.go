package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/common"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeCommon creates all common use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeCommon(
	repos *domain.CommonRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
) (*common.CommonUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return common.NewCommonUseCases(
		repos.Attribute,
		repos.Category,
		i18nSvc,
		idSvc,
	), nil
}
