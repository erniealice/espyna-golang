//go:build postgresql

package communication

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	conversationParticipantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_participant"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ConversationParticipant, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres conversation_participant repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresConversationParticipantRepository(dbOps, tableName), nil
	})
}

// PostgresConversationParticipantRepository implements conversation_participant
// CRUD operations using PostgreSQL. v1 seam: the table + adapter ship now but no
// use cases query it until v2 (team inboxes).
type PostgresConversationParticipantRepository struct {
	conversationParticipantpb.UnimplementedConversationParticipantDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresConversationParticipantRepository creates a new repository.
func NewPostgresConversationParticipantRepository(dbOps interfaces.DatabaseOperation, tableName string) conversationParticipantpb.ConversationParticipantDomainServiceServer {
	if tableName == "" {
		tableName = "conversation_participant"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresConversationParticipantRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateConversationParticipant creates a new participant row.
func (r *PostgresConversationParticipantRepository) CreateConversationParticipant(ctx context.Context, req *conversationParticipantpb.CreateConversationParticipantRequest) (*conversationParticipantpb.CreateConversationParticipantResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("conversation_participant data is required")
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
		return nil, fmt.Errorf("failed to create conversation_participant: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	participant := &conversationParticipantpb.ConversationParticipant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, participant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationParticipantpb.CreateConversationParticipantResponse{
		Data:    []*conversationParticipantpb.ConversationParticipant{participant},
		Success: true,
	}, nil
}

// ReadConversationParticipant retrieves a participant by ID.
func (r *PostgresConversationParticipantRepository) ReadConversationParticipant(ctx context.Context, req *conversationParticipantpb.ReadConversationParticipantRequest) (*conversationParticipantpb.ReadConversationParticipantResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_participant ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation_participant: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("conversation_participant with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	participant := &conversationParticipantpb.ConversationParticipant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, participant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationParticipantpb.ReadConversationParticipantResponse{
		Data:    []*conversationParticipantpb.ConversationParticipant{participant},
		Success: true,
	}, nil
}

// UpdateConversationParticipant updates a participant.
func (r *PostgresConversationParticipantRepository) UpdateConversationParticipant(ctx context.Context, req *conversationParticipantpb.UpdateConversationParticipantRequest) (*conversationParticipantpb.UpdateConversationParticipantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_participant ID is required")
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
		return nil, fmt.Errorf("failed to update conversation_participant: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	participant := &conversationParticipantpb.ConversationParticipant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, participant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationParticipantpb.UpdateConversationParticipantResponse{
		Data:    []*conversationParticipantpb.ConversationParticipant{participant},
		Success: true,
	}, nil
}

// DeleteConversationParticipant soft-deletes a participant.
func (r *PostgresConversationParticipantRepository) DeleteConversationParticipant(ctx context.Context, req *conversationParticipantpb.DeleteConversationParticipantRequest) (*conversationParticipantpb.DeleteConversationParticipantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_participant ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete conversation_participant: %w", err)
	}

	return &conversationParticipantpb.DeleteConversationParticipantResponse{
		Success: true,
	}, nil
}

// ListConversationParticipants lists participants.
func (r *PostgresConversationParticipantRepository) ListConversationParticipants(ctx context.Context, req *conversationParticipantpb.ListConversationParticipantsRequest) (*conversationParticipantpb.ListConversationParticipantsResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation_participants: %w", err)
	}

	var participants []*conversationParticipantpb.ConversationParticipant
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		participant := &conversationParticipantpb.ConversationParticipant{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, participant); err != nil {
			continue
		}
		participants = append(participants, participant)
	}

	return &conversationParticipantpb.ListConversationParticipantsResponse{
		Data:    participants,
		Success: true,
	}, nil
}

// GetConversationParticipantListPageData composes over ListConversationParticipants.
func (r *PostgresConversationParticipantRepository) GetConversationParticipantListPageData(
	ctx context.Context,
	req *conversationParticipantpb.GetConversationParticipantListPageDataRequest,
) (*conversationParticipantpb.GetConversationParticipantListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation_participant list page data request is required")
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

	listResp, err := r.ListConversationParticipants(ctx, &conversationParticipantpb.ListConversationParticipantsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation_participants for page data: %w", err)
	}
	participants := listResp.GetData()

	totalItems := int32(len(participants))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &conversationParticipantpb.GetConversationParticipantListPageDataResponse{
		ConversationParticipantList: participants,
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

// GetConversationParticipantItemPageData composes over ReadConversationParticipant.
func (r *PostgresConversationParticipantRepository) GetConversationParticipantItemPageData(
	ctx context.Context,
	req *conversationParticipantpb.GetConversationParticipantItemPageDataRequest,
) (*conversationParticipantpb.GetConversationParticipantItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation_participant item page data request is required")
	}
	if req.ConversationParticipantId == "" {
		return nil, fmt.Errorf("conversation_participant ID is required")
	}

	rr, err := r.ReadConversationParticipant(ctx, &conversationParticipantpb.ReadConversationParticipantRequest{Data: &conversationParticipantpb.ConversationParticipant{Id: req.ConversationParticipantId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("conversation_participant with ID '%s' not found", req.ConversationParticipantId)
	}

	return &conversationParticipantpb.GetConversationParticipantItemPageDataResponse{
		ConversationParticipant: rr.GetData()[0],
		Success:                 true,
	}, nil
}

// NewConversationParticipantRepository creates a new repository (old-style constructor).
func NewConversationParticipantRepository(db *sql.DB, tableName string) conversationParticipantpb.ConversationParticipantDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresConversationParticipantRepository(dbOps, tableName)
}
