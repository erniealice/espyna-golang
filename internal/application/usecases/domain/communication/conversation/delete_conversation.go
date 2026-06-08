package conversation

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// DeleteConversationRepositories groups all repository dependencies.
type DeleteConversationRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	User         userpb.UserDomainServiceServer
}

// DeleteConversationServices groups all business service dependencies.
type DeleteConversationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteConversationUseCase handles soft-deleting a conversation.
type DeleteConversationUseCase struct {
	repositories DeleteConversationRepositories
	services     DeleteConversationServices
}

// NewDeleteConversationUseCase creates a new DeleteConversationUseCase.
func NewDeleteConversationUseCase(repos DeleteConversationRepositories, svcs DeleteConversationServices) *DeleteConversationUseCase {
	return &DeleteConversationUseCase{repositories: repos, services: svcs}
}

// Execute performs the delete conversation operation.
func (uc *DeleteConversationUseCase) Execute(ctx context.Context, req *conversationpb.DeleteConversationRequest) (*conversationpb.DeleteConversationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Conversation, entityid.ActionDelete); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *conversationpb.DeleteConversationResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("%s: %w",
					contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator,
						"conversation.errors.deletion_failed", "Conversation deletion failed [DEFAULT]"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *DeleteConversationUseCase) executeCore(ctx context.Context, req *conversationpb.DeleteConversationRequest) (*conversationpb.DeleteConversationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.id_required", "Conversation ID is required [DEFAULT]"))
	}

	return uc.repositories.Conversation.DeleteConversation(ctx, req)
}
