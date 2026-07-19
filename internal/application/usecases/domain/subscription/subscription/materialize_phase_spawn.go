package subscription

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

// phaseSpawnDeps is the exact dependency set the shared phase/task materialization
// seam needs. Both MaterializeJobsForSubscription and
// MaterializeInstanceJobsForSubscription delegate to spawnPhasesAndTasksForJob so
// the (previously byte-identical, duplicated) W-SPAWN body lives in ONE place —
// closing the drift hazard codex flagged and giving the approval-lifecycle
// hardening a single home.
type phaseSpawnDeps struct {
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask  jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
	JobPhase         jobphasepb.JobPhaseDomainServiceServer
	JobTask          jobtaskpb.JobTaskDomainServiceServer
	IDGenerator      ports.IDGenerator
}

// templatePhaseSpawnLocker is the narrow optional interface the PostgreSQL
// job_phase adapter satisfies. It carries the W-SPAWN lifecycle hardening that
// only the real database path can enforce:
//
//   - LockTemplatePhasesForSpawn pre-locks the referenced job_template_phase
//     parents FOR UPDATE in sorted id order (the same parent mutex the
//     transitions take, in the same global lock order) INSIDE the ambient
//     transaction — it fails closed when no transaction is active, which is how
//     "phase creation requires the ambient transaction" is enforced for the
//     enforcing provider (the mock/firestore providers do not implement this
//     interface, so their non-transactional test paths are unchanged).
//   - JobHardFrozen rejects a late spawn into a hard-frozen target (a closed
//     academic-year schedule or an active authoritative final) — new phases must
//     never be materialised into finalized history.
type templatePhaseSpawnLocker interface {
	LockTemplatePhasesForSpawn(ctx context.Context, templatePhaseIDs []string) error
	JobHardFrozen(ctx context.Context, jobID, wsID string) (bool, error)
}

// requireTxForTemplateSpawn fail-closes when the ENFORCING (postgres) provider is
// active but transaction support is absent — checked BEFORE the first Job write
// (codex §5 HIGH). On the enforcing path spawnPhasesAndTasksForJob must hold the
// parent job_template_phase locks inside an ambient transaction, but that check
// fires only AFTER the Job row is created; a nil/miswired transactor would
// otherwise autocommit a partial graph (an orphan Job with no phases) before
// failing. Mock/firestore providers do NOT implement templatePhaseSpawnLocker, so
// their existing non-transactional path is preserved.
func requireTxForTemplateSpawn(jobPhase jobphasepb.JobPhaseDomainServiceServer, tx ports.Transactor) error {
	if _, enforcing := jobPhase.(templatePhaseSpawnLocker); !enforcing {
		return nil
	}
	if tx == nil || !tx.SupportsTransactions() {
		return fmt.Errorf("materialize_jobs: template-backed spawn requires transaction support on the enforcing provider — fail closed before the first job write")
	}
	return nil
}

