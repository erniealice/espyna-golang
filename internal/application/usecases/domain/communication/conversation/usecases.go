package conversation

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ConversationRepositories groups all repository dependencies for conversation use cases.
type ConversationRepositories struct {
	Conversation conversationpb.ConversationDomainServiceServer
	Client       clientpb.ClientDomainServiceServer // IDOR gate + FK validation
	User         userpb.UserDomainServiceServer     // assigned_to / created_by FK validation
}

// ConversationServices groups all business service dependencies.
type ConversationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all conversation-related use cases.
type UseCases struct {
	CreateConversation          *CreateConversationUseCase
	ReadConversation            *ReadConversationUseCase
	UpdateConversation          *UpdateConversationUseCase
	DeleteConversation          *DeleteConversationUseCase
	ListConversations           *ListConversationsUseCase
	GetConversationListPageData *GetConversationListPageDataUseCase
	GetConversationItemPageData *GetConversationItemPageDataUseCase
	SetConversationStatus       *SetConversationStatusUseCase
	AssignConversation          *AssignConversationUseCase
}

// NewUseCases creates a new collection of conversation use cases.
func NewUseCases(
	repositories ConversationRepositories,
	services ConversationServices,
	transactionService ports.Transactor,
) *UseCases {
	return &UseCases{
		CreateConversation: NewCreateConversationUseCase(
			CreateConversationRepositories(repositories),
			CreateConversationServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator, IDGenerator: services.IDGenerator},
		),
		ReadConversation: NewReadConversationUseCase(
			ReadConversationRepositories(repositories),
			ReadConversationServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
		UpdateConversation: NewUpdateConversationUseCase(
			UpdateConversationRepositories(repositories),
			UpdateConversationServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
		DeleteConversation: NewDeleteConversationUseCase(
			DeleteConversationRepositories(repositories),
			DeleteConversationServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
		ListConversations: NewListConversationsUseCase(
			ListConversationsRepositories(repositories),
			ListConversationsServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
		GetConversationListPageData: NewGetConversationListPageDataUseCase(
			GetConversationListPageDataRepositories{Conversation: repositories.Conversation},
			GetConversationListPageDataServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
		GetConversationItemPageData: NewGetConversationItemPageDataUseCase(
			GetConversationItemPageDataRepositories{Conversation: repositories.Conversation},
			GetConversationItemPageDataServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
		SetConversationStatus: NewSetConversationStatusUseCase(
			SetConversationStatusRepositories(repositories),
			SetConversationStatusServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
		AssignConversation: NewAssignConversationUseCase(
			AssignConversationRepositories(repositories),
			AssignConversationServices{Authorizer: services.Authorizer, Transactor: transactionService, Translator: services.Translator},
		),
	}
}
