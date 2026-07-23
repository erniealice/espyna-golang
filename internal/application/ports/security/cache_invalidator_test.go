package security

import "testing"

// recordingInvalidator records the ids passed to each invalidation method.
type recordingInvalidator struct {
	users    []string
	bindings []string
}

func (r *recordingInvalidator) InvalidateUser(userID string) { r.users = append(r.users, userID) }
func (r *recordingInvalidator) InvalidateBinding(bindingID string) {
	r.bindings = append(r.bindings, bindingID)
}

// TestNoOpPermissionCacheInvalidator verifies the no-op default never panics and
// records nothing (it has no state) — it is safe wherever no cache is wired.
func TestNoOpPermissionCacheInvalidator(t *testing.T) {
	var inv PermissionCacheInvalidator = NewNoOpPermissionCacheInvalidator()
	// Must not panic for any input.
	inv.InvalidateUser("user-1")
	inv.InvalidateUser("")
	inv.InvalidateBinding("wu-1")
	inv.InvalidateBinding("")
}

// TestSettablePermissionCacheInvalidator_NoOpUntilSet verifies the holder is a
// silent no-op before SetDelegate — the pre-wired boot state — so use cases
// constructed before the loader exists do not fail.
func TestSettablePermissionCacheInvalidator_NoOpUntilSet(t *testing.T) {
	holder := NewSettablePermissionCacheInvalidator()
	// No delegate set: these must be no-ops (no panic, nothing forwarded).
	holder.InvalidateUser("user-1")
	holder.InvalidateBinding("wu-1")
}

// TestSettablePermissionCacheInvalidator_ForwardsAfterSet verifies that once the
// delegate is set, both grains forward to it verbatim — the wiring that makes a
// grant change reach the live cache (P10/D2).
func TestSettablePermissionCacheInvalidator_ForwardsAfterSet(t *testing.T) {
	holder := NewSettablePermissionCacheInvalidator()
	rec := &recordingInvalidator{}
	holder.SetDelegate(rec)

	holder.InvalidateUser("user-1")
	holder.InvalidateUser("user-2")
	holder.InvalidateBinding("wu-9")

	if len(rec.users) != 2 || rec.users[0] != "user-1" || rec.users[1] != "user-2" {
		t.Fatalf("InvalidateUser not forwarded verbatim: got %v", rec.users)
	}
	if len(rec.bindings) != 1 || rec.bindings[0] != "wu-9" {
		t.Fatalf("InvalidateBinding not forwarded verbatim: got %v", rec.bindings)
	}
}

// TestSettablePermissionCacheInvalidator_ClearDelegate verifies SetDelegate(nil)
// reverts the holder to the no-op state (no forwarding, no panic).
func TestSettablePermissionCacheInvalidator_ClearDelegate(t *testing.T) {
	holder := NewSettablePermissionCacheInvalidator()
	rec := &recordingInvalidator{}
	holder.SetDelegate(rec)
	holder.SetDelegate(nil)

	holder.InvalidateUser("user-1")
	holder.InvalidateBinding("wu-1")

	if len(rec.users) != 0 || len(rec.bindings) != 0 {
		t.Fatalf("delegate still forwarding after clear: users=%v bindings=%v", rec.users, rec.bindings)
	}
}
