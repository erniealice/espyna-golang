//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

// RA2 P1 (server-owned publish lifecycle) regression tests, all DB-free:
//   - Create forces DRAFT + unversioned/provisional + strips publish audit.
//   - Update freezes immutable fields once a row is PUBLISHED/DEPRECATED.
//   - the resolver SQL carries its fail-closed predicate shape (T1).
//   - the publish flip SQL carries the draft-only guard + version allocation.
//
// recordingDBOps (proto_map_h1_test.go) captures the data map handed to the
// write path and serves a configurable Read result.

// ── Create forces the DRAFT lifecycle ────────────────────────────────────────

func TestCreateBinding_ForcesDraftUnversionedStripsPublishAudit(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobOutcomeSummaryDocumentTemplateRepository(fake, "job_outcome_summary_document_template")

	// A caller supplies a hostile PUBLISHED status + a fake version + publish audit.
	_, err := r.CreateJobOutcomeSummaryDocumentTemplate(context.Background(), &pb.CreateJobOutcomeSummaryDocumentTemplateRequest{
		Data: &pb.JobOutcomeSummaryDocumentTemplate{
			Id:                 "b-1",
			DocumentTemplateId: "dt-1",
			VersionStatus:      enums.VersionStatus_VERSION_STATUS_PUBLISHED,
			Version:            99,
			PublishedBy:        strptr("attacker"),
		},
	})
	if err != nil {
		t.Fatalf("CreateJobOutcomeSummaryDocumentTemplate returned error: %v", err)
	}
	if got := fake.lastCreate["version_status"]; got != versionStatusDraft {
		t.Errorf("Create must force version_status=DRAFT, got %v", got)
	}
	if got := fmt.Sprintf("%v", fake.lastCreate["version"]); got != "0" {
		t.Errorf("Create must force provisional version=0, got %v", got)
	}
	if _, ok := fake.lastCreate["published_by"]; ok {
		t.Error("Create must strip client-supplied published_by")
	}
	if _, ok := fake.lastCreate["published_at"]; ok {
		t.Error("Create must strip client-supplied published_at")
	}
}

// ── Update: unconditional scope/lifecycle strip (Q3) ─────────────────────────

// TestUpdateBinding_ScopeFieldsNeverReachWrite_EvenWhenRowReadsDraft pins the
// Q3 contract: the generic Update strips scope + lifecycle fields for ALL rows,
// WITHOUT reading the current lifecycle state. The fake's Read reports DRAFT —
// the exact snapshot a publish/Update TOCTOU would present — and the scope
// fields still never reach the write path, so the race window is structurally
// closed (no read-check-write to interleave with).
func TestUpdateBinding_ScopeFieldsNeverReachWrite_EvenWhenRowReadsDraft(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{
		"id":             "b-1",
		"version_status": versionStatusDraft,
	}}
	r := NewPostgresJobOutcomeSummaryDocumentTemplateRepository(fake, "job_outcome_summary_document_template")

	ms := int64(1700000000000)
	_, err := r.UpdateJobOutcomeSummaryDocumentTemplate(context.Background(), &pb.UpdateJobOutcomeSummaryDocumentTemplateRequest{
		Data: &pb.JobOutcomeSummaryDocumentTemplate{
			Id:                  "b-1",
			DocumentTemplateId:  "dt-HOSTILE",   // scope — must be dropped
			PriceScheduleId:     strptr("ps-X"), // scope — must be dropped
			SupersedesBindingId: strptr("b-0"),  // scope — must be dropped
			Version:             42,             // server-owned — must be dropped
			PublishedBy:         strptr("attacker"),
			DateModified:        &ms, // audit stamp — must survive
		},
	})
	if err != nil {
		t.Fatalf("UpdateJobOutcomeSummaryDocumentTemplate returned error: %v", err)
	}
	for _, frozen := range []string{
		"document_template_id", "price_schedule_id", "supersedes_binding_id",
		"validity_start", "validity_end", "version", "version_status",
		"published_at", "published_by", "workspace_id", "id",
	} {
		if _, ok := fake.lastUpdate[frozen]; ok {
			t.Errorf("scope/lifecycle field %q must not reach the write path even for a DRAFT row (got %v)", frozen, fake.lastUpdate[frozen])
		}
	}
	if _, ok := fake.lastUpdate["date_modified"]; !ok {
		t.Error("audit stamp date_modified must survive the unconditional filter")
	}
}

