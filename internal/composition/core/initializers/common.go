package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/common"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeCommon creates all common use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeCommon(
	repos *domain.CommonRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*common.CommonUseCases, error) {
	// Use the domain's constructor which properly handles all use case creation
	return common.NewCommonUseCases(
		repos.Attribute,
		repos.Category,
		i18nSvc,
		idSvc,
	), nil
}
