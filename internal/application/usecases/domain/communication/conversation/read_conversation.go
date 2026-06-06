package conversation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ReadConversationRepositories groups all repository dependencies.
type ReadConversationRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	User         userpb.UserDomainServiceServer
}

// ReadConversationServices groups all business service dependencies.
type ReadConversationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadConversationUseCase handles reading a single conversation header (with IDOR check).
type ReadConversationUseCase struct {
	repositories ReadConversationRepositories
	services     ReadConversationServices
}

// NewReadConversationUseCase creates a new ReadConversationUseCase.
func NewReadConversationUseCase(repos ReadConversationRepositories, svcs ReadConversationServices) *ReadConversationUseCase {
	return &ReadConversationUseCase{repositories: repos, services: svcs}
}

// Execute performs the read conversation operation.
func (uc *ReadConversationUseCase) Execute(ctx context.Context, req *conversationpb.ReadConversationRequest) (*conversationpb.ReadConversationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityConversation, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.id_required", "Conversation ID is required [DEFAULT]"))
	}

	resp, err := uc.repositories.Conversation.ReadConversation(ctx, req)
	if err != nil {
		if contextutil.Contains(err.Error(), "not found") {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
				"conversation.errors.not_found", map[string]interface{}{"conversationId": req.Data.Id}, "Conversation not found [DEFAULT]"))
		}
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
			"conversation.errors.not_found", map[string]interface{}{"conversationId": req.Data.Id}, "Conversation not found [DEFAULT]"))
	}

	return resp, nil
}
