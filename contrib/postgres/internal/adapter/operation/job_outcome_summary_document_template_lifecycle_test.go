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

// ── Update freezes a published lineage ───────────────────────────────────────

func TestUpdateBinding_FreezesImmutableFieldsOnPublishedRow(t *testing.T) {
	// Current persisted row is PUBLISHED → immutable except active/date_modified.
	fake := &recordingDBOps{result: map[string]any{
		"id":             "b-1",
		"version_status": versionStatusPublished,
	}}
	r := NewPostgresJobOutcomeSummaryDocumentTemplateRepository(fake, "job_outcome_summary_document_template")

	_, err := r.UpdateJobOutcomeSummaryDocumentTemplate(context.Background(), &pb.UpdateJobOutcomeSummaryDocumentTemplateRequest{
		Data: &pb.JobOutcomeSummaryDocumentTemplate{
			Id:                 "b-1",
			DocumentTemplateId: "dt-HOSTILE",   // immutable — must be dropped
			PriceScheduleId:    strptr("ps-X"), // immutable — must be dropped
			Version:            42,             // immutable — must be dropped
			Active:             true,           // mutable — must survive (proto3 omits the false zero-value)
		},
	})
	if err != nil {
		t.Fatalf("UpdateJobOutcomeSummaryDocumentTemplate returned error: %v", err)
	}
	for _, frozen := range []string{"document_template_id", "price_schedule_id", "version", "version_status", "id"} {
		if _, ok := fake.lastUpdate[frozen]; ok {
			t.Errorf("published row: immutable field %q must not reach the write path (got %v)", frozen, fake.lastUpdate[frozen])
		}
	}
	if _, ok := fake.lastUpdate["active"]; !ok {
		t.Error("published row: mutable field active must survive the filter")
	}
}

func TestUpdateBinding_DraftRowStaysFullyMutable(t *testing.T) {
	// Current persisted row is a DRAFT → fully mutable.
	fake := &recordingDBOps{result: map[string]any{
		"id":             "b-1",
		"version_status": versionStatusDraft,
	}}
	r := NewPostgresJobOutcomeSummaryDocumentTemplateRepository(fake, "job_outcome_summary_document_template")

	_, err := r.UpdateJobOutcomeSummaryDocumentTemplate(context.Background(), &pb.UpdateJobOutcomeSummaryDocumentTemplateRequest{
		Data: &pb.JobOutcomeSummaryDocumentTemplate{
			Id:                 "b-1",
			DocumentTemplateId: "dt-2",
			Version:            7, // server-owned — must be stripped even on a draft
		},
	})
	if err != nil {
		t.Fatalf("UpdateJobOutcomeSummaryDocumentTemplate returned error: %v", err)
	}
	if got := fake.lastUpdate["document_template_id"]; got != "dt-2" {
		t.Errorf("draft row: document_template_id must remain mutable, got %v", got)
	}
	// version is server-owned (only Publish allocates it): stripped unconditionally
	// so a publish/Update TOCTOU can never clobber a published lineage's version
	// (B4 codex finding #3).
	if _, ok := fake.lastUpdate["version"]; ok {
		t.Errorf("draft row: server-owned version must be stripped from Update (got %v)", fake.lastUpdate["version"])
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
