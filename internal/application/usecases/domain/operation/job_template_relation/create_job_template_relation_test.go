package job_template_relation

// Cross-workspace guard tests for the job_template_relation write/read paths.
// In-package mocks keep the tests build-tag-free (red-team HIGH #4).

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

// ----- mocks ---------------------------------------------------------------

type mockRelationRepo struct {
	jobtemplaterelationpb.UnimplementedJobTemplateRelationDomainServiceServer
	createCalls int
	existing    *jobtemplaterelationpb.JobTemplateRelation
	// listRels, when set, is returned verbatim by ListByParent so tests can probe
	// the child-template scope filter with malformed (foreign-child) edges.
	listRels []*jobtemplaterelationpb.JobTemplateRelation
}

func (m *mockRelationRepo) CreateJobTemplateRelation(_ context.Context, req *jobtemplaterelationpb.CreateJobTemplateRelationRequest) (*jobtemplaterelationpb.CreateJobTemplateRelationResponse, error) {
	m.createCalls++
	return &jobtemplaterelationpb.CreateJobTemplateRelationResponse{Success: true, Data: []*jobtemplaterelationpb.JobTemplateRelation{req.GetData()}}, nil
}

func (m *mockRelationRepo) ReadJobTemplateRelation(_ context.Context, _ *jobtemplaterelationpb.ReadJobTemplateRelationRequest) (*jobtemplaterelationpb.ReadJobTemplateRelationResponse, error) {
	if m.existing == nil {
		return &jobtemplaterelationpb.ReadJobTemplateRelationResponse{Success: true}, nil
	}
	return &jobtemplaterelationpb.ReadJobTemplateRelationResponse{Success: true, Data: []*jobtemplaterelationpb.JobTemplateRelation{m.existing}}, nil
}

func (m *mockRelationRepo) ListByParent(_ context.Context, req *jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest) (*jobtemplaterelationpb.ListJobTemplateRelationsByParentResponse, error) {
	if m.listRels != nil {
		return &jobtemplaterelationpb.ListJobTemplateRelationsByParentResponse{Success: true, JobTemplateRelations: m.listRels}, nil
	}
	return &jobtemplaterelationpb.ListJobTemplateRelationsByParentResponse{
		Success:              true,
		JobTemplateRelations: []*jobtemplaterelationpb.JobTemplateRelation{{Id: "rel-1", ParentTemplateId: req.GetParentTemplateId()}},
	}, nil
}

// mockTemplateRepo maps template id -> workspace id.
type mockTemplateRepo struct {
	jobtemplatepb.UnimplementedJobTemplateDomainServiceServer
	ws map[string]string
}

func (m *mockTemplateRepo) ReadJobTemplate(_ context.Context, req *jobtemplatepb.ReadJobTemplateRequest) (*jobtemplatepb.ReadJobTemplateResponse, error) {
	id := req.GetData().GetId()
	ws, ok := m.ws[id]
	if !ok {
		return &jobtemplatepb.ReadJobTemplateResponse{Success: true}, nil
	}
	w := ws
	return &jobtemplatepb.ReadJobTemplateResponse{Success: true, Data: []*jobtemplatepb.JobTemplate{{Id: id, WorkspaceId: &w}}}, nil
}

// ----- helpers -------------------------------------------------------------

type stubIDRel struct{}

func (stubIDRel) GenerateID() string                   { return "rel-new" }
func (stubIDRel) GenerateIDWithPrefix(p string) string { return p + "-new" }
func (stubIDRel) IsEnabled() bool                      { return true }
func (stubIDRel) GetProviderInfo() string              { return "stub" }

type noTxnRel struct{}

func (noTxnRel) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (noTxnRel) SupportsTransactions() bool               { return false }
func (noTxnRel) IsTransactionActive(context.Context) bool { return false }

func newCreateRelUC(relRepo *mockRelationRepo, tmplRepo *mockTemplateRepo) *CreateJobTemplateRelationUseCase {
	return NewCreateJobTemplateRelationUseCase(
		CreateJobTemplateRelationRepositories{JobTemplateRelation: relRepo, JobTemplate: tmplRepo},
		CreateJobTemplateRelationServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Transactor:       noTxnRel{},
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDRel{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func ctxWS(ws string) context.Context {
	return appcontext.WithWorkspaceID(context.Background(), ws)
}

func createReq(parent, child string) *jobtemplaterelationpb.CreateJobTemplateRelationRequest {
	return &jobtemplaterelationpb.CreateJobTemplateRelationRequest{
		Data: &jobtemplaterelationpb.JobTemplateRelation{ParentTemplateId: parent, ChildTemplateId: child},
	}
}

// ----- tests ---------------------------------------------------------------

func TestCreateRelation_SameWorkspace_Success(t *testing.T) {
	relRepo := &mockRelationRepo{}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-1", "child-1": "ws-1"}}
	uc := newCreateRelUC(relRepo, tmplRepo)

	if _, err := uc.Execute(ctxWS("ws-1"), createReq("parent-1", "child-1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if relRepo.createCalls != 1 {
		t.Fatalf("expected create to be called once, got %d", relRepo.createCalls)
	}
}

func TestCreateRelation_CrossWorkspaceParent_Rejects(t *testing.T) {
	relRepo := &mockRelationRepo{}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-2", "child-1": "ws-1"}}
	uc := newCreateRelUC(relRepo, tmplRepo)

	_, err := uc.Execute(ctxWS("ws-1"), createReq("parent-1", "child-1"))
	if err == nil {
		t.Fatalf("expected cross-workspace parent rejection")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("expected workspace error, got %q", err.Error())
	}
	if relRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", relRepo.createCalls)
	}
}

func TestCreateRelation_CrossWorkspaceChild_Rejects(t *testing.T) {
	relRepo := &mockRelationRepo{}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-1", "child-1": "ws-2"}}
	uc := newCreateRelUC(relRepo, tmplRepo)

	_, err := uc.Execute(ctxWS("ws-1"), createReq("parent-1", "child-1"))
	if err == nil {
		t.Fatalf("expected cross-workspace child rejection")
	}
	if relRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", relRepo.createCalls)
	}
}

func TestCreateRelation_MissingWorkspaceContext_Rejects(t *testing.T) {
	relRepo := &mockRelationRepo{}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "", "child-1": ""}}
	uc := newCreateRelUC(relRepo, tmplRepo)

	// Empty ctx workspace must fail-closed even against empty-workspace templates.
	_, err := uc.Execute(context.Background(), createReq("parent-1", "child-1"))
	if err == nil {
		t.Fatalf("expected fail-closed rejection with empty workspace context")
	}
	if relRepo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d calls", relRepo.createCalls)
	}
}

