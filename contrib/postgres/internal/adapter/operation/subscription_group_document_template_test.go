//go:build postgresql

package operation

import (
	"strings"
	"testing"
	"time"

	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

func TestSubscriptionGroupDocumentTemplateMapperPreservesBigintAuditFields(t *testing.T) {
	t.Parallel()
	validityStart := time.Date(2026, 8, 10, 1, 2, 3, 0, time.UTC).UnixMilli()
	validityEnd := validityStart + int64(time.Hour/time.Millisecond)
	publishedAt := int64(1_754_788_923_000)
	dateCreated := publishedAt - 1000
	dateModified := publishedAt + 1000
	result := map[string]any{
		"id":                   "binding-1",
		"workspace_id":         "workspace-1",
		"document_template_id": "document-template-1",
		"render_profile":       pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1.String(),
		"version":              float64(1),
		"version_status":       enums.VersionStatus_VERSION_STATUS_PUBLISHED.String(),
		"validity_start":       validityStart,
		"validity_end":         validityEnd,
		"published_at":         publishedAt,
		"date_created":         dateCreated,
		"date_modified":        dateModified,
		"active":               true,
	}

	item, err := subscriptionGroupDocumentTemplateFromResult(result)
	if err != nil {
		t.Fatalf("map binding result: %v", err)
	}
	if got := item.GetValidityStart().AsTime().UTC().UnixMilli(); got != validityStart {
		t.Fatalf("validity_start=%d; want %d", got, validityStart)
	}
	if got := item.GetValidityEnd().AsTime().UTC().UnixMilli(); got != validityEnd {
		t.Fatalf("validity_end=%d; want %d", got, validityEnd)
	}
	if item.GetPublishedAt() != publishedAt || item.GetDateCreated() != dateCreated || item.GetDateModified() != dateModified {
		t.Fatalf("audit millis changed: published=%d created=%d modified=%d", item.GetPublishedAt(), item.GetDateCreated(), item.GetDateModified())
	}
	for _, field := range []string{"published_at", "date_created", "date_modified"} {
		if _, ok := result[field].(int64); !ok {
			t.Fatalf("%s was converted away from bigint: %T", field, result[field])
		}
	}
}

func TestFindApplicableSubscriptionGroupDocumentTemplateSQLRanksAllFallbackBuckets(t *testing.T) {
	t.Parallel()
	query := strings.Join(strings.Fields(findApplicableSubscriptionGroupDocumentTemplateSQL()), " ")
	for _, required := range []string{
		"b.plan_id IS NULL AND b.price_schedule_id = rs.price_schedule_id THEN 2",
		"b.plan_id IS NULL AND b.price_schedule_id IS NULL THEN 3",
		"NULLIF(btrim(dt.storage_container), '') IS NOT NULL",
		"NULLIF(btrim(dt.storage_key), '') IS NOT NULL",
	} {
		if !strings.Contains(query, required) {
			t.Fatalf("resolver SQL missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"rs.plan_id IS NULL AND b.plan_id IS NULL",
		"rs.price_schedule_id IS NULL AND b.price_schedule_id IS NULL THEN 3",
	} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("resolver SQL retains broken fallback predicate %q", forbidden)
		}
	}
}
