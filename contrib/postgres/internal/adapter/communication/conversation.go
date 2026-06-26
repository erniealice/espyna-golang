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
	conversationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Conversation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres conversation repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresConversationRepository(dbOps, tableName), nil
	})
}

// PostgresConversationRepository implements conversation CRUD operations using PostgreSQL.
type PostgresConversationRepository struct {
	conversationpb.UnimplementedConversationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresConversationRepository creates a new PostgreSQL conversation repository.
func NewPostgresConversationRepository(dbOps interfaces.DatabaseOperation, tableName string) conversationpb.ConversationDomainServiceServer {
	if tableName == "" {
		tableName = "conversation"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresConversationRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateConversation creates a new conversation.
func (r *PostgresConversationRepository) CreateConversation(ctx context.Context, req *conversationpb.CreateConversationRequest) (*conversationpb.CreateConversationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("conversation data is required")
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
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	conv := &conversationpb.Conversation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, conv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationpb.CreateConversationResponse{
		Data:    []*conversationpb.Conversation{conv},
		Success: true,
	}, nil
}

// ReadConversation retrieves a conversation by ID.
func (r *PostgresConversationRepository) ReadConversation(ctx context.Context, req *conversationpb.ReadConversationRequest) (*conversationpb.ReadConversationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("conversation with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	conv := &conversationpb.Conversation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, conv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationpb.ReadConversationResponse{
		Data:    []*conversationpb.Conversation{conv},
		Success: true,
	}, nil
}

// UpdateConversation updates a conversation.
func (r *PostgresConversationRepository) UpdateConversation(ctx context.Context, req *conversationpb.UpdateConversationRequest) (*conversationpb.UpdateConversationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation ID is required")
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
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	conv := &conversationpb.Conversation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, conv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationpb.UpdateConversationResponse{
		Data:    []*conversationpb.Conversation{conv},
		Success: true,
	}, nil
}

// DeleteConversation soft-deletes a conversation via dbOps.Delete.
func (r *PostgresConversationRepository) DeleteConversation(ctx context.Context, req *conversationpb.DeleteConversationRequest) (*conversationpb.DeleteConversationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete conversation: %w", err)
	}

	return &conversationpb.DeleteConversationResponse{
		Success: true,
	}, nil
}

// ListConversations lists conversations. The WorkspaceAwareOperations layer
// injects the workspace_id predicate from context; the use case adds the
// client_id IDOR filter for portal callers.
func (r *PostgresConversationRepository) ListConversations(ctx context.Context, req *conversationpb.ListConversationsRequest) (*conversationpb.ListConversationsResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	var convs []*conversationpb.Conversation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		conv := &conversationpb.Conversation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, conv); err != nil {
			continue
		}
		convs = append(convs, conv)
	}

	return &conversationpb.ListConversationsResponse{
		Data:    convs,
		Success: true,
	}, nil
}

// GetConversationListPageData composes over ListConversations and computes
// pagination metadata locally.
func (r *PostgresConversationRepository) GetConversationListPageData(
	ctx context.Context,
	req *conversationpb.GetConversationListPageDataRequest,
) (*conversationpb.GetConversationListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation list page data request is required")
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

	listResp, err := r.ListConversations(ctx, &conversationpb.ListConversationsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations for page data: %w", err)
	}
	convs := listResp.GetData()

	totalItems := int32(len(convs))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &conversationpb.GetConversationListPageDataResponse{
		ConversationList: convs,
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

// GetConversationItemPageData composes over ReadConversation.
func (r *PostgresConversationRepository) GetConversationItemPageData(
	ctx context.Context,
	req *conversationpb.GetConversationItemPageDataRequest,
) (*conversationpb.GetConversationItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation item page data request is required")
	}
	if req.ConversationId == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	rr, err := r.ReadConversation(ctx, &conversationpb.ReadConversationRequest{Data: &conversationpb.Conversation{Id: req.ConversationId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("conversation with ID '%s' not found", req.ConversationId)
	}

	return &conversationpb.GetConversationItemPageDataResponse{
		Conversation: rr.GetData()[0],
		Success:      true,
	}, nil
}

// NewConversationRepository creates a new PostgreSQL conversation repository (old-style constructor).
func NewConversationRepository(db *sql.DB, tableName string) conversationpb.ConversationDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresConversationRepository(dbOps, tableName)
}
