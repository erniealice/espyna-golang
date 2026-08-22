package plan_job_template

import (
	"context"
	"sort"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/plan_job_template"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

type fakeRepo struct {
	pb.UnimplementedPlanJobTemplateDomainServiceServer
	rows map[string]*pb.PlanJobTemplate
}

func (r *fakeRepo) CreatePlanJobTemplate(_ context.Context, req *pb.CreatePlanJobTemplateRequest) (*pb.CreatePlanJobTemplateResponse, error) {
	r.rows[req.Data.GetId()] = req.Data
	return &pb.CreatePlanJobTemplateResponse{Success: true, Data: []*pb.PlanJobTemplate{req.Data}}, nil
}
func (r *fakeRepo) ReadPlanJobTemplate(_ context.Context, req *pb.ReadPlanJobTemplateRequest) (*pb.ReadPlanJobTemplateResponse, error) {
	row := r.rows[req.Data.GetId()]
	return &pb.ReadPlanJobTemplateResponse{Success: row != nil, Data: []*pb.PlanJobTemplate{row}}, nil
}
func (r *fakeRepo) UpdatePlanJobTemplate(_ context.Context, req *pb.UpdatePlanJobTemplateRequest) (*pb.UpdatePlanJobTemplateResponse, error) {
	r.rows[req.Data.GetId()] = req.Data
	return &pb.UpdatePlanJobTemplateResponse{Success: true, Data: []*pb.PlanJobTemplate{req.Data}}, nil
}
func (r *fakeRepo) DeletePlanJobTemplate(_ context.Context, req *pb.DeletePlanJobTemplateRequest) (*pb.DeletePlanJobTemplateResponse, error) {
	delete(r.rows, req.Data.GetId())
	return &pb.DeletePlanJobTemplateResponse{Success: true}, nil
}
func (r *fakeRepo) ListPlanJobTemplatesByPlan(_ context.Context, req *pb.ListPlanJobTemplatesByPlanRequest) (*pb.ListPlanJobTemplatesByPlanResponse, error) {
	var rows []*pb.PlanJobTemplate
	for _, row := range r.rows {
		if row.GetPlanId() == req.GetPlanId() && row.GetActive() {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].GetSequenceOrder() < rows[j].GetSequenceOrder() })
	return &pb.ListPlanJobTemplatesByPlanResponse{Success: true, PlanJobTemplates: rows}, nil
}

type fakePlanRepo struct {
	planpb.UnimplementedPlanDomainServiceServer
}

func (fakePlanRepo) ReadPlan(_ context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	id := req.Data.GetId()
	return &planpb.ReadPlanResponse{Success: true, Data: []*planpb.Plan{{Id: &id, WorkspaceId: strp("ws-1")}}}, nil
}

type fakeTemplateRepo struct {
	jobtemplatepb.UnimplementedJobTemplateDomainServiceServer
}

func (fakeTemplateRepo) ReadJobTemplate(_ context.Context, req *jobtemplatepb.ReadJobTemplateRequest) (*jobtemplatepb.ReadJobTemplateResponse, error) {
	return &jobtemplatepb.ReadJobTemplateResponse{Success: true, Data: []*jobtemplatepb.JobTemplate{{Id: req.Data.GetId(), WorkspaceId: strp("ws-1")}}}, nil
}
func strp(v string) *string { return &v }

func newTestUseCases(repo *fakeRepo) *UseCases {
	return NewUseCases(Repositories{PlanJobTemplate: repo, Plan: fakePlanRepo{}, JobTemplate: fakeTemplateRepo{}}, Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), IDGenerator: ports.NewNoOpIDGenerator(), ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())})
}

func TestPlanJobTemplateCreate_ReadsByPlan(t *testing.T) {
	repo := &fakeRepo{rows: map[string]*pb.PlanJobTemplate{}}
	uc := newTestUseCases(repo)
	ctx := contextutil.WithWorkspaceID(context.Background(), "ws-1")
	created, err := uc.Create(ctx, &pb.CreatePlanJobTemplateRequest{Data: &pb.PlanJobTemplate{PlanId: "plan-1", JobTemplateId: "tpl-1", CompositionEntryPattern: pb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_BUNDLE_ENTRY}})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.GetData()) != 1 {
		t.Fatalf("expected one row")
	}
	listed, err := uc.ListByPlan(ctx, &pb.ListPlanJobTemplatesByPlanRequest{PlanId: "plan-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.GetPlanJobTemplates()) != 1 || listed.GetPlanJobTemplates()[0].GetJobTemplateId() != "tpl-1" {
		t.Fatalf("round trip failed: %+v", listed)
	}
}

func TestPlanJobTemplateListByPlanOrdersBySequence(t *testing.T) {
	repo := &fakeRepo{rows: map[string]*pb.PlanJobTemplate{"b": {Id: "b", Active: true, PlanId: "p", JobTemplateId: "tb", SequenceOrder: 2}, "a": {Id: "a", Active: true, PlanId: "p", JobTemplateId: "ta", SequenceOrder: 1}}}
	resp, err := newTestUseCases(repo).ListByPlan(context.Background(), &pb.ListPlanJobTemplatesByPlanRequest{PlanId: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.GetPlanJobTemplates()[0].GetJobTemplateId(); got != "ta" {
		t.Fatalf("first=%s", got)
	}
}

func TestBackfillPlanJobTemplate_AllIntentionalRows(t *testing.T) {
	in := LegacyPlanComposition{PlanID: "p", WorkspaceID: "ws", RootTemplateID: "empty", RootIsEmpty: true, Relations: []LegacyRelation{{ChildTemplateID: "b", SequenceOrder: 2, Active: true}, {ChildTemplateID: "a", SequenceOrder: 1, Active: true}}}
	rows := BuildBackfillEntries(in, map[string]bool{})
	if len(rows) != 2 || rows[0].GetJobTemplateId() != "a" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
	if got := BuildBackfillEntries(in, map[string]bool{"p|a": true, "p|b": true}); len(got) != 0 {
		t.Fatalf("rerun inserted %d", len(got))
	}
}
func TestBackfillPlanJobTemplate_NoActionWhenNoIntentionalRows(t *testing.T) {
	if got := BuildBackfillEntries(LegacyPlanComposition{}, map[string]bool{}); len(got) != 0 {
		t.Fatalf("expected no-op")
	}
}
