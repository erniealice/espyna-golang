//go:build postgresql

package operation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// DB-free unit coverage for the two writeLifecycleAudit methods added for
// codex-review-impl2.out.md finding 9 (schema-proposal.md §9.4): Publish /
// Deprecate (rating_description_set.go) and Relink / Unlink
// (rating_description_set_product_plan.go) each write a durable semantic
// audit event inside the caller's ambient transaction, mandatory (fail
// closed on a nil audit dependency or missing trusted actor), with the
// actor derived from the trusted identity rather than any middleware-
// supplied audit context. These tests exercise writeLifecycleAudit directly
// (no *sql.DB / *sql.Tx required — LogEntry is the only dependency), mirroring
// how PostgresJobPhaseRepository.writeTransitionAudit is unit-tested.

// recordingAuditService captures every LogEntry call; optionally fails.
type recordingAuditService struct {
	calls    []*infraports.AuditLogRequest
	failWith error
}

func (r *recordingAuditService) LogEntry(_ context.Context, req *infraports.AuditLogRequest) error {
	r.calls = append(r.calls, req)
	return r.failWith
}
func (r *recordingAuditService) ListByEntity(_ context.Context, _ *infraports.ListAuditRequest) (*infraports.ListAuditResponse, error) {
	return &infraports.ListAuditResponse{}, nil
}
func (r *recordingAuditService) ListByActor(_ context.Context, _ *infraports.ListByActorRequest) (*infraports.ListAuditResponse, error) {
	return &infraports.ListAuditResponse{}, nil
}

func trustedActorCtx(userID, workspaceID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: userID, WorkspaceID: workspaceID})
}

// --- PostgresRatingDescriptionSetRepository.writeLifecycleAudit -----------

func TestRatingDescriptionSet_WriteLifecycleAudit_FailsClosedWhenAuditNil(t *testing.T) {
	r := &PostgresRatingDescriptionSetRepository{audit: nil}
	err := r.writeLifecycleAudit(trustedActorCtx("user-1", "ws-1"), "ws-1", "set-1",
		entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionPublish), "PublishRatingDescriptionSet",
		ratingDescriptionSetVersionStatusDraft, versionStatusPublishedValue)
	if err == nil || !strings.Contains(err.Error(), "audit dependency is absent") {
		t.Fatalf("expected a fail-closed audit-absent error, got %v", err)
	}
}

func TestRatingDescriptionSet_WriteLifecycleAudit_FailsClosedWithoutTrustedActor(t *testing.T) {
	audit := &recordingAuditService{}
	r := &PostgresRatingDescriptionSetRepository{audit: audit}
	// No identity on ctx at all.
	err := r.writeLifecycleAudit(context.Background(), "ws-1", "set-1",
		entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionPublish), "PublishRatingDescriptionSet",
		ratingDescriptionSetVersionStatusDraft, versionStatusPublishedValue)
	if err == nil || !strings.Contains(err.Error(), "no trusted actor identity") {
		t.Fatalf("expected a fail-closed no-actor error, got %v", err)
	}
	if len(audit.calls) != 0 {
		t.Fatalf("expected no LogEntry call before the actor check, got %d", len(audit.calls))
	}
}

func TestRatingDescriptionSet_WriteLifecycleAudit_PropagatesLogEntryFailure(t *testing.T) {
	sentinel := fmt.Errorf("simulated audit backend failure")
	audit := &recordingAuditService{failWith: sentinel}
	r := &PostgresRatingDescriptionSetRepository{audit: audit}
	err := r.writeLifecycleAudit(trustedActorCtx("user-1", "ws-1"), "ws-1", "set-1",
		entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionDeprecate), "DeprecateRatingDescriptionSet",
		versionStatusPublishedValue, versionStatusDeprecatedValue)
	if err == nil || !strings.Contains(err.Error(), sentinel.Error()) {
		t.Fatalf("expected the LogEntry error to propagate, got %v", err)
	}
}

func TestRatingDescriptionSet_WriteLifecycleAudit_EventFields(t *testing.T) {
	audit := &recordingAuditService{}
	r := &PostgresRatingDescriptionSetRepository{audit: audit}
	err := r.writeLifecycleAudit(trustedActorCtx("user-42", "ws-9"), "ws-9", "set-7",
		entityid.EntityPermission(entityid.RatingDescriptionSet, entityid.ActionPublish), "PublishRatingDescriptionSet",
		ratingDescriptionSetVersionStatusDraft, versionStatusPublishedValue)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(audit.calls) != 1 {
		t.Fatalf("expected exactly one LogEntry call, got %d", len(audit.calls))
	}
	got := audit.calls[0]
	if got.WorkspaceID != "ws-9" {
		t.Fatalf("expected workspace_id ws-9, got %q", got.WorkspaceID)
	}
	if got.EntityType != entityid.RatingDescriptionSet || got.EntityID != "set-7" {
		t.Fatalf("expected entity_type/id rating_description_set/set-7, got %q/%q", got.EntityType, got.EntityID)
	}
	if got.PermissionCode != "rating_description_set:publish" || got.UseCase != "PublishRatingDescriptionSet" {
		t.Fatalf("unexpected permission_code/use_case: %q/%q", got.PermissionCode, got.UseCase)
	}
	foundTransition := false
	for _, fc := range got.FieldChanges {
		if fc.FieldName == "version_status" {
			foundTransition = true
			if fc.OldValue != ratingDescriptionSetVersionStatusDraft || fc.NewValue != versionStatusPublishedValue {
				t.Fatalf("unexpected version_status old/new: %q -> %q", fc.OldValue, fc.NewValue)
			}
		}
	}
	if !foundTransition {
		t.Fatalf("expected a version_status field change, got %+v", got.FieldChanges)
	}
}

