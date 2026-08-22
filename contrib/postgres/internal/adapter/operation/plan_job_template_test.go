//go:build postgresql

package operation

import (
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/plan_job_template"
	"testing"
)

func TestPostgresPlanJobTemplateListByPlanOrder(t *testing.T) {
	rows := filterAndSortPlanJobTemplates([]*pb.PlanJobTemplate{{Id: "b", Active: true, PlanId: "p", SequenceOrder: 2}, {Id: "a", Active: true, PlanId: "p", SequenceOrder: 1}, {Id: "x", Active: true, PlanId: "q"}}, "p")
	if len(rows) != 2 || rows[0].GetId() != "a" {
		t.Fatalf("unexpected order: %+v", rows)
	}
}
