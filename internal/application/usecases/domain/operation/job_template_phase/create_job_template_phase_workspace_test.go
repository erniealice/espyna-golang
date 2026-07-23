package job_template_phase

// Cross-workspace FK guard tests for the job_template_phase write path: the
// owning job_template AND any scoring_scheme must live in the caller's workspace
// (red-team HIGH #2). In-package mocks keep the tests build-tag-free.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	scoringschemepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
)

// ----- mocks ---------------------------------------------------------------

type mockPhaseRepo struct {
	pb.UnimplementedJobTemplatePhaseDomainServiceServer
	createCalls int
}

func (m *mockPhaseRepo) CreateJobTemplatePhase(_ context.Context, req *pb.CreateJobTemplatePhaseRequest) (*pb.CreateJobTemplatePhaseResponse, error) {
	m.createCalls++
	return &pb.CreateJobTemplatePhaseResponse{Success: true, Data: []*pb.JobTemplatePhase{req.GetData()}}, nil
}

type mockPhaseTemplateRepo struct {
	jobtemplatepb.UnimplementedJobTemplateDomainServiceServer
	ws map[string]string
}

func (m *mockPhaseTemplateRepo) ReadJobTemplate(_ context.Context, req *jobtemplatepb.ReadJobTemplateRequest) (*jobtemplatepb.ReadJobTemplateResponse, error) {
	id := req.GetData().GetId()
	ws, ok := m.ws[id]
	if !ok {
		return &jobtemplatepb.ReadJobTemplateResponse{Success: true}, nil
	}
	w := ws
	return &jobtemplatepb.ReadJobTemplateResponse{Success: true, Data: []*jobtemplatepb.JobTemplate{{Id: id, WorkspaceId: &w}}}, nil
}

type mockSchemeRepo struct {
	scoringschemepb.UnimplementedScoringSchemeDomainServiceServer
	ws map[string]string
}

func (m *mockSchemeRepo) ReadScoringScheme(_ context.Context, req *scoringschemepb.ReadScoringSchemeRequest) (*scoringschemepb.ReadScoringSchemeResponse, error) {
	id := req.GetData().GetId()
	ws, ok := m.ws[id]
	if !ok {
		return &scoringschemepb.ReadScoringSchemeResponse{Success: true}, nil
	}
	w := ws
	return &scoringschemepb.ReadScoringSchemeResponse{Success: true, Data: []*scoringschemepb.ScoringScheme{{Id: id, WorkspaceId: &w}}}, nil
}

// ----- helpers -------------------------------------------------------------

type stubIDPhase struct{}

func (stubIDPhase) GenerateID() string                   { return "phase-new" }
func (stubIDPhase) GenerateIDWithPrefix(p string) string { return p + "-new" }
func (stubIDPhase) IsEnabled() bool                      { return true }
func (stubIDPhase) GetProviderInfo() string              { return "stub" }

func newCreatePhaseUC(repo *mockPhaseRepo, tmplWS, schemeWS string) *CreateJobTemplatePhaseUseCase {
	return NewCreateJobTemplatePhaseUseCase(
		CreateJobTemplatePhaseRepositories{
			JobTemplatePhase: repo,
			JobTemplate:      &mockPhaseTemplateRepo{ws: map[string]string{"tmpl-1": tmplWS}},
			ScoringScheme:    &mockSchemeRepo{ws: map[string]string{"scheme-1": schemeWS}},
		},
		CreateJobTemplatePhaseServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDPhase{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func phaseReq(scheme string) *pb.CreateJobTemplatePhaseRequest {
	data := &pb.JobTemplatePhase{Name: "Semester 1", JobTemplateId: "tmpl-1"}
	if scheme != "" {
		s := scheme
		data.ScoringSchemeId = &s
	}
	return &pb.CreateJobTemplatePhaseRequest{Data: data}
}

// ----- tests ---------------------------------------------------------------

func TestCreatePhase_SameWorkspace_Success(t *testing.T) {
	repo := &mockPhaseRepo{}
	uc := newCreatePhaseUC(repo, "ws-1", "ws-1")
	if _, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), phaseReq("scheme-1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected create once, got %d", repo.createCalls)
	}
}

func TestCreatePhase_ForeignOwningTemplate_Rejects(t *testing.T) {
	repo := &mockPhaseRepo{}
	uc := newCreatePhaseUC(repo, "ws-2", "ws-1") // owning template lives in ws-2
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), phaseReq("scheme-1"))
	if err == nil {
		t.Fatalf("expected foreign owning-template rejection")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("expected workspace error, got %q", err.Error())
	}
	if repo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d", repo.createCalls)
	}
}

func TestCreatePhase_ForeignScoringScheme_Rejects(t *testing.T) {
	repo := &mockPhaseRepo{}
	uc := newCreatePhaseUC(repo, "ws-1", "ws-2") // scheme lives in ws-2
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), phaseReq("scheme-1"))
	if err == nil {
		t.Fatalf("expected foreign scoring-scheme rejection")
	}
	if repo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d", repo.createCalls)
	}
}

func TestCreatePhase_EmptyScoringScheme_Passes(t *testing.T) {
	// The scoring_scheme FK is optional; an empty value must not trip the guard.
	repo := &mockPhaseRepo{}
	uc := newCreatePhaseUC(repo, "ws-1", "ws-1")
	if _, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), phaseReq("")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected create once, got %d", repo.createCalls)
	}
}
