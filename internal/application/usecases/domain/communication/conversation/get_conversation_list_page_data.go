package conversation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Conversation,
		Action: entityid.ActionList,
	}); err != nil {
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
