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

// SetConversationStatusRepositories groups all repository dependencies.
type SetConversationStatusRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	User         userpb.UserDomainServiceServer
}

// SetConversationStatusServices groups all business service dependencies.
type SetConversationStatusServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// SetConversationStatusUseCase transitions a conversation between lifecycle
// states. Verb-folds to conversation:update (entities.md D.6).
type SetConversationStatusUseCase struct {
	repositories SetConversationStatusRepositories
	services     SetConversationStatusServices
}

// NewSetConversationStatusUseCase creates a new SetConversationStatusUseCase.
func NewSetConversationStatusUseCase(repos SetConversationStatusRepositories, svcs SetConversationStatusServices) *SetConversationStatusUseCase {
	return &SetConversationStatusUseCase{repositories: repos, services: svcs}
}

// allowedTransitions is the LOCKED transition map. CLOSED -> OPEN is allowed
// (re-openable, Q-MSG-6). A transition to the same state is a no-op (allowed).
var allowedTransitions = map[conversationpb.ConversationStatus]map[conversationpb.ConversationStatus]bool{
	conversationpb.ConversationStatus_CONVERSATION_STATUS_OPEN: {
		conversationpb.ConversationStatus_CONVERSATION_STATUS_IN_PROGRESS: true,
		conversationpb.ConversationStatus_CONVERSATION_STATUS_RESOLVED:    true,
		conversationpb.ConversationStatus_CONVERSATION_STATUS_CLOSED:      true,
	},
	conversationpb.ConversationStatus_CONVERSATION_STATUS_IN_PROGRESS: {
		conversationpb.ConversationStatus_CONVERSATION_STATUS_OPEN:     true,
		conversationpb.ConversationStatus_CONVERSATION_STATUS_RESOLVED: true,
		conversationpb.ConversationStatus_CONVERSATION_STATUS_CLOSED:   true,
	},
	conversationpb.ConversationStatus_CONVERSATION_STATUS_RESOLVED: {
		conversationpb.ConversationStatus_CONVERSATION_STATUS_IN_PROGRESS: true,
		conversationpb.ConversationStatus_CONVERSATION_STATUS_CLOSED:      true,
		conversationpb.ConversationStatus_CONVERSATION_STATUS_OPEN:        true,
	},
	conversationpb.ConversationStatus_CONVERSATION_STATUS_CLOSED: {
		// Re-openable.
		conversationpb.ConversationStatus_CONVERSATION_STATUS_OPEN:        true,
		conversationpb.ConversationStatus_CONVERSATION_STATUS_IN_PROGRESS: true,
	},
}

// Execute performs the set conversation status operation.
func (uc *SetConversationStatusUseCase) Execute(ctx context.Context, req *conversationpb.UpdateConversationRequest) (*conversationpb.UpdateConversationResponse, error) {
	// Verb-fold: conversation:transition -> conversation:update.
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityConversation, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.id_required", "Conversation ID is required [DEFAULT]"))
	}

	target := req.Data.Status
	if target == conversationpb.ConversationStatus_CONVERSATION_STATUS_UNSPECIFIED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.status_required", "A target status is required [DEFAULT]"))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *conversationpb.UpdateConversationResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req, target)
			if err != nil {
				return fmt.Errorf("%s: %w",
					contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator,
						"conversation.errors.transition_failed", "Conversation status transition failed [DEFAULT]"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req, target)
}

func (uc *SetConversationStatusUseCase) executeCore(ctx context.Context, req *conversationpb.UpdateConversationRequest, target conversationpb.ConversationStatus) (*conversationpb.UpdateConversationResponse, error) {
	// Load current state for the transition guard + IDOR scoping.
	readResp, err := uc.repositories.Conversation.ReadConversation(ctx, &conversationpb.ReadConversationRequest{Data: &conversationpb.Conversation{Id: req.Data.Id}})
	if err != nil {
		return nil, err
	}
	if len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
			"conversation.errors.not_found", map[string]interface{}{"conversationId": req.Data.Id}, "Conversation not found [DEFAULT]"))
	}
	current := readResp.Data[0]

	if current.Status != target {
		if allowed, ok := allowedTransitions[current.Status]; !ok || !allowed[target] {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
				"conversation.validation.invalid_transition",
				map[string]interface{}{"from": current.Status.String(), "to": target.String()},
				"Invalid status transition [DEFAULT]"))
		}
	}

	// Apply only the status change; do not let the caller mutate IDOR anchors here.
	current.Status = target
	now := time.Now().UnixMilli()
	current.DateModified = &now

	return uc.repositories.Conversation.UpdateConversation(ctx, &conversationpb.UpdateConversationRequest{Data: current})
}