// TestUpdateBinding_ActiveTrueCannotResurrect pins the resurrection guard:
// protojson omits the false zero-value, so the generic route could only ever
// SET active — i.e. restore a soft-deleted draft through the public update
// surface. active=true is therefore dropped from the payload; deactivation
// stays the draft-only Delete transaction's job.
func TestUpdateBinding_ActiveTrueCannotResurrect(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobOutcomeSummaryDocumentTemplateRepository(fake, "job_outcome_summary_document_template")

	ms := int64(1700000000000)
	_, err := r.UpdateJobOutcomeSummaryDocumentTemplate(context.Background(), &pb.UpdateJobOutcomeSummaryDocumentTemplateRequest{
		Data: &pb.JobOutcomeSummaryDocumentTemplate{
			Id:           "b-1",
			Active:       true, // resurrection attempt — must be dropped
			DateModified: &ms,
		},
	})
	if err != nil {
		t.Fatalf("UpdateJobOutcomeSummaryDocumentTemplate returned error: %v", err)
	}
	if got, ok := fake.lastUpdate["active"]; ok {
		t.Errorf("active=true must not reach the write path (resurrection guard), got %v", got)
	}
}

// TestUpdateBinding_ScopeOnlyPayloadFailsClosed: a payload carrying ONLY
// immutable fields filters down to nothing — the adapter refuses it rather
// than issuing a malformed empty UPDATE.
func TestUpdateBinding_ScopeOnlyPayloadFailsClosed(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobOutcomeSummaryDocumentTemplateRepository(fake, "job_outcome_summary_document_template")

	_, err := r.UpdateJobOutcomeSummaryDocumentTemplate(context.Background(), &pb.UpdateJobOutcomeSummaryDocumentTemplateRequest{
		Data: &pb.JobOutcomeSummaryDocumentTemplate{
			Id:                 "b-1",
			DocumentTemplateId: "dt-HOSTILE",
			Version:            9,
		},
	})
	if err == nil {
		t.Fatal("a scope-only update payload must fail closed (no generically mutable fields)")
	}
	if fake.lastUpdate != nil {
		t.Errorf("no write must be issued for a scope-only payload, got %v", fake.lastUpdate)
	}
}

// ── Resolver SQL shape (T1) ──────────────────────────────────────────────────

func TestFindApplicableSQL_HasFailClosedPredicateShape(t *testing.T) {
	q := findApplicableSQL()
	mustContain := map[string]string{
		"binding table (entityid)":  entityid_JobOutcomeSummaryDocumentTemplate(),
		"document_template table":   "document_template",
		"price_schedule table":      "price_schedule",
		"tenant gate":               "b.workspace_id = $1",
		"published gate":            "b.version_status = $3",
		"active gate":               "b.active = true",
		"validity start half-open":  "b.validity_start <= $4",
		"validity end half-open":    "$4 < b.validity_end",
		"exact-before-fallback ord": "ORDER BY match_rank, b.version DESC",
		"ambiguity guard":           "LIMIT 2",
		"scope match":               "b.price_schedule_id = rs.price_schedule_id OR b.price_schedule_id IS NULL",
	}
	for name, sub := range mustContain {
		if !strings.Contains(q, sub) {
			t.Errorf("resolver SQL missing %s: %q", name, sub)
		}
	}
}

// ── Publish flip SQL shape + version allocation ──────────────────────────────

func TestPublishFlipSQL_IsDraftOnlyGuarded(t *testing.T) {
	q := publishFlipSQL()
	if !strings.Contains(q, "version_status = $7") {
		t.Errorf("publish flip must carry the draft-only predicate ($7), got: %q", q)
	}
	if !strings.Contains(q, "WHERE id = $5 AND workspace_id = $6") {
		t.Errorf("publish flip must be id + workspace scoped, got: %q", q)
	}
	if !strings.Contains(q, entityid_JobOutcomeSummaryDocumentTemplate()) {
		t.Errorf("publish flip must target the binding table via entityid, got: %q", q)
	}
}

func TestNextBindingVersion_AllocatesFromPublishedMax(t *testing.T) {
	cases := []struct {
		name string
		max  sql.NullInt32
		want int32
	}{
		{"no prior published sibling → 1", sql.NullInt32{}, 1},
		{"max published 1 → 2", sql.NullInt32{Int32: 1, Valid: true}, 2},
		{"max published 3 → 4", sql.NullInt32{Int32: 3, Valid: true}, 4},
	}
	for _, c := range cases {
		if got := nextBindingVersion(c.max); got != c.want {
			t.Errorf("%s: nextBindingVersion = %d, want %d", c.name, got, c.want)
		}
	}
}

// entityid_JobOutcomeSummaryDocumentTemplate returns the binding table name
// literal for substring assertions without importing entityid at the top (the
// adapter already imports it; this keeps the test's intent local).
func entityid_JobOutcomeSummaryDocumentTemplate() string {
	return "job_outcome_summary_document_template"
}
