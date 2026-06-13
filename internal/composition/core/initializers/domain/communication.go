package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	communication "github.com/erniealice/espyna-golang/internal/application/usecases/domain/communication"
	repodomain "github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeCommunication creates all communication use cases from provider
// repositories. This is composition logic — it wires infrastructure (providers)
// to application (use cases).
func InitializeCommunication(
	repos *repodomain.CommunicationRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*communication.CommunicationUseCases, error) {
	return communication.NewCommunicationUseCases(
		repos.Conversation,
		repos.ConversationPost,
		repos.ConversationReadReceipt,
		repos.Client,
		repos.User,
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		actionGate,
	), nil
}
