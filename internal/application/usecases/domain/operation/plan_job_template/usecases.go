package plan_job_template

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/plan_job_template"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

type Repositories struct {
	PlanJobTemplate pb.PlanJobTemplateDomainServiceServer
	Plan            planpb.PlanDomainServiceServer
	JobTemplate     jobtemplatepb.JobTemplateDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UseCases struct {
	Create               func(context.Context, *pb.CreatePlanJobTemplateRequest) (*pb.CreatePlanJobTemplateResponse, error)
	Read                 func(context.Context, *pb.ReadPlanJobTemplateRequest) (*pb.ReadPlanJobTemplateResponse, error)
	Update               func(context.Context, *pb.UpdatePlanJobTemplateRequest) (*pb.UpdatePlanJobTemplateResponse, error)
	Delete               func(context.Context, *pb.DeletePlanJobTemplateRequest) (*pb.DeletePlanJobTemplateResponse, error)
	List                 func(context.Context, *pb.ListPlanJobTemplatesRequest) (*pb.ListPlanJobTemplatesResponse, error)
	ListByPlan           func(context.Context, *pb.ListPlanJobTemplatesByPlanRequest) (*pb.ListPlanJobTemplatesByPlanResponse, error)
	BuildBackfillEntries func(LegacyPlanComposition, map[string]bool) []*pb.PlanJobTemplate
}

func NewUseCases(repos Repositories, services Services) *UseCases {
	uc := &useCases{repos: repos, services: services}
	if repos.PlanJobTemplate == nil {
		return &UseCases{BuildBackfillEntries: BuildBackfillEntries}
	}
	return &UseCases{
		Create: uc.create, Read: uc.read, Update: uc.update, Delete: uc.delete,
		List: uc.list, ListByPlan: uc.listByPlan,
		BuildBackfillEntries: BuildBackfillEntries,
	}
}

type useCases struct {
	repos    Repositories
	services Services
}

func (uc *useCases) gate(ctx context.Context, action string) error {
	if uc.services.ActionGatekeeper == nil {
		return errors.New("plan_job_template action gate unavailable")
	}
	return uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.PlanJobTemplate, Action: action})
}

func (uc *useCases) create(ctx context.Context, req *pb.CreatePlanJobTemplateRequest) (*pb.CreatePlanJobTemplateResponse, error) {
	if err := uc.gate(ctx, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.GetPlanId() == "" || req.Data.GetJobTemplateId() == "" {
		return nil, errors.New("plan and job template are required")
	}
	if req.Data.GetCompositionEntryPattern() == pb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_UNSPECIFIED {
		return nil, errors.New("composition entry pattern is required")
	}
	ws := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if ws == "" {
		return nil, errors.New("workspace context is required")
	}
	if err := uc.requireReferencesInWorkspace(ctx, ws, req.Data.GetPlanId(), req.Data.GetJobTemplateId()); err != nil {
		return nil, err
	}
	if req.Data.GetId() == "" {
		if uc.services.IDGenerator != nil {
			req.Data.Id = uc.services.IDGenerator.GenerateID()
		} else {
			req.Data.Id = "pjt-" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
		}
	}
	now := time.Now()
	ms, stamp := now.UnixMilli(), now.Format(time.RFC3339)
	req.Data.DateCreated, req.Data.DateCreatedString, req.Data.DateModified, req.Data.DateModifiedString = &ms, &stamp, &ms, &stamp
	req.Data.WorkspaceId = ws
	req.Data.Active = true
	return uc.repos.PlanJobTemplate.CreatePlanJobTemplate(ctx, req)
}

func (uc *useCases) read(ctx context.Context, req *pb.ReadPlanJobTemplateRequest) (*pb.ReadPlanJobTemplateResponse, error) {
	if err := uc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.repos.PlanJobTemplate.ReadPlanJobTemplate(ctx, req)
}

func (uc *useCases) update(ctx context.Context, req *pb.UpdatePlanJobTemplateRequest) (*pb.UpdatePlanJobTemplateResponse, error) {
	if err := uc.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, errors.New("plan_job_template ID is required")
	}
	read, err := uc.repos.PlanJobTemplate.ReadPlanJobTemplate(ctx, &pb.ReadPlanJobTemplateRequest{Data: &pb.PlanJobTemplate{Id: req.Data.GetId()}})
	if err != nil || len(read.GetData()) == 0 {
		return nil, errors.New("plan_job_template not found")
	}
	existing := read.GetData()[0]
	ws := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if ws == "" || existing.GetWorkspaceId() != ws {
		return nil, errors.New("plan_job_template is outside workspace")
	}
	if req.Data.GetPlanId() == "" {
		req.Data.PlanId = existing.GetPlanId()
	}
	if req.Data.GetJobTemplateId() == "" {
		req.Data.JobTemplateId = existing.GetJobTemplateId()
	}
	if req.Data.GetCompositionEntryPattern() == 0 {
		req.Data.CompositionEntryPattern = existing.GetCompositionEntryPattern()
	}
	req.Data.WorkspaceId = ws
	now := time.Now()
	ms, stamp := now.UnixMilli(), now.Format(time.RFC3339)
	req.Data.DateModified, req.Data.DateModifiedString = &ms, &stamp
	if err := uc.requireReferencesInWorkspace(ctx, ws, req.Data.GetPlanId(), req.Data.GetJobTemplateId()); err != nil {
		return nil, err
	}
	return uc.repos.PlanJobTemplate.UpdatePlanJobTemplate(ctx, req)
}

