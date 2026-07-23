package http

import (
	"context"
	"testing"
)

// countingQuery is a PermissionQuery that returns a fixed code set and counts
// how many times it is hit — so a test can tell a cache HIT (no call) from a
// MISS (a call), which is how it observes eviction.
type countingQuery struct{ calls int }

func (q *countingQuery) GetUserPermissionCodes(
	_ context.Context,
	_ string, _ string,
	_ PrincipalType,
	_ string,
	_ string, _ string,
) ([]string, error) {
	q.calls++
	return []string{"x:read"}, nil
}

// prime loads one cache entry and returns the query-call count delta (1 = miss).
func prime(t *testing.T, l *DBPermissionLoader, userID, bindingID string) {
	t.Helper()
	if _, err := l.GetUserPermissionCodes(context.Background(), userID, "ws-1", PrincipalTypeOperatorStaff, bindingID, "", ""); err != nil {
		t.Fatalf("prime %s/%s: %v", userID, bindingID, err)
	}
}

// isCached reports whether (userID, bindingID) is served from cache (no query
// call) by observing the query counter across one lookup.
func isCached(t *testing.T, l *DBPermissionLoader, q *countingQuery, userID, bindingID string) bool {
	t.Helper()
	before := q.calls
	if _, err := l.GetUserPermissionCodes(context.Background(), userID, "ws-1", PrincipalTypeOperatorStaff, bindingID, "", ""); err != nil {
		t.Fatalf("lookup %s/%s: %v", userID, bindingID, err)
	}
	return q.calls == before // no new call ⇒ served from cache
}

// TestDBPermissionLoader_InvalidateBinding_Scoped proves InvalidateBinding drops
// ONLY the targeted binding's entry — the same-workspace neighbour stays cached
// (P10/D2 audit-gate: no over-clear).
func TestDBPermissionLoader_InvalidateBinding_Scoped(t *testing.T) {
	q := &countingQuery{}
	l := NewDBPermissionLoader(q)
	prime(t, l, "user-A", "wu-A")
	prime(t, l, "user-B", "wu-B")

	l.InvalidateBinding("wu-A")

	if isCached(t, l, q, "user-A", "wu-A") {
		t.Fatalf("wu-A should have been evicted (expected a cache miss)")
	}
	if !isCached(t, l, q, "user-B", "wu-B") {
		t.Fatalf("wu-B must remain cached — InvalidateBinding over-cleared")
	}
}

// TestDBPermissionLoader_InvalidateUser_Scoped proves InvalidateUser drops only
// the targeted user's entries and leaves other users cached.
func TestDBPermissionLoader_InvalidateUser_Scoped(t *testing.T) {
	q := &countingQuery{}
	l := NewDBPermissionLoader(q)
	prime(t, l, "user-A", "wu-A")
	prime(t, l, "user-B", "wu-B")

	l.InvalidateUser("user-A")

	if isCached(t, l, q, "user-A", "wu-A") {
		t.Fatalf("user-A should have been evicted (expected a cache miss)")
	}
	if !isCached(t, l, q, "user-B", "wu-B") {
		t.Fatalf("user-B must remain cached — InvalidateUser over-cleared")
	}
}

// TestDBPermissionLoader_InvalidateEmptyNoop proves empty-id calls never flush
// the cache.
func TestDBPermissionLoader_InvalidateEmptyNoop(t *testing.T) {
	q := &countingQuery{}
	l := NewDBPermissionLoader(q)
	prime(t, l, "user-A", "wu-A")

	l.InvalidateBinding("")
	l.InvalidateUser("")

	if !isCached(t, l, q, "user-A", "wu-A") {
		t.Fatalf("empty-id invalidation must be a no-op, but the entry was dropped")
	}
}
