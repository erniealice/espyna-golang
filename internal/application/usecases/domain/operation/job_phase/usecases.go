package job_phase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
)

// JobPhaseRepositories groups all repository dependencies.
//
// BillingEvent is optional — only required to power the OnJobPhaseCompleted
// hook (milestone-billing plan §3 + flow.md §11). When nil, the hook is a
// no-op.
type JobPhaseRepositories struct {
	JobPhase     pb.JobPhaseDomainServiceServer
	BillingEvent billingeventpb.BillingEventDomainServiceServer
}

// JobPhaseServices groups all business service dependencies
type JobPhaseServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all job-phase-related use cases
type UseCases struct {
	CreateJobPhase          *CreateJobPhaseUseCase
	ReadJobPhase            *ReadJobPhaseUseCase
	UpdateJobPhase          *UpdateJobPhaseUseCase
	DeleteJobPhase          *DeleteJobPhaseUseCase
	ListJobPhases           *ListJobPhasesUseCase
	GetJobPhaseListPageData *GetJobPhaseListPageDataUseCase
	GetJobPhaseItemPageData *GetJobPhaseItemPageDataUseCase
	ListByJob               *ListByJobUseCase
}

// NewUseCases creates a new collection of job phase use cases
func NewUseCases(
	repositories JobPhaseRepositories,
	services JobPhaseServices,
) *UseCases {
	return &UseCases{
		CreateJobPhase: &CreateJobPhaseUseCase{
			repositories: CreateJobPhaseRepositories{JobPhase: repositories.JobPhase},
			services: CreateJobPhaseServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		},
		ReadJobPhase: &ReadJobPhaseUseCase{
			repositories: ReadJobPhaseRepositories{JobPhase: repositories.JobPhase},
			services: ReadJobPhaseServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		},
		UpdateJobPhase: &UpdateJobPhaseUseCase{
			repositories: UpdateJobPhaseRepositories{
				JobPhase:     repositories.JobPhase,
				BillingEvent: repositories.BillingEvent,
			},
			services: UpdateJobPhaseServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		},
		DeleteJobPhase: &DeleteJobPhaseUseCase{
			repositories: DeleteJobPhaseRepositories{JobPhase: repositories.JobPhase},
			services: DeleteJobPhaseServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		},
		ListJobPhases: &ListJobPhasesUseCase{
			repositories: ListJobPhasesRepositories{JobPhase: repositories.JobPhase},
			services: ListJobPhasesServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		},
		GetJobPhaseListPageData: &GetJobPhaseListPageDataUseCase{
			repositories: GetJobPhaseListPageDataRepositories{JobPhase: repositories.JobPhase},
			services: GetJobPhaseListPageDataServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		},
		GetJobPhaseItemPageData: &GetJobPhaseItemPageDataUseCase{
			repositories: GetJobPhaseItemPageDataRepositories{JobPhase: repositories.JobPhase},
			services: GetJobPhaseItemPageDataServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		},
		ListByJob: &ListByJobUseCase{
			repositories: ListByJobRepositories{JobPhase: repositories.JobPhase},
			services: ListByJobServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
	}
}

// ---- CreateJobPhase ----

type CreateJobPhaseRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}
type CreateJobPhaseServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}
type CreateJobPhaseUseCase struct {
	repositories CreateJobPhaseRepositories
	services     CreateJobPhaseServices
}

func (uc *CreateJobPhaseUseCase) Execute(ctx context.Context, req *pb.CreateJobPhaseRequest) (*pb.CreateJobPhaseResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.data_required", "job phase data is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.name_required", "job phase name is required [DEFAULT]"))
	}
	if req.Data.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.job_id_required", "job ID is required [DEFAULT]"))
	}

	now := time.Now()
	if uc.services.IDGenerator != nil {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	} else {
		req.Data.Id = fmt.Sprintf("job_phase-%d", now.UnixNano())
	}
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)
	req.Data.DateCreated = &dc
	req.Data.DateCreatedString = &dcs
	req.Data.DateModified = &dc
	req.Data.DateModifiedString = &dcs
	req.Data.Active = true

	if uc.services.Transactor != nil {
		var result *pb.CreateJobPhaseResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.JobPhase.CreateJobPhase(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.JobPhase.CreateJobPhase(ctx, req)
}

// ---- ReadJobPhase ----

type ReadJobPhaseRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}
type ReadJobPhaseServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ReadJobPhaseUseCase struct {
	repositories ReadJobPhaseRepositories
	services     ReadJobPhaseServices
}

