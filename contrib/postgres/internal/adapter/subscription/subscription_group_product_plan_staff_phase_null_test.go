//go:build postgresql

package subscription

import (
	"context"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// capturingOps is a DB-free stand-in for interfaces.DatabaseOperation that records
// the payload map an adapter hands to Update(). Only Update is exercised; every
// other method is a hard failure so a test that accidentally takes a different
// code path is loud rather than silently green.
type capturingOps struct {
	t       *testing.T
	updated map[string]any
}

func (c *capturingOps) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	c.updated = data
	return map[string]any{"id": id}, nil
}

func (c *capturingOps) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	c.t.Fatalf("unexpected Create on capturingOps")
	return nil, nil
}

func (c *capturingOps) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	c.t.Fatalf("unexpected Read on capturingOps")
	return nil, nil
}

func (c *capturingOps) Delete(ctx context.Context, tableName string, id string) error {
	c.t.Fatalf("unexpected Delete on capturingOps")
	return nil
}

func (c *capturingOps) HardDelete(ctx context.Context, tableName string, id string) error {
	c.t.Fatalf("unexpected HardDelete on capturingOps")
	return nil
}

func (c *capturingOps) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	c.t.Fatalf("unexpected List on capturingOps")
	return nil, nil
}

func (c *capturingOps) Query(ctx context.Context, tableName string, query interfaces.QueryBuilder) ([]map[string]any, error) {
	c.t.Fatalf("unexpected Query on capturingOps")
	return nil, nil
}

func (c *capturingOps) QueryOne(ctx context.Context, tableName string, query interfaces.QueryBuilder) (map[string]any, error) {
	c.t.Fatalf("unexpected QueryOne on capturingOps")
	return nil, nil
}

func strptr(s string) *string { return &s }

// phaseValuesReaching collects the values of every key in an adapter write payload
// that canonicalizes to the job_template_phase_id COLUMN.
//
// The canonicalization is deliberately the SAME one the write path applies:
// PostgresOperations.Update() runs normalizeKeys (camelToSnake, exposed here as
// postgresCore.CamelToSnake) over the payload before building the SET clause, so
// the protojson key "jobTemplatePhaseId" and the literal key
// "job_template_phase_id" both address the one FK column. Asserting on the raw
// key spelling would let a payload that carries BOTH spellings pass while the
// value that actually reaches SQL depends on Go's randomized map iteration order.
func phaseValuesReaching(data map[string]any) []any {
	var vals []any
	for k, v := range data {
		if postgresCore.CamelToSnake(k) == "job_template_phase_id" {
			vals = append(vals, v)
		}
	}
	return vals
}

// TestUpdateSubscriptionGroupProductPlanStaff_PhaseNullTranslation is the Bug #3
// regression guard (audit gap W-G1, go-unit half).
//
// job_template_phase_id is a proto3 `optional string` FK column where NULL means
// "all phases". protojson has no scalar-null form: a caller that deliberately
// CLEARS the phase (a non-nil pointer to "") marshals as the JSON string "", and
// PostgresOperations.Update() writes present keys verbatim — so the literal empty
// string hits the FK constraint and 500s the ordinary "clear phase back to All
// Phases" drawer edit. The adapter is the one place that still holds the caller's
// ORIGINAL pointer, so it must translate non-nil-but-empty into a real Go nil,
// the only value that serializes to SQL NULL.
//
// The three pointer states are the entire input space of that translation.
func TestUpdateSubscriptionGroupProductPlanStaff_PhaseNullTranslation(t *testing.T) {
	tests := []struct {
		name string
		// phase is the caller's JobTemplatePhaseId pointer on the update request.
		phase *string
		// wantPresent is whether the phase column should appear in the payload at
		// all. A partial update that never mentioned the field must not touch it.
		wantPresent bool
		// wantValue is the value that must reach the column when present.
		// Go nil is the ONLY representation that becomes SQL NULL.
		wantValue any
	}{
		{
			name:        "nil pointer leaves the phase column untouched",
			phase:       nil,
			wantPresent: false,
		},
		{
			name:        "non-nil empty pointer clears the phase to SQL NULL",
			phase:       strptr(""),
			wantPresent: true,
			wantValue:   nil,
		},
		{
			name:        "non-nil populated pointer writes the phase id verbatim",
			phase:       strptr("jtp-scope-1"),
			wantPresent: true,
			wantValue:   "jtp-scope-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ops := &capturingOps{t: t}
			repo := NewPostgresSubscriptionGroupProductPlanStaffRepository(ops, "subscription_group_product_plan_staff")

			_, err := repo.UpdateSubscriptionGroupProductPlanStaff(context.Background(), &pb.UpdateSubscriptionGroupProductPlanStaffRequest{
				Data: &pb.SubscriptionGroupProductPlanStaff{
					Id:                 "sgpps-1",
					JobTemplatePhaseId: tc.phase,
				},
			})
			if err != nil {
				t.Fatalf("UpdateSubscriptionGroupProductPlanStaff: %v", err)
			}
			if ops.updated == nil {
				t.Fatalf("adapter never called dbOps.Update — nothing was captured")
			}

			vals := phaseValuesReaching(ops.updated)

			if !tc.wantPresent {
				if len(vals) != 0 {
					t.Fatalf("Bug #3 regression guard: a nil JobTemplatePhaseId pointer must leave job_template_phase_id out of the update payload entirely (partial update must not touch an unmentioned column), but the payload carries %#v (payload=%#v)", vals, ops.updated)
				}
				return
			}

			if len(vals) == 0 {
				t.Fatalf("Bug #3 regression guard: JobTemplatePhaseId=%q was set by the caller but no key in the update payload addresses job_template_phase_id — the write would be silently dropped (payload=%#v)", *tc.phase, ops.updated)
			}
			// More than one key canonicalizing to the same column means the value
			// that reaches SQL is decided by randomized map iteration order.
			if len(vals) > 1 {
				t.Fatalf("Bug #3 regression guard: %d keys in the update payload canonicalize to job_template_phase_id (%#v) — normalizeKeys collapses them and Go's randomized map order decides which value reaches SQL, so the NULL translation is non-deterministic (payload=%#v)", len(vals), vals, ops.updated)
			}
			if vals[0] != tc.wantValue {
				if tc.wantValue == nil {
					t.Fatalf("Bug #3 regression guard: a non-nil-but-empty JobTemplatePhaseId pointer (clear phase back to All Phases) must reach the write path as Go nil — the only value that serializes to SQL NULL — but got %#v; the literal empty string violates the job_template_phase_id FK and 500s the drawer edit", vals[0])
				}
				t.Fatalf("Bug #3 regression guard: job_template_phase_id = %#v, want %#v — a populated phase pointer must be written verbatim and must NOT be swept into the NULL translation", vals[0], tc.wantValue)
			}
		})
	}
}