// preLockSpawnGraph is the GRAPH-WIDE parent pre-lock (FIX-5 / codex §5 HIGH +
// codex-rereview.md "Exact E1 mixed-state recovery"). BEFORE the first job/phase/
// task write of a materialization, it enumerates EVERY job_template_phase referenced
// by the whole spawn graph (across every template the graph will spawn), dedupes
// them, and takes them ALL FOR UPDATE in one global id order via the ambient
// transaction (LockTemplatePhasesForSpawn sorts + `ORDER BY id FOR UPDATE`).
//
// The per-job lock inside spawnPhasesAndTasksForJob is "too local and too late" — it
// locked a single template's parents only AFTER that job row had already been
// created, so a concurrent spawn or approval transition could interleave between two
// jobs' locks. Acquiring the whole graph's parents up front, in one global order,
// before any child write removes that lock-ordering window and serializes the entire
// graph against any concurrent transition sharing a parent.
//
// Gated on the templatePhaseSpawnLocker capability: the mock/firestore providers do
// not implement it, so this is a no-op for them (their non-transactional test paths
// are unaffected). On the postgres path LockTemplatePhasesForSpawn fails closed
// without an ambient transaction, so the graph pre-lock can only take effect inside
// the tx. Idempotent on an empty/duplicate template set.
func preLockSpawnGraph(ctx context.Context, deps phaseSpawnDeps, templateIDs []string) error {
	locker, ok := deps.JobPhase.(templatePhaseSpawnLocker)
	if !ok || deps.JobTemplatePhase == nil {
		return nil
	}
	seenTpl := make(map[string]struct{}, len(templateIDs))
	seenPhase := make(map[string]struct{})
	var parentIDs []string
	for _, tid := range templateIDs {
		if tid == "" {
			continue
		}
		if _, dup := seenTpl[tid]; dup {
			continue
		}
		seenTpl[tid] = struct{}{}
		resp, err := deps.JobTemplatePhase.ListByJobTemplate(ctx,
			&jobtemplatephasepb.ListByJobTemplateRequest{JobTemplateId: tid})
		if err != nil {
			return fmt.Errorf("spawn_prelock_list_template_phases (template=%s): %w", tid, err)
		}
		if resp == nil {
			continue
		}
		for _, tp := range resp.GetJobTemplatePhases() {
			id := tp.GetId()
			if id == "" {
				continue
			}
			if _, dup := seenPhase[id]; dup {
				continue
			}
			seenPhase[id] = struct{}{}
			parentIDs = append(parentIDs, id)
		}
	}
	if len(parentIDs) == 0 {
		return nil
	}
	// LockTemplatePhasesForSpawn sorts the ids and issues one `ORDER BY id FOR UPDATE`,
	// so the whole graph's parents are acquired in a single global id order before any
	// child write of the materialization.
	if err := locker.LockTemplatePhasesForSpawn(ctx, parentIDs); err != nil {
		return fmt.Errorf("spawn_prelock_graph: %w", err)
	}
	return nil
}