func (uc *ReadJobPhaseUseCase) Execute(ctx context.Context, req *pb.ReadJobPhaseRequest) (*pb.ReadJobPhaseResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.id_required", "job phase ID is required"))
	}
	result, err := uc.repositories.JobPhase.ReadJobPhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.errors.not_found", "job phase not found [DEFAULT]"))
	}
	return result, nil
}

// ---- UpdateJobPhase ----

type UpdateJobPhaseRepositories struct {
	JobPhase     pb.JobPhaseDomainServiceServer
	BillingEvent billingeventpb.BillingEventDomainServiceServer
}
type UpdateJobPhaseServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type UpdateJobPhaseUseCase struct {
	repositories UpdateJobPhaseRepositories
	services     UpdateJobPhaseServices
}

func (uc *UpdateJobPhaseUseCase) Execute(ctx context.Context, req *pb.UpdateJobPhaseRequest) (*pb.UpdateJobPhaseResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.id_required", "job phase ID is required"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.name_required", "job phase name is required"))
	}

	// Snapshot the prior status so we can detect the PHASE_COMPLETED transition
	// for the OnJobPhaseCompleted hook (milestone-billing plan §3 / flow.md §11).
	priorStatus := pb.PhaseStatus_PHASE_STATUS_UNSPECIFIED
	if existing, err := uc.repositories.JobPhase.ReadJobPhase(ctx, &pb.ReadJobPhaseRequest{Data: &pb.JobPhase{Id: req.Data.Id}}); err == nil &&
		existing != nil && len(existing.GetData()) > 0 {
		priorStatus = existing.GetData()[0].GetStatus()
	}

	now := time.Now()
	dm := now.UnixMilli()
	dms := now.Format(time.RFC3339)
	req.Data.DateModified = &dm
	req.Data.DateModifiedString = &dms

	_, err := uc.repositories.JobPhase.UpdateJobPhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.errors.update_failed", "job phase update failed [DEFAULT]"))
	}

	// OnJobPhaseCompleted hook — one-way (revert does NOT roll back any
	// already-billed milestone). Fires only when status crosses into
	// COMPLETED for the first time.
	if priorStatus != pb.PhaseStatus_PHASE_STATUS_COMPLETED &&
		req.Data.GetStatus() == pb.PhaseStatus_PHASE_STATUS_COMPLETED {
		uc.fireOnJobPhaseCompleted(ctx, req.Data)
	}

	return &pb.UpdateJobPhaseResponse{Success: true, Data: []*pb.JobPhase{req.Data}}, nil
}

// fireOnJobPhaseCompleted advances every billing_event row linked to the
// supplied phase from UNSPECIFIED → READY. Two transitions are wired:
//
//  1. Milestone-billing path: ListByJobPhase → mark UNSPECIFIED events READY
//     with trigger=PHASE_COMPLETED. (Original behaviour.)
//  2. AD_HOC × PER_OCCURRENCE path: when ALL phases of the parent Job are
//     COMPLETED, ListByJob → mark UNSPECIFIED events READY with
//     trigger=VISIT_COMPLETED. AD_HOC events have job_phase_id = NULL by
//     design (ad-hoc plan §2.6), so the milestone path can't see them.
//
// Best-effort: the hook never blocks the phase update itself, even if any
// individual billing_event update fails.
//
// Codex MAJ-3 fix.
// See docs/plan/20260501-ad-hoc-subscription-billing/plan.md §3.5.
func (uc *UpdateJobPhaseUseCase) fireOnJobPhaseCompleted(ctx context.Context, phase *pb.JobPhase) {
	if uc.repositories.BillingEvent == nil || phase == nil || phase.GetId() == "" {
		return
	}

	// (1) Milestone path — ListByJobPhase.
	if resp, err := uc.repositories.BillingEvent.ListByJobPhase(ctx, &billingeventpb.ListBillingEventsByJobPhaseRequest{
		JobPhaseId: phase.GetId(),
	}); err == nil && resp != nil {
		for _, ev := range resp.GetBillingEvents() {
			if ev.GetStatus() != billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_UNSPECIFIED {
				continue
			}
			if _, err := uc.repositories.BillingEvent.SetStatus(ctx, &billingeventpb.SetBillingEventStatusRequest{
				BillingEventId: ev.GetId(),
				Status:         billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_READY,
				Trigger:        billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_PHASE_COMPLETED,
			}); err != nil {
				log.Printf("OnJobPhaseCompleted: failed to advance billing_event %s: %v", ev.GetId(), err)
			}
		}
	}

	// (2) AD_HOC path — fires only when every sibling phase under the same
	// Job is COMPLETED. The ListByJob lookup excludes the milestone-event
	// case because milestone events DO have a job_phase_id and were already
	// caught in (1) above (UNSPECIFIED → READY filter is idempotent).
	jobID := phase.GetJobId()
	if jobID == "" {
		return
	}
	siblingResp, err := uc.repositories.JobPhase.ListByJob(ctx, &pb.ListJobPhasesByJobRequest{JobId: jobID})
	if err != nil || siblingResp == nil {
		return
	}
	for _, sibling := range siblingResp.GetJobPhases() {
		if !sibling.GetActive() {
			continue
		}
		if sibling.GetStatus() != pb.PhaseStatus_PHASE_STATUS_COMPLETED {
			return // at least one phase still pending — defer the trigger
		}
	}
	// All phases COMPLETED → flip any AD_HOC events on this Job.
	jobEvents, err := uc.repositories.BillingEvent.ListByJob(ctx, &billingeventpb.ListBillingEventsByJobRequest{
		JobId: jobID,
	})
	if err != nil || jobEvents == nil {
		return
	}
	for _, ev := range jobEvents.GetBillingEvents() {
		if ev.GetStatus() != billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_UNSPECIFIED {
			continue
		}
		// Defensive — milestone events already caught in (1) and are now
		// READY; only AD_HOC events with NULL job_phase_id remain at
		// UNSPECIFIED here.
		if _, err := uc.repositories.BillingEvent.SetStatus(ctx, &billingeventpb.SetBillingEventStatusRequest{
			BillingEventId: ev.GetId(),
			Status:         billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_READY,
			Trigger:        billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_VISIT_COMPLETED,
		}); err != nil {
			log.Printf("OnJobPhaseCompleted (AD_HOC): failed to advance billing_event %s: %v", ev.GetId(), err)
		}
	}
}

