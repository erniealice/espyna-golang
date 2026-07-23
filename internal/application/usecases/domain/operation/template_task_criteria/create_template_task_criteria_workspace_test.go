package template_task_criteria

// Cross-workspace guard tests for the template_task_criteria write path. The
// tenant scope is inherited via task -> phase -> template -> workspace_id
// (red-team HIGH #4). In-package mocks keep the tests build-tag-free.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

// ----- mocks ---------------------------------------------------------------

type mockCriteriaRepo struct {
	pb.UnimplementedTemplateTaskCriteriaDomainServiceServer
	createCalls int
}

func (m *mockCriteriaRepo) CreateTemplateTaskCriteria(_ context.Context, req *pb.CreateTemplateTaskCriteriaRequest) (*pb.CreateTemplateTaskCriteriaResponse, error) {
	m.createCalls++
	return &pb.CreateTemplateTaskCriteriaResponse{Success: true, Data: []*pb.TemplateTaskCriteria{req.GetData()}}, nil
}

type mockTaskRepo struct {
	jobtemplatetaskpb.UnimplementedJobTemplateTaskDomainServiceServer
	phaseByTask map[string]string
}

func (m *mockTaskRepo) ReadJobTemplateTask(_ context.Context, req *jobtemplatetaskpb.ReadJobTemplateTaskRequest) (*jobtemplatetaskpb.ReadJobTemplateTaskResponse, error) {
	id := req.GetData().GetId()
	phase, ok := m.phaseByTask[id]
	if !ok {
		return &jobtemplatetaskpb.ReadJobTemplateTaskResponse{Success: true}, nil
	}
	return &jobtemplatetaskpb.ReadJobTemplateTaskResponse{Success: true, Data: []*jobtemplatetaskpb.JobTemplateTask{{Id: id, JobTemplatePhaseId: phase}}}, nil
}

type mockPhaseRepo struct {
	jobtemplatephasepb.UnimplementedJobTemplatePhaseDomainServiceServer
	templateByPhase map[string]string
}

func (m *mockPhaseRepo) ReadJobTemplatePhase(_ context.Context, req *jobtemplatephasepb.ReadJobTemplatePhaseRequest) (*jobtemplatephasepb.ReadJobTemplatePhaseResponse, error) {
	id := req.GetData().GetId()
	tmpl, ok := m.templateByPhase[id]
	if !ok {
		return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true}, nil
	}
	return &jobtemplatephasepb.ReadJobTemplatePhaseResponse{Success: true, Data: []*jobtemplatephasepb.JobTemplatePhase{{Id: id, JobTemplateId: tmpl}}}, nil
}

type mockTmplRepo struct {
	jobtemplatepb.UnimplementedJobTemplateDomainServiceServer
	wsByTemplate map[string]string
}

func (m *mockTmplRepo) ReadJobTemplate(_ context.Context, req *jobtemplatepb.ReadJobTemplateRequest) (*jobtemplatepb.ReadJobTemplateResponse, error) {
	id := req.GetData().GetId()
	ws, ok := m.wsByTemplate[id]
	if !ok {
		return &jobtemplatepb.ReadJobTemplateResponse{Success: true}, nil
	}
	w := ws
	return &jobtemplatepb.ReadJobTemplateResponse{Success: true, Data: []*jobtemplatepb.JobTemplate{{Id: id, WorkspaceId: &w}}}, nil
}

// mockCriterionRepo maps outcome_criteria id -> workspace id.
type mockCriterionRepo struct {
	outcomecriteriapb.UnimplementedOutcomeCriteriaDomainServiceServer
	wsByCriterion map[string]string
}

func (m *mockCriterionRepo) ReadOutcomeCriteria(_ context.Context, req *outcomecriteriapb.ReadOutcomeCriteriaRequest) (*outcomecriteriapb.ReadOutcomeCriteriaResponse, error) {
	id := req.GetData().GetId()
	ws, ok := m.wsByCriterion[id]
	if !ok {
		return &outcomecriteriapb.ReadOutcomeCriteriaResponse{Success: true}, nil
	}
	w := ws
	return &outcomecriteriapb.ReadOutcomeCriteriaResponse{Success: true, Data: []*outcomecriteriapb.OutcomeCriteria{{Id: id, WorkspaceId: &w}}}, nil
}

