package schema

import (
	"testing"

	sgppspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// TestClassifySubscriptionGroupProductPlanStaffV2Columns is the M2 adapter
// prerequisite (docs/plan/20260724-section-assignment-merged/espyna.md §1b):
// "register the three NEW sgpps columns in the postgres adapter's filter map
// so List can filter on subscription_group_product_plan_id /
// product_plan_staff_id / job_template_phase_id".
//
// This codebase's reflectionless CRUD path (Plan 2,
// docs/plan/20260530-reflectionless-crud/) derives the persisted/filterable
// column set entirely from the proto descriptor via Classify() — there is no
// separate hand-maintained per-entity filter allowlist to edit (verified:
// PostgresOperations.List / buildFilterConditions validate a filter Field only
// through ValidateSQLIdent, accepting any column the schema registry knows
// about). So the "registration" this prerequisite asks for is automatically
// satisfied the moment the proto fields exist (already true — the sgpps proto
// carries f12/f13/f14 as plain optional string scalars, esqyma.md §1). This
// test pins that fact as a regression guard: if a future proto edit ever moves
// these fields behind a nested message or a (db).ignore annotation, List
// filtering on them would silently stop working and this test would catch it.
func TestClassifySubscriptionGroupProductPlanStaffV2Columns(t *testing.T) {
	md := (&sgppspb.SubscriptionGroupProductPlanStaff{}).ProtoReflect().Descriptor()
	cols := Classify(md)
	have := names(cols)

	for _, want := range []string{
		"subscription_group_product_plan_id", // f12 — the class FK
		"product_plan_staff_id",              // f13 — the eligibility FK
		"job_template_phase_id",              // f14 — the optional phase-scope FK
	} {
		if !have[want] {
			t.Errorf("v2 class-edge column %q must be classified (and therefore List-filterable) — got %v", want, have)
		}
	}

	// Sanity: the legacy dual-write triple stays classified too — the M2 dual-
	// write depends on these columns remaining real, writable columns.
	for _, want := range []string{"subscription_group_id", "product_plan_id", "staff_id"} {
		if !have[want] {
			t.Errorf("legacy dual-write column %q must remain classified — got %v", want, have)
		}
	}
}
