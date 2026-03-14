package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/operation"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeOperation creates all operation use cases from provider repositories.
func InitializeOperation(
	repos *domain.OperationRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*operation.OperationUseCases, error) {
	return operation.NewUseCases(
		operation.OperationRepositories{
			Job:              repos.Job,
			JobPhase:         repos.JobPhase,
			JobTask:          repos.JobTask,
			JobTemplate:      repos.JobTemplate,
			JobTemplatePhase: repos.JobTemplatePhase,
			JobTemplateTask:  repos.JobTemplateTask,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
