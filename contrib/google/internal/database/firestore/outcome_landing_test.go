package firestore

import (
	"testing"

	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

func landingDoc(values ...any) firestoreLandingDocument {
	result := firestoreLandingDocument{}
	for index := 0; index+1 < len(values); index += 2 {
		result[values[index].(string)] = values[index+1]
	}
	return result
}

func TestProjectFirestoreOutcomeLandingWorkspaceWideActiveAndHistorical(t *testing.T) {
	activeOnly := true
	snapshot := firestoreLandingSnapshot{
		priceSchedules: []firestoreLandingDocument{
			landingDoc("id", "ps-z", "name", "Zulu", "active", false),
			landingDoc("id", "ps-a", "name", "Alpha", "active", true, "sort_order", int64(1)),
		},
		subscriptionGroups: []firestoreLandingDocument{
			landingDoc("id", "g-z", "price_schedule_id", "ps-z", "name", "Historic", "active", false),
			landingDoc("id", "g-a", "price_schedule_id", "ps-a", "name", "Current", "active", true),
		},
		members: []firestoreLandingDocument{
			landingDoc("id", "m-a1", "subscription_group_id", "g-a", "subscription_id", "s-a1", "client_id", "c-a1", "active", true),
			landingDoc("id", "m-a2", "subscription_group_id", "g-a", "subscription_id", "s-a2", "client_id", "c-a2", "active", true),
			landingDoc("id", "m-z", "subscription_group_id", "g-z", "subscription_id", "s-z", "client_id", "c-z", "active", false),
		},
		subscriptions: []firestoreLandingDocument{
			landingDoc("id", "s-a1", "client_id", "c-a1", "active", true),
			landingDoc("id", "s-a2", "client_id", "c-a2", "active", true),
			landingDoc("id", "s-z", "client_id", "c-z", "active", false),
		},
		jobs: []firestoreLandingDocument{
			landingDoc("id", "j-a1", "origin_type", landingOriginSubscription, "origin_id", "s-a1", "client_id", "c-a1", "job_template_id", "t-a", "active", true),
			landingDoc("id", "j-a2", "origin_type", landingOriginSubscription, "origin_id", "s-a2", "client_id", "c-a2", "job_template_id", "t-a", "active", true),
			landingDoc("id", "j-z", "origin_type", landingOriginSubscription, "origin_id", "s-z", "client_id", "c-z", "job_template_id", "t-retired", "active", false),
		},
		jobTemplates: []firestoreLandingDocument{
			landingDoc("id", "t-a", "active", true),
		},
	}
	requestIdentity := &identity.RequestIdentity{WorkspaceID: "ws"}

	rows := projectFirestoreOutcomeLanding(snapshot, requestIdentity, &exportpb.ListSubscriptionGroupOutcomeLandingRequest{}, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].GetSubscriptionGroupId() != "g-a" || rows[0].GetMemberCount() != 2 || rows[0].GetJobTemplateCount() != 1 {
		t.Fatalf("current row = %+v", rows[0])
	}
	if rows[1].GetSubscriptionGroupId() != "g-z" || rows[1].GetMemberCount() != 1 || rows[1].GetJobTemplateCount() != 1 {
		t.Fatalf("historical row = %+v", rows[1])
	}

	rows = projectFirestoreOutcomeLanding(snapshot, requestIdentity, &exportpb.ListSubscriptionGroupOutcomeLandingRequest{PriceScheduleActive: &activeOnly}, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true})
	if len(rows) != 1 || rows[0].GetSubscriptionGroupId() != "g-a" {
		t.Fatalf("active rows = %+v, want g-a only", rows)
	}
}

