package subscription

import (
	"context"
	"testing"

	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
	planjobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/plan_job_template"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func bundleEntry(id, template string, order int32) *planjobtemplatepb.PlanJobTemplate {
	return &planjobtemplatepb.PlanJobTemplate{Id: id, Active: true, PlanId: "plan-1", JobTemplateId: template, SequenceOrder: order, CompositionEntryPattern: planjobtemplatepb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_BUNDLE_ENTRY}
}
func standaloneEntry(id, template string) *planjobtemplatepb.PlanJobTemplate {
	return &planjobtemplatepb.PlanJobTemplate{Id: id, Active: true, PlanId: "plan-1", JobTemplateId: template, CompositionEntryPattern: planjobtemplatepb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_STANDALONE_ENTRY}
}

func TestMaterializeJobs_PCS03_CompositionPreferredOverLegacy(t *testing.T) {
	f := newFixture(t, fixtureOpts{planJobTemplateID: "legacy", billingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME, composition: []*planjobtemplatepb.PlanJobTemplate{bundleEntry("e1", "course", 1)}, templates: map[string]*jobtemplatepb.JobTemplate{"legacy": makeTemplate("legacy", "Legacy", true), "course": makeTemplate("course", "Course", true)}})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetSpawnedJobs()) != 1 || resp.GetSpawnedJobs()[0].GetJobTemplateId() != "course" {
		t.Fatalf("composition was not preferred: %+v", resp.GetSpawnedJobs())
	}
}

func TestMaterializeJobs_PCS04_BundleEntriesParentlessAndOrdered(t *testing.T) {
	f := newFixture(t, fixtureOpts{billingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME, composition: []*planjobtemplatepb.PlanJobTemplate{bundleEntry("e1", "math", 1), bundleEntry("e2", "science", 2)}, templates: map[string]*jobtemplatepb.JobTemplate{"math": makeTemplate("math", "Math", true), "science": makeTemplate("science", "Science", true)}})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetSpawnedJobs()) != 2 {
		t.Fatalf("jobs=%d", len(resp.GetSpawnedJobs()))
	}
	for _, job := range resp.GetSpawnedJobs() {
		if job.GetParentJobId() != "" || job.GetOriginId() != "sub-1" {
			t.Fatalf("bundle job not co-equal: %+v", job)
		}
	}
	if resp.GetSpawnedJobs()[0].GetJobTemplateId() != "math" || resp.GetSpawnedJobs()[1].GetJobTemplateId() != "science" {
		t.Fatalf("wrong order")
	}
}

func TestMaterializeJobs_PCS05_StandaloneKeepsRealChildren(t *testing.T) {
	f := newFixture(t, fixtureOpts{billingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME, composition: []*planjobtemplatepb.PlanJobTemplate{standaloneEntry("e1", "root")}, relations: []*jobtemplaterelationpb.JobTemplateRelation{makeRelation("root", "child", 1)}, templates: map[string]*jobtemplatepb.JobTemplate{"root": makeTemplate("root", "Root", true), "child": makeTemplate("child", "Child", true)}})
	resp, err := f.uc.Execute(context.Background(), &subscriptionpb.MaterializeJobsForSubscriptionRequest{SubscriptionId: "sub-1", SpawnJobs: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetSpawnedJobs()) != 2 || resp.GetSpawnedJobs()[1].GetParentJobId() != resp.GetSpawnedJobs()[0].GetId() {
		t.Fatalf("standalone graph changed: %+v", resp.GetSpawnedJobs())
	}
}

func TestMaterializeInstanceJobs_PCS04_StandaloneCompositionResolvesWithoutLegacyRoot(t *testing.T) {
	f := newInstFixture(t, instFixtureOpts{subActive: true, billingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING, billingCycleValue: 1, billingCycleUnit: "month", visitsPerCycle: 1, composition: []*planjobtemplatepb.PlanJobTemplate{standaloneEntry("e1", "visit")}, templates: map[string]*jobtemplatepb.JobTemplate{"visit": makeTemplate("visit", "Visit", true)}})
	resp, err := f.uc.executeInternal(context.Background(), materializeInstanceJobsInternalRequest{SubscriptionId: "sub-1", CyclePeriodStart: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.SpawnedCycles) != 1 || len(resp.SpawnedCycles[0].Jobs) != 1 || resp.SpawnedCycles[0].Jobs[0].GetJobTemplateId() != "visit" {
		t.Fatalf("instance composition failed: %+v", resp)
	}
}
