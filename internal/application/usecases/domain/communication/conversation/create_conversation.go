package conversation

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
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// CreateConversationRepositories groups all repository dependencies.
type CreateConversationRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	User         userpb.UserDomainServiceServer
}

// CreateConversationServices groups all business service dependencies.
type CreateConversationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateConversationUseCase handles starting a new conversation thread/ticket.
type CreateConversationUseCase struct {
	repositories CreateConversationRepositories
	services     CreateConversationServices
}

// NewCreateConversationUseCase creates a new CreateConversationUseCase.
func NewCreateConversationUseCase(repos CreateConversationRepositories, svcs CreateConversationServices) *CreateConversationUseCase {
	return &CreateConversationUseCase{repositories: repos, services: svcs}
}

// Execute performs the create conversation operation.
func (uc *CreateConversationUseCase) Execute(ctx context.Context, req *conversationpb.CreateConversationRequest) (*conversationpb.CreateConversationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Conversation, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *conversationpb.CreateConversationResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("%s: %w",
					contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator,
						"conversation.errors.creation_failed", "Conversation creation failed [DEFAULT]"), err)
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

func (uc *CreateConversationUseCase) executeCore(ctx context.Context, req *conversationpb.CreateConversationRequest) (*conversationpb.CreateConversationResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.data_required", "Conversation data is required [DEFAULT]"))
	}

	conv := uc.applyBusinessLogic(ctx, req.Data)

	if err := uc.validateBusinessRules(ctx, conv); err != nil {
		return nil, err
	}

	return uc.repositories.Conversation.CreateConversation(ctx, &conversationpb.CreateConversationRequest{Data: conv})
}

// applyBusinessLogic enriches the conversation with ID, audit fields, session-
// stamped IDOR anchors, and lifecycle defaults.
//
// Invariant I1 / directive Q-MSG-5: client_id is stamped from the session's
// acting-as scope, NEVER trusted from the request body. A staff caller (empty
// acting-as scope) keeps the body-supplied client_id (the thread is opened on
// behalf of a chosen client), but a portal caller's scope always wins.
func (uc *CreateConversationUseCase) applyBusinessLogic(ctx context.Context, conv *conversationpb.Conversation) *conversationpb.Conversation {
	now := time.Now()

	if conv.Id == "" && uc.services.IDGenerator != nil {
		conv.Id = uc.services.IDGenerator.GenerateID()
	}

	// I1: stamp client_id from session acting-as scope when present.
	if actingClientID := contextutil.GetActingAsClientIDFromContext(ctx); actingClientID != "" {
		conv.ClientId = actingClientID
	}

	// Stamp workspace_id + created_by from session when not already set.
	if conv.WorkspaceId == "" {
		conv.WorkspaceId = contextutil.ExtractWorkspaceIDFromContext(ctx)
	}
	if conv.CreatedByUserId == "" {
		conv.CreatedByUserId = contextutil.ExtractUserIDFromContext(ctx)
	}

	// Default lifecycle: status = OPEN, active = true.
	if conv.Status == conversationpb.ConversationStatus_CONVERSATION_STATUS_UNSPECIFIED {
		conv.Status = conversationpb.ConversationStatus_CONVERSATION_STATUS_OPEN
	}
	conv.Active = true

	ts := now.UnixMilli()
	conv.DateCreated = &ts
	conv.DateModified = &ts

	return conv
}

// validateBusinessRules enforces required-field + IDOR-anchor constraints.
func (uc *CreateConversationUseCase) validateBusinessRules(ctx context.Context, conv *conversationpb.Conversation) error {
	if conv.Subject == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.subject_required", "Conversation subject is required [DEFAULT]"))
	}
	// IDOR anchors must be non-empty (mirrors the DB CHECK; directive F).
	if conv.WorkspaceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.workspace_required", "Conversation workspace is required [DEFAULT]"))
	}
	if conv.ClientId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.client_required", "Conversation client is required [DEFAULT]"))
	}
	return nil
}
