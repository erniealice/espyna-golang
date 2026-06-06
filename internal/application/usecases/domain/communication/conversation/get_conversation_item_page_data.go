package conversation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
)

// GetConversationItemPageDataRepositories groups all repository dependencies.
type GetConversationItemPageDataRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
}

// GetConversationItemPageDataServices groups all business service dependencies.
type GetConversationItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetConversationItemPageDataUseCase handles conversation item page data.
type GetConversationItemPageDataUseCase struct {
	repositories GetConversationItemPageDataRepositories
	services     GetConversationItemPageDataServices
}

// NewGetConversationItemPageDataUseCase creates a new use case.
func NewGetConversationItemPageDataUseCase(repos GetConversationItemPageDataRepositories, svcs GetConversationItemPageDataServices) *GetConversationItemPageDataUseCase {
	return &GetConversationItemPageDataUseCase{repositories: repos, services: svcs}
}

// Execute performs the get conversation item page data operation.
func (uc *GetConversationItemPageDataUseCase) Execute(ctx context.Context, req *conversationpb.GetConversationItemPageDataRequest) (*conversationpb.GetConversationItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityConversation, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.ConversationId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.id_required", "Conversation ID is required [DEFAULT]"))
	}

	return uc.repositories.Conversation.GetConversationItemPageData(ctx, req)
}
