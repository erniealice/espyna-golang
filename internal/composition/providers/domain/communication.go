package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Communication domain
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationParticipantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_participant"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	conversationReadReceiptpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_read_receipt"

	// Cross-domain dependencies
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// CommunicationRepositories contains all communication domain repositories.
// Active entities: Conversation, ConversationPost, ConversationReadReceipt.
// Seam (v2-queried, no use cases in v1): ConversationParticipant.
// Cross-domain: Client (IDOR gate), User (assigned_to / created_by / sender).
type CommunicationRepositories struct {
	Conversation            conversationpb.ConversationDomainServiceServer
	ConversationPost        conversationPostpb.ConversationPostDomainServiceServer
	ConversationReadReceipt conversationReadReceiptpb.ConversationReadReceiptDomainServiceServer
	ConversationParticipant conversationParticipantpb.ConversationParticipantDomainServiceServer
	// Cross-domain dependencies
	Client clientpb.ClientDomainServiceServer
	User   userpb.UserDomainServiceServer
}

// NewCommunicationRepositories creates and returns a new set of CommunicationRepositories.
func NewCommunicationRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*CommunicationRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	convRepo, err := repoCreator.CreateRepository(entityid.Conversation, conn, tableConfig.TableName(entityid.Conversation))
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation repository: %w", err)
	}

	convPostRepo, err := repoCreator.CreateRepository(entityid.ConversationPost, conn, tableConfig.TableName(entityid.ConversationPost))
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation_post repository: %w", err)
	}

	convReceiptRepo, err := repoCreator.CreateRepository(entityid.ConversationReadReceipt, conn, tableConfig.TableName(entityid.ConversationReadReceipt))
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation_read_receipt repository: %w", err)
	}

	convParticipantRepo, err := repoCreator.CreateRepository(entityid.ConversationParticipant, conn, tableConfig.TableName(entityid.ConversationParticipant))
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation_participant repository: %w", err)
	}

	// Cross-domain repositories
	clientRepo, err := repoCreator.CreateRepository(entityid.Client, conn, tableConfig.TableName(entityid.Client))
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	userRepo, err := repoCreator.CreateRepository(entityid.User, conn, tableConfig.TableName(entityid.User))
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}

	return &CommunicationRepositories{
		Conversation:            convRepo.(conversationpb.ConversationDomainServiceServer),
		ConversationPost:        convPostRepo.(conversationPostpb.ConversationPostDomainServiceServer),
		ConversationReadReceipt: convReceiptRepo.(conversationReadReceiptpb.ConversationReadReceiptDomainServiceServer),
		ConversationParticipant: convParticipantRepo.(conversationParticipantpb.ConversationParticipantDomainServiceServer),
		Client:                  clientRepo.(clientpb.ClientDomainServiceServer),
		User:                    userRepo.(userpb.UserDomainServiceServer),
	}, nil
}
