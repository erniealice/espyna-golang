package conversation_read_receipt

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	conversationReadReceiptpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_read_receipt"
)

// ComputeConversationUnreadRepositories groups all repository dependencies.
type ComputeConversationUnreadRepositories struct {
	ConversationReadReceipt conversationReadReceiptpb.ConversationReadReceiptDomainServiceServer
	ConversationPost        conversationPostpb.ConversationPostDomainServiceServer
	Conversation            conversationpb.ConversationDomainServiceServer
}

// ComputeConversationUnreadServices groups all business service dependencies.
type ComputeConversationUnreadServices struct {
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UnreadResult is the per-conversation unread count returned to the notifications
// aggregator. It is a pure projection (no mutable counter, W1 doctrine).
type UnreadResult struct {
	ConversationID string
	UnreadCount    int32
}

// ComputeConversationUnreadUseCase computes unread counts using the SINGLE
// monotonic cursor (last_read_post_id, ordered by (sent_at, id)) — codex H2.
// It has TWO branches (D.3):
//   - client: posts after the cursor, excluding the reader's own posts;
//   - staff:  posts authored by a client (sender_principal_type=client) after the
//     cursor, workspace-scoped (optionally narrowed to assigned-to-me).
//
// Internal use case (no UI verb): callers are the /api/notifications aggregator
// and the inbox badge, not an HTTP route.
type ComputeConversationUnreadUseCase struct {
	repositories ComputeConversationUnreadRepositories
	services     ComputeConversationUnreadServices
}

// NewComputeConversationUnreadUseCase creates a new use case.
func NewComputeConversationUnreadUseCase(repos ComputeConversationUnreadRepositories, svcs ComputeConversationUnreadServices) *ComputeConversationUnreadUseCase {
	return &ComputeConversationUnreadUseCase{repositories: repos, services: svcs}
}

// ExecuteForClient computes unread for a client reader of a single conversation:
// posts after the cursor, excluding the reader's own posts.
func (uc *ComputeConversationUnreadUseCase) ExecuteForClient(ctx context.Context, conversationID, readerPrincipalType, readerPrincipalID, readerUserID string) (*UnreadResult, error) {
	cursorSentAt, cursorID, err := uc.cursorTuple(ctx, conversationID, readerPrincipalType, readerPrincipalID)
	if err != nil {
		return nil, err
	}

	posts, err := uc.listPosts(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	var unread int32
	for _, p := range posts {
		if !afterCursor(p, cursorSentAt, cursorID) {
			continue
		}
		// Exclude the reader's own posts.
		if readerUserID != "" && p.GetSenderUserId() == readerUserID {
			continue
		}
		unread++
	}

	return &UnreadResult{ConversationID: conversationID, UnreadCount: unread}, nil
}

// ExecuteForStaff computes unread for a staff reader of a single conversation:
// only client-authored posts after the cursor count.
func (uc *ComputeConversationUnreadUseCase) ExecuteForStaff(ctx context.Context, conversationID, readerPrincipalType, readerPrincipalID string) (*UnreadResult, error) {
	cursorSentAt, cursorID, err := uc.cursorTuple(ctx, conversationID, readerPrincipalType, readerPrincipalID)
	if err != nil {
		return nil, err
	}

	posts, err := uc.listPosts(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	var unread int32
	for _, p := range posts {
		if !afterCursor(p, cursorSentAt, cursorID) {
			continue
		}
		// Staff branch (redteam B4 / Q-MSG-15): only client-authored posts count.
		if p.GetSenderPrincipalType() != conversationPostpb.SenderPrincipalType_SENDER_PRINCIPAL_TYPE_CLIENT {
			continue
		}
		unread++
	}

	return &UnreadResult{ConversationID: conversationID, UnreadCount: unread}, nil
}

// cursorTuple resolves the (sent_at, id) tuple of the reader's last_read_post_id.
// When no receipt or no cursor exists the tuple is the zero tuple → all posts unread.
func (uc *ComputeConversationUnreadUseCase) cursorTuple(ctx context.Context, conversationID, readerPrincipalType, readerPrincipalID string) (int64, string, error) {
	receipts, err := uc.repositories.ConversationReadReceipt.ListConversationReadReceipts(ctx, &conversationReadReceiptpb.ListConversationReadReceiptsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				stringEq("conversation_id", conversationID),
				stringEq("reader_principal_type", readerPrincipalType),
				stringEq("reader_principal_id", readerPrincipalID),
			},
		},
	})
	if err != nil {
		return 0, "", err
	}
	if receipts == nil || len(receipts.Data) == 0 {
		return 0, "", nil
	}
	cursorPostID := receipts.Data[0].GetLastReadPostId()
	if cursorPostID == "" {
		return 0, "", nil
	}

	postResp, err := uc.repositories.ConversationPost.ReadConversationPost(ctx, &conversationPostpb.ReadConversationPostRequest{Data: &conversationPostpb.ConversationPost{Id: cursorPostID}})
	if err != nil || postResp == nil || len(postResp.Data) == 0 {
		// Cursor post missing → treat as zero tuple (all unread).
		return 0, "", nil
	}
	cursor := postResp.Data[0]
	return cursor.GetSentAt(), cursor.GetId(), nil
}

func (uc *ComputeConversationUnreadUseCase) listPosts(ctx context.Context, conversationID string) ([]*conversationPostpb.ConversationPost, error) {
	resp, err := uc.repositories.ConversationPost.ListConversationPosts(ctx, &conversationPostpb.ListConversationPostsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{stringEq("conversation_id", conversationID)},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.Data, nil
}

// afterCursor reports whether the post's (sent_at, id) tuple is strictly greater
// than the cursor tuple.
func afterCursor(p *conversationPostpb.ConversationPost, cursorSentAt int64, cursorID string) bool {
	if p.GetSentAt() != cursorSentAt {
		return p.GetSentAt() > cursorSentAt
	}
	return p.GetId() > cursorID
}

func stringEq(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{Value: value, Operator: commonpb.StringOperator_STRING_EQUALS},
		},
	}
}
