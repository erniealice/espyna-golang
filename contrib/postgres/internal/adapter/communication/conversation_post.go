//go:build postgresql

package communication

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	conversationPostpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_post"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ConversationPost, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres conversation_post repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresConversationPostRepository(dbOps, tableName), nil
	})
}

// PostgresConversationPostRepository implements conversation_post CRUD operations using PostgreSQL.
type PostgresConversationPostRepository struct {
	conversationPostpb.UnimplementedConversationPostDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresConversationPostRepository creates a new PostgreSQL conversation_post repository.
func NewPostgresConversationPostRepository(dbOps interfaces.DatabaseOperation, tableName string) conversationPostpb.ConversationPostDomainServiceServer {
	if tableName == "" {
		tableName = "conversation_post"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresConversationPostRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateConversationPost inserts a post and bumps the parent conversation's
// last_post_at using GREATEST(COALESCE(last_post_at, 0), :sent_at) so concurrent
// sends cannot regress the monotone inbox-sort timestamp (codex M3 / invariant I3).
// The bump runs inside the same caller-supplied transaction context as the
// insert (the use case wraps both in ports.Transactor.ExecuteInTransaction).
func (r *PostgresConversationPostRepository) CreateConversationPost(ctx context.Context, req *conversationPostpb.CreateConversationPostRequest) (*conversationPostpb.CreateConversationPostResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("conversation_post data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation_post: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	post := &conversationPostpb.ConversationPost{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, post); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	// I3 / M3: monotone parent bump in the same connection/txn.
	if r.db != nil && post.GetConversationId() != "" && post.GetSentAt() != 0 {
		const bump = `UPDATE conversation
			SET last_post_at = GREATEST(COALESCE(last_post_at, 0), $2),
			    date_modified = now()
			WHERE id = $1`
		if _, err := r.db.ExecContext(ctx, bump, post.GetConversationId(), post.GetSentAt()); err != nil {
			return nil, fmt.Errorf("failed to bump conversation last_post_at: %w", err)
		}
	}

	return &conversationPostpb.CreateConversationPostResponse{
		Data:    []*conversationPostpb.ConversationPost{post},
		Success: true,
	}, nil
}

// ReadConversationPost retrieves a conversation_post by ID.
func (r *PostgresConversationPostRepository) ReadConversationPost(ctx context.Context, req *conversationPostpb.ReadConversationPostRequest) (*conversationPostpb.ReadConversationPostResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_post ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation_post: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("conversation_post with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	post := &conversationPostpb.ConversationPost{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, post); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationPostpb.ReadConversationPostResponse{
		Data:    []*conversationPostpb.ConversationPost{post},
		Success: true,
	}, nil
}

// UpdateConversationPost updates a conversation_post.
func (r *PostgresConversationPostRepository) UpdateConversationPost(ctx context.Context, req *conversationPostpb.UpdateConversationPostRequest) (*conversationPostpb.UpdateConversationPostResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_post ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation_post: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	post := &conversationPostpb.ConversationPost{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, post); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationPostpb.UpdateConversationPostResponse{
		Data:    []*conversationPostpb.ConversationPost{post},
		Success: true,
	}, nil
}

// DeleteConversationPost soft-deletes a conversation_post via dbOps.Delete.
func (r *PostgresConversationPostRepository) DeleteConversationPost(ctx context.Context, req *conversationPostpb.DeleteConversationPostRequest) (*conversationPostpb.DeleteConversationPostResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_post ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete conversation_post: %w", err)
	}

	return &conversationPostpb.DeleteConversationPostResponse{
		Success: true,
	}, nil
}

// ListConversationPosts lists posts. The use case is responsible for scoping
// to the parent conversation + IDOR anchors before calling this.
func (r *PostgresConversationPostRepository) ListConversationPosts(ctx context.Context, req *conversationPostpb.ListConversationPostsRequest) (*conversationPostpb.ListConversationPostsResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation_posts: %w", err)
	}

	var posts []*conversationPostpb.ConversationPost
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		post := &conversationPostpb.ConversationPost{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, post); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	return &conversationPostpb.ListConversationPostsResponse{
		Data:    posts,
		Success: true,
	}, nil
}

// GetConversationPostListPageData composes over ListConversationPosts.
func (r *PostgresConversationPostRepository) GetConversationPostListPageData(
	ctx context.Context,
	req *conversationPostpb.GetConversationPostListPageDataRequest,
) (*conversationPostpb.GetConversationPostListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation_post list page data request is required")
	}

	limit := int32(50)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
		}
	}

	listResp, err := r.ListConversationPosts(ctx, &conversationPostpb.ListConversationPostsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation_posts for page data: %w", err)
	}
	posts := listResp.GetData()

	totalItems := int32(len(posts))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &conversationPostpb.GetConversationPostListPageDataResponse{
		ConversationPostList: posts,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetConversationPostItemPageData composes over ReadConversationPost.
func (r *PostgresConversationPostRepository) GetConversationPostItemPageData(
	ctx context.Context,
	req *conversationPostpb.GetConversationPostItemPageDataRequest,
) (*conversationPostpb.GetConversationPostItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation_post item page data request is required")
	}
	if req.ConversationPostId == "" {
		return nil, fmt.Errorf("conversation_post ID is required")
	}

	rr, err := r.ReadConversationPost(ctx, &conversationPostpb.ReadConversationPostRequest{Data: &conversationPostpb.ConversationPost{Id: req.ConversationPostId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("conversation_post with ID '%s' not found", req.ConversationPostId)
	}

	return &conversationPostpb.GetConversationPostItemPageDataResponse{
		ConversationPost: rr.GetData()[0],
		Success:          true,
	}, nil
}

// NewConversationPostRepository creates a new PostgreSQL conversation_post repository (old-style constructor).
func NewConversationPostRepository(db *sql.DB, tableName string) conversationPostpb.ConversationPostDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresConversationPostRepository(dbOps, tableName)
}
