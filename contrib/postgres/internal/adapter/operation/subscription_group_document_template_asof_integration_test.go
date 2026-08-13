//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	operationv1 "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const sgdtTestWS = "sgdt-int-ws"
const sgdtOtherTestWS = "sgdt-int-ws-other"

func openSGDTIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	var ok string
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('subscription_group_document_template')::text").Scan(&ok); err != nil || ok == "" {
		db.Close()
		t.Skip("subscription_group_document_template not present")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func assertNoSGDTResidue(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{
		"subscription_group_document_template",
		"document_template",
		"job_category",
		"price_schedule",
		"plan",
		"workspace",
	} {
		var n int
		query := fmt.Sprintf("SELECT count(*) FROM %s WHERE id LIKE 'sgdt-int-%%'", table)
		if table == "workspace" {
			query = "SELECT count(*) FROM workspace WHERE id LIKE 'sgdt-int-%%'"
		}
		if err := db.QueryRow(query).Scan(&n); err != nil {
			t.Fatalf("residue check for %s failed: %v", table, err)
		}
		if n != 0 {
			t.Fatalf("rollback residue in %s: %d rows", table, n)
		}
	}
}

func execSGDTQuery(ctx context.Context, ops interfaces.DatabaseOperation, q string, args ...any) (sql.Result, error) {
	return ocitestExec(ctx, ops, q, args...)
}

func sgdtExecutor(ctx context.Context, ops interfaces.DatabaseOperation) (sqlexec.DBExecutor, error) {
	ex, ok := ops.(interface {
		GetExecutor(context.Context) sqlexec.DBExecutor
	})
	if !ok {
		return nil, fmt.Errorf("ops does not expose GetExecutor")
	}
	return ex.GetExecutor(ctx), nil
}

func TestIntegration_SubscriptionGroupDocumentTemplate_CreateRejectsForeignDocumentTemplateAndStripsCallerFields(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template").(*PostgresSubscriptionGroupDocumentTemplateRepository)

	r := fmt.Errorf("sgdt-int: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := execSGDTQuery(txCtx, wsOps, q, args...)
			return err
		}
		ex, err := sgdtExecutor(txCtx, wsOps)
		if err != nil {
			return err
		}

		if err := exec(`INSERT INTO workspace (id) VALUES ($1), ($2) ON CONFLICT (id) DO NOTHING`, sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO job_category
		        (id, workspace_id, name, code)
		        VALUES
		        ('sgdt-int-cat-main', $1, 'Main Category', 'main'),
		        ('sgdt-int-cat-other', $2, 'Other Category', 'other')`, sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed job categories: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES
		        ('sgdt-int-dt-main', $1, 'SGDT local', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/subscription/main.docx', 'active', true),
		        ('sgdt-int-dt-foreign', $2, 'SGDT foreign', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/subscription/foreign.docx', 'active', true)`,
			sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed document templates: %w", err)
		}

		ctxMain := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "intruder", WorkspaceID: sgdtTestWS})
		if _, err := repo.CreateSubscriptionGroupDocumentTemplate(ctxMain, &pb.CreateSubscriptionGroupDocumentTemplateRequest{
			Data: &pb.SubscriptionGroupDocumentTemplate{
				Id:                  "sgdt-int-create-attempt",
				DocumentTemplateId:  "sgdt-int-dt-foreign",
				RenderProfile:       pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
				JobCategoryId:       strptr("sgdt-int-cat-main"),
				CreatedBy:           strptr("attacker"),
				SupersedesBindingId: strptr("sgdt-int-bad"),
			},
		}); err == nil {
			return fmt.Errorf("foreign document references must fail at create")
		}

		resp, err := repo.CreateSubscriptionGroupDocumentTemplate(ctxMain, &pb.CreateSubscriptionGroupDocumentTemplateRequest{
			Data: &pb.SubscriptionGroupDocumentTemplate{
				Id:                  "sgdt-int-create-ok",
				DocumentTemplateId:  "sgdt-int-dt-main",
				RenderProfile:       pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
				JobCategoryId:       strptr("sgdt-int-cat-main"),
				CreatedBy:           strptr("attacker"),
				SupersedesBindingId: strptr("sgdt-int-bad"),
				VersionStatus:       operationv1.VersionStatus_VERSION_STATUS_PUBLISHED,
				Version:             42,
				PublishedBy:         strptr("attacker"),
			},
		})
		if err != nil {
			return fmt.Errorf("create valid binding: %w", err)
		}
		if got := resp.GetData()[0].GetId(); got != "sgdt-int-create-ok" {
			return fmt.Errorf("unexpected binding id %q", got)
		}

		var gotVersionStatus string
		var gotVersion int32
		var createdBy sql.NullString
		var supersedes sql.NullString
		var publishedAt sql.NullInt64
		var publishedBy sql.NullString
		if err := ex.QueryRowContext(txCtx,
			`SELECT version_status, version, created_by, supersedes_binding_id, published_at, published_by
		           FROM subscription_group_document_template WHERE id = $1`,
			"sgdt-int-create-ok").Scan(&gotVersionStatus, &gotVersion, &createdBy, &supersedes, &publishedAt, &publishedBy); err != nil {
			return fmt.Errorf("read created row: %w", err)
		}
		if gotVersionStatus != versionStatusDraft {
			return fmt.Errorf("create must force DRAFT, got %q", gotVersionStatus)
		}
		if gotVersion != 0 {
			return fmt.Errorf("create must force version 0, got %d", gotVersion)
		}
		if !createdBy.Valid || createdBy.String != "intruder" {
			return fmt.Errorf("created_by must be client-authenticated context principal, got %v", createdBy)
		}
		if supersedes.Valid {
			return fmt.Errorf("supersedes_binding_id must be stripped, got %q", supersedes.String)
		}
		if publishedAt.Valid {
			return fmt.Errorf("published_at must be stripped at create")
		}
		if publishedBy.Valid {
			return fmt.Errorf("published_by must be stripped at create")
		}
		return r
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}

	assertNoSGDTResidue(t, db)
}

