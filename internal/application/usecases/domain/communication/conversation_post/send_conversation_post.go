package conversation_post

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// SendConversationPostRepositories groups all repository dependencies.
type SendConversationPostRepositories struct {
	ConversationPost conversationPostpb.ConversationPostDomainServiceServer
	Conversation     conversationpb.ConversationDomainServiceServer
	Client           clientpb.ClientDomainServiceServer
}

// SendConversationPostServices groups all business service dependencies.
type SendConversationPostServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// SendConversationPostUseCase appends a post to a thread. It is idempotent
// (requires a non-empty client_token), copies the IDOR anchors from the parent,
// bumps conversation.last_post_at in the same transaction (via the adapter), and
// triggers a best-effort notification.
type SendConversationPostUseCase struct {
	repositories SendConversationPostRepositories
	services     SendConversationPostServices
	notify       *NotifyConversationPostUseCase
}

// NewSendConversationPostUseCase creates a new SendConversationPostUseCase.
func NewSendConversationPostUseCase(repos SendConversationPostRepositories, svcs SendConversationPostServices, notify *NotifyConversationPostUseCase) *SendConversationPostUseCase {
	return &SendConversationPostUseCase{repositories: repos, services: svcs, notify: notify}
}

// Execute performs the send conversation post operation.
func (uc *SendConversationPostUseCase) Execute(ctx context.Context, req *conversationPostpb.CreateConversationPostRequest) (*conversationPostpb.CreateConversationPostResponse, error) {
	// Verb: conversation_post:create.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ConversationPost,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_post.validation.data_required", "Conversation post data is required [DEFAULT]"))
	}

	// I2 / codex H3: REQUIRE a non-empty client_token for portal/composer posts.
	// Reject (422-class validation error) BEFORE any DB write — a raw POST
	// without a token is never inserted.
	if req.Data.GetClientToken() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_post.validation.client_token_required", "A client token is required to send a message [DEFAULT]"))
	}

	if req.Data.GetConversationId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_post.validation.conversation_required", "A conversation is required [DEFAULT]"))
	}

	if req.Data.GetBody() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_post.validation.body_required", "Message body is required [DEFAULT]"))
	}

	var result *conversationPostpb.CreateConversationPostResponse
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("%s: %w",
					contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator,
						"conversation_post.errors.send_failed", "Sending the message failed [DEFAULT]"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		res, err := uc.executeCore(ctx, req)
		if err != nil {
			return nil, err
		}
		result = res
	}

	// Best-effort notification AFTER the post is committed.
	if uc.notify != nil && result != nil && len(result.Data) > 0 {
		_ = uc.notify.Execute(ctx, result.Data[0])
	}

	return result, nil
}

func (uc *SendConversationPostUseCase) executeCore(ctx context.Context, req *conversationPostpb.CreateConversationPostRequest) (*conversationPostpb.CreateConversationPostResponse, error) {
	// Load the parent for the IDOR anchors (denorm copy — directive H) + IDOR check.
	parentResp, err := uc.repositories.Conversation.ReadConversation(ctx, &conversationpb.ReadConversationRequest{Data: &conversationpb.Conversation{Id: req.Data.GetConversationId()}})
	if err != nil {
		return nil, err
	}
	if len(parentResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
			"conversation.errors.not_found", map[string]interface{}{"conversationId": req.Data.GetConversationId()}, "Conversation not found [DEFAULT]"))
	}
	parent := parentResp.Data[0]

	// Client IDOR: a portal caller may only post to their own thread.
	if actingClientID := contextutil.GetActingAsClientIDFromContext(ctx); actingClientID != "" && parent.GetClientId() != actingClientID {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_post.errors.forbidden", "You are not allowed to post to this conversation [DEFAULT]"))
	}

	post := uc.applyBusinessLogic(ctx, req.Data, parent)

	// Create the (draft) post FIRST; the adapter bumps the parent
	// last_post_at = GREATEST(...) in the SAME transaction (I3 / M3).
	return uc.repositories.ConversationPost.CreateConversationPost(ctx, &conversationPostpb.CreateConversationPostRequest{Data: post})
}

// applyBusinessLogic enriches the post: ID, denormalized IDOR anchors copied
// from the parent, session-stamped sender, default source = PORTAL, sent_at.
func (uc *SendConversationPostUseCase) applyBusinessLogic(ctx context.Context, post *conversationPostpb.ConversationPost, parent *conversationpb.Conversation) *conversationPostpb.ConversationPost {
	now := time.Now()

	if post.Id == "" && uc.services.IDGenerator != nil {
		post.Id = uc.services.IDGenerator.GenerateID()
	}

	// Directive H: copy workspace_id + client_id from the parent at create.
	post.WorkspaceId = parent.GetWorkspaceId()
	post.ClientId = parent.GetClientId()

	// Stamp the sender from the session.
	if post.SenderUserId == "" {
		post.SenderUserId = contextutil.ExtractUserIDFromContext(ctx)
	}
	// Default principal type: a portal acting-as-client caller is CLIENT, else STAFF.
	if post.SenderPrincipalType == conversationPostpb.SenderPrincipalType_SENDER_PRINCIPAL_TYPE_UNSPECIFIED {
		if contextutil.GetActingAsClientIDFromContext(ctx) != "" {
			post.SenderPrincipalType = conversationPostpb.SenderPrincipalType_SENDER_PRINCIPAL_TYPE_CLIENT
		} else {
			post.SenderPrincipalType = conversationPostpb.SenderPrincipalType_SENDER_PRINCIPAL_TYPE_STAFF
		}
	}

	// Default source = PORTAL (v1). EMAIL is a v2 seam.
	if post.SourceType == conversationPostpb.ConversationSourceType_CONVERSATION_SOURCE_TYPE_UNSPECIFIED {
		post.SourceType = conversationPostpb.ConversationSourceType_CONVERSATION_SOURCE_TYPE_PORTAL
	}

	post.Active = true

	ts := now.UnixMilli()
	if post.SentAt == nil {
		post.SentAt = &ts
	}
	post.DateCreated = &ts
	post.DateModified = &ts

	return post
}
