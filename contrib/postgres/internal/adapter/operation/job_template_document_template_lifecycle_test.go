//go:build postgresql

package operation

import (
	"fmt"
	"strings"
	"testing"

	"context"

	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_document_template"
)

// RA2 P1 (server-owned publish lifecycle) regression tests for the sheet-family
// binding (JOSDT sibling, 20260720), all DB-free:
//   - Create forces DRAFT + unversioned/provisional + strips publish audit.
//   - Update unconditionally strips scope/lifecycle fields (incl. job_category_id).
//   - the resolver SQL carries its fail-closed predicate shape + the 4-tier
//     category-aware match_rank (T1).
//   - the publish flip SQL carries the draft-only guard.
//
// recordingDBOps (proto_map_h1_test.go) captures the data map handed to the
// write path and serves a configurable Read result. strptr / nextBindingVersion
// are shared package helpers.

// ── Create forces the DRAFT lifecycle ────────────────────────────────────────

func TestCreateJTDTBinding_ForcesDraftUnversionedStripsPublishAudit(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobTemplateDocumentTemplateRepository(fake, "job_template_document_template")

	// A caller supplies a hostile PUBLISHED status + a fake version + publish audit.
	_, err := r.CreateJobTemplateDocumentTemplate(context.Background(), &pb.CreateJobTemplateDocumentTemplateRequest{
		Data: &pb.JobTemplateDocumentTemplate{
			Id:                 "b-1",
			DocumentTemplateId: "dt-1",
			JobCategoryId:      strptr("cat-academic"),
			VersionStatus:      enums.VersionStatus_VERSION_STATUS_PUBLISHED,
			Version:            99,
			PublishedBy:        strptr("attacker"),
		},
	})
	if err != nil {
		t.Fatalf("CreateJobTemplateDocumentTemplate returned error: %v", err)
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
	// The sheet-shape axis is a legitimate create-time scope field and must survive.
	if got := fmt.Sprintf("%v", fake.lastCreate["job_category_id"]); got != "cat-academic" {
		t.Errorf("Create must persist the job_category_id scope, got %v", got)
	}
}

// TestCreateJTDTBinding_EmptyCategoryBecomesNull pins the empty-optional-FK
// normalization: an empty job_category_id from a form (any-shape fallback) must
// become SQL NULL so the FK constraint holds.
func TestCreateJTDTBinding_EmptyCategoryBecomesNull(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobTemplateDocumentTemplateRepository(fake, "job_template_document_template")

	_, err := r.CreateJobTemplateDocumentTemplate(context.Background(), &pb.CreateJobTemplateDocumentTemplateRequest{
		Data: &pb.JobTemplateDocumentTemplate{
			Id:                 "b-1",
			DocumentTemplateId: "dt-1",
			JobCategoryId:      strptr(""),
			PriceScheduleId:    strptr(""),
		},
	})
	if err != nil {
		t.Fatalf("CreateJobTemplateDocumentTemplate returned error: %v", err)
	}
	if got, ok := fake.lastCreate["job_category_id"]; !ok || got != nil {
		t.Errorf("empty job_category_id must normalize to SQL NULL, got %v (present=%v)", got, ok)
	}
	if got, ok := fake.lastCreate["price_schedule_id"]; !ok || got != nil {
		t.Errorf("empty price_schedule_id must normalize to SQL NULL, got %v (present=%v)", got, ok)
	}
}

// ── Update: unconditional scope/lifecycle strip (Q3) ─────────────────────────

// TestUpdateJTDTBinding_ScopeFieldsNeverReachWrite_EvenWhenRowReadsDraft pins the
// Q3 contract: the generic Update strips scope + lifecycle fields for ALL rows,
// WITHOUT reading the current lifecycle state. job_category_id is a scope field
// and must be dropped too.
func TestUpdateJTDTBinding_ScopeFieldsNeverReachWrite_EvenWhenRowReadsDraft(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{
		"id":             "b-1",
		"version_status": versionStatusDraft,
	}}
	r := NewPostgresJobTemplateDocumentTemplateRepository(fake, "job_template_document_template")

	ms := int64(1700000000000)
	_, err := r.UpdateJobTemplateDocumentTemplate(context.Background(), &pb.UpdateJobTemplateDocumentTemplateRequest{
		Data: &pb.JobTemplateDocumentTemplate{
			Id:                  "b-1",
			DocumentTemplateId:  "dt-HOSTILE",         // scope — must be dropped
			PriceScheduleId:     strptr("ps-X"),       // scope — must be dropped
			JobCategoryId:       strptr("cat-HOSTILE"), // scope — must be dropped
			SupersedesBindingId: strptr("b-0"),        // scope — must be dropped
			Version:             42,                   // server-owned — must be dropped
			PublishedBy:         strptr("attacker"),
			DateModified:        &ms, // audit stamp — must survive
		},
	})
	if err != nil {
		t.Fatalf("UpdateJobTemplateDocumentTemplate returned error: %v", err)
	}
	for _, frozen := range []string{
		"document_template_id", "price_schedule_id", "job_category_id", "supersedes_binding_id",
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

// TestUpdateJTDTBinding_ActiveTrueCannotResurrect pins the resurrection guard:
// active=true is dropped from the payload; deactivation stays the draft-only
// Delete transaction's job.
func TestUpdateJTDTBinding_ActiveTrueCannotResurrect(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobTemplateDocumentTemplateRepository(fake, "job_template_document_template")

	ms := int64(1700000000000)
	_, err := r.UpdateJobTemplateDocumentTemplate(context.Background(), &pb.UpdateJobTemplateDocumentTemplateRequest{
		Data: &pb.JobTemplateDocumentTemplate{
			Id:           "b-1",
			Active:       true, // resurrection attempt — must be dropped
			DateModified: &ms,
		},
	})
	if err != nil {
		t.Fatalf("UpdateJobTemplateDocumentTemplate returned error: %v", err)
	}
	if got, ok := fake.lastUpdate["active"]; ok {
		t.Errorf("active=true must not reach the write path (resurrection guard), got %v", got)
	}
}

// TestUpdateJTDTBinding_ScopeOnlyPayloadFailsClosed: a payload carrying ONLY
// immutable fields filters down to nothing — the adapter refuses it.
func TestUpdateJTDTBinding_ScopeOnlyPayloadFailsClosed(t *testing.T) {
	fake := &recordingDBOps{result: map[string]any{"id": "b-1"}}
	r := NewPostgresJobTemplateDocumentTemplateRepository(fake, "job_template_document_template")

	_, err := r.UpdateJobTemplateDocumentTemplate(context.Background(), &pb.UpdateJobTemplateDocumentTemplateRequest{
		Data: &pb.JobTemplateDocumentTemplate{
			Id:                 "b-1",
			DocumentTemplateId: "dt-HOSTILE",
			JobCategoryId:      strptr("cat-HOSTILE"),
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

// ── Resolver SQL shape + category axis (T1) ──────────────────────────────────

func TestFindApplicableJTDTSQL_HasFailClosedPredicateShape(t *testing.T) {
	q := findApplicableJobTemplateDocumentTemplateSQL()
	mustContain := map[string]string{
		"binding table (entityid)":  entityid_JobTemplateDocumentTemplate(),
		"document_template table":   "document_template",
		"price_schedule table":      "price_schedule",
		"job_category table":        "job_category",
		"tenant gate":               "b.workspace_id = $1",
		"published gate":            "b.version_status = $3",
		"active gate":               "b.active = true",
		"validity start half-open":  "b.validity_start <= $4",
		"validity end half-open":    "$4 < b.validity_end",
		"exact-before-fallback ord": "ORDER BY match_rank, b.version DESC",
		"ambiguity guard":           "LIMIT 2",
		"schedule scope match":      "b.price_schedule_id = rs.price_schedule_id OR b.price_schedule_id IS NULL",
		"category scope match":      "b.job_category_id = rs.job_category_id OR b.job_category_id IS NULL",
		"category param":            "NULLIF($5, '')",
	}
	for name, sub := range mustContain {
		if !strings.Contains(q, sub) {
			t.Errorf("resolver SQL missing %s: %q", name, sub)
		}
	}
}

// TestFindApplicableJTDTSQL_FourTierMatchRank pins the most-specific-wins rank
// ordering (category ≻ schedule, per Q2): the four rank branches are present so
// category-exact+schedule-fallback (rank 1) outranks category-fallback+
// schedule-exact (rank 2) via the ORDER BY match_rank asc.
func TestFindApplicableJTDTSQL_FourTierMatchRank(t *testing.T) {
	q := findApplicableJobTemplateDocumentTemplateSQL()
	// rank 0: category exact + schedule exact.
	if !strings.Contains(q, "THEN 0") {
		t.Error("resolver SQL missing rank-0 (category+schedule exact) branch")
	}
	// rank 1: category exact + schedule fallback.
	if !strings.Contains(q, "b.price_schedule_id IS NULL THEN 1") {
		t.Error("resolver SQL missing rank-1 (category exact + schedule fallback) branch")
	}
	// rank 2: category fallback + schedule exact.
	if !strings.Contains(q, "b.job_category_id IS NULL") || !strings.Contains(q, "THEN 2") {
		t.Error("resolver SQL missing rank-2 (category fallback + schedule exact) branch")
	}
	// rank 3: both fallback.
	if !strings.Contains(q, "ELSE 3") {
		t.Error("resolver SQL missing rank-3 (workspace-wide fallback) branch")
	}
}

// ── Publish flip SQL shape ───────────────────────────────────────────────────

func TestPublishFlipJTDTSQL_IsDraftOnlyGuarded(t *testing.T) {
	q := jobTemplateDocumentTemplatePublishFlipSQL()
	if !strings.Contains(q, "version_status = $7") {
		t.Errorf("publish flip must carry the draft-only predicate ($7), got: %q", q)
	}
	if !strings.Contains(q, "WHERE id = $5 AND workspace_id = $6") {
		t.Errorf("publish flip must be id + workspace scoped, got: %q", q)
	}
	if !strings.Contains(q, entityid_JobTemplateDocumentTemplate()) {
		t.Errorf("publish flip must target the binding table via entityid, got: %q", q)
	}
}

// entityid_JobTemplateDocumentTemplate returns the binding table name literal for
// substring assertions without importing entityid at the top.
func entityid_JobTemplateDocumentTemplate() string {
	return "job_template_document_template"
}