// --- PostgresRatingDescriptionSetProductPlanRepository.writeLifecycleAudit -

func TestRatingDescriptionSetProductPlan_WriteLifecycleAudit_FailsClosedWhenAuditNil(t *testing.T) {
	r := &PostgresRatingDescriptionSetProductPlanRepository{audit: nil}
	err := r.writeLifecycleAudit(trustedActorCtx("user-1", "ws-1"), "ws-1", "link-1",
		entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionCreate),
		"RelinkRatingDescriptionSetProductPlan", "test reason", nil)
	if err == nil || !strings.Contains(err.Error(), "audit dependency is absent") {
		t.Fatalf("expected a fail-closed audit-absent error, got %v", err)
	}
}

func TestRatingDescriptionSetProductPlan_WriteLifecycleAudit_FailsClosedWithoutTrustedActor(t *testing.T) {
	audit := &recordingAuditService{}
	r := &PostgresRatingDescriptionSetProductPlanRepository{audit: audit}
	err := r.writeLifecycleAudit(context.Background(), "ws-1", "link-1",
		entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionDelete),
		"UnlinkRatingDescriptionSetProductPlan", "", nil)
	if err == nil || !strings.Contains(err.Error(), "no trusted actor identity") {
		t.Fatalf("expected a fail-closed no-actor error, got %v", err)
	}
	if len(audit.calls) != 0 {
		t.Fatalf("expected no LogEntry call before the actor check, got %d", len(audit.calls))
	}
}

func TestRatingDescriptionSetProductPlan_WriteLifecycleAudit_PropagatesLogEntryFailure(t *testing.T) {
	sentinel := fmt.Errorf("simulated audit backend failure")
	audit := &recordingAuditService{failWith: sentinel}
	r := &PostgresRatingDescriptionSetProductPlanRepository{audit: audit}
	err := r.writeLifecycleAudit(trustedActorCtx("user-1", "ws-1"), "ws-1", "link-1",
		entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionCreate),
		"RelinkRatingDescriptionSetProductPlan", "AY rollover", nil)
	if err == nil || !strings.Contains(err.Error(), sentinel.Error()) {
		t.Fatalf("expected the LogEntry error to propagate, got %v", err)
	}
}

func TestRatingDescriptionSetProductPlan_WriteLifecycleAudit_EventFields(t *testing.T) {
	audit := &recordingAuditService{}
	r := &PostgresRatingDescriptionSetProductPlanRepository{audit: audit}
	changes := []infraports.AuditFieldChange{
		{FieldName: "product_plan_id", FieldType: 1, OldValue: "", NewValue: "pp-1"},
		{FieldName: "price_schedule_id", FieldType: 1, OldValue: "", NewValue: "ps-1"},
		{FieldName: "rating_description_set_id", FieldType: 1, OldValue: "", NewValue: "set-1"},
		{FieldName: "old_link_id", FieldType: 1, OldValue: "", NewValue: "link-0"},
		{FieldName: "new_link_id", FieldType: 1, OldValue: "", NewValue: "link-1"},
	}
	err := r.writeLifecycleAudit(trustedActorCtx("user-42", "ws-9"), "ws-9", "link-1",
		entityid.EntityPermission(entityid.RatingDescriptionSetProductPlan, entityid.ActionUpdate),
		"RelinkRatingDescriptionSetProductPlan", "AY rollover", changes)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(audit.calls) != 1 {
		t.Fatalf("expected exactly one LogEntry call, got %d", len(audit.calls))
	}
	got := audit.calls[0]
	if got.WorkspaceID != "ws-9" {
		t.Fatalf("expected workspace_id ws-9, got %q", got.WorkspaceID)
	}
	if got.EntityType != entityid.RatingDescriptionSetProductPlan || got.EntityID != "link-1" {
		t.Fatalf("expected entity_type/id rating_description_set_product_plan/link-1, got %q/%q", got.EntityType, got.EntityID)
	}
	if got.Reason != "AY rollover" {
		t.Fatalf("expected reason %q, got %q", "AY rollover", got.Reason)
	}
	if got.UseCase != "RelinkRatingDescriptionSetProductPlan" {
		t.Fatalf("unexpected use_case: %q", got.UseCase)
	}
	if len(got.FieldChanges) != len(changes) {
		t.Fatalf("expected %d field changes, got %d", len(changes), len(got.FieldChanges))
	}
}

func TestRelinkAuditAction_MatchesUseCasePermissionChoice(t *testing.T) {
	if got := relinkAuditAction(""); got != entityid.ActionCreate {
		t.Fatalf("expected ActionCreate for an empty currentID (first link), got %q", got)
	}
	if got := relinkAuditAction("link-0"); got != entityid.ActionUpdate {
		t.Fatalf("expected ActionUpdate for a non-empty currentID (replacement), got %q", got)
	}
}
