package operation

// narrowStaffReportsToReachableJobs is the row-scope policy for the narrow
// report reads (outcome landing, composite export matrix, document resolver)
// when the caller is a STAFF principal that is not workspace-wide.
//
// false (owner decision 2026-09-21, docs/plan/20260921-section-manager-report-card-access):
// a STAFF principal that holds an ACTIVE section assignment
// (subscription_group_workspace_user for the session's own workspace_user,
// enforced in SQL by the visible_groups / group_context EXISTS gate) sees the
// WHOLE assigned section — every student and subject — not only the jobs
// reachable from its staff record. The section assignment remains the fail-closed
// row gate: no assignment, no rows.
//
// true restores the earlier 20260809 behavior (assignment ∩ reachable-job graph,
// principalscope.StaffReachableJobClause). Flipping it is a security-relevant
// policy change: it also changes the CSV/PDF downloads that share these queries.
const narrowStaffReportsToReachableJobs = false
