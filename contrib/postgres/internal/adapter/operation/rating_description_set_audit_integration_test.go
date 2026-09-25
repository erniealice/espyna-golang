//go:build postgresql

package operation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

// Integration coverage for codex-review-impl2.out.md finding 9 / schema-
// proposal.md §9.4 ("Audit failure rolls back the operation" / "row diffs
// join the ambient transaction"). Reuses the W3 lifecycle harness (openW3LifecycleIntegrationDB,
// tm.RunInTransaction, the seedW3Lifecycle* fixture helpers, sgdtExecutor) from
// rating_description_set_w3_lifecycle_integration_test.go — same package, same
// TEST_DATABASE_URL / rolled-back-transaction convention, no residue.

// failingAuditService always fails LogEntry — proves that when the semantic
// audit write inside RelinkLocked's ambient transaction fails, the calling
// use case's real transaction rollback (not the test's own manual sentinel)
// discards the relink's SQL side effects too.
type failingAuditService struct{ calls int }

func (f *failingAuditService) LogEntry(_ context.Context, _ *infraports.AuditLogRequest) error {
	f.calls++
	return fmt.Errorf("simulated audit backend failure")
}
func (f *failingAuditService) ListByEntity(_ context.Context, _ *infraports.ListAuditRequest) (*infraports.ListAuditResponse, error) {
	return &infraports.ListAuditResponse{}, nil
}
func (f *failingAuditService) ListByActor(_ context.Context, _ *infraports.ListByActorRequest) (*infraports.ListAuditResponse, error) {
	return &infraports.ListAuditResponse{}, nil
}

// TestIntegration_RatingDescriptionSetProductPlan_RelinkAuditFailureRollsBackInsert
// proves the rollback half of "audit failure rolls back the operation":
// RelinkLocked's step-6 insert (+ step-5 deactivate) must not persist when
// the step-7 semantic audit write fails. Unlike the sibling W3 lifecycle
// tests, this one lets tm.RunInTransaction perform a REAL rollback driven by
// the actual returned error (rather than an always-fail sentinel), then reads
// back OUTSIDE that transaction to prove nothing committed.
func TestIntegration_RatingDescriptionSetProductPlan_RelinkAuditFailureRollsBackInsert(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	audit := &failingAuditService{}
	linkRepo := &PostgresRatingDescriptionSetProductPlanRepository{
		dbOps: wsOps, db: db, tableName: "rating_description_set_product_plan", audit: audit,
	}

	const ws = "w3lc-audit-fail-ws"
	const scale = "w3lc-audit-fail-scale"
	const setPublished = "w3lc-audit-fail-set"
	const pp = "w3lc-audit-fail-pp"
	const schedule = "w3lc-audit-fail-schedule"
	const linkID = "w3lc-audit-fail-newlink"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, setPublished, ws, scale, versionStatusPublishedValue)
		seedW3LifecycleOffering(t, ex, txCtx, "w3lc-audit-fail-product", "w3lc-audit-fail-plan", pp, ws)
		seedW3LifecyclePriceSchedule(t, ex, txCtx, schedule, ws)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-audit-fail-user", WorkspaceID: ws})
		_, relinkErr := linkRepo.RelinkLocked(ctx, &linkpb.RatingDescriptionSetProductPlan{
			Id: linkID, ProductPlanId: pp, PriceScheduleId: schedule, RatingDescriptionSetId: setPublished,
		}, nil, "test reason")
		if relinkErr == nil {
			return fmt.Errorf("expected relink to fail when the audit write fails")
		}
		if !strings.Contains(relinkErr.Error(), "simulated audit backend failure") {
			return fmt.Errorf("expected the simulated audit error to surface, got: %w", relinkErr)
		}
		// Return the REAL error (not a sentinel) so RunInTransaction performs an
		// actual rollback — the assertions below then run against a completely
		// separate, un-transacted read.
		return relinkErr
	})
	if err == nil || !strings.Contains(err.Error(), "simulated audit backend failure") {
		t.Fatalf("expected the transaction to fail closed on the audit error, got: %v", err)
	}
	if audit.calls == 0 {
		t.Fatalf("expected the audit service to have been invoked")
	}

	var residue int
	if scanErr := db.QueryRow(`SELECT count(*) FROM rating_description_set_product_plan WHERE id = $1`, linkID).Scan(&residue); scanErr != nil {
		t.Fatalf("post-rollback residue check: %v", scanErr)
	}
	if residue != 0 {
		t.Fatalf("expected no link row to persist after an audit failure caused a rollback, found %d", residue)
	}
}

