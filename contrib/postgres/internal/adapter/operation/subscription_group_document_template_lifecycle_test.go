//go:build postgresql

package operation

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

// Canonical rollout for the subscription-group binding query/publish SQL.

func TestFindApplicableSubscriptionGroupDocumentTemplateSQL_BucketsAndStorefrontGuards(t *testing.T) {
	q := strings.Join(strings.Fields(findApplicableSubscriptionGroupDocumentTemplateSQL()), " ")

	for _, required := range []string{
		"b.job_category_id IS NOT NULL",
		"b.job_category_id = rs.job_category_id",
		"b.plan_id = rs.plan_id AND b.price_schedule_id = rs.price_schedule_id THEN 0",
		"b.plan_id = rs.plan_id AND b.price_schedule_id IS NULL THEN 1",
		"b.plan_id IS NULL AND b.price_schedule_id = rs.price_schedule_id THEN 2",
		"b.plan_id IS NULL AND b.price_schedule_id IS NULL THEN 3",
		"ORDER BY match_rank, b.version DESC",
		"LIMIT 2",
		"AND (b.validity_start IS NULL OR b.validity_start <= $7)",
		"AND (b.validity_end IS NULL OR $7 < b.validity_end)",
		"dt.document_purpose = $8",
		"NULLIF(btrim(dt.storage_container), '') IS NOT NULL",
		"NULLIF(btrim(dt.storage_key), '') IS NOT NULL",
	} {
		if !strings.Contains(q, required) {
			t.Fatalf("applicability SQL missing %q", required)
		}
	}
}

func TestSubscriptionGroupDocumentTemplatePublishFlipSQL_IsDraftOnlyAndTransactional(t *testing.T) {
	q := subscriptionGroupDocumentTemplatePublishFlipSQL()
	for _, required := range []string{
		"version_status = $1",
		"version = $2",
		"published_at = $3",
		"published_by = $4",
		"validity_start = COALESCE(validity_start, $5)",
		"supersedes_binding_id = $6",
		"WHERE id = $7 AND workspace_id = $8",
		"AND version_status = $9",
	} {
		if !strings.Contains(q, required) {
			t.Fatalf("publish SQL missing %q", required)
		}
	}

	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(&recordingDBOps{}, "subscription_group_document_template")
	_, err := repo.PublishSubscriptionGroupDocumentTemplate(
		identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "sgdt-lifecycle-ws", UserID: "u"}),
		&pb.PublishSubscriptionGroupDocumentTemplateRequest{Id: "sgdt-lifecycle-b"},
	)
	if err == nil {
		t.Fatal("publish must fail outside an active transaction")
	}
	if !strings.Contains(err.Error(), "publish requires an active transaction") {
		t.Fatalf("expected transaction gate, got %v", err)
	}
}

func TestFindApplicableSubscriptionGroupDocumentTemplate_WithoutWorkspaceIdentityFailsClosed(t *testing.T) {
	repo := NewPostgresSubscriptionGroupDocumentTemplateRepository(&recordingDBOps{}, "subscription_group_document_template")
	_, err := repo.FindApplicableSubscriptionGroupDocumentTemplate(context.Background(), &pb.FindApplicableSubscriptionGroupDocumentTemplateRequest{
		RenderProfile:   pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
		JobCategoryId:   strptr("cat-test"),
		PlanId:          strptr("pl-test"),
		PriceScheduleId: strptr("ps-test"),
		DocumentPurpose: subscriptionGroupOutcomeSummaryDocumentPurpose,
	})
	if err == nil {
		t.Fatal("findable must fail closed without workspace identity")
	}
	if !strings.Contains(err.Error(), "workspace identity required") && !strings.Contains(err.Error(), "binding resolver requires direct SQL access") {
		t.Fatalf("expected workspace-related fail-closed, got: %v", err)
	}
}
