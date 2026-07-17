package grade_compute

import (
	"context"
	"errors"
	"fmt"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	scoringcomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component"
	scoringcomponentcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
)

// pageWindow mirrors the DB list windowing (1-based offset page + limit) so the
// fakes below page exactly like the real adapter the use case drives.
func pageWindow(total int, p *commonpb.PaginationRequest) (start, end int) {
	page, limit := 1, int(scopeReadPageSize)
	if p != nil {
		if p.Limit > 0 {
			limit = int(p.Limit)
		}
		if off := p.GetOffset(); off != nil && off.Page > 0 {
			page = int(off.Page)
		}
	}
	start = (page - 1) * limit
	if start > total {
		start = total
	}
	end = start + limit
	if end > total {
		end = total
	}
	return start, end
}

// fakeSCCRepo is a paginating ScoringComponentCriteria stub. Embedding the
// generated Unimplemented server satisfies the rest of the interface.
type fakeSCCRepo struct {
	scoringcomponentcriteriapb.UnimplementedScoringComponentCriteriaDomainServiceServer
	rows      []*scoringcomponentcriteriapb.ScoringComponentCriteria
	err       error
	listCalls int
}

func (f *fakeSCCRepo) ListScoringComponentCriterias(_ context.Context, req *scoringcomponentcriteriapb.ListScoringComponentCriteriasRequest) (*scoringcomponentcriteriapb.ListScoringComponentCriteriasResponse, error) {
	f.listCalls++
	if f.err != nil {
		return nil, f.err
	}
	start, end := pageWindow(len(f.rows), req.GetPagination())
	return &scoringcomponentcriteriapb.ListScoringComponentCriteriasResponse{Data: f.rows[start:end], Success: true}, nil
}

// fakeSCRepo is a paginating ScoringComponent stub.
type fakeSCRepo struct {
	scoringcomponentpb.UnimplementedScoringComponentDomainServiceServer
	rows []*scoringcomponentpb.ScoringComponent
	err  error
}

func (f *fakeSCRepo) ListScoringComponents(_ context.Context, req *scoringcomponentpb.ListScoringComponentsRequest) (*scoringcomponentpb.ListScoringComponentsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	start, end := pageWindow(len(f.rows), req.GetPagination())
	return &scoringcomponentpb.ListScoringComponentsResponse{Data: f.rows[start:end], Success: true}, nil
}

// TestInScopeCriteria_PaginationCompleteness locks the HIGH fix: a scheme whose
// component<->criteria graph spills past a single default page must still be read
// COMPLETELY. A criterion living only on the second page cannot be truncated.
func TestInScopeCriteria_PaginationCompleteness(t *testing.T) {
	const scheme = "scheme-1"
	var rows []*scoringcomponentcriteriapb.ScoringComponentCriteria
	for i := 0; i < 150; i++ {
		rows = append(rows, &scoringcomponentcriteriapb.ScoringComponentCriteria{
			Id:                 fmt.Sprintf("scc-%d", i),
			ScoringSchemeId:    scheme,
			ScoringComponentId: "comp-1",
			OutcomeCriteriaId:  fmt.Sprintf("crit-%d", i),
			Active:             true,
		})
	}
	sccRepo := &fakeSCCRepo{rows: rows}
	scRepo := &fakeSCRepo{rows: []*scoringcomponentpb.ScoringComponent{
		{Id: "comp-1", ScoringSchemeId: scheme, Active: true},
	}}
	uc := &ComputePhaseOutcomeUseCase{repositories: Repositories{ScoringComponentCriteria: sccRepo, ScoringComponent: scRepo}}

	set, err := uc.inScopeCriteria(context.Background(), scheme)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set) != 150 {
		t.Fatalf("expected 150 criteria from a complete cross-page read, got %d", len(set))
	}
	if !set["crit-120"] {
		t.Fatal("a criterion on the second page (crit-120) was truncated — read is not complete")
	}
	if sccRepo.listCalls < 2 {
		t.Fatalf("expected the read to page (>=2 list calls), got %d", sccRepo.listCalls)
	}
}

// TestInScopeCriteria_InactiveParentComponentExcluded locks that the scope is the
// ACTIVE component graph: junctions under an inactive parent, or under a component
// belonging to another scheme, are excluded.
func TestInScopeCriteria_InactiveParentComponentExcluded(t *testing.T) {
	const scheme = "scheme-1"
	sccRepo := &fakeSCCRepo{rows: []*scoringcomponentcriteriapb.ScoringComponentCriteria{
		{Id: "scc-a", ScoringSchemeId: scheme, ScoringComponentId: "comp-active", OutcomeCriteriaId: "crit-active", Active: true},
		{Id: "scc-i", ScoringSchemeId: scheme, ScoringComponentId: "comp-inactive", OutcomeCriteriaId: "crit-inactive", Active: true},
		{Id: "scc-o", ScoringSchemeId: scheme, ScoringComponentId: "comp-other-scheme", OutcomeCriteriaId: "crit-otherscheme", Active: true},
	}}
	scRepo := &fakeSCRepo{rows: []*scoringcomponentpb.ScoringComponent{
		{Id: "comp-active", ScoringSchemeId: scheme, Active: true},
		{Id: "comp-inactive", ScoringSchemeId: scheme, Active: false},
		{Id: "comp-other-scheme", ScoringSchemeId: "scheme-2", Active: true},
	}}
	uc := &ComputePhaseOutcomeUseCase{repositories: Repositories{ScoringComponentCriteria: sccRepo, ScoringComponent: scRepo}}

	set, err := uc.inScopeCriteria(context.Background(), scheme)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !set["crit-active"] {
		t.Fatal("criterion under the active in-scheme component must be in scope")
	}
	if set["crit-inactive"] {
		t.Fatal("criterion under an INACTIVE parent component must be excluded")
	}
	if set["crit-otherscheme"] {
		t.Fatal("criterion whose parent component belongs to another scheme must be excluded")
	}
	if len(set) != 1 {
		t.Fatalf("expected exactly one in-scope criterion, got %d", len(set))
	}
}

// TestInScopeCriteria_JunctionReadErrorPropagates: a junction read failure must
// surface as an error so the caller fails closed (attempts recompute / reports
// stale) rather than acking the cell as authoritatively not-applicable.
func TestInScopeCriteria_JunctionReadErrorPropagates(t *testing.T) {
	const scheme = "scheme-1"
	sccRepo := &fakeSCCRepo{err: errors.New("boom")}
	scRepo := &fakeSCRepo{rows: []*scoringcomponentpb.ScoringComponent{{Id: "comp-1", ScoringSchemeId: scheme, Active: true}}}
	uc := &ComputePhaseOutcomeUseCase{repositories: Repositories{ScoringComponentCriteria: sccRepo, ScoringComponent: scRepo}}

	if _, err := uc.inScopeCriteria(context.Background(), scheme); err == nil {
		t.Fatal("a junction read failure must propagate as an error")
	}
}

// TestInScopeCriteria_ComponentReadErrorPropagates: a component read failure must
// likewise propagate.
func TestInScopeCriteria_ComponentReadErrorPropagates(t *testing.T) {
	const scheme = "scheme-1"
	sccRepo := &fakeSCCRepo{}
	scRepo := &fakeSCRepo{err: errors.New("boom")}
	uc := &ComputePhaseOutcomeUseCase{repositories: Repositories{ScoringComponentCriteria: sccRepo, ScoringComponent: scRepo}}

	if _, err := uc.inScopeCriteria(context.Background(), scheme); err == nil {
		t.Fatal("a component read failure must propagate as an error")
	}
}