func (uc *useCases) delete(ctx context.Context, req *pb.DeletePlanJobTemplateRequest) (*pb.DeletePlanJobTemplateResponse, error) {
	if err := uc.gate(ctx, entityid.ActionDelete); err != nil {
		return nil, err
	}
	return uc.repos.PlanJobTemplate.DeletePlanJobTemplate(ctx, req)
}
func (uc *useCases) list(ctx context.Context, req *pb.ListPlanJobTemplatesRequest) (*pb.ListPlanJobTemplatesResponse, error) {
	if err := uc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	return uc.repos.PlanJobTemplate.ListPlanJobTemplates(ctx, req)
}

// listByPlan is deliberately ungated: it is an internal composition read used
// by already-authorized Plan and Subscription views. No standalone HTTP route
// exposes it, and the workspace-aware adapter still scopes every row.
func (uc *useCases) listByPlan(ctx context.Context, req *pb.ListPlanJobTemplatesByPlanRequest) (*pb.ListPlanJobTemplatesByPlanResponse, error) {
	return uc.repos.PlanJobTemplate.ListPlanJobTemplatesByPlan(ctx, req)
}

func (uc *useCases) requireReferencesInWorkspace(ctx context.Context, ws, planID, templateID string) error {
	if uc.repos.Plan == nil || uc.repos.JobTemplate == nil {
		return errors.New("plan_job_template reference repositories unavailable")
	}
	p, err := uc.repos.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{Data: &planpb.Plan{Id: &planID}})
	if err != nil || len(p.GetData()) == 0 || p.GetData()[0].GetWorkspaceId() != ws {
		return errors.New("plan is outside workspace")
	}
	t, err := uc.repos.JobTemplate.ReadJobTemplate(ctx, &jobtemplatepb.ReadJobTemplateRequest{Data: &jobtemplatepb.JobTemplate{Id: templateID}})
	if err != nil || len(t.GetData()) == 0 || t.GetData()[0].GetWorkspaceId() != ws {
		return errors.New("job template is outside workspace")
	}
	return nil
}

type LegacyRelation struct {
	ChildTemplateID string
	SequenceOrder   int32
	Active          bool
}
type LegacyPlanComposition struct {
	PlanID, WorkspaceID, RootTemplateID string
	RootIsEmpty                         bool
	Relations                           []LegacyRelation
}

// BuildBackfillEntries is the deterministic, provider-independent PCS-P2 core.
// Existing keys are "planID|templateID" and make reruns no-ops.
func BuildBackfillEntries(in LegacyPlanComposition, existing map[string]bool) []*pb.PlanJobTemplate {
	if in.PlanID == "" || in.WorkspaceID == "" || in.RootTemplateID == "" {
		return nil
	}
	var out []*pb.PlanJobTemplate
	appendEntry := func(templateID string, order int32, pattern pb.PlanJobTemplateCompositionEntryPattern) {
		key := in.PlanID + "|" + templateID
		if templateID == "" || existing[key] {
			return
		}
		existing[key] = true
		out = append(out, &pb.PlanJobTemplate{Id: "pjt-" + strings.ReplaceAll(key, "|", "-"), Active: true, PlanId: in.PlanID, JobTemplateId: templateID, SequenceOrder: order, CompositionEntryPattern: pattern, WorkspaceId: in.WorkspaceID})
	}
	if in.RootIsEmpty {
		for _, rel := range in.Relations {
			if rel.Active {
				appendEntry(rel.ChildTemplateID, rel.SequenceOrder, pb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_BUNDLE_ENTRY)
			}
		}
	} else {
		appendEntry(in.RootTemplateID, 0, pb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_STANDALONE_ENTRY)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].GetSequenceOrder() != out[j].GetSequenceOrder() {
			return out[i].GetSequenceOrder() < out[j].GetSequenceOrder()
		}
		return out[i].GetJobTemplateId() < out[j].GetJobTemplateId()
	})
	return out
}
