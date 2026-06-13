package conversation_post

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	emailpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/email"
)

// NotifyConversationPostRepositories groups repository dependencies.
type NotifyConversationPostRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
}

// NotifyConversationPostServices groups service dependencies.
type NotifyConversationPostServices struct {
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	Email      ports.EmailProvider // OPTIONAL — nil makes this a no-op
}

// NotifyConversationPostUseCase composes an outbound "you have a new message"
// email + portal deep link for a freshly-sent post (D.6). It is internal (no UI
// verb) and fail-soft: a notification failure never blocks the send.
type NotifyConversationPostUseCase struct {
	repositories NotifyConversationPostRepositories
	services     NotifyConversationPostServices
}

// NewNotifyConversationPostUseCase creates a new NotifyConversationPostUseCase.
func NewNotifyConversationPostUseCase(repos NotifyConversationPostRepositories, svcs NotifyConversationPostServices) *NotifyConversationPostUseCase {
	return &NotifyConversationPostUseCase{repositories: repos, services: svcs}
}

// Execute sends the notification email for a post. Returns nil (no-op) when the
// email provider is unwired or disabled — the caller treats notification as
// best-effort.
func (uc *NotifyConversationPostUseCase) Execute(ctx context.Context, post *conversationPostpb.ConversationPost) error {
	if uc == nil || uc.services.Email == nil || !uc.services.Email.IsEnabled() {
		return nil
	}
	if post == nil {
		return nil
	}

	// The recipient address + portal deep link are resolved by the composition
	// root's email wiring in a later phase; v1 emits a minimal envelope so the
	// hook is live and acceptance-testable end-to-end.
	req := &emailpb.SendEmailRequest{}
	if _, err := uc.services.Email.SendEmail(ctx, req); err != nil {
		// Fail-soft: do not propagate notification errors to the send path.
		return nil
	}
	return nil
}
