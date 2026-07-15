//go:build postgresql

package operation

import (
	"context"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
)

// Gate H1 regression tests: a client-supplied camelCase workspaceId must never
// survive a Create (the trusted canonical workspace wins) nor mutate a row on
// Update, and two source keys that canonicalize to the same column must fail
// loudly instead of nondeterministically overwriting one another.

func strptr(s string) *string { return &s }

// ── proto_map canonicalization + collision guard ─────────────────────────────

func TestCanonicalizeColumnKeys_SnakeCasesTopLevelKeys(t *testing.T) {
	out, err := canonicalizeColumnKeys(map[string]any{
		"workspaceId":        "ws-1",
		"documentTemplateId": "dt-1",
		"name":               "Academic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, col := range []string{"workspace_id", "document_template_id", "name"} {
		if _, ok := out[col]; !ok {
			t.Errorf("expected canonical key %q in output, got %v", col, out)
		}
	}
	if _, ok := out["workspaceId"]; ok {
		t.Error("camelCase workspaceId leaked into canonical output")
	}
}

func TestCanonicalizeColumnKeys_CollisionErrors(t *testing.T) {
	// A camelCase key and its snake_case twin both canonicalize to workspace_id.
	_, err := canonicalizeColumnKeys(map[string]any{
		"workspaceId":  "attacker-ws",
		"workspace_id": "trusted-ws",
	})
	if err == nil {
		t.Fatal("expected a collision error for workspaceId + workspace_id, got nil")
	}
}

func TestProtoToMap_EmitsCanonicalWorkspaceKey(t *testing.T) {
	data, err := protoToMap(&pb.JobCategory{Id: "jc-1", WorkspaceId: strptr("attacker-ws")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := data["workspace_id"]; got != "attacker-ws" {
		t.Errorf("expected canonical workspace_id key, got %v (full map %v)", got, data)
	}
	if _, ok := data["workspaceId"]; ok {
		t.Error("protoToMap leaked camelCase workspaceId")
	}
}

// ── adapter Create/Update strip the client workspace key ─────────────────────

// recordingDBOps captures the data map handed to the write path so tests can
// assert the adapter stripped the client workspace key before delegating.
type recordingDBOps struct {
	lastCreate map[string]any
	lastUpdate map[string]any
	result     map[string]any
}

func (f *recordingDBOps) Create(_ context.Context, _ string, data map[string]any) (map[string]any, error) {
	f.lastCreate = data
	if f.result != nil {
		return f.result, nil
	}
	return data, nil
}

func (f *recordingDBOps) Update(_ context.Context, _ string, _ string, data map[string]any) (map[string]any, error) {
	f.lastUpdate = data
	if f.result != nil {
		return f.result, nil
	}
	return data, nil
}

func (f *recordingDBOps) Read(_ context.Context, _ string, _ string) (map[string]any, error) {
	return f.result, nil
}
func (f *recordingDBOps) Delete(_ context.Context, _ string, _ string) error     { return nil }
func (f *recordingDBOps) HardDelete(_ context.Context, _ string, _ string) error { return nil }
func (f *recordingDBOps) List(_ context.Context, _ string, _ *interfaces.ListParams) (*interfaces.ListResult, error) {
	return &interfaces.ListResult{}, nil
}
func (f *recordingDBOps) Query(_ context.Context, _ string, _ interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, nil
}
func (f *recordingDBOps) QueryOne(_ context.Context, _ string, _ interfaces.QueryBuilder) (map[string]any, error) {
	return nil, nil
}

func TestCreateJobCategory_StripsClientWorkspaceKey(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "jc-1"}}
	r := NewPostgresJobCategoryRepository(fake, "job_category")

	_, err := r.CreateJobCategory(context.Background(), &pb.CreateJobCategoryRequest{
		Data: &pb.JobCategory{Id: "jc-1", WorkspaceId: strptr("attacker-ws")},
	})
	if err != nil {
		t.Fatalf("CreateJobCategory returned error: %v", err)
	}
	if _, ok := fake.lastCreate["workspace_id"]; ok {
		t.Error("client workspace_id survived CreateJobCategory")
	}
	if _, ok := fake.lastCreate["workspaceId"]; ok {
		t.Error("client workspaceId survived CreateJobCategory")
	}
}

func TestUpdateJobCategory_StripsClientWorkspaceKey(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "jc-1"}}
	r := NewPostgresJobCategoryRepository(fake, "job_category")

	_, err := r.UpdateJobCategory(context.Background(), &pb.UpdateJobCategoryRequest{
		Data: &pb.JobCategory{Id: "jc-1", WorkspaceId: strptr("attacker-ws")},
	})
	if err != nil {
		t.Fatalf("UpdateJobCategory returned error: %v", err)
	}
	if _, ok := fake.lastUpdate["workspace_id"]; ok {
		t.Error("client workspace_id survived UpdateJobCategory (would mutate tenant anchor)")
	}
	if _, ok := fake.lastUpdate["workspaceId"]; ok {
		t.Error("client workspaceId survived UpdateJobCategory")
	}
}
