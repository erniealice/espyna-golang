package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/communication"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	conversationReadReceiptpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_read_receipt"
)

// ConfigureCommunicationDomain configures routes for the Communication domain.
func ConfigureCommunicationDomain(commUseCases *communication.CommunicationUseCases) contracts.DomainRouteConfiguration {
	if commUseCases == nil {
		fmt.Printf("WARNING: Communication use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "communication",
			Prefix:  "/communication",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	routes := []contracts.RouteConfiguration{}

	// Conversation routes
	if commUseCases.Conversation != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/create", Handler: contracts.NewGenericHandler(commUseCases.Conversation.CreateConversation, &conversationpb.CreateConversationRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/read", Handler: contracts.NewGenericHandler(commUseCases.Conversation.ReadConversation, &conversationpb.ReadConversationRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/update", Handler: contracts.NewGenericHandler(commUseCases.Conversation.UpdateConversation, &conversationpb.UpdateConversationRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/delete", Handler: contracts.NewGenericHandler(commUseCases.Conversation.DeleteConversation, &conversationpb.DeleteConversationRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/list", Handler: contracts.NewGenericHandler(commUseCases.Conversation.ListConversations, &conversationpb.ListConversationsRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/get-list-page-data", Handler: contracts.NewGenericHandler(commUseCases.Conversation.GetConversationListPageData, &conversationpb.GetConversationListPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/get-item-page-data", Handler: contracts.NewGenericHandler(commUseCases.Conversation.GetConversationItemPageData, &conversationpb.GetConversationItemPageDataRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/set-status", Handler: contracts.NewGenericHandler(commUseCases.Conversation.SetConversationStatus, &conversationpb.UpdateConversationRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation/assign", Handler: contracts.NewGenericHandler(commUseCases.Conversation.AssignConversation, &conversationpb.UpdateConversationRequest{})},
		)
	}

	// ConversationPost routes
	if commUseCases.ConversationPost != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation-post/send", Handler: contracts.NewGenericHandler(commUseCases.ConversationPost.SendConversationPost, &conversationPostpb.CreateConversationPostRequest{})},
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation-post/list", Handler: contracts.NewGenericHandler(commUseCases.ConversationPost.ListConversationPosts, &conversationPostpb.ListConversationPostsRequest{})},
		)
	}

	// ConversationReadReceipt routes
	if commUseCases.ConversationReadReceipt != nil {
		routes = append(routes,
			contracts.RouteConfiguration{Method: "POST", Path: "/api/communication/conversation-read-receipt/mark-read", Handler: contracts.NewGenericHandler(commUseCases.ConversationReadReceipt.MarkConversationRead, &conversationReadReceiptpb.CreateConversationReadReceiptRequest{})},
		)
	}

	// NOTE: ConversationParticipant has NO routes in v1 (seam entity, queried in v2 only).

	return contracts.DomainRouteConfiguration{
		Domain:  "communication",
		Prefix:  "/communication",
		Enabled: true,
		Routes:  routes,
	}
}