func TestListByParent_CrossWorkspace_Rejects(t *testing.T) {
	relRepo := &mockRelationRepo{}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-2"}}
	uc := NewListByParentUseCase(
		ListByParentRepositories{JobTemplateRelation: relRepo, JobTemplate: tmplRepo},
		ListByParentServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)

	_, err := uc.Execute(ctxWS("ws-1"), &jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest{ParentTemplateId: "parent-1"})
	if err == nil {
		t.Fatalf("expected cross-workspace list rejection (IDOR)")
	}
}

func TestListByParent_SameWorkspace_Success(t *testing.T) {
	relRepo := &mockRelationRepo{}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-1"}}
	uc := NewListByParentUseCase(
		ListByParentRepositories{JobTemplateRelation: relRepo, JobTemplate: tmplRepo},
		ListByParentServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)

	resp, err := uc.Execute(ctxWS("ws-1"), &jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest{ParentTemplateId: "parent-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetJobTemplateRelations()) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(resp.GetJobTemplateRelations()))
	}
}

func newReadRelUC(relRepo *mockRelationRepo, tmplRepo *mockTemplateRepo) *ReadJobTemplateRelationUseCase {
	return NewReadJobTemplateRelationUseCase(
		ReadJobTemplateRelationRepositories{JobTemplateRelation: relRepo, JobTemplate: tmplRepo},
		ReadJobTemplateRelationServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func TestReadRelation_ForeignChild_Rejects(t *testing.T) {
	// A malformed pre-existing edge: parent in-workspace, child foreign. The read
	// must not reveal the foreign child through the Spawn Graph.
	relRepo := &mockRelationRepo{existing: &jobtemplaterelationpb.JobTemplateRelation{Id: "rel-1", ParentTemplateId: "parent-1", ChildTemplateId: "child-1"}}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-1", "child-1": "ws-2"}}
	uc := newReadRelUC(relRepo, tmplRepo)

	_, err := uc.Execute(ctxWS("ws-1"), &jobtemplaterelationpb.ReadJobTemplateRelationRequest{Data: &jobtemplaterelationpb.JobTemplateRelation{Id: "rel-1"}})
	if err == nil {
		t.Fatalf("expected foreign-child rejection (Spawn Graph leak)")
	}
}

func TestReadRelation_SameWorkspaceBothEnds_Success(t *testing.T) {
	relRepo := &mockRelationRepo{existing: &jobtemplaterelationpb.JobTemplateRelation{Id: "rel-1", ParentTemplateId: "parent-1", ChildTemplateId: "child-1"}}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-1", "child-1": "ws-1"}}
	uc := newReadRelUC(relRepo, tmplRepo)

	if _, err := uc.Execute(ctxWS("ws-1"), &jobtemplaterelationpb.ReadJobTemplateRelationRequest{Data: &jobtemplaterelationpb.JobTemplateRelation{Id: "rel-1"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListByParent_FiltersForeignChildEdges(t *testing.T) {
	// Parent in-workspace, but the returned edges mix an in-workspace child with a
	// malformed foreign-child edge. Only the in-workspace child must survive.
	relRepo := &mockRelationRepo{listRels: []*jobtemplaterelationpb.JobTemplateRelation{
		{Id: "rel-ok", ParentTemplateId: "parent-1", ChildTemplateId: "child-ok"},
		{Id: "rel-leak", ParentTemplateId: "parent-1", ChildTemplateId: "child-foreign"},
	}}
	tmplRepo := &mockTemplateRepo{ws: map[string]string{"parent-1": "ws-1", "child-ok": "ws-1", "child-foreign": "ws-2"}}
	uc := NewListByParentUseCase(
		ListByParentRepositories{JobTemplateRelation: relRepo, JobTemplate: tmplRepo},
		ListByParentServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)

	resp, err := uc.Execute(ctxWS("ws-1"), &jobtemplaterelationpb.ListJobTemplateRelationsByParentRequest{ParentTemplateId: "parent-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := resp.GetJobTemplateRelations()
	if len(got) != 1 {
		t.Fatalf("expected 1 in-workspace edge after filtering, got %d", len(got))
	}
	if got[0].GetId() != "rel-ok" {
		t.Errorf("expected rel-ok to survive, got %q", got[0].GetId())
	}
}
