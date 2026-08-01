//go:build sqlserver

package operation

import (
	"context"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// Delivery-group narrow (ListJobPhasesRequest field 5) on the SQL Server
// provider — the locked completion contract's defense-in-depth item: this
// provider implements neither the group predicate nor pagination on
// ListJobPhases, so a NON-EMPTY narrow must be an explicit ERROR (never a
// silently unnarrowed Success: true), while an absent/empty narrow keeps the
// legacy path byte-identical (filters-only ListParams, exactly one List call).

// rejectFakeDBOps records List invocations and returns a fixed row set.
type rejectFakeDBOps struct {
	rows           []map[string]any
	listCalls      int
	lastListParams *interfaces.ListParams
}

func (f *rejectFakeDBOps) Create(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *rejectFakeDBOps) Read(context.Context, string, string) (map[string]any, error) {
	return nil, nil
}
func (f *rejectFakeDBOps) Update(context.Context, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *rejectFakeDBOps) Delete(context.Context, string, string) error     { return nil }
func (f *rejectFakeDBOps) HardDelete(context.Context, string, string) error { return nil }
func (f *rejectFakeDBOps) List(_ context.Context, _ string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	f.listCalls++
	f.lastListParams = params
	return &interfaces.ListResult{Data: f.rows}, nil
}
func (f *rejectFakeDBOps) Query(context.Context, string, interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, nil
}
func (f *rejectFakeDBOps) QueryOne(context.Context, string, interfaces.QueryBuilder) (map[string]any, error) {
	return nil, nil
}

func narrowStrPtr(s string) *string { return &s }

// TestSQLServerListJobPhases_NonEmptyNarrowIsRejected: a non-empty field 5 must
// error BEFORE any List call — an ignored narrow under Success: true is the
// exact fail-open this rejection exists to make unrepresentable.
func TestSQLServerListJobPhases_NonEmptyNarrowIsRejected(t *testing.T) {
	fake := &rejectFakeDBOps{rows: []map[string]any{{"id": "p1"}}}
	r := NewSQLServerJobPhaseRepository(fake, "job_phase")

	resp, err := r.ListJobPhases(context.Background(), &pb.ListJobPhasesRequest{
		SubscriptionGroupId: narrowStrPtr("grp-A"),
	})
	if err == nil {
		t.Fatalf("non-empty narrow must be refused on this provider, got success with %d rows", len(resp.GetData()))
	}
	if resp != nil {
		t.Errorf("a refused narrow must not return a response, got %+v", resp)
	}
	if fake.listCalls != 0 {
		t.Errorf("must refuse BEFORE listing, got %d List calls", fake.listCalls)
	}
}

// TestSQLServerListJobPhases_AbsentOrEmptyNarrowKeepsLegacyPath: with the field
// absent or present-but-empty the provider must issue exactly one List whose
// params carry ONLY the caller's Filters pointer (this method has never
// forwarded pagination/sort) — byte-identical legacy behavior.
func TestSQLServerListJobPhases_AbsentOrEmptyNarrowKeepsLegacyPath(t *testing.T) {
	filters := &commonpb.FilterRequest{}
	for _, tc := range []struct {
		name string
		req  *pb.ListJobPhasesRequest
	}{
		{"field absent", &pb.ListJobPhasesRequest{Filters: filters}},
		{"field present but empty", &pb.ListJobPhasesRequest{Filters: filters, SubscriptionGroupId: narrowStrPtr("")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &rejectFakeDBOps{rows: []map[string]any{{"id": "p1"}, {"id": "p2"}}}
			r := NewSQLServerJobPhaseRepository(fake, "job_phase")

			resp, err := r.ListJobPhases(context.Background(), tc.req)
			if err != nil {
				t.Fatalf("unnarrowed list must not error: %v", err)
			}
			if fake.listCalls != 1 {
				t.Fatalf("expected exactly one List, got %d", fake.listCalls)
			}
			if fake.lastListParams == nil || fake.lastListParams.Filters != filters {
				t.Errorf("params must forward the caller's exact Filters pointer, got %+v", fake.lastListParams)
			}
			if fake.lastListParams.Pagination != nil || fake.lastListParams.Sort != nil || fake.lastListParams.Search != nil {
				t.Errorf("legacy path forwards ONLY filters, got %+v", fake.lastListParams)
			}
			if got := len(resp.GetData()); got != 2 {
				t.Errorf("legacy row set changed: got %d rows want 2", got)
			}
		})
	}
}

// TestSQLServerListJobPhases_NilRequestStillSafe: req == nil must keep the
// legacy nil-params path (the narrow getter is nil-safe).
func TestSQLServerListJobPhases_NilRequestStillSafe(t *testing.T) {
	fake := &rejectFakeDBOps{}
	r := NewSQLServerJobPhaseRepository(fake, "job_phase")
	if _, err := r.ListJobPhases(context.Background(), nil); err != nil {
		t.Fatalf("nil request must not error: %v", err)
	}
	if fake.listCalls != 1 {
		t.Fatalf("expected exactly one List, got %d", fake.listCalls)
	}
	if fake.lastListParams != nil {
		t.Errorf("nil request must yield nil ListParams, got %+v", fake.lastListParams)
	}
}
