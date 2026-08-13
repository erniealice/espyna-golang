//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

const (
	deletePairIntegrationWorkspace = "sgdt-int-delete-ws"
	deletePairIntegrationCategory  = "sgdt-int-delete-category"
	deletePairIntegrationArtifact  = "sgdt-int-delete-artifact"
	deletePairIntegrationBinding   = "sgdt-int-delete-binding"
	deletePairIntegrationShared    = "sgdt-int-delete-shared-binding"
	deletePairIntegrationProfile   = "RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1"
)

func seedDeletePairFixture(t *testing.T, ex sqlexec.DBExecutor, includeShared bool) {
	t.Helper()
	if _, err := ex.ExecContext(context.Background(), `INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, deletePairIntegrationWorkspace); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if _, err := ex.ExecContext(context.Background(), `INSERT INTO job_category (id, workspace_id, name, code) VALUES ($1, $2, 'Delete Pair', 'delete-pair')`, deletePairIntegrationCategory, deletePairIntegrationWorkspace); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	if _, err := ex.ExecContext(context.Background(), `INSERT INTO document_template
		(id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		VALUES ($1, $2, 'Delete Pair Artifact', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/grade-delete.docx', 'active', true)`, deletePairIntegrationArtifact, deletePairIntegrationWorkspace); err != nil {
		t.Fatalf("seed artifact: %v", err)
	}
	if _, err := ex.ExecContext(context.Background(), `INSERT INTO subscription_group_document_template
		(id, workspace_id, document_template_id, render_profile, job_category_id, version, version_status, active, date_created, date_modified)
		VALUES ($1, $2, $3, $4, $5, 0, $6, true, 111111, 111111)`,
		deletePairIntegrationBinding, deletePairIntegrationWorkspace, deletePairIntegrationArtifact,
		deletePairIntegrationProfile, deletePairIntegrationCategory, versionStatusDraft); err != nil {
		t.Fatalf("seed draft binding: %v", err)
	}
	if includeShared {
		if _, err := ex.ExecContext(context.Background(), `INSERT INTO subscription_group_document_template
			(id, workspace_id, document_template_id, render_profile, job_category_id, version, version_status, active, date_created, date_modified)
			VALUES ($1, $2, $3, $4, $5, 0, $6, true, 111112, 111112)`,
			deletePairIntegrationShared, deletePairIntegrationWorkspace, deletePairIntegrationArtifact,
			deletePairIntegrationProfile, deletePairIntegrationCategory, versionStatusDraft); err != nil {
			t.Fatalf("seed shared binding: %v", err)
		}
	}
}

func cleanupDeletePairFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM subscription_group_document_template WHERE id LIKE 'sgdt-int-delete-%'`); err != nil {
		t.Errorf("cleanup bindings: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM document_template WHERE id LIKE 'sgdt-int-delete-%'`); err != nil {
		t.Errorf("cleanup artifacts: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM job_category WHERE id LIKE 'sgdt-int-delete-%'`); err != nil {
		t.Errorf("cleanup categories: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM workspace WHERE id LIKE 'sgdt-int-delete-%'`); err != nil {
		t.Errorf("cleanup workspaces: %v", err)
	}
}

func TestIntegration_SubscriptionGroupDocumentTemplate_DeleteDraftPair_UnsharedSoftDeletesBindingAndArtifactWithinTransaction(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template").(*PostgresSubscriptionGroupDocumentTemplateRepository)

	rollback := fmt.Errorf("sgdt-int-delete: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, err := sgdtExecutor(txCtx, wsOps)
		if err != nil {
			return err
		}
		seedDeletePairFixture(t, ex, false)
		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-delete-user", WorkspaceID: deletePairIntegrationWorkspace})
		artifact, err := repo.DeleteDraftPair(ctx, deletePairIntegrationBinding)
		if err != nil {
			return fmt.Errorf("delete unshared draft pair: %w", err)
		}
		if artifact.GetId() != deletePairIntegrationArtifact || artifact.GetStorageContainer() != "templates" || artifact.GetStorageKey() != "templates/grade-delete.docx" {
			return fmt.Errorf("returned artifact locator = (%q,%q,%q), want exact committed locator", artifact.GetId(), artifact.GetStorageContainer(), artifact.GetStorageKey())
		}
		if artifact.GetActive() {
			return fmt.Errorf("returned artifact must be inactive")
		}

		var bindingActive, artifactActive bool
		if err := ex.QueryRowContext(txCtx, `SELECT active FROM subscription_group_document_template WHERE id = $1`, deletePairIntegrationBinding).Scan(&bindingActive); err != nil {
			return fmt.Errorf("read deleted binding: %w", err)
		}
		if err := ex.QueryRowContext(txCtx, `SELECT active FROM document_template WHERE id = $1`, deletePairIntegrationArtifact).Scan(&artifactActive); err != nil {
			return fmt.Errorf("read deleted artifact: %w", err)
		}
		if bindingActive || artifactActive {
			return fmt.Errorf("delete must atomically set binding/artifact inactive: binding=%v artifact=%v", bindingActive, artifactActive)
		}
		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected intentional outer rollback, got %v", err)
	}
	assertNoSGDTResidue(t, db)
}

func TestIntegration_SubscriptionGroupDocumentTemplate_DeleteDraftPair_RejectsSharedArtifactWithoutMutation(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template").(*PostgresSubscriptionGroupDocumentTemplateRepository)

	rollback := fmt.Errorf("sgdt-int-delete: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, err := sgdtExecutor(txCtx, wsOps)
		if err != nil {
			return err
		}
		seedDeletePairFixture(t, ex, true)
		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-delete-user", WorkspaceID: deletePairIntegrationWorkspace})
		if _, err := repo.DeleteDraftPair(ctx, deletePairIntegrationBinding); err == nil {
			return fmt.Errorf("shared artifact delete must fail")
		} else if !strings.Contains(err.Error(), "referenced by another active binding") {
			return fmt.Errorf("unexpected shared-reference error: %w", err)
		}

		var firstActive, secondActive, artifactActive bool
		if err := ex.QueryRowContext(txCtx, `SELECT active FROM subscription_group_document_template WHERE id = $1`, deletePairIntegrationBinding).Scan(&firstActive); err != nil {
			return fmt.Errorf("read first binding after rejection: %w", err)
		}
		if err := ex.QueryRowContext(txCtx, `SELECT active FROM subscription_group_document_template WHERE id = $1`, deletePairIntegrationShared).Scan(&secondActive); err != nil {
			return fmt.Errorf("read shared binding after rejection: %w", err)
		}
		if err := ex.QueryRowContext(txCtx, `SELECT active FROM document_template WHERE id = $1`, deletePairIntegrationArtifact).Scan(&artifactActive); err != nil {
			return fmt.Errorf("read artifact after rejection: %w", err)
		}
		if !firstActive || !secondActive || !artifactActive {
			return fmt.Errorf("shared-reference rejection must preserve active rows: first=%v second=%v artifact=%v", firstActive, secondActive, artifactActive)
		}
		return rollback
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected intentional outer rollback, got %v", err)
	}
	assertNoSGDTResidue(t, db)
}

func TestIntegration_SubscriptionGroupDocumentTemplate_DeleteDraftPair_CreateValidationKeyShareBlocksConcurrentArtifactUpdate(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	cleanupDeletePairFixture(t, db)
	seedDeletePairFixture(t, db, false)
	t.Cleanup(func() { cleanupDeletePairFixture(t, db) })

	connA, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open first session: %v", err)
	}
	defer connA.Close()
	connB, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("open second session: %v", err)
	}
	defer connB.Close()

	txA, err := connA.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin create-validation transaction: %v", err)
	}
	defer txA.Rollback() //nolint:errcheck — lock regression is rollback-only

	request := &pb.SubscriptionGroupDocumentTemplate{
		DocumentTemplateId: deletePairIntegrationArtifact,
		RenderProfile:      pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
		JobCategoryId:      strptr(deletePairIntegrationCategory),
	}
	if err := validateBindingReferencesWithExecutor(context.Background(), txA, deletePairIntegrationWorkspace, request); err != nil {
		t.Fatalf("create-side validation should hold FOR KEY SHARE: %v", err)
	}

	txB, err := connB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin concurrent update transaction: %v", err)
	}
	defer txB.Rollback() //nolint:errcheck — timeout regression is rollback-only

	lockCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	started := time.Now()
	var lockedID string
	err = txB.QueryRowContext(lockCtx, `SELECT id FROM document_template WHERE id = $1 FOR UPDATE`, deletePairIntegrationArtifact).Scan(&lockedID)
	if err == nil {
		t.Fatalf("artifact FOR UPDATE unexpectedly acquired while create validation held FOR KEY SHARE")
	}
	if elapsed := time.Since(started); elapsed < 200*time.Millisecond {
		t.Fatalf("artifact FOR UPDATE failed too quickly (%s), lock blocking was not demonstrated: %v", elapsed, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(strings.ToLower(err.Error()), "cancel") {
		t.Fatalf("expected bounded context cancellation from blocked FOR UPDATE, got %v", err)
	}
}
