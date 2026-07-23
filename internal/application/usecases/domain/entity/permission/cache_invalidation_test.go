package permission

import "testing"

type recorder struct {
	users    []string
	bindings []string
}

func (r *recorder) InvalidateUser(id string)    { r.users = append(r.users, id) }
func (r *recorder) InvalidateBinding(id string) { r.bindings = append(r.bindings, id) }

// TestCreateInvalidateProvenanceCache verifies a permission-definition create
// evicts the provenance user's cached codes (P10/D2), and unwired / empty inputs
// are silent no-ops.
func TestCreateInvalidateProvenanceCache(t *testing.T) {
	rec := &recorder{}
	uc := &CreatePermissionUseCase{services: CreatePermissionServices{PermissionCacheInvalidator: rec}}
	uc.invalidateProvenanceCache("user-admin")
	uc.invalidateProvenanceCache("") // no-op

	if len(rec.users) != 1 || rec.users[0] != "user-admin" {
		t.Fatalf("expected only user-admin evicted, got %v", rec.users)
	}
	if len(rec.bindings) != 0 {
		t.Fatalf("permission provenance eviction must not touch bindings, got %v", rec.bindings)
	}

	// nil invalidator: no panic.
	(&CreatePermissionUseCase{}).invalidateProvenanceCache("user-admin")
}

// TestUpdateInvalidateProvenanceCache mirrors the create path for updates.
func TestUpdateInvalidateProvenanceCache(t *testing.T) {
	rec := &recorder{}
	uc := &UpdatePermissionUseCase{services: UpdatePermissionServices{PermissionCacheInvalidator: rec}}
	uc.invalidateProvenanceCache("user-admin")
	if len(rec.users) != 1 || rec.users[0] != "user-admin" {
		t.Fatalf("expected only user-admin evicted, got %v", rec.users)
	}
}
