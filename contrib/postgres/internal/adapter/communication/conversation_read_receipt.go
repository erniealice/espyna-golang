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
	conversationReadReceiptpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/communication/conversation_read_receipt"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ConversationReadReceipt, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres conversation_read_receipt repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresConversationReadReceiptRepository(dbOps, tableName), nil
	})
}

// PostgresConversationReadReceiptRepository implements conversation_read_receipt
// operations using PostgreSQL. Create is an UPSERT keyed on the principal-scoped
// unique key (invariant I4).
type PostgresConversationReadReceiptRepository struct {
	conversationReadReceiptpb.UnimplementedConversationReadReceiptDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresConversationReadReceiptRepository creates a new repository.
func NewPostgresConversationReadReceiptRepository(dbOps interfaces.DatabaseOperation, tableName string) conversationReadReceiptpb.ConversationReadReceiptDomainServiceServer {
	if tableName == "" {
		tableName = "conversation_read_receipt"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresConversationReadReceiptRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateConversationReadReceipt upserts the per-reader high-water mark cursor
// keyed on (conversation_id, reader_principal_type, reader_principal_id) — the
// principal-scoped unique key (D.3 / invariant I4). When the raw *sql.DB is
// unavailable it falls back to a plain insert via dbOps.Create.
func (r *PostgresConversationReadReceiptRepository) CreateConversationReadReceipt(ctx context.Context, req *conversationReadReceiptpb.CreateConversationReadReceiptRequest) (*conversationReadReceiptpb.CreateConversationReadReceiptResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("conversation_read_receipt data is required")
	}

	if r.db != nil {
		const upsert = `
			INSERT INTO conversation_read_receipt
				(id, conversation_id, reader_principal_type, reader_principal_id,
				 user_id, workspace_id, last_read_post_id, last_read_at, active,
				 date_created, date_modified)
			VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, true, now(), now())
			ON CONFLICT (conversation_id, reader_principal_type, reader_principal_id)
			DO UPDATE SET
				last_read_post_id = EXCLUDED.last_read_post_id,
				last_read_at      = EXCLUDED.last_read_at,
				date_modified     = now()
			RETURNING id, conversation_id, reader_principal_type, reader_principal_id,
				user_id, workspace_id, COALESCE(last_read_post_id, ''), COALESCE(last_read_at, 0), active`

		d := req.Data
		row := r.db.QueryRowContext(ctx, upsert,
			d.GetId(), d.GetConversationId(), d.GetReaderPrincipalType(), d.GetReaderPrincipalId(),
			d.GetUserId(), d.GetWorkspaceId(), d.GetLastReadPostId(), d.GetLastReadAt(),
		)

		var (
			id, convID, readerType, readerID, userID, wsID, lastReadPostID string
			lastReadAt                                                     int64
			active                                                         bool
		)
		if err := row.Scan(&id, &convID, &readerType, &readerID, &userID, &wsID, &lastReadPostID, &lastReadAt, &active); err != nil {
			return nil, fmt.Errorf("failed to upsert conversation_read_receipt: %w", err)
		}

		out := &conversationReadReceiptpb.ConversationReadReceipt{
			Id:                  id,
			ConversationId:      convID,
			ReaderPrincipalType: readerType,
			ReaderPrincipalId:   readerID,
			UserId:              userID,
			WorkspaceId:         wsID,
			Active:              active,
		}
		if lastReadPostID != "" {
			out.LastReadPostId = &lastReadPostID
		}
		if lastReadAt != 0 {
			out.LastReadAt = &lastReadAt
		}

		return &conversationReadReceiptpb.CreateConversationReadReceiptResponse{
			Data:    []*conversationReadReceiptpb.ConversationReadReceipt{out},
			Success: true,
		}, nil
	}

	// Fallback: plain insert via common operations.
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
		return nil, fmt.Errorf("failed to create conversation_read_receipt: %w", err)
	}
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	receipt := &conversationReadReceiptpb.ConversationReadReceipt{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, receipt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &conversationReadReceiptpb.CreateConversationReadReceiptResponse{
		Data:    []*conversationReadReceiptpb.ConversationReadReceipt{receipt},
		Success: true,
	}, nil
}

// ReadConversationReadReceipt retrieves a receipt by ID.
func (r *PostgresConversationReadReceiptRepository) ReadConversationReadReceipt(ctx context.Context, req *conversationReadReceiptpb.ReadConversationReadReceiptRequest) (*conversationReadReceiptpb.ReadConversationReadReceiptResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_read_receipt ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation_read_receipt: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("conversation_read_receipt with ID '%s' not found", req.Data.Id)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	receipt := &conversationReadReceiptpb.ConversationReadReceipt{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, receipt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationReadReceiptpb.ReadConversationReadReceiptResponse{
		Data:    []*conversationReadReceiptpb.ConversationReadReceipt{receipt},
		Success: true,
	}, nil
}

// UpdateConversationReadReceipt updates a receipt.
func (r *PostgresConversationReadReceiptRepository) UpdateConversationReadReceipt(ctx context.Context, req *conversationReadReceiptpb.UpdateConversationReadReceiptRequest) (*conversationReadReceiptpb.UpdateConversationReadReceiptResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_read_receipt ID is required")
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
		return nil, fmt.Errorf("failed to update conversation_read_receipt: %w", err)
	}

	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	receipt := &conversationReadReceiptpb.ConversationReadReceipt{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, receipt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &conversationReadReceiptpb.UpdateConversationReadReceiptResponse{
		Data:    []*conversationReadReceiptpb.ConversationReadReceipt{receipt},
		Success: true,
	}, nil
}

// DeleteConversationReadReceipt soft-deletes a receipt.
func (r *PostgresConversationReadReceiptRepository) DeleteConversationReadReceipt(ctx context.Context, req *conversationReadReceiptpb.DeleteConversationReadReceiptRequest) (*conversationReadReceiptpb.DeleteConversationReadReceiptResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("conversation_read_receipt ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete conversation_read_receipt: %w", err)
	}

	return &conversationReadReceiptpb.DeleteConversationReadReceiptResponse{
		Success: true,
	}, nil
}

// ListConversationReadReceipts lists receipts.
func (r *PostgresConversationReadReceiptRepository) ListConversationReadReceipts(ctx context.Context, req *conversationReadReceiptpb.ListConversationReadReceiptsRequest) (*conversationReadReceiptpb.ListConversationReadReceiptsResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation_read_receipts: %w", err)
	}

	var receipts []*conversationReadReceiptpb.ConversationReadReceipt
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}

		receipt := &conversationReadReceiptpb.ConversationReadReceipt{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, receipt); err != nil {
			continue
		}
		receipts = append(receipts, receipt)
	}

	return &conversationReadReceiptpb.ListConversationReadReceiptsResponse{
		Data:    receipts,
		Success: true,
	}, nil
}

