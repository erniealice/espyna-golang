package subscription

import (
	"context"
	"testing"

	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

// spawnLockerFake is a JobPhase adapter stand-in that ALSO implements
// templatePhaseSpawnLocker, capturing the exact id set handed to
// LockTemplatePhasesForSpawn so the graph-wide pre-lock (FIX-5) can be asserted
// without a database.
type spawnLockerFake struct {
	jobphasepb.UnimplementedJobPhaseDomainServiceServer
	lockedIDs [][]string
}

func (f *spawnLockerFake) LockTemplatePhasesForSpawn(_ context.Context, ids []string) error {
	f.lockedIDs = append(f.lockedIDs, append([]string(nil), ids...))
	return nil
}
func (f *spawnLockerFake) JobHardFrozen(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// plainJobPhaseFake does NOT implement templatePhaseSpawnLocker — it stands for the
// mock/firestore providers, which must be unaffected (pre-lock is a no-op).
type plainJobPhaseFake struct {
	jobphasepb.UnimplementedJobPhaseDomainServiceServer
}

// templatePhaseFake returns canned template phases per template id.
type templatePhaseFake struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	byTemplate map[string][]string // templateID -> phase ids
}

func (f *templatePhaseFake) ListByJobTemplate(_ context.Context, req *jobtemplatephasepb.ListByJobTemplateRequest) (*jobtemplatephasepb.ListByJobTemplateResponse, error) {
	var phases []*jobtemplatephasepb.JobTemplatePhase
	for _, id := range f.byTemplate[req.GetJobTemplateId()] {
		phases = append(phases, &jobtemplatephasepb.JobTemplatePhase{Id: id})
	}
	return &jobtemplatephasepb.ListByJobTemplateResponse{JobTemplatePhases: phases}, nil
}

// TestPreLockSpawnGraph_LocksEveryReferencedParentDeduped proves the pre-lock
// enumerates EVERY job_template_phase across ALL templates in the graph, dedupes
// them, and hands the whole set to LockTemplatePhasesForSpawn in ONE call (before
// any child write). The adapter's LockTemplatePhasesForSpawn then sorts + issues one
// `ORDER BY id FOR UPDATE`.
func TestPreLockSpawnGraph_LocksEveryReferencedParentDeduped(t *testing.T) {
	locker := &spawnLockerFake{}
	tpl := &templatePhaseFake{byTemplate: map[string][]string{
		"tA": {"p3", "p1"},
		"tB": {"p2", "p1"}, // p1 duplicated across templates
	}}
	deps := phaseSpawnDeps{JobPhase: locker, JobTemplatePhase: tpl}

	// templateIDs intentionally repeats tA to prove template-level dedupe too.
	if err := preLockSpawnGraph(context.Background(), deps, []string{"tA", "tB", "tA", ""}); err != nil {
		t.Fatalf("preLockSpawnGraph: %v", err)
	}
	if len(locker.lockedIDs) != 1 {
		t.Fatalf("expected exactly one graph-wide lock call, got %d", len(locker.lockedIDs))
	}
	got := locker.lockedIDs[0]
	seen := map[string]int{}
	for _, id := range got {
		seen[id]++
	}
	for _, want := range []string{"p1", "p2", "p3"} {
		if seen[want] != 1 {
			t.Fatalf("parent %s must be locked exactly once, got %d (all=%v)", want, seen[want], got)
		}
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 deduped parents, got %d (%v)", len(got), got)
	}
}

// TestPreLockSpawnGraph_MockProviderNoOp proves that a JobPhase provider which does
// not implement templatePhaseSpawnLocker (mock/firestore) is a no-op — the pre-lock
// never runs, so those non-transactional test paths stay unaffected.
func TestPreLockSpawnGraph_MockProviderNoOp(t *testing.T) {
	tpl := &templatePhaseFake{byTemplate: map[string][]string{"tA": {"p1"}}}
	deps := phaseSpawnDeps{JobPhase: &plainJobPhaseFake{}, JobTemplatePhase: tpl}
	if err := preLockSpawnGraph(context.Background(), deps, []string{"tA"}); err != nil {
		t.Fatalf("mock provider pre-lock must be a no-op, got %v", err)
	}
}
