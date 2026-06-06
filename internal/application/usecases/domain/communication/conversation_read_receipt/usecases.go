package conversation_read_receipt

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	conversationReadReceiptpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_read_receipt"
)

// ConversationReadReceiptRepositories groups all repository dependencies.
type ConversationReadReceiptRepositories struct {
	ConversationReadReceipt conversationReadReceiptpb.ConversationReadReceiptDomainServiceServer
	Conversation            conversationpb.ConversationDomainServiceServer
	ConversationPost        conversationPostpb.ConversationPostDomainServiceServer // unread computation
}

// ConversationReadReceiptServices groups all business service dependencies.
type ConversationReadReceiptServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all conversation_read_receipt-related use cases.
type UseCases struct {
	MarkConversationRead      *MarkConversationReadUseCase
	ComputeConversationUnread *ComputeConversationUnreadUseCase
}

// NewUseCases creates a new collection of conversation_read_receipt use cases.
func NewUseCases(
	repositories ConversationReadReceiptRepositories,
	services ConversationReadReceiptServices,
	transactionService ports.Transactor,
) *UseCases {
	return &UseCases{
		MarkConversationRead: NewMarkConversationReadUseCase(
			MarkConversationReadRepositories{
				ConversationReadReceipt: repositories.ConversationReadReceipt,
				Conversation:            repositories.Conversation,
			},
			MarkConversationReadServices{
				Authorizer:  services.Authorizer,
				Transactor:  transactionService,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ComputeConversationUnread: NewComputeConversationUnreadUseCase(
			ComputeConversationUnreadRepositories{
				ConversationReadReceipt: repositories.ConversationReadReceipt,
				ConversationPost:        repositories.ConversationPost,
				Conversation:            repositories.Conversation,
			},
			ComputeConversationUnreadServices{
				Translator: services.Translator,
			},
		),
	}
}
