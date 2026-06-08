package conversation_read_receipt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationReadReceiptpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_read_receipt"
)

// MarkConversationReadRepositories groups all repository dependencies.
type MarkConversationReadRepositories struct {
	ConversationReadReceipt conversationReadReceiptpb.ConversationReadReceiptDomainServiceServer
	Conversation            conversationpb.ConversationDomainServiceServer
}

// MarkConversationReadServices groups all business service dependencies.
type MarkConversationReadServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// MarkConversationReadUseCase upserts the per-reader high-water mark cursor,
// principal-scoped (Q-MSG-12 / I4). Verb-folds to conversation_post:read.
type MarkConversationReadUseCase struct {
	repositories MarkConversationReadRepositories
	services     MarkConversationReadServices
}

// NewMarkConversationReadUseCase creates a new MarkConversationReadUseCase.
func NewMarkConversationReadUseCase(repos MarkConversationReadRepositories, svcs MarkConversationReadServices) *MarkConversationReadUseCase {
	return &MarkConversationReadUseCase{repositories: repos, services: svcs}
}

// Execute performs the mark-conversation-read upsert.
func (uc *MarkConversationReadUseCase) Execute(ctx context.Context, req *conversationReadReceiptpb.CreateConversationReadReceiptRequest) (*conversationReadReceiptpb.CreateConversationReadReceiptResponse, error) {
	// Verb-fold: conversation_post:mark_read -> conversation_post:read.
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.ConversationPost, entityid.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_read_receipt.validation.data_required", "Read receipt data is required [DEFAULT]"))
	}
	if req.Data.GetConversationId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_read_receipt.validation.conversation_required", "A conversation is required [DEFAULT]"))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *conversationReadReceiptpb.CreateConversationReadReceiptResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("%s: %w",
					contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator,
						"conversation_read_receipt.errors.mark_failed", "Marking the conversation read failed [DEFAULT]"), err)
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

func (uc *MarkConversationReadUseCase) executeCore(ctx context.Context, req *conversationReadReceiptpb.CreateConversationReadReceiptRequest) (*conversationReadReceiptpb.CreateConversationReadReceiptResponse, error) {
	// IDOR: resolve the parent + (for portal callers) assert ownership.
	parentResp, err := uc.repositories.Conversation.ReadConversation(ctx, &conversationpb.ReadConversationRequest{Data: &conversationpb.Conversation{Id: req.Data.GetConversationId()}})
	if err != nil {
		return nil, err
	}
	if len(parentResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
			"conversation.errors.not_found", map[string]interface{}{"conversationId": req.Data.GetConversationId()}, "Conversation not found [DEFAULT]"))
	}
	parent := parentResp.Data[0]

	if actingClientID := contextutil.GetActingAsClientIDFromContext(ctx); actingClientID != "" && parent.GetClientId() != actingClientID {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation_read_receipt.errors.forbidden", "You are not allowed to mark this conversation read [DEFAULT]"))
	}

	receipt := req.Data
	if receipt.Id == "" && uc.services.IDGenerator != nil {
		receipt.Id = uc.services.IDGenerator.GenerateID()
	}

	// Stamp principal-scoped reader + session identity. The reader principal is
	// the unique key (with conversation_id); never trust the body for it.
	if receipt.GetUserId() == "" {
		receipt.UserId = contextutil.ExtractUserIDFromContext(ctx)
	}
	// Directive H: copy workspace_id from the parent.
	receipt.WorkspaceId = parent.GetWorkspaceId()
	receipt.Active = true

	now := time.Now().UnixMilli()
	if receipt.DateCreated == nil {
		receipt.DateCreated = &now
	}
	receipt.DateModified = &now
	receipt.LastReadAt = &now

	// The adapter performs the ON CONFLICT upsert on the principal-scoped key.
	return uc.repositories.ConversationReadReceipt.CreateConversationReadReceipt(ctx, &conversationReadReceiptpb.CreateConversationReadReceiptRequest{Data: receipt})
}