// GetConversationReadReceiptListPageData composes over ListConversationReadReceipts.
func (r *PostgresConversationReadReceiptRepository) GetConversationReadReceiptListPageData(
	ctx context.Context,
	req *conversationReadReceiptpb.GetConversationReadReceiptListPageDataRequest,
) (*conversationReadReceiptpb.GetConversationReadReceiptListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation_read_receipt list page data request is required")
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

	listResp, err := r.ListConversationReadReceipts(ctx, &conversationReadReceiptpb.ListConversationReadReceiptsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversation_read_receipts for page data: %w", err)
	}
	receipts := listResp.GetData()

	totalItems := int32(len(receipts))
	totalPages := int32(1)
	if limit > 0 && totalItems == limit {
		totalPages = page + 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &conversationReadReceiptpb.GetConversationReadReceiptListPageDataResponse{
		ConversationReadReceiptList: receipts,
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

// GetConversationReadReceiptItemPageData composes over ReadConversationReadReceipt.
func (r *PostgresConversationReadReceiptRepository) GetConversationReadReceiptItemPageData(
	ctx context.Context,
	req *conversationReadReceiptpb.GetConversationReadReceiptItemPageDataRequest,
) (*conversationReadReceiptpb.GetConversationReadReceiptItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get conversation_read_receipt item page data request is required")
	}
	if req.ConversationReadReceiptId == "" {
		return nil, fmt.Errorf("conversation_read_receipt ID is required")
	}

	rr, err := r.ReadConversationReadReceipt(ctx, &conversationReadReceiptpb.ReadConversationReadReceiptRequest{Data: &conversationReadReceiptpb.ConversationReadReceipt{Id: req.ConversationReadReceiptId}})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("conversation_read_receipt with ID '%s' not found", req.ConversationReadReceiptId)
	}

	return &conversationReadReceiptpb.GetConversationReadReceiptItemPageDataResponse{
		ConversationReadReceipt: rr.GetData()[0],
		Success:                 true,
	}, nil
}

// NewConversationReadReceiptRepository creates a new repository (old-style constructor).
func NewConversationReadReceiptRepository(db *sql.DB, tableName string) conversationReadReceiptpb.ConversationReadReceiptDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresConversationReadReceiptRepository(dbOps, tableName)
}
