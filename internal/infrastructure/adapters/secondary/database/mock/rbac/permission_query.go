//go:build mock_db

package rbac

import (
	"context"
	"sync"

	"github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockPermissionQuery implements security.PermissionQuery with an in-memory
// map keyed by userID:workspaceID. Intended for tests and dev-mode mock_db
// builds where no real RBAC tables exist.
//
// Tests can populate entries via SetCodes; unpopulated lookups return an
// empty slice (the PermissionLoader treats that as "no permissions").
type MockPermissionQuery struct {
	mu    sync.RWMutex
	codes map[string][]string
}

func NewMockPermissionQuery() *MockPermissionQuery {
	return &MockPermissionQuery{codes: make(map[string][]string)}
}

var _ security.PermissionQuery = (*MockPermissionQuery)(nil)

func (m *MockPermissionQuery) GetUserPermissionCodes(ctx context.Context, userID, workspaceID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	codes, ok := m.codes[cacheKey(userID, workspaceID)]
	if !ok {
		return []string{}, nil
	}
	out := make([]string, len(codes))
	copy(out, codes)
	return out, nil
}

// SetCodes seeds the mock with a specific permission set for a user/workspace.
// Safe to call concurrently.
func (m *MockPermissionQuery) SetCodes(userID, workspaceID string, codes []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dup := make([]string, len(codes))
	copy(dup, codes)
	m.codes[cacheKey(userID, workspaceID)] = dup
}

// Clear removes all seeded entries — convenient for test cleanup.
func (m *MockPermissionQuery) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes = make(map[string][]string)
}

func cacheKey(userID, workspaceID string) string {
	return userID + ":" + workspaceID
}

// Package-level singleton so tests can grab the same instance the auth adapter
// uses. Tests call SharedMock().SetCodes(...) to seed permissions.
var sharedMock = NewMockPermissionQuery()

// SharedMock returns the singleton MockPermissionQuery registered with the
// factory. Tests use this to seed permission data.
func SharedMock() *MockPermissionQuery {
	return sharedMock
}

func init() {
	registry.RegisterPermissionQueryFactory(func(db any) any {
		return sharedMock
	})
}
