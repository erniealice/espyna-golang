package communication

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

	conversationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/communication/conversation"
	conversationPostUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/communication/conversation_post"
	conversationReceiptUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/communication/conversation_read_receipt"

	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	conversationReceiptpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_read_receipt"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// CommunicationUseCases contains all communication-domain use cases.
//
// Active entities: Conversation, ConversationPost, ConversationReadReceipt.
// ConversationParticipant ships proto + adapter + provider but NO use cases in
// v1 (v2-queried seam).
type CommunicationUseCases struct {
	Conversation            *conversationUseCases.UseCases
	ConversationPost        *conversationPostUseCases.UseCases
	ConversationReadReceipt *conversationReceiptUseCases.UseCases
}

// NewCommunicationUseCases wires all communication use cases from raw repo/service
// dependencies.
func NewCommunicationUseCases(
	convRepo conversationpb.ConversationDomainServiceServer,
	convPostRepo conversationPostpb.ConversationPostDomainServiceServer,
	convReceiptRepo conversationReceiptpb.ConversationReadReceiptDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	userRepo userpb.UserDomainServiceServer,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
) *CommunicationUseCases {
	return &CommunicationUseCases{
		Conversation: conversationUseCases.NewUseCases(
			conversationUseCases.ConversationRepositories{Conversation: convRepo, Client: clientRepo, User: userRepo},
			conversationUseCases.ConversationServices{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idSvc},
			txSvc,
		),
		ConversationPost: conversationPostUseCases.NewUseCases(
			conversationPostUseCases.ConversationPostRepositories{Conversation: convRepo, ConversationPost: convPostRepo, Client: clientRepo},
			conversationPostUseCases.ConversationPostServices{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idSvc},
			txSvc,
		),
		ConversationReadReceipt: conversationReceiptUseCases.NewUseCases(
			conversationReceiptUseCases.ConversationReadReceiptRepositories{ConversationReadReceipt: convReceiptRepo, Conversation: convRepo, ConversationPost: convPostRepo},
			conversationReceiptUseCases.ConversationReadReceiptServices{Authorizer: authSvc, Transactor: txSvc, Translator: i18nSvc, IDGenerator: idSvc},
			txSvc,
		),
	}
}