// ---- DeleteJobPhase ----

type DeleteJobPhaseRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}
type DeleteJobPhaseServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type DeleteJobPhaseUseCase struct {
	repositories DeleteJobPhaseRepositories
	services     DeleteJobPhaseServices
}

func (uc *DeleteJobPhaseUseCase) Execute(ctx context.Context, req *pb.DeleteJobPhaseRequest) (*pb.DeleteJobPhaseResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.id_required", "job phase ID is required"))
	}
	result, err := uc.repositories.JobPhase.DeleteJobPhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.errors.deletion_failed", "job phase deletion failed [DEFAULT]"))
	}
	return result, nil
}

// ---- ListJobPhases ----

type ListJobPhasesRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}
type ListJobPhasesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ListJobPhasesUseCase struct {
	repositories ListJobPhasesRepositories
	services     ListJobPhasesServices
}

func (uc *ListJobPhasesUseCase) Execute(ctx context.Context, req *pb.ListJobPhasesRequest) (*pb.ListJobPhasesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.JobPhase.ListJobPhases(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.errors.list_failed", "job phase listing failed [DEFAULT]"))
	}
	return result, nil
}

// ---- GetJobPhaseListPageData ----

type GetJobPhaseListPageDataRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}
type GetJobPhaseListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type GetJobPhaseListPageDataUseCase struct {
	repositories GetJobPhaseListPageDataRepositories
	services     GetJobPhaseListPageDataServices
}

func (uc *GetJobPhaseListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobPhaseListPageDataRequest) (*pb.GetJobPhaseListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.request_required", "request is required"))
	}
	return uc.repositories.JobPhase.GetJobPhaseListPageData(ctx, req)
}

// ---- GetJobPhaseItemPageData ----

type GetJobPhaseItemPageDataRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}
type GetJobPhaseItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type GetJobPhaseItemPageDataUseCase struct {
	repositories GetJobPhaseItemPageDataRepositories
	services     GetJobPhaseItemPageDataServices
}

func (uc *GetJobPhaseItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobPhaseItemPageDataRequest) (*pb.GetJobPhaseItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.JobPhaseId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.id_required", "job phase ID is required"))
	}
	return uc.repositories.JobPhase.GetJobPhaseItemPageData(ctx, req)
}

// ---- ListByJob (extra RPC) ----

type ListByJobRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}
type ListByJobServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ListByJobUseCase struct {
	repositories ListByJobRepositories
	services     ListByJobServices
}

func (uc *ListByJobUseCase) Execute(ctx context.Context, req *pb.ListJobPhasesByJobRequest) (*pb.ListJobPhasesByJobResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_phase", Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil || req.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_phase.validation.job_id_required", "job ID is required"))
	}
	return uc.repositories.JobPhase.ListByJob(ctx, req)
}
