package conversation

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ListConversationsRepositories groups all repository dependencies.
type ListConversationsRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	User         userpb.UserDomainServiceServer
}

// ListConversationsServices groups all business service dependencies.
type ListConversationsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListConversationsUseCase handles listing conversations with role-scoped IDOR filtering.
type ListConversationsUseCase struct {
	repositories ListConversationsRepositories
	services     ListConversationsServices
}

// NewListConversationsUseCase creates a new ListConversationsUseCase.
func NewListConversationsUseCase(repos ListConversationsRepositories, svcs ListConversationsServices) *ListConversationsUseCase {
	return &ListConversationsUseCase{repositories: repos, services: svcs}
}

// Execute performs the list conversations operation.
//
// D.6 scoping: a client caller (non-empty acting-as scope) is hard-filtered to
// client_id = acting_as_client_id; a staff caller is workspace-wide (the
// WorkspaceAwareOperations layer applies the workspace predicate from context).
func (uc *ListConversationsUseCase) Execute(ctx context.Context, req *conversationpb.ListConversationsRequest) (*conversationpb.ListConversationsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Conversation, entityid.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		req = &conversationpb.ListConversationsRequest{}
	}

	// Client IDOR filter: pin client_id to the session's acting-as scope so a
	// portal caller can never list another client's threads (Q-MSG-5 / I1).
	if actingClientID := contextutil.GetActingAsClientIDFromContext(ctx); actingClientID != "" {
		req.Filters = appendClientIDFilter(req.Filters, actingClientID)
	}

	resp, err := uc.repositories.Conversation.ListConversations(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.errors.list_failed", "Failed to retrieve conversations [DEFAULT]"))
	}

	return resp, nil
}

// appendClientIDFilter adds a client_id equality predicate to the filter set.
func appendClientIDFilter(filters *commonpb.FilterRequest, clientID string) *commonpb.FilterRequest {
	clientFilter := &commonpb.TypedFilter{
		Field: "client_id",
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{
				Value:    clientID,
				Operator: commonpb.StringOperator_STRING_EQUALS,
			},
		},
	}
	if filters == nil {
		return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{clientFilter}}
	}
	filters.Filters = append(filters.Filters, clientFilter)
	return filters
}