// TestIntegration_RatingDescriptionSetProductPlan_RelinkWritesAuditEvent
// proves the positive half of finding 9: a successful RelinkLocked writes a
// real audit_trail.audit_entry (+ audit_trail.audit_field_change) row for the
// new link, visible within the SAME ambient (uncommitted) transaction, before
// the test's own rollback-sentinel discards everything.
func TestIntegration_RatingDescriptionSetProductPlan_RelinkWritesAuditEvent(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	// fix3-backend (codex-review-impl3 #4): exercise the REAL registered,
	// audited factory (rating_description_set_product_plan.go init() —
	// NewAuditedWorkspaceAwareOperations), not a hand-built unaudited repo, so
	// a wiring regression in the factory fails this test. wsOps above is only
	// the fixture executor.
	linkRepo := fix3RegisteredRepo[*PostgresRatingDescriptionSetProductPlanRepository](t, db, entityid.RatingDescriptionSetProductPlan)

	// ws must be a real UUID string: this test's RelinkLocked call reaches the
	// success path, which writes a real audit_trail.audit_entry row whose
	// workspace_id column is typed uuid (production workspace ids are always
	// real UUIDs — uuidv7). Every other id below stays a plain string.
	const ws = "00000000-0000-0000-0000-00000000a002"
	const scale = "w3lc-audit-ok-scale"
	const setPublished = "w3lc-audit-ok-set"
	const pp = "w3lc-audit-ok-pp"
	const schedule = "w3lc-audit-ok-schedule"
	const linkID = "w3lc-audit-ok-newlink"
	const actor = "w3lc-audit-ok-user"
	const reason = "AY rollover"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, setPublished, ws, scale, versionStatusPublishedValue)
		seedW3LifecycleOffering(t, ex, txCtx, "w3lc-audit-ok-product", "w3lc-audit-ok-plan", pp, ws)
		seedW3LifecyclePriceSchedule(t, ex, txCtx, schedule, ws)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: actor, WorkspaceID: ws})
		item, relinkErr := linkRepo.RelinkLocked(ctx, &linkpb.RatingDescriptionSetProductPlan{
			Id: linkID, ProductPlanId: pp, PriceScheduleId: schedule, RatingDescriptionSetId: setPublished,
		}, nil, reason)
		if relinkErr != nil {
			return fmt.Errorf("expected relink success, got: %w", relinkErr)
		}
		if !item.GetActive() {
			return fmt.Errorf("expected the new link to be active")
		}

		var auditCount int
		if scanErr := ex.QueryRowContext(txCtx,
			`SELECT count(*) FROM audit_trail.audit_entry
			 WHERE entity_type = 'rating_description_set_product_plan' AND entity_id = $1
			   AND use_case = 'RelinkRatingDescriptionSetProductPlan' AND actor_id = $2
			   AND workspace_id = $3 AND reason = $4`,
			linkID, actor, ws, reason).Scan(&auditCount); scanErr != nil {
			return fmt.Errorf("count audit_entry: %w", scanErr)
		}
		if auditCount != 1 {
			return fmt.Errorf("expected exactly one audit_entry row for the relink, found %d", auditCount)
		}

		// The generic row-diff event from the audited factory's dbOps.Create
		// (the new link insert) must carry the TRUSTED workspace id — it was
		// NULL before fix3 (audit_adapter.go trustedAuditWorkspaceID).
		var genericCount int
		if scanErr := ex.QueryRowContext(txCtx,
			`SELECT count(*) FROM audit_trail.audit_entry
			 WHERE entity_type = 'rating_description_set_product_plan' AND entity_id = $1
			   AND method_name = 'PostgresOperations.Create' AND workspace_id = $2`,
			linkID, ws).Scan(&genericCount); scanErr != nil {
			return fmt.Errorf("count generic audit_entry: %w", scanErr)
		}
		if genericCount != 1 {
			return fmt.Errorf("expected one workspace-stamped generic create audit_entry from the registered factory, found %d", genericCount)
		}

		var fieldChangeCount int
		if scanErr := ex.QueryRowContext(txCtx,
			`SELECT count(*) FROM audit_trail.audit_field_change afc
			 JOIN audit_trail.audit_entry ae ON ae.id = afc.audit_entry_id
			 WHERE ae.entity_type = 'rating_description_set_product_plan' AND ae.entity_id = $1`,
			linkID).Scan(&fieldChangeCount); scanErr != nil {
			return fmt.Errorf("count audit_field_change: %w", scanErr)
		}
		if fieldChangeCount == 0 {
			return fmt.Errorf("expected at least one audit_field_change row for the relink")
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

// TestIntegration_RatingDescriptionSet_PublishThroughScaleLockWritesAuditEvent
// answers the lead's compatibility question directly with a real DB: fix2-
// backend's descriptor lock protocol (LockScoreScaleForPublish ->
// LockRatingDescriptionSetForUpdate -> VerifyRatingDescriptionSetPublishScale,
// rating_description_set_publish_lock.go) is defined on the SAME concrete
// *PostgresRatingDescriptionSetRepository this package's constructor returns
// — fix2-audit never wraps that repository in a decorator/proxy type (only
// the internal dbOps field's construction changed), so the lock methods and
// the audited conditionalStatusTransition below coexist on one struct value
// with no pass-through needed. This runs the real lock sequence end-to-end and
// asserts the resulting PublishRatingDescriptionSetIfDraft call still writes
// its semantic audit event.
func TestIntegration_RatingDescriptionSet_PublishThroughScaleLockWritesAuditEvent(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	setRepo := NewPostgresRatingDescriptionSetRepository(wsOps, "rating_description_set").(*PostgresRatingDescriptionSetRepository)

	const ws = "00000000-0000-0000-0000-00000000a003"
	const scale = "w3lc-publish-audit-scale"
	const draftSet = "w3lc-publish-audit-set"
	const actor = "w3lc-publish-audit-user"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, draftSet, ws, scale, ratingDescriptionSetVersionStatusDraft)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: actor, WorkspaceID: ws})

		// The exact sequence PublishRatingDescriptionSetUseCase.Execute runs
		// (publish_rating_description_set.go): scale FOR SHARE, then the set
		// FOR UPDATE, then re-verify, then the conditional transition.
		lockedScaleID, err := setRepo.LockScoreScaleForPublish(ctx, draftSet)
		if err != nil {
			return fmt.Errorf("lock scale: %w", err)
		}
		if lockedScaleID != scale {
			return fmt.Errorf("expected locked scale %q, got %q", scale, lockedScaleID)
		}
		if _, err := setRepo.LockRatingDescriptionSetForUpdate(ctx, draftSet); err != nil {
			return fmt.Errorf("lock set: %w", err)
		}
		if err := setRepo.VerifyRatingDescriptionSetPublishScale(ctx, draftSet, lockedScaleID); err != nil {
			return fmt.Errorf("verify scale: %w", err)
		}
		published, err := setRepo.PublishRatingDescriptionSetIfDraft(ctx, draftSet)
		if err != nil {
			return fmt.Errorf("publish: %w", err)
		}
		if published.GetVersionStatus().String() != "VERSION_STATUS_PUBLISHED" {
			return fmt.Errorf("expected VERSION_STATUS_PUBLISHED, got %v", published.GetVersionStatus())
		}

		var auditCount int
		if scanErr := ex.QueryRowContext(txCtx,
			`SELECT count(*) FROM audit_trail.audit_entry
			 WHERE entity_type = 'rating_description_set' AND entity_id = $1
			   AND use_case = 'PublishRatingDescriptionSet' AND actor_id = $2 AND workspace_id = $3`,
			draftSet, actor, ws).Scan(&auditCount); scanErr != nil {
			return fmt.Errorf("count audit_entry: %w", scanErr)
		}
		if auditCount != 1 {
			return fmt.Errorf("expected exactly one audit_entry row for the publish, found %d", auditCount)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}
