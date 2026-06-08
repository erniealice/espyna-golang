package conversation_post

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
)

// ListConversationPostsRepositories groups all repository dependencies.
type ListConversationPostsRepositories struct {
	ConversationPost conversationPostpb.ConversationPostDomainServiceServer
	Conversation     conversationpb.ConversationDomainServiceServer
}

// ListConversationPostsServices groups all business service dependencies.
type ListConversationPostsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListConversationPostsUseCase lists posts in a thread, ordered, with IDOR.
type ListConversationPostsUseCase struct {
	repositories ListConversationPostsRepositories
	services     ListConversationPostsServices
}

// NewListConversationPostsUseCase creates a new ListConversationPostsUseCase.
func NewListConversationPostsUseCase(repos ListConversationPostsRepositories, svcs ListConversationPostsServices) *ListConversationPostsUseCase {
	return &ListConversationPostsUseCase{repositories: repos, services: svcs}
}

// Execute performs the list conversation posts operation.
//
// IDOR (D.6): the request MUST carry a conversation_id filter. For a client
// caller the parent's client_id must equal the acting-as scope; a staff caller
// is workspace-scoped (the WorkspaceAwareOperations layer enforces the workspace
// predicate). Verb: conversation_post:read.
func (uc *ListConversationPostsUseCase) Execute(ctx context.Context, req *conversationPostpb.ListConversationPostsRequest) (*conversationPostpb.ListConversationPostsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.ConversationPost, entityid.ActionRead); err != nil {
		return nil, err
	}

	if req == nil {
		req = &conversationPostpb.ListConversationPostsRequest{}
	}

	conversationID := extractConversationIDFilter(req.Filters)
	if conversationID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_post.validation.conversation_required", "A conversation is required [DEFAULT]"))
	}

	// Resolve the parent for the IDOR check.
	parentResp, err := uc.repositories.Conversation.ReadConversation(ctx, &conversationpb.ReadConversationRequest{Data: &conversationpb.Conversation{Id: conversationID}})
	if err != nil {
		return nil, err
	}
	if len(parentResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
			"conversation.errors.not_found", map[string]interface{}{"conversationId": conversationID}, "Conversation not found [DEFAULT]"))
	}
	parent := parentResp.Data[0]

	// Client IDOR: a portal caller may only read their own thread's posts.
	if actingClientID := contextutil.GetActingAsClientIDFromContext(ctx); actingClientID != "" && parent.GetClientId() != actingClientID {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_post.errors.forbidden", "You are not allowed to read this conversation [DEFAULT]"))
	}

	return uc.repositories.ConversationPost.ListConversationPosts(ctx, req)
}

// extractConversationIDFilter pulls the conversation_id equality value from the
// filter set, if present.
func extractConversationIDFilter(filters *commonpb.FilterRequest) string {
	if filters == nil {
		return ""
	}
	for _, f := range filters.Filters {
		if f == nil || f.Field != "conversation_id" {
			continue
		}
		if sf, ok := f.FilterType.(*commonpb.TypedFilter_StringFilter); ok && sf.StringFilter != nil {
			return sf.StringFilter.Value
		}
	}
	return ""
}
