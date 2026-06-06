package conversation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
)

// GetConversationListPageDataRepositories groups all repository dependencies.
type GetConversationListPageDataRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
}

// GetConversationListPageDataServices groups all business service dependencies.
type GetConversationListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetConversationListPageDataUseCase handles conversation list page data.
type GetConversationListPageDataUseCase struct {
	repositories GetConversationListPageDataRepositories
	services     GetConversationListPageDataServices
}

// NewGetConversationListPageDataUseCase creates a new use case.
func NewGetConversationListPageDataUseCase(repos GetConversationListPageDataRepositories, svcs GetConversationListPageDataServices) *GetConversationListPageDataUseCase {
	return &GetConversationListPageDataUseCase{repositories: repos, services: svcs}
}

// Execute performs the get conversation list page data operation.
func (uc *GetConversationListPageDataUseCase) Execute(ctx context.Context, req *conversationpb.GetConversationListPageDataRequest) (*conversationpb.GetConversationListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityConversation, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.request_required", "Request is required [DEFAULT]"))
	}

	// Client IDOR filter: pin client_id to the session acting-as scope (I1).
	if actingClientID := contextutil.GetActingAsClientIDFromContext(ctx); actingClientID != "" {
		req.Filters = appendClientIDFilter(req.Filters, actingClientID)
	}

	return uc.repositories.Conversation.GetConversationListPageData(ctx, req)
}
