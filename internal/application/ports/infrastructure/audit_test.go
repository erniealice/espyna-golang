package infrastructure

import (
	"context"
	"testing"
)

// mockAuditService captures LogEntry calls for assertion.
type mockAuditService struct {
	calls []*AuditLogRequest
}

func (m *mockAuditService) LogEntry(_ context.Context, req *AuditLogRequest) error {
	m.calls = append(m.calls, req)
	return nil
}

func (m *mockAuditService) ListByEntity(_ context.Context, _ *ListAuditRequest) (*ListAuditResponse, error) {
	return &ListAuditResponse{}, nil
}

// fieldMap converts a slice of AuditFieldChange to a map keyed by FieldName
// so tests can look up changes by field name without caring about slice order.
func fieldMap(changes []AuditFieldChange) map[string]AuditFieldChange {
	m := make(map[string]AuditFieldChange, len(changes))
	for _, c := range changes {
		m[c.FieldName] = c
	}
	return m
}

// TestDiffAndLog_NilService verifies that a nil AuditService returns nil without panicking.
func TestDiffAndLog_NilService(t *testing.T) {
	err := DiffAndLog(context.Background(), nil, DiffAndLogRequest{
		Action:  2,
		OldData: map[string]any{"name": "Alice"},
		NewData: map[string]any{"name": "Bob"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

// TestDiffAndLog_Insert verifies Action=1 (INSERT) produces field changes with empty OldValue.
func TestDiffAndLog_Insert(t *testing.T) {
	svc := &mockAuditService{}
	err := DiffAndLog(context.Background(), svc, DiffAndLogRequest{
		WorkspaceID: "ws1",
		EntityType:  "client",
		EntityID:    "c1",
		Action:      1,
		NewData: map[string]any{
			"name":   "Alice",
			"status": "active",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.calls) != 1 {
		t.Fatalf("expected 1 LogEntry call, got %d", len(svc.calls))
	}

	req := svc.calls[0]
	if len(req.FieldChanges) != 2 {
		t.Fatalf("expected 2 field changes, got %d", len(req.FieldChanges))
	}

	fm := fieldMap(req.FieldChanges)

	name, ok := fm["name"]
	if !ok {
		t.Fatal("expected field 'name' in changes")
	}
	if name.OldValue != "" {
		t.Errorf("INSERT 'name' OldValue: want \"\", got %q", name.OldValue)
	}
	if name.NewValue != "Alice" {
		t.Errorf("INSERT 'name' NewValue: want \"Alice\", got %q", name.NewValue)
	}

	status, ok := fm["status"]
	if !ok {
		t.Fatal("expected field 'status' in changes")
	}
	if status.OldValue != "" {
		t.Errorf("INSERT 'status' OldValue: want \"\", got %q", status.OldValue)
	}
	if status.NewValue != "active" {
		t.Errorf("INSERT 'status' NewValue: want \"active\", got %q", status.NewValue)
	}
}

// TestDiffAndLog_Update_OnlyChangedFields verifies Action=2 (UPDATE) only logs fields that changed.
func TestDiffAndLog_Update_OnlyChangedFields(t *testing.T) {
	svc := &mockAuditService{}
	err := DiffAndLog(context.Background(), svc, DiffAndLogRequest{
		Action: 2,
		OldData: map[string]any{
			"name":   "Alice",
			"status": "active",
		},
		NewData: map[string]any{
			"name":   "Alice",   // unchanged
			"status": "suspended", // changed
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := svc.calls[0]
	fm := fieldMap(req.FieldChanges)

	if _, ok := fm["name"]; ok {
		t.Error("unchanged field 'name' should not appear in changes")
	}

	status, ok := fm["status"]
	if !ok {
		t.Fatal("changed field 'status' must appear in changes")
	}
	if status.OldValue != "active" {
		t.Errorf("status OldValue: want \"active\", got %q", status.OldValue)
	}
	if status.NewValue != "suspended" {
		t.Errorf("status NewValue: want \"suspended\", got %q", status.NewValue)
	}
	if len(req.FieldChanges) != 1 {
		t.Errorf("expected exactly 1 change, got %d", len(req.FieldChanges))
	}
}

// TestDiffAndLog_Update_CapturesRemovedFields verifies that a field present in OldData
// but absent in NewData is captured with an empty NewValue.
func TestDiffAndLog_Update_CapturesRemovedFields(t *testing.T) {
	svc := &mockAuditService{}
	err := DiffAndLog(context.Background(), svc, DiffAndLogRequest{
		Action: 2,
		OldData: map[string]any{
			"name":  "Alice",
			"notes": "VIP",
		},
		NewData: map[string]any{
			"name": "Alice", // notes removed
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := svc.calls[0]
	fm := fieldMap(req.FieldChanges)

	notes, ok := fm["notes"]
	if !ok {
		t.Fatal("removed field 'notes' must appear in changes")
	}
	if notes.OldValue != "VIP" {
		t.Errorf("notes OldValue: want \"VIP\", got %q", notes.OldValue)
	}
	if notes.NewValue != "" {
		t.Errorf("notes NewValue: want \"\", got %q", notes.NewValue)
	}
}

// TestDiffAndLog_ExcludedFields verifies that ExcludedFields are filtered out.
func TestDiffAndLog_ExcludedFields(t *testing.T) {
	svc := &mockAuditService{}
	err := DiffAndLog(context.Background(), svc, DiffAndLogRequest{
		Action: 1,
		NewData: map[string]any{
			"username":      "alice",
			"password_hash": "secret-hash",
			"api_key":       "sk-1234",
		},
		ExcludedFields: map[string]bool{
			"password_hash": true,
			"api_key":       true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := svc.calls[0]
	fm := fieldMap(req.FieldChanges)

	if _, ok := fm["password_hash"]; ok {
		t.Error("'password_hash' should be excluded from changes")
	}
	if _, ok := fm["api_key"]; ok {
		t.Error("'api_key' should be excluded from changes")
	}
	if _, ok := fm["username"]; !ok {
		t.Error("non-excluded field 'username' must appear in changes")
	}
	if len(req.FieldChanges) != 1 {
		t.Errorf("expected 1 change after exclusions, got %d", len(req.FieldChanges))
	}
}

// TestDiffAndLog_Delete verifies Action=3 (DELETE) produces changes with empty NewValue.
func TestDiffAndLog_Delete(t *testing.T) {
	svc := &mockAuditService{}
	err := DiffAndLog(context.Background(), svc, DiffAndLogRequest{
		Action: 3,
		OldData: map[string]any{
			"name":   "Alice",
			"status": "active",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := svc.calls[0]
	if len(req.FieldChanges) != 2 {
		t.Fatalf("expected 2 field changes for DELETE, got %d", len(req.FieldChanges))
	}

	fm := fieldMap(req.FieldChanges)

	name, ok := fm["name"]
	if !ok {
		t.Fatal("expected field 'name' in changes")
	}
	if name.OldValue != "Alice" {
		t.Errorf("DELETE 'name' OldValue: want \"Alice\", got %q", name.OldValue)
	}
	if name.NewValue != "" {
		t.Errorf("DELETE 'name' NewValue: want \"\", got %q", name.NewValue)
	}

	status, ok := fm["status"]
	if !ok {
		t.Fatal("expected field 'status' in changes")
	}
	if status.OldValue != "active" {
		t.Errorf("DELETE 'status' OldValue: want \"active\", got %q", status.OldValue)
	}
	if status.NewValue != "" {
		t.Errorf("DELETE 'status' NewValue: want \"\", got %q", status.NewValue)
	}
}

// TestDiffAndLog_BoolSerialization verifies booleans serialize to "true" / "false".
func TestDiffAndLog_BoolSerialization(t *testing.T) {
	svc := &mockAuditService{}
	err := DiffAndLog(context.Background(), svc, DiffAndLogRequest{
		Action: 2,
		OldData: map[string]any{
			"verified": false,
		},
		NewData: map[string]any{
			"verified": true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := svc.calls[0]
	fm := fieldMap(req.FieldChanges)

	v, ok := fm["verified"]
	if !ok {
		t.Fatal("expected field 'verified' in changes")
	}
	if v.OldValue != "false" {
		t.Errorf("bool false OldValue: want \"false\", got %q", v.OldValue)
	}
	if v.NewValue != "true" {
		t.Errorf("bool true NewValue: want \"true\", got %q", v.NewValue)
	}
}

// TestDiffAndLog_NilSerialization verifies nil values serialize to "".
func TestDiffAndLog_NilSerialization(t *testing.T) {
	svc := &mockAuditService{}
	err := DiffAndLog(context.Background(), svc, DiffAndLogRequest{
		Action: 2,
		OldData: map[string]any{
			"bio": "some text",
		},
		NewData: map[string]any{
			"bio": nil,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := svc.calls[0]
	fm := fieldMap(req.FieldChanges)

	bio, ok := fm["bio"]
	if !ok {
		t.Fatal("expected field 'bio' in changes")
	}
	if bio.OldValue != "some text" {
		t.Errorf("bio OldValue: want \"some text\", got %q", bio.OldValue)
	}
	if bio.NewValue != "" {
		t.Errorf("nil NewValue: want \"\", got %q", bio.NewValue)
	}
}
