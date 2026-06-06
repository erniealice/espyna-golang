package conversation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// UpdateConversationRepositories groups all repository dependencies.
type UpdateConversationRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	User         userpb.UserDomainServiceServer
}

// UpdateConversationServices groups all business service dependencies.
type UpdateConversationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateConversationUseCase handles updating a conversation header.
type UpdateConversationUseCase struct {
	repositories UpdateConversationRepositories
	services     UpdateConversationServices
}

// NewUpdateConversationUseCase creates a new UpdateConversationUseCase.
func NewUpdateConversationUseCase(repos UpdateConversationRepositories, svcs UpdateConversationServices) *UpdateConversationUseCase {
	return &UpdateConversationUseCase{repositories: repos, services: svcs}
}

// Execute performs the update conversation operation.
func (uc *UpdateConversationUseCase) Execute(ctx context.Context, req *conversationpb.UpdateConversationRequest) (*conversationpb.UpdateConversationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityConversation, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *conversationpb.UpdateConversationResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("%s: %w",
					contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator,
						"conversation.errors.update_failed", "Conversation update failed [DEFAULT]"), err)
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

func (uc *UpdateConversationUseCase) executeCore(ctx context.Context, req *conversationpb.UpdateConversationRequest) (*conversationpb.UpdateConversationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.id_required", "Conversation ID is required [DEFAULT]"))
	}

	now := time.Now().UnixMilli()
	req.Data.DateModified = &now

	return uc.repositories.Conversation.UpdateConversation(ctx, req)
}
