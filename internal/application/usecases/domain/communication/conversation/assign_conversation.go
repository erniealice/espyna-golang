package conversation

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
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// AssignConversationRepositories groups all repository dependencies.
type AssignConversationRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	User         userpb.UserDomainServiceServer
}

// AssignConversationServices groups all business service dependencies.
type AssignConversationServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// AssignConversationUseCase stamps assigned_to_user_id and (optionally) auto-
// advances status to IN_PROGRESS. Verb-folds to conversation:update (D.6),
// mirroring AssignActivity.
type AssignConversationUseCase struct {
	repositories AssignConversationRepositories
	services     AssignConversationServices
}

// NewAssignConversationUseCase creates a new AssignConversationUseCase.
func NewAssignConversationUseCase(repos AssignConversationRepositories, svcs AssignConversationServices) *AssignConversationUseCase {
	return &AssignConversationUseCase{repositories: repos, services: svcs}
}

// Execute performs the assign conversation operation.
func (uc *AssignConversationUseCase) Execute(ctx context.Context, req *conversationpb.UpdateConversationRequest) (*conversationpb.UpdateConversationResponse, error) {
	// Verb-fold: conversation:assign -> conversation:update.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Conversation,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.id_required", "Conversation ID is required [DEFAULT]"))
	}
	if req.Data.GetAssignedToUserId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"conversation.validation.assignee_required", "An assignee user is required [DEFAULT]"))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *conversationpb.UpdateConversationResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("%s: %w",
					contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator,
						"conversation.errors.assign_failed", "Conversation assignment failed [DEFAULT]"), err)
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

func (uc *AssignConversationUseCase) executeCore(ctx context.Context, req *conversationpb.UpdateConversationRequest) (*conversationpb.UpdateConversationResponse, error) {
	assignee := req.Data.GetAssignedToUserId()

	// Validate the assignee user exists + is active.
	if uc.repositories.User != nil {
		userResp, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{Data: &userpb.User{Id: assignee}})
		if err != nil || userResp == nil || len(userResp.Data) == 0 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
				"conversation.validation.assignee_not_found", map[string]interface{}{"userId": assignee}, "Assignee user not found [DEFAULT]"))
		}
	}

	// Load current state for IDOR scoping + status auto-advance.
	readResp, err := uc.repositories.Conversation.ReadConversation(ctx, &conversationpb.ReadConversationRequest{Data: &conversationpb.Conversation{Id: req.Data.Id}})
	if err != nil {
		return nil, err
	}
	if len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator,
			"conversation.errors.not_found", map[string]interface{}{"conversationId": req.Data.Id}, "Conversation not found [DEFAULT]"))
	}
	current := readResp.Data[0]

	current.AssignedToUserId = &assignee
	// Auto-advance OPEN -> IN_PROGRESS on first assignment (mirrors AssignActivity).
	if current.Status == conversationpb.ConversationStatus_CONVERSATION_STATUS_OPEN {
		current.Status = conversationpb.ConversationStatus_CONVERSATION_STATUS_IN_PROGRESS
	}
	now := time.Now().UnixMilli()
	current.DateModified = &now

	return uc.repositories.Conversation.UpdateConversation(ctx, &conversationpb.UpdateConversationRequest{Data: current})
}
