package task_outcome

import "context"

// cellWriteGuard is the optional adapter capability that enforces the matrix
// CELL-WRITE lock protocol inside the ambient transaction: it locks the cell's
// parent job_template_phase FOR SHARE and the owning job_phase row FOR UPDATE,
// then rechecks trusted workspace + ancestry + approval status (IN_PROGRESS) +
// hard-frozen, failing closed if the cell is not editable (codex FIX-FIRST 2 /
// plan §4.2 cell-write paragraph). The PostgreSQL task_outcome adapter implements
// it; mock/firestore providers do not (they carry no approval lock protocol), so
// the guard is skipped for them and their existing behaviour is preserved.
type cellWriteGuard interface {
	GuardCellWrite(ctx context.Context, jobTaskID string) error
}

// isGuardedRepo reports whether the repository enforces the cell-write lock
// protocol (the approval-aware PostgreSQL path). When true, the task_outcome
// write MUST run inside a transaction — there is NO nontransactional fallback.
func isGuardedRepo(repo any) (cellWriteGuard, bool) {
	g, ok := repo.(cellWriteGuard)
	return g, ok
}
