package job_template

// Cross-workspace FK guard + version_status round-trip tests for the job_template
// write path (red-team HIGH #2 / MED-2). In-package mocks keep the tests
// build-tag-free.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// ----- mocks ---------------------------------------------------------------

type mockTemplateRepo struct {
	pb.UnimplementedJobTemplateDomainServiceServer
	createCalls int
	lastCreated *pb.JobTemplate
}

func (m *mockTemplateRepo) CreateJobTemplate(_ context.Context, req *pb.CreateJobTemplateRequest) (*pb.CreateJobTemplateResponse, error) {
	m.createCalls++
	m.lastCreated = req.GetData()
	return &pb.CreateJobTemplateResponse{Success: true, Data: []*pb.JobTemplate{req.GetData()}}, nil
}

type mockCategoryRepo struct {
	jobcategorypb.UnimplementedJobCategoryDomainServiceServer
	wsByCategory map[string]string
}

func (m *mockCategoryRepo) ReadJobCategory(_ context.Context, req *jobcategorypb.ReadJobCategoryRequest) (*jobcategorypb.ReadJobCategoryResponse, error) {
	id := req.GetData().GetId()
	ws, ok := m.wsByCategory[id]
	if !ok {
		return &jobcategorypb.ReadJobCategoryResponse{Success: true}, nil
	}
	w := ws
	return &jobcategorypb.ReadJobCategoryResponse{Success: true, Data: []*jobcategorypb.JobCategory{{Id: id, WorkspaceId: &w}}}, nil
}

type mockProductRepo struct {
	productpb.UnimplementedProductDomainServiceServer
	wsByProduct map[string]string
}

func (m *mockProductRepo) ReadProduct(_ context.Context, req *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	id := req.GetData().GetId()
	ws, ok := m.wsByProduct[id]
	if !ok {
		return &productpb.ReadProductResponse{Success: true}, nil
	}
	w := ws
	return &productpb.ReadProductResponse{Success: true, Data: []*productpb.Product{{Id: id, WorkspaceId: &w}}}, nil
}

// ----- helpers -------------------------------------------------------------

type stubIDTmpl struct{}

func (stubIDTmpl) GenerateID() string                   { return "jt-new" }
func (stubIDTmpl) GenerateIDWithPrefix(p string) string { return p + "-new" }
func (stubIDTmpl) IsEnabled() bool                      { return true }
func (stubIDTmpl) GetProviderInfo() string              { return "stub" }

func newCreateTmplUC(repo *mockTemplateRepo, catWS, prodWS string) *CreateJobTemplateUseCase {
	return NewCreateJobTemplateUseCase(
		CreateJobTemplateRepositories{
			JobTemplate: repo,
			JobCategory: &mockCategoryRepo{wsByCategory: map[string]string{"cat-1": catWS}},
			Product:     &mockProductRepo{wsByProduct: map[string]string{"prod-1": prodWS}},
		},
		CreateJobTemplateServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      stubIDTmpl{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
			// Transactor left nil so executeCore runs directly.
		},
	)
}

func tmplReq(category, product string, status enumspb.VersionStatus) *pb.CreateJobTemplateRequest {
	data := &pb.JobTemplate{Name: "Grade 10"}
	if category != "" {
		c := category
		data.JobCategoryId = &c
	}
	if product != "" {
		p := product
		data.OutputProductId = &p
	}
	if status != enumspb.VersionStatus_VERSION_STATUS_UNSPECIFIED {
		s := status
		data.VersionStatus = &s
	}
	return &pb.CreateJobTemplateRequest{Data: data}
}

// ----- tests ---------------------------------------------------------------

func TestCreateTemplate_SameWorkspaceFKs_Success(t *testing.T) {
	repo := &mockTemplateRepo{}
	uc := newCreateTmplUC(repo, "ws-1", "ws-1")
	if _, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), tmplReq("cat-1", "prod-1", enumspb.VersionStatus_VERSION_STATUS_DRAFT)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected create once, got %d", repo.createCalls)
	}
}

func TestCreateTemplate_ForeignCategory_Rejects(t *testing.T) {
	repo := &mockTemplateRepo{}
	uc := newCreateTmplUC(repo, "ws-2", "ws-1") // category lives in ws-2
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), tmplReq("cat-1", "prod-1", enumspb.VersionStatus_VERSION_STATUS_DRAFT))
	if err == nil {
		t.Fatalf("expected foreign-category rejection")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("expected workspace error, got %q", err.Error())
	}
	if repo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d", repo.createCalls)
	}
}

func TestCreateTemplate_ForeignProduct_Rejects(t *testing.T) {
	repo := &mockTemplateRepo{}
	uc := newCreateTmplUC(repo, "ws-1", "ws-2") // product lives in ws-2
	_, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), tmplReq("cat-1", "prod-1", enumspb.VersionStatus_VERSION_STATUS_DRAFT))
	if err == nil {
		t.Fatalf("expected foreign-product rejection")
	}
	if repo.createCalls != 0 {
		t.Errorf("create must not run on rejection, got %d", repo.createCalls)
	}
}

func TestCreateTemplate_VersionStatusRoundTrips(t *testing.T) {
	repo := &mockTemplateRepo{}
	uc := newCreateTmplUC(repo, "ws-1", "ws-1")
	// No FKs set (both optional) — isolate the version_status persistence path.
	if _, err := uc.Execute(appcontext.WithWorkspaceID(context.Background(), "ws-1"), tmplReq("", "", enumspb.VersionStatus_VERSION_STATUS_PUBLISHED)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastCreated == nil {
		t.Fatalf("expected a persisted template")
	}
	if repo.lastCreated.GetVersionStatus() != enumspb.VersionStatus_VERSION_STATUS_PUBLISHED {
		t.Errorf("expected version_status PUBLISHED to survive create, got %v", repo.lastCreated.GetVersionStatus())
	}
}
