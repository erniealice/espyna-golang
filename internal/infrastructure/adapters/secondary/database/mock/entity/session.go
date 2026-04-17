//go:build mock_db

package entity

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
)

// MockSessionRepository implements sessionpb.SessionDomainServiceServer using an in-memory store.
type MockSessionRepository struct {
	sessionpb.UnimplementedSessionDomainServiceServer
	sessions    map[string]*sessionpb.Session // Keyed by session id
	mu          sync.RWMutex
	initialized bool
	processor   *listdata.ListDataProcessor
}

// NewMockSessionRepository creates a new in-memory session repository.
func NewMockSessionRepository() sessionpb.SessionDomainServiceServer {
	repo := &MockSessionRepository{
		sessions:  make(map[string]*sessionpb.Session),
		processor: listdata.NewListDataProcessor(),
	}
	return repo
}

// CreateSession stores a new session in the in-memory map.
func (r *MockSessionRepository) CreateSession(ctx context.Context, req *sessionpb.CreateSessionRequest) (*sessionpb.CreateSessionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create session request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("session data is required")
	}
	if req.Data.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Data.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	sessionID := fmt.Sprintf("session-%d-%d", now.UnixNano(), len(r.sessions))

	nowMs := now.UnixMilli()
	nowStr := now.Format(time.RFC3339)

	newSession := &sessionpb.Session{
		Id:                 sessionID,
		UserId:             req.Data.UserId,
		Token:              req.Data.Token,
		WorkspaceUserId:    req.Data.WorkspaceUserId,
		WorkspaceId:        req.Data.WorkspaceId,
		ExpiresAt:          req.Data.ExpiresAt,
		Active:             true,
		DateCreated:        &nowMs,
		DateCreatedString:  &nowStr,
		DateModified:       &nowMs,
		DateModifiedString: &nowStr,
	}

	r.sessions[sessionID] = newSession

	return &sessionpb.CreateSessionResponse{
		Data:    []*sessionpb.Session{newSession},
		Success: true,
	}, nil
}

// ReadSession retrieves a session by id.
func (r *MockSessionRepository) ReadSession(ctx context.Context, req *sessionpb.ReadSessionRequest) (*sessionpb.ReadSessionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read session request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session id is required")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("session with id '%s' not found", req.Data.Id)
	}

	return &sessionpb.ReadSessionResponse{
		Data:    []*sessionpb.Session{session},
		Success: true,
	}, nil
}

// UpdateSession updates an existing session's mutable fields.
func (r *MockSessionRepository) UpdateSession(ctx context.Context, req *sessionpb.UpdateSessionRequest) (*sessionpb.UpdateSessionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update session request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session id is required for update")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.sessions[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("session with id '%s' not found", req.Data.Id)
	}

	now := time.Now()
	nowMs := now.UnixMilli()
	nowStr := now.Format(time.RFC3339)

	updated := &sessionpb.Session{
		Id:                 req.Data.Id,
		UserId:             req.Data.UserId,
		Token:              req.Data.Token,
		WorkspaceUserId:    req.Data.WorkspaceUserId,
		WorkspaceId:        req.Data.WorkspaceId,
		ExpiresAt:          req.Data.ExpiresAt,
		Active:             req.Data.Active,
		DateCreated:        existing.DateCreated,       // Preserve original
		DateCreatedString:  existing.DateCreatedString, // Preserve original
		DateModified:       &nowMs,
		DateModifiedString: &nowStr,
	}

	r.sessions[req.Data.Id] = updated

	return &sessionpb.UpdateSessionResponse{
		Data:    []*sessionpb.Session{updated},
		Success: true,
	}, nil
}

// DeleteSession removes a session from the in-memory store.
func (r *MockSessionRepository) DeleteSession(ctx context.Context, req *sessionpb.DeleteSessionRequest) (*sessionpb.DeleteSessionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete session request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("session id is required for deletion")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[req.Data.Id]; !exists {
		return nil, fmt.Errorf("session with id '%s' not found", req.Data.Id)
	}

	delete(r.sessions, req.Data.Id)

	return &sessionpb.DeleteSessionResponse{
		Success: true,
	}, nil
}

// ListSessions returns all sessions with optional filter/sort/pagination via ListDataProcessor.
func (r *MockSessionRepository) ListSessions(ctx context.Context, req *sessionpb.ListSessionsRequest) (*sessionpb.ListSessionsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*sessionpb.Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		items = append(items, s)
	}

	result, err := r.processor.ProcessListRequest(
		items,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process session list: %w", err)
	}

	processed := make([]*sessionpb.Session, len(result.Items))
	for i, item := range result.Items {
		if typed, ok := item.(*sessionpb.Session); ok {
			processed[i] = typed
		}
	}

	return &sessionpb.ListSessionsResponse{
		Data:    processed,
		Success: true,
	}, nil
}

func init() {
	registry.RegisterRepositoryFactory("mock", entityid.Session, func(conn any, tableName string) (any, error) {
		return NewMockSessionRepository(), nil
	})
}