func TestProjectFirestoreOutcomeLandingStaffRequiresExactMembershipGrantAndReachableJob(t *testing.T) {
	snapshot := firestoreLandingSnapshot{
		priceSchedules:     []firestoreLandingDocument{landingDoc("id", "ps", "name", "Current", "active", true)},
		subscriptionGroups: []firestoreLandingDocument{landingDoc("id", "g", "price_schedule_id", "ps", "name", "Section", "active", true)},
		workspaceUsers: []firestoreLandingDocument{
			landingDoc("id", "wu", "workspace_id", "ws", "user_id", "user", "active", true),
		},
		subscriptionGroupWorkspaceUsers: []firestoreLandingDocument{
			landingDoc("id", "grant", "workspace_id", "ws", "workspace_user_id", "wu", "subscription_group_id", "g", "active", true),
		},
		members:       []firestoreLandingDocument{landingDoc("id", "m", "subscription_group_id", "g", "subscription_id", "s", "client_id", "c", "active", true)},
		subscriptions: []firestoreLandingDocument{landingDoc("id", "s", "client_id", "c", "active", true)},
		jobs: []firestoreLandingDocument{
			landingDoc("id", "j", "origin_type", landingOriginSubscription, "origin_id", "s", "client_id", "c", "job_template_id", "t", "output_product_id", "product", "active", true),
		},
		jobTemplates: []firestoreLandingDocument{landingDoc("id", "t", "output_product_id", "product", "active", true)},
		productPlans: []firestoreLandingDocument{
			landingDoc("id", "pp", "product_id", "product", "active", true),
		},
		classEdges: []firestoreLandingDocument{
			landingDoc("id", "edge", "subscription_group_id", "g", "product_plan_id", "pp", "product_plan_staff_id", "pps", "staff_id", "legacy", "active", true),
		},
		productPlanStaffs: []firestoreLandingDocument{
			landingDoc("id", "pps", "staff_id", "staff", "active", true),
		},
	}
	requestIdentity := &identity.RequestIdentity{
		WorkspaceID:     "ws",
		WorkspaceUserID: "wu",
		UserID:          "user",
		PrincipalType:   landingPrincipalTypeStaff,
		PrincipalID:     "staff",
	}

	rows := projectFirestoreOutcomeLanding(snapshot, requestIdentity, &exportpb.ListSubscriptionGroupOutcomeLandingRequest{}, ports.SubscriptionGroupOutcomeExportScope{})
	if len(rows) != 1 || rows[0].GetMemberCount() != 1 || rows[0].GetJobTemplateCount() != 1 {
		t.Fatalf("staff rows = %+v, want one granted/reachable row", rows)
	}

	missingMembership := *requestIdentity
	missingMembership.WorkspaceUserID = ""
	if rows := projectFirestoreOutcomeLanding(snapshot, &missingMembership, &exportpb.ListSubscriptionGroupOutcomeLandingRequest{}, ports.SubscriptionGroupOutcomeExportScope{}); len(rows) != 0 {
		t.Fatalf("missing membership rows = %+v, want none", rows)
	}

	snapshot.productPlanStaffs[0]["active"] = false
	rows = projectFirestoreOutcomeLanding(snapshot, requestIdentity, &exportpb.ListSubscriptionGroupOutcomeLandingRequest{}, ports.SubscriptionGroupOutcomeExportScope{})
	if len(rows) != 1 || rows[0].GetJobTemplateCount() != 0 {
		t.Fatalf("revoked eligibility rows = %+v, want visible group with zero reachable templates", rows)
	}
}

func TestFirestoreOutcomeLandingUsesConfiguredCollectionNames(t *testing.T) {
	tables := registry.NewTableConfig("tenant_", map[string]string{entityid.Job: "work"})
	query := newFirestoreSubscriptionGroupOutcomeLandingQuery(nil, tables)
	if got := query.tables.TableName(entityid.Job); got != "tenant_work" {
		t.Fatalf("job collection = %q, want tenant_work", got)
	}
	if got := query.tables.TableName(entityid.SubscriptionGroup); got != "tenant_subscription_group" {
		t.Fatalf("subscription group collection = %q, want tenant_subscription_group", got)
	}
}