// ----- helpers -------------------------------------------------------------

type stubIDCrit struct{}

func (stubIDCrit) GenerateID() string                   { return "ttc-new" }
func (stubIDCrit) GenerateIDWithPrefix(p string) string { return p + "-new" }
func (stubIDCrit) IsEnabled() bool                      { return true }
func (stubIDCrit) GetProviderInfo() string              { return "stub" }

// newCreateCritUC wires a use case whose task chain resolves to template
// workspace tmplWS and whose pinned criterion oc-1 lives in critWS. Callers pass
// equal values for the same-workspace happy path and divergent values to probe
// each cross-workspace edge.
func newCreateCritUC(critRepo *mockCriteriaRepo, tmplWS, critWS string) *CreateTemplateTaskCriteriaUseCase {
	return NewCreateTemplateTaskCriteriaUseCase(
		CreateTemplateTaskCriteriaRepositories{
			TemplateTaskCriteria: critRepo,
			JobTemplateTask:      &mockTaskRepo{phaseByTask: map[string]string{"task-1": "phase-1"}},
			JobTemplatePhase:     &mockPhaseRepo{templateByPhase: map[string]string{"phase-1": "tmpl-1"}},
			JobTemplate:          &mockTmplRepo{wsByTemplate: map[string]string{"tmpl-1": tmplWS}},
			OutcomeCriteria:      &mockCriterionRepo{wsByCriterion: map[string]string{"oc-1": critWS}},
		},
		CreateTemplateTaskCriteriaServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDCrit{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func critReq() *pb.CreateTemplateTaskCriteriaRequest {
	return &pb.CreateTemplateTaskCriteriaRequest{
		Data: &pb.TemplateTaskCriteria{JobTemplateTaskId: "task-1", OutcomeCriteriaId: "oc-1"},
	}
}

// ----- tests ---------------------------------------------------------------

func TestCreateCriteria_SameWorkspace_Success(t *testing.T) {
	critRepo := &mockCriteriaRepo{}
	uc := newCreateCritUC(critRepo, "ws-1", "ws-1")
	if _, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), critReq()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if critRepo.createCalls != 1 {
		t.Fatalf("expected create once, got %d", critRepo.createCalls)
	}
}

func TestCreateCriteria_CrossWorkspaceChain_Rejects(t *testing.T) {
	critRepo := &mockCriteriaRepo{}
	uc := newCreateCritUC(critRepo, "ws-2", "ws-1") // owning template lives in ws-2
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), critReq())
	if err == nil {
		t.Fatalf("expected cross-workspace chain rejection")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("expected workspace error, got %q", err.Error())
	}
	if critRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", critRepo.createCalls)
	}
}

func TestCreateCriteria_ForeignCriterion_Rejects(t *testing.T) {
	critRepo := &mockCriteriaRepo{}
	// Task chain resolves in-workspace, but the pinned criterion belongs to ws-2.
	uc := newCreateCritUC(critRepo, "ws-1", "ws-2")
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), critReq())
	if err == nil {
		t.Fatalf("expected foreign-criterion rejection (cross-workspace pin)")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("expected workspace error, got %q", err.Error())
	}
	if critRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", critRepo.createCalls)
	}
}

func TestCreateCriteria_MissingWorkspaceContext_Rejects(t *testing.T) {
	critRepo := &mockCriteriaRepo{}
	uc := newCreateCritUC(critRepo, "ws-1", "ws-1")
	_, err := uc.Execute(context.Background(), critReq())
	if err == nil {
		t.Fatalf("expected fail-closed rejection with empty workspace context")
	}
	if critRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", critRepo.createCalls)
	}
}
