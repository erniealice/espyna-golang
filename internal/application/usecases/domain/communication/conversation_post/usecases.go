package conversation_post

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

// ConversationPostRepositories groups all repository dependencies for
// conversation_post use cases.
type ConversationPostRepositories struct {
	ConversationPost conversationPostpb.ConversationPostDomainServiceServer
	Conversation     conversationpb.ConversationDomainServiceServer // parent IDOR + last_post_at
	Client           clientpb.ClientDomainServiceServer             // IDOR gate
}

// ConversationPostServices groups all business service dependencies.
type ConversationPostServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
	// Email is OPTIONAL: when non-nil, NotifyConversationPost sends an
	// outbound "new message" email. Nil leaves notification a no-op (the
	// email provider is wired by the composition root in a later phase).
	Email ports.EmailProvider
}

// UseCases contains all conversation_post-related use cases.
type UseCases struct {
	SendConversationPost   *SendConversationPostUseCase
	ListConversationPosts  *ListConversationPostsUseCase
	NotifyConversationPost *NotifyConversationPostUseCase
}

// NewUseCases creates a new collection of conversation_post use cases.
func NewUseCases(
	repositories ConversationPostRepositories,
	services ConversationPostServices,
	transactionService ports.Transactor,
) *UseCases {
	notify := NewNotifyConversationPostUseCase(
		NotifyConversationPostRepositories{Conversation: repositories.Conversation},
		NotifyConversationPostServices{Translator: services.Translator, Email: services.Email},
	)

	return &UseCases{
		SendConversationPost: NewSendConversationPostUseCase(
			SendConversationPostRepositories{
				ConversationPost: repositories.ConversationPost,
				Conversation:     repositories.Conversation,
				Client:           repositories.Client,
			},
			SendConversationPostServices{
				Authorizer:  services.Authorizer,
				Transactor:  transactionService,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
			notify,
		),
		ListConversationPosts: NewListConversationPostsUseCase(
			ListConversationPostsRepositories{
				ConversationPost: repositories.ConversationPost,
				Conversation:     repositories.Conversation,
			},
			ListConversationPostsServices{
				Authorizer: services.Authorizer,
				Transactor: transactionService,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		NotifyConversationPost: notify,
	}
}