func TestIntegration_SubscriptionGroupDocumentTemplate_Update_MissingOrForeignWorkspaceCausesNoMutation(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template")

	r := fmt.Errorf("sgdt-int: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := execSGDTQuery(txCtx, wsOps, q, args...)
			return err
		}
		ex, err := sgdtExecutor(txCtx, wsOps)
		if err != nil {
			return err
		}

		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO job_category (id, name, workspace_id, code) VALUES ('sgdt-int-update-cat', 'Cat', $1, 'cat')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed category: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES ('sgdt-int-update-dt', $1, 'template', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/subscription/update.docx', 'active', true)`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed template: %w", err)
		}
		if err := exec(`INSERT INTO subscription_group_document_template
		        (id, workspace_id, document_template_id, render_profile, job_category_id, version, version_status, active, date_created, date_modified)
		        VALUES ('sgdt-int-update', $1, 'sgdt-int-update-dt',
		                'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1',
		                'sgdt-int-update-cat', 0, $2, true, 111111, 111111)`, sgdtTestWS, versionStatusDraft); err != nil {
			return fmt.Errorf("seed draft binding: %w", err)
		}

		if _, err := repo.UpdateSubscriptionGroupDocumentTemplate(context.Background(), &pb.UpdateSubscriptionGroupDocumentTemplateRequest{Data: &pb.SubscriptionGroupDocumentTemplate{Id: "sgdt-int-update"}}); err == nil {
			return fmt.Errorf("update without workspace should fail closed")
		}
		if _, err := repo.UpdateSubscriptionGroupDocumentTemplate(identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "sgdt-test", WorkspaceID: sgdtOtherTestWS}), &pb.UpdateSubscriptionGroupDocumentTemplateRequest{Data: &pb.SubscriptionGroupDocumentTemplate{Id: "sgdt-int-update"}}); err == nil {
			return fmt.Errorf("update under foreign workspace should fail")
		}

		var version int32
		var dateModified int64
		if err := ex.QueryRowContext(txCtx, `SELECT version, date_modified FROM subscription_group_document_template WHERE id = $1`, "sgdt-int-update").Scan(&version, &dateModified); err != nil {
			return fmt.Errorf("read mutated row: %w", err)
		}
		if version != 0 {
			return fmt.Errorf("version should remain unchanged, got %d", version)
		}
		if dateModified != 111111 {
			return fmt.Errorf("date_modified should remain unchanged, got %d", dateModified)
		}
		return r
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}

	assertNoSGDTResidue(t, db)
}