// spawnPhasesAndTasksForJob materialises JobPhase + JobTask rows from a
// JobTemplate. New phases are born IN_PROGRESS with null approval audit: the
// spawn never sets an approval field, and the job_phase adapter strips any
// approval/audit key from a generic create (server-owned lifecycle), so an
// advanced sibling makes the sheet visibly mixed/Attention rather than forging an
// approval. Predecessor phase IDs are remapped from template-phase IDs to the
// freshly minted phase IDs.
func spawnPhasesAndTasksForJob(
	ctx context.Context, deps phaseSpawnDeps, dc int64, dcs string, job *jobpb.Job, templateID string,
) error {
	phaseResp, err := deps.JobTemplatePhase.ListByJobTemplate(ctx,
		&jobtemplatephasepb.ListByJobTemplateRequest{JobTemplateId: templateID})
	if err != nil {
		return fmt.Errorf("list_template_phases (template=%s): %w", templateID, err)
	}
	tplPhases := []*jobtemplatephasepb.JobTemplatePhase{}
	if phaseResp != nil {
		tplPhases = phaseResp.GetJobTemplatePhases()
	}
	sort.SliceStable(tplPhases, func(i, j int) bool {
		return tplPhases[i].GetPhaseOrder() < tplPhases[j].GetPhaseOrder()
	})

	// W-SPAWN lifecycle hardening (postgres path only — codex "W-SPAWN fresh
	// finding" + "Exact E1 mixed-state recovery" + P3 §A5): reject a late spawn into
	// a hard-frozen target. The job_template_phase parent lock is NO LONGER taken
	// here — codex P3 §A5 rejected the late per-job re-list/lock (it ran AFTER the
	// Job row existed and re-enumerated, so a parent added after the up-front
	// enumeration would be locked late, outside the global order). Every spawn entry
	// point (non-cyclic root+relations, cyclic-shell onboarding children, instance
	// cycle+once-at-start, ad-hoc usage) now calls preLockSpawnGraph BEFORE its first
	// Job write, which captures the WHOLE graph's parents in ONE global id order.
	// This recheck of hard-frozen therefore runs UNDER those already-held locks.
	// Gated on the narrow interface so mock/firestore providers are unaffected.
	//
	// INVARIANT: spawnPhasesAndTasksForJob MUST be reached only after
	// preLockSpawnGraph has locked this job's template parents in the same tx.
	if locker, ok := deps.JobPhase.(templatePhaseSpawnLocker); ok && len(tplPhases) > 0 {
		if job != nil && job.GetId() != "" {
			frozen, ferr := locker.JobHardFrozen(ctx, job.GetId(), job.GetWorkspaceId())
			if ferr != nil {
				return fmt.Errorf("spawn_hard_frozen_probe (job=%s): %w", job.GetId(), ferr)
			}
			if frozen {
				return fmt.Errorf("spawn_rejected: job %s is hard-frozen (closed schedule or authoritative final) — cannot materialise phases into finalized history", job.GetId())
			}
		}
	}

	// Mark this context as the trusted spawn seam: the parent job_template_phase
	// rows are now locked FOR UPDATE (postgres path), so the adapter permits the
	// template-backed CreateJobPhase that a generic create rejects. The marker is
	// internal and unforgeable by any downstream module.
	ctx = approvalctx.WithTrustedSpawn(ctx)

	phaseIDMap := make(map[string]string, len(tplPhases))

	for _, tp := range tplPhases {
		var phaseID string
		if deps.IDGenerator != nil {
			phaseID = deps.IDGenerator.GenerateID()
		} else {
			phaseID = fmt.Sprintf("phase-%d", time.Now().UnixNano())
		}
		tplPhaseID := tp.GetId()
		// Propagate the scoring scheme from the template phase. A NULL scheme makes
		// the spawned phase invisible to the education grading pipeline (grade
		// sheet, phase/year-final compute, report card). Nil-safe copy.
		var scoringSchemeID *string
		if tp.ScoringSchemeId != nil {
			v := tp.GetScoringSchemeId()
			scoringSchemeID = &v
		}
		phase := &jobphasepb.JobPhase{
			Id:                 phaseID,
			JobId:              job.GetId(),
			Name:               tp.GetName(),
			PhaseOrder:         tp.GetPhaseOrder(),
			Status:             jobphasepb.PhaseStatus_PHASE_STATUS_PENDING,
			Active:             true,
			TemplatePhaseId:    &tplPhaseID,
			ScoringSchemeId:    scoringSchemeID,
			DateCreated:        &dc,
			DateCreatedString:  &dcs,
			DateModified:       &dc,
			DateModifiedString: &dcs,
		}
		if tp.PredecessorTemplatePhaseId != nil && *tp.PredecessorTemplatePhaseId != "" {
			if mapped, ok := phaseIDMap[*tp.PredecessorTemplatePhaseId]; ok {
				v := mapped
				phase.PredecessorPhaseId = &v
			}
		}
		if _, err := deps.JobPhase.CreateJobPhase(ctx,
			&jobphasepb.CreateJobPhaseRequest{Data: phase}); err != nil {
			return fmt.Errorf("create_job_phase (template_phase=%s): %w", tplPhaseID, err)
		}
		phaseIDMap[tplPhaseID] = phaseID

		taskResp, err := deps.JobTemplateTask.ListByPhase(ctx,
			&jobtemplatetaskpb.ListJobTemplateTasksByPhaseRequest{JobTemplatePhaseId: tplPhaseID})
		if err != nil {
			return fmt.Errorf("list_template_tasks (phase=%s): %w", tplPhaseID, err)
		}
		tplTasks := []*jobtemplatetaskpb.JobTemplateTask{}
		if taskResp != nil {
			tplTasks = taskResp.GetJobTemplateTasks()
		}
		sort.SliceStable(tplTasks, func(i, j int) bool {
			return tplTasks[i].GetStepOrder() < tplTasks[j].GetStepOrder()
		})
		for _, tt := range tplTasks {
			var taskID string
			if deps.IDGenerator != nil {
				taskID = deps.IDGenerator.GenerateID()
			} else {
				taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
			}
			tplTaskID := tt.GetId()
			task := &jobtaskpb.JobTask{
				Id:                 taskID,
				JobPhaseId:         phaseID,
				Name:               tt.GetName(),
				StepOrder:          tt.GetStepOrder(),
				Status:             jobtaskpb.TaskStatus_TASK_STATUS_PENDING,
				IsAdHoc:            false,
				Active:             true,
				TemplateTaskId:     &tplTaskID,
				DateCreated:        &dc,
				DateCreatedString:  &dcs,
				DateModified:       &dc,
				DateModifiedString: &dcs,
			}
			if _, err := deps.JobTask.CreateJobTask(ctx,
				&jobtaskpb.CreateJobTaskRequest{Data: task}); err != nil {
				return fmt.Errorf("create_job_task (template_task=%s): %w", tplTaskID, err)
			}
		}
	}
	return nil
}