func TestIntegration_SubscriptionGroupDocumentTemplate_FindApplicable_BucketsAndCrossWorkspaceAndAmbiguity(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template").(*PostgresSubscriptionGroupDocumentTemplateRepository)

	boundary := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	v1Start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	r := fmt.Errorf("sgdt-int: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := execSGDTQuery(txCtx, wsOps, q, args...)
			return err
		}

		if err := exec(`INSERT INTO workspace (id) VALUES ($1), ($2) ON CONFLICT (id) DO NOTHING`, sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed workspaces: %w", err)
		}
		if err := exec(`INSERT INTO job_category (id, name, workspace_id, code)
		        VALUES
		        ('sgdt-int-main-cat', 'Main', $1, 'main'),
		        ('sgdt-int-other-cat', 'Other', $2, 'other')`, sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed categories: %w", err)
		}
		if err := exec(`INSERT INTO plan (id, workspace_id, name) VALUES ('sgdt-int-plan', $1, 'Plan A'), ('sgdt-int-other-plan', $2, 'Plan B')`, sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed plans: %w", err)
		}
		if err := exec(`INSERT INTO price_schedule (id, workspace_id, name)
		        VALUES ('sgdt-int-ps', $1, 'PS A'), ('sgdt-int-other-ps', $2, 'PS B')`, sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed price schedules: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES
		        ('sgdt-int-main-dt-0', $1, 'Main 0', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/main0.docx', 'active', true),
		        ('sgdt-int-main-dt-1', $1, 'Main 1', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/main1.docx', 'active', true),
		        ('sgdt-int-main-dt-2', $1, 'Main 2', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/main2.docx', 'active', true),
		        ('sgdt-int-main-dt-3', $1, 'Main 3', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/main3.docx', 'active', true),
		        ('sgdt-int-main-dt-blank', $1, 'Blank', 'docx', 'subscription_group_outcome_summary', '   ', '   ', 'active', true),
		        ('sgdt-int-main-dt-amb', $1, 'Amb', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/amb.docx', 'active', true),
		        ('sgdt-int-other-dt-0', $2, 'Other 0', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/other0.docx', 'active', true)`, sgdtTestWS, sgdtOtherTestWS); err != nil {
			return fmt.Errorf("seed document templates: %w", err)
		}

		if err := exec(`INSERT INTO subscription_group_document_template
		        (id, workspace_id, document_template_id, render_profile, price_schedule_id, plan_id, job_category_id, version, version_status, validity_start, validity_end, active)
		        VALUES
		        ('sgdt-int-main-r0', $1, 'sgdt-int-main-dt-0', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-ps', 'sgdt-int-plan', 'sgdt-int-main-cat', 1, $3, $4, NULL, true),
		        ('sgdt-int-main-r1', $1, 'sgdt-int-main-dt-1', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', NULL, 'sgdt-int-plan', 'sgdt-int-main-cat', 1, $3, $4, NULL, true),
		        ('sgdt-int-main-r2', $1, 'sgdt-int-main-dt-2', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-ps', NULL, 'sgdt-int-main-cat', 1, $3, $4, NULL, true),
		        ('sgdt-int-main-r3', $1, 'sgdt-int-main-dt-3', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', NULL, NULL, 'sgdt-int-main-cat', 1, $3, $4, NULL, true),
		        ('sgdt-int-main-r0-blank-storage', $1, 'sgdt-int-main-dt-blank', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-ps', 'sgdt-int-plan', 'sgdt-int-main-cat', 2, $3, $4, NULL, true),
		        ('sgdt-int-main-r3-amb', $1, 'sgdt-int-main-dt-amb', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', NULL, NULL, 'sgdt-int-main-cat', 2, $3, $4, NULL, true),
		        ('sgdt-int-other-r0', $2, 'sgdt-int-other-dt-0', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-other-ps', 'sgdt-int-other-plan', 'sgdt-int-other-cat', 1, $3, $4, NULL, true)`,
			sgdtTestWS, sgdtOtherTestWS, versionStatusPublished, v1Start); err != nil {
			return fmt.Errorf("seed bindings: %w", err)
		}

		resolve := func(ctx context.Context, wsPlan, wsPs string, asOf time.Time) (*pb.FindApplicableSubscriptionGroupDocumentTemplateResponse, error) {
			return repo.FindApplicableSubscriptionGroupDocumentTemplate(ctx, &pb.FindApplicableSubscriptionGroupDocumentTemplateRequest{
				RenderProfile:   pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
				PriceScheduleId: strptr(wsPs),
				PlanId:          strptr(wsPlan),
				JobCategoryId:   strptr("sgdt-int-main-cat"),
				AsOf:            timestamppb.New(asOf),
				DocumentPurpose: subscriptionGroupOutcomeSummaryDocumentPurpose,
			})
		}

		wsMain := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-test", WorkspaceID: sgdtTestWS})

		resp, err := resolve(wsMain, "sgdt-int-plan", "sgdt-int-ps", boundary)
		if err != nil {
			return fmt.Errorf("resolve local rank0: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "sgdt-int-main-r0" {
			return fmt.Errorf("expected rank0 on local workspace, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		if err := exec(`UPDATE subscription_group_document_template SET active = false WHERE id = 'sgdt-int-main-r0'`); err != nil {
			return fmt.Errorf("disable rank0: %w", err)
		}
		resp, err = resolve(wsMain, "sgdt-int-plan", "sgdt-int-ps", boundary)
		if err != nil {
			return fmt.Errorf("resolve local fallback rank1: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "sgdt-int-main-r1" {
			return fmt.Errorf("expected rank1 on local workspace, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		if err := exec(`UPDATE subscription_group_document_template SET active = false WHERE id = 'sgdt-int-main-r1'`); err != nil {
			return fmt.Errorf("disable rank1: %w", err)
		}
		resp, err = resolve(wsMain, "sgdt-int-plan", "sgdt-int-ps", boundary)
		if err != nil {
			return fmt.Errorf("resolve local fallback rank2: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "sgdt-int-main-r2" {
			return fmt.Errorf("expected rank2 on local workspace, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		if err := exec(`UPDATE subscription_group_document_template SET active = false WHERE id = 'sgdt-int-main-r2'`); err != nil {
			return fmt.Errorf("disable rank2: %w", err)
		}
		if _, err = resolve(wsMain, "sgdt-int-plan", "sgdt-int-ps", boundary); err == nil || !strings.Contains(err.Error(), "ambiguous applicable binding") {
			return fmt.Errorf("overlapping equal-rank versions must fail closed, got: %v", err)
		}
		if err := exec(`UPDATE subscription_group_document_template SET active = false WHERE id = 'sgdt-int-main-r3-amb'`); err != nil {
			return fmt.Errorf("disable ambiguous rank3 sibling: %w", err)
		}
		resp, err = resolve(wsMain, "sgdt-int-plan", "sgdt-int-ps", boundary)
		if err != nil {
			return fmt.Errorf("resolve local fallback rank3: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "sgdt-int-main-r3" {
			return fmt.Errorf("expected rank3 on local workspace, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		if err := exec(`UPDATE subscription_group_document_template SET active = false WHERE id = 'sgdt-int-main-r3'`); err != nil {
			return fmt.Errorf("disable rank3: %w", err)
		}
		resp, err = resolve(wsMain, "sgdt-int-plan", "sgdt-int-ps", boundary)
		if err != nil {
			return fmt.Errorf("resolve local with only blank high-rank storage: %w", err)
		}
		if resp.GetFound() {
			return fmt.Errorf("blank storage-only binding must not be resolved, got %q", resp.GetBinding().GetId())
		}

		if err := exec(`SAVEPOINT amb`); err != nil {
			return fmt.Errorf("savepoint: %w", err)
		}
		dupErr := exec(`INSERT INTO subscription_group_document_template
		        (id, workspace_id, document_template_id, render_profile, price_schedule_id, plan_id, job_category_id, version, version_status, validity_start, validity_end, active)
		        VALUES ('sgdt-int-main-r0-ambig', $1, 'sgdt-int-main-dt-amb', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-ps', 'sgdt-int-plan', 'sgdt-int-main-cat', 2, $2, $3, NULL, true)`,
			sgdtTestWS, versionStatusPublished, boundary)
		if dupErr == nil {
			return fmt.Errorf("an equal-rank same-version sibling must be rejected by uq_sgdt_pub_version")
		}
		if !strings.Contains(dupErr.Error(), "uq_sgdt_pub_version") {
			return fmt.Errorf("ambiguous sibling must fail on uq_sgdt_pub_version, got: %v", dupErr)
		}
		if err := exec(`ROLLBACK TO SAVEPOINT amb`); err != nil {
			return fmt.Errorf("rollback to savepoint: %w", err)
		}

		// Foreign-workspace artifact must be isolated.
		resp, err = repo.FindApplicableSubscriptionGroupDocumentTemplate(
			identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-test", WorkspaceID: sgdtOtherTestWS}),
			&pb.FindApplicableSubscriptionGroupDocumentTemplateRequest{
				RenderProfile:   pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
				PriceScheduleId: strptr("sgdt-int-other-ps"),
				PlanId:          strptr("sgdt-int-other-plan"),
				JobCategoryId:   strptr("sgdt-int-other-cat"),
				AsOf:            timestamppb.New(boundary),
				DocumentPurpose: subscriptionGroupOutcomeSummaryDocumentPurpose,
			},
		)
		if err != nil {
			return fmt.Errorf("resolve other workspace failed: %w", err)
		}
		if !resp.GetFound() || resp.GetBinding().GetId() != "sgdt-int-other-r0" {
			return fmt.Errorf("expected other workspace binding, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		if _, err := repo.FindApplicableSubscriptionGroupDocumentTemplate(identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-test", WorkspaceID: sgdtTestWS}), &pb.FindApplicableSubscriptionGroupDocumentTemplateRequest{
			RenderProfile:   pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			PriceScheduleId: strptr("sgdt-int-ps"),
			PlanId:          strptr("sgdt-int-plan"),
			JobCategoryId:   strptr("sgdt-int-main-cat"),
			AsOf:            timestamppb.New(boundary),
			DocumentPurpose: "wrong-purpose",
		}); err == nil {
			return fmt.Errorf("wrong purpose should fail")
		} else if !strings.Contains(err.Error(), "document_purpose must be") {
			return fmt.Errorf("wrong purpose error wrong: %v", err)
		}

		if _, err := repo.FindApplicableSubscriptionGroupDocumentTemplate(context.Background(), &pb.FindApplicableSubscriptionGroupDocumentTemplateRequest{
			RenderProfile:   pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			PriceScheduleId: strptr("sgdt-int-ps"),
			PlanId:          strptr("sgdt-int-plan"),
			JobCategoryId:   strptr("sgdt-int-main-cat"),
			AsOf:            timestamppb.New(boundary),
			DocumentPurpose: subscriptionGroupOutcomeSummaryDocumentPurpose,
		}); err == nil {
			return fmt.Errorf("missing workspace identity should fail")
		}

		resp, err = resolve(wsMain, "sgdt-int-plan", "sgdt-int-ps", v1Start.Add(-time.Millisecond))
		if err != nil {
			return fmt.Errorf("unexpected error at pre-range check: %w", err)
		}
		if resp.GetFound() {
			return fmt.Errorf("before as_of start should be no result, got %q", resp.GetBinding().GetId())
		}

		return r
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}

	assertNoSGDTResidue(t, db)
}

func TestIntegration_SubscriptionGroupDocumentTemplate_Publish_ClosesPredecessorAndResolvesAtBoundary(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template").(*PostgresSubscriptionGroupDocumentTemplateRepository)

	r := fmt.Errorf("sgdt-int: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := execSGDTQuery(txCtx, wsOps, q, args...)
			return err
		}
		ex, err := sgdtExecutor(txCtx, wsOps)
		if err != nil {
			return err
		}

		a := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		b := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO job_category (id, workspace_id, name, code) VALUES ('sgdt-int-pub-cat', $1, 'Category', 'cat')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed category: %w", err)
		}
		if err := exec(`INSERT INTO plan (id, workspace_id, name) VALUES ('sgdt-int-pub-plan', $1, 'Plan')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed plan: %w", err)
		}
		if err := exec(`INSERT INTO price_schedule (id, workspace_id, name) VALUES ('sgdt-int-pub-ps', $1, 'PS')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed price schedule: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES
		        ('sgdt-int-pub-dt-1', $1, 'prev', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/pub-1.docx', 'active', true),
		        ('sgdt-int-pub-dt-2', $1, 'next', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/pub-2.docx', 'active', true)`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed templates: %w", err)
		}
		if err := exec(`INSERT INTO subscription_group_document_template
		        (id, workspace_id, document_template_id, render_profile, price_schedule_id, plan_id, job_category_id, version, version_status, validity_start, validity_end, active)
		        VALUES
		        ('sgdt-int-pub-prev', $1, 'sgdt-int-pub-dt-1', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-pub-ps', 'sgdt-int-pub-plan', 'sgdt-int-pub-cat', 1, $2, $3, NULL, true),
		        ('sgdt-int-pub-draft', $1, 'sgdt-int-pub-dt-2', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-pub-ps', 'sgdt-int-pub-plan', 'sgdt-int-pub-cat', 0, $4, $5, NULL, true)`,
			sgdtTestWS, versionStatusPublished, a, versionStatusDraft, b); err != nil {
			return fmt.Errorf("seed publish lineage: %w", err)
		}

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-publisher", WorkspaceID: sgdtTestWS})
		publishResp, err := repo.PublishSubscriptionGroupDocumentTemplate(ctx, &pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "sgdt-int-pub-draft"})
		if err != nil {
			return fmt.Errorf("publish should succeed: %w", err)
		}
		if publishResp.GetData() == nil {
			return fmt.Errorf("publish response missing data")
		}
		if got := publishResp.GetData().GetVersion(); got != 2 {
			return fmt.Errorf("published version should be 2, got %d", got)
		}
		if got := publishResp.GetData().GetSupersedesBindingId(); got != "sgdt-int-pub-prev" {
			return fmt.Errorf("published binding should supersede previous, got %q", got)
		}

		var prevEnd sql.NullTime
		if err := ex.QueryRowContext(txCtx, `SELECT validity_end FROM subscription_group_document_template WHERE id = $1`, "sgdt-int-pub-prev").Scan(&prevEnd); err != nil {
			return fmt.Errorf("read predecessor: %w", err)
		}
		if !prevEnd.Valid || !prevEnd.Time.Equal(b) {
			return fmt.Errorf("predecessor validity_end must close at publish boundary %v, got %v", b, prevEnd.Time)
		}
		resolve := func(asOf time.Time) (*pb.FindApplicableSubscriptionGroupDocumentTemplateResponse, error) {
			return repo.FindApplicableSubscriptionGroupDocumentTemplate(identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-reader", WorkspaceID: sgdtTestWS}), &pb.FindApplicableSubscriptionGroupDocumentTemplateRequest{
				RenderProfile:   pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
				PriceScheduleId: strptr("sgdt-int-pub-ps"),
				PlanId:          strptr("sgdt-int-pub-plan"),
				JobCategoryId:   strptr("sgdt-int-pub-cat"),
				AsOf:            timestamppb.New(asOf),
				DocumentPurpose: subscriptionGroupOutcomeSummaryDocumentPurpose,
			})
		}
		if resp, err := resolve(b.Add(-time.Millisecond)); err != nil {
			return fmt.Errorf("predecessor resolve: %w", err)
		} else if !resp.GetFound() || resp.GetBinding().GetId() != "sgdt-int-pub-prev" {
			return fmt.Errorf("before publish boundary should resolve predecessor, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}
		if resp, err := resolve(b); err != nil {
			return fmt.Errorf("current resolve: %w", err)
		} else if !resp.GetFound() || resp.GetBinding().GetId() != "sgdt-int-pub-draft" {
			return fmt.Errorf("at/after boundary should resolve current, got found=%v id=%q", resp.GetFound(), resp.GetBinding().GetId())
		}

		return r
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}

	assertNoSGDTResidue(t, db)
}

func TestIntegration_SubscriptionGroupDocumentTemplate_Publish_RejectsOverlappingPredecessorsWithoutMutation(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template")

	r := fmt.Errorf("sgdt-int: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := execSGDTQuery(txCtx, wsOps, q, args...)
			return err
		}
		ex, err := sgdtExecutor(txCtx, wsOps)
		if err != nil {
			return err
		}

		t := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		b := t.Add(24 * time.Hour)
		c := b.Add(24 * time.Hour)
		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO job_category (id, workspace_id, name, code) VALUES ('sgdt-int-conflict-cat', $1, 'Category', 'cat')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed category: %w", err)
		}
		if err := exec(`INSERT INTO plan (id, workspace_id, name) VALUES ('sgdt-int-conflict-plan', $1, 'Plan')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed plan: %w", err)
		}
		if err := exec(`INSERT INTO price_schedule (id, workspace_id, name) VALUES ('sgdt-int-conflict-ps', $1, 'PS')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed schedule: %w", err)
		}
		if err := exec(`INSERT INTO document_template (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES ('sgdt-int-conflict-dt-1', $1, 'd1', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/pub-1.docx', 'active', true),
		                ('sgdt-int-conflict-dt-2', $1, 'd2', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/pub-2.docx', 'active', true),
		                ('sgdt-int-conflict-dt-3', $1, 'd3', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/pub-3.docx', 'active', true)`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed templates: %w", err)
		}
		if err := exec(`INSERT INTO subscription_group_document_template
		        (id, workspace_id, document_template_id, render_profile, price_schedule_id, plan_id, job_category_id, version, version_status, validity_start, validity_end, active)
		        VALUES
		        ('sgdt-int-conflict-prev-1', $1, 'sgdt-int-conflict-dt-1', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-conflict-ps', 'sgdt-int-conflict-plan', 'sgdt-int-conflict-cat', 1, $2, $3, $4, true),
		        ('sgdt-int-conflict-prev-2', $1, 'sgdt-int-conflict-dt-2', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-conflict-ps', 'sgdt-int-conflict-plan', 'sgdt-int-conflict-cat', 2, $2, $3, $4, true),
		        ('sgdt-int-conflict-draft', $1, 'sgdt-int-conflict-dt-3', 'RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1', 'sgdt-int-conflict-ps', 'sgdt-int-conflict-plan', 'sgdt-int-conflict-cat', 0, $5, $6, NULL, true)`,
			sgdtTestWS, versionStatusPublished, t, c, versionStatusDraft, b); err != nil {
			return fmt.Errorf("seed conflict lineage: %w", err)
		}

		_, err = repo.PublishSubscriptionGroupDocumentTemplate(identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-publisher", WorkspaceID: sgdtTestWS}), &pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "sgdt-int-conflict-draft"})
		if err == nil {
			return fmt.Errorf("overlapping predecessors should fail publish")
		}
		if !strings.Contains(err.Error(), "ambiguous publication bucket") {
			return fmt.Errorf("expected ambiguous predecessor error, got: %v", err)
		}

		var targetVersion int32
		var targetStatus string
		var targetSup sql.NullString
		if err := ex.QueryRowContext(txCtx, `SELECT version, version_status, supersedes_binding_id FROM subscription_group_document_template WHERE id = 'sgdt-int-conflict-draft'`).Scan(&targetVersion, &targetStatus, &targetSup); err != nil {
			return fmt.Errorf("read target after failed publish: %w", err)
		}
		if targetVersion != 0 || targetStatus != versionStatusDraft || targetSup.Valid {
			return fmt.Errorf("failed publish must keep target draft/unrelated, got version=%d status=%s supersedes=%v", targetVersion, targetStatus, targetSup)
		}
		return r
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}

	assertNoSGDTResidue(t, db)
}

func TestIntegration_SubscriptionGroupDocumentTemplate_Publish_RejectsBackdatedOverlapWithLaterSiblingWithoutMutation(t *testing.T) {
	db := openSGDTIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(wsOps, "subscription_group_document_template")

	r := fmt.Errorf("sgdt-int: intentional rollback")
	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		exec := func(q string, args ...any) error {
			_, err := execSGDTQuery(txCtx, wsOps, q, args...)
			return err
		}
		ex, err := sgdtExecutor(txCtx, wsOps)
		if err != nil {
			return err
		}

		a := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		b := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		c := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		const profile = "RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1"

		if err := exec(`INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed workspace: %w", err)
		}
		if err := exec(`INSERT INTO job_category (id, workspace_id, name, code) VALUES ('sgdt-int-backdated-cat', $1, 'Category', 'cat')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed category: %w", err)
		}
		if err := exec(`INSERT INTO plan (id, workspace_id, name) VALUES ('sgdt-int-backdated-plan', $1, 'Plan')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed plan: %w", err)
		}
		if err := exec(`INSERT INTO price_schedule (id, workspace_id, name) VALUES ('sgdt-int-backdated-ps', $1, 'PS')`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed price schedule: %w", err)
		}
		if err := exec(`INSERT INTO document_template
		        (id, workspace_id, name, template_type, document_purpose, storage_container, storage_key, status, active)
		        VALUES
		        ('sgdt-int-backdated-dt-a', $1, 'A', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/backdated-a.docx', 'active', true),
		        ('sgdt-int-backdated-dt-b', $1, 'B', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/backdated-b.docx', 'active', true),
		        ('sgdt-int-backdated-dt-c', $1, 'C', 'docx', 'subscription_group_outcome_summary', 'templates', 'templates/backdated-c.docx', 'active', true)`, sgdtTestWS); err != nil {
			return fmt.Errorf("seed templates: %w", err)
		}
		if err := exec(`INSERT INTO subscription_group_document_template
		        (id, workspace_id, document_template_id, render_profile, price_schedule_id, plan_id, job_category_id,
		         version, version_status, validity_start, validity_end, supersedes_binding_id, active, published_at, published_by)
		        VALUES
		        ('sgdt-int-backdated-a', $1, 'sgdt-int-backdated-dt-a', $2, 'sgdt-int-backdated-ps', 'sgdt-int-backdated-plan', 'sgdt-int-backdated-cat',
		         1, $3, $4, NULL, NULL, true, 101, 'seed-a'),
		        ('sgdt-int-backdated-b', $1, 'sgdt-int-backdated-dt-b', $2, 'sgdt-int-backdated-ps', 'sgdt-int-backdated-plan', 'sgdt-int-backdated-cat',
		         2, $3, $5, NULL, 'sgdt-int-backdated-a', true, 102, 'seed-b'),
		        ('sgdt-int-backdated-c', $1, 'sgdt-int-backdated-dt-c', $2, 'sgdt-int-backdated-ps', 'sgdt-int-backdated-plan', 'sgdt-int-backdated-cat',
		         0, $6, $7, NULL, NULL, true, NULL, NULL)`,
			sgdtTestWS, profile, versionStatusPublished, a, b, versionStatusDraft, c); err != nil {
			return fmt.Errorf("seed backdated sibling lineage: %w", err)
		}

		_, err = repo.PublishSubscriptionGroupDocumentTemplate(
			identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "sgdt-publisher", WorkspaceID: sgdtTestWS}),
			&pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "sgdt-int-backdated-c"},
		)
		if err == nil {
			return fmt.Errorf("backdated publication overlapping a later sibling should fail")
		}
		if !strings.Contains(err.Error(), "publication interval overlaps a later published sibling") {
			return fmt.Errorf("expected later-sibling overlap error, got: %v", err)
		}

		type bindingSnapshot struct {
			status     string
			version    int32
			start      sql.NullTime
			end        sql.NullTime
			supersedes sql.NullString
		}
		readSnapshot := func(id string) (bindingSnapshot, error) {
			var got bindingSnapshot
			err := ex.QueryRowContext(txCtx, `SELECT version_status, version, validity_start, validity_end, supersedes_binding_id
			        FROM subscription_group_document_template WHERE id = $1`, id).Scan(
				&got.status, &got.version, &got.start, &got.end, &got.supersedes)
			return got, err
		}
		assertSnapshot := func(id string, got bindingSnapshot, wantStatus string, wantVersion int32, wantStart time.Time, wantSupersedes sql.NullString) error {
			if got.status != wantStatus || got.version != wantVersion || !got.start.Valid || !got.start.Time.Equal(wantStart) || got.end.Valid || got.supersedes != wantSupersedes {
				return fmt.Errorf("%s mutated after rejected publish: status=%s version=%d start=%v end=%v supersedes=%v", id, got.status, got.version, got.start, got.end, got.supersedes)
			}
			return nil
		}
		if got, err := readSnapshot("sgdt-int-backdated-a"); err != nil {
			return fmt.Errorf("read A after failed publish: %w", err)
		} else if err := assertSnapshot("A", got, versionStatusPublished, 1, a, sql.NullString{}); err != nil {
			return err
		}
		if got, err := readSnapshot("sgdt-int-backdated-b"); err != nil {
			return fmt.Errorf("read B after failed publish: %w", err)
		} else if err := assertSnapshot("B", got, versionStatusPublished, 2, b, sql.NullString{String: "sgdt-int-backdated-a", Valid: true}); err != nil {
			return err
		}

		var target struct {
			active      bool
			status      string
			version     int32
			start       sql.NullTime
			end         sql.NullTime
			supersedes  sql.NullString
			publishedAt sql.NullInt64
			publishedBy sql.NullString
		}
		if err := ex.QueryRowContext(txCtx, `SELECT active, version_status, version, validity_start, validity_end,
		        supersedes_binding_id, published_at, published_by
		        FROM subscription_group_document_template WHERE id = 'sgdt-int-backdated-c'`).Scan(
			&target.active, &target.status, &target.version, &target.start, &target.end,
			&target.supersedes, &target.publishedAt, &target.publishedBy); err != nil {
			return fmt.Errorf("read C after failed publish: %w", err)
		}
		if !target.active || target.status != versionStatusDraft || target.version != 0 || !target.start.Valid || !target.start.Time.Equal(c) || target.end.Valid || target.supersedes.Valid || target.publishedAt.Valid || target.publishedBy.Valid {
			return fmt.Errorf("C mutated after rejected publish: active=%v status=%s version=%d start=%v end=%v supersedes=%v published_at=%v published_by=%v", target.active, target.status, target.version, target.start, target.end, target.supersedes, target.publishedAt, target.publishedBy)
		}
		return r
	})
	if err == nil || !strings.Contains(err.Error(), "intentional rollback") {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}

	assertNoSGDTResidue(t, db)
}
