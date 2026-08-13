package subscription_group_document_template

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

var (
	permittedDocumentTemplateCreate = entityid.EntityPermission(entityid.DocumentTemplate, entityid.ActionCreate)
	permittedSGDTCreate             = entityid.EntityPermission(entityid.SubscriptionGroupDocumentTemplate, entityid.ActionCreate)
)

type testAuthorizer struct {
	enabled bool
	perms   map[string]bool
	asked   []string
}

func (a *testAuthorizer) IsEnabled() bool { return a.enabled }

func (a *testAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	a.asked = append(a.asked, permission)
	if a.perms == nil {
		return false, nil
	}
	return a.perms[permission], nil
}

type testTransactor struct {
	supports         bool
	executeCalls     int
	rollbackExpected bool
	rolledBack       bool
}

func (t *testTransactor) ExecuteInTransaction(_ context.Context, op func(context.Context) error) error {
	t.executeCalls++
	if err := op(context.Background()); err != nil {
		if t.rollbackExpected {
			t.rolledBack = true
			return fmt.Errorf("rollback requested: %w", err)
		}
		return err
	}
	return nil
}

func (t *testTransactor) SupportsTransactions() bool { return t.supports }
func (t *testTransactor) IsTransactionActive(context.Context) bool {
	return false
}

type fakeDocumentTemplateRepo struct {
	documenttemplatepb.UnimplementedDocumentTemplateDomainServiceServer

	createCalls      int
	createErr        error
	createResponseFn func(*documenttemplatepb.DocumentTemplate) *documenttemplatepb.CreateDocumentTemplateResponse
	order            *[]string
	lastCreateData   *documenttemplatepb.DocumentTemplate
}

func (r *fakeDocumentTemplateRepo) CreateDocumentTemplate(_ context.Context, req *documenttemplatepb.CreateDocumentTemplateRequest) (*documenttemplatepb.CreateDocumentTemplateResponse, error) {
	r.createCalls++
	if r.order != nil {
		*r.order = append(*r.order, "artifact")
	}
	r.lastCreateData = req.GetData()
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.createResponseFn != nil {
		return r.createResponseFn(req.GetData()), nil
	}
	return &documenttemplatepb.CreateDocumentTemplateResponse{Success: true, Data: []*documenttemplatepb.DocumentTemplate{req.GetData()}}, nil
}

type fakeBindingRepo struct {
	pb.UnimplementedSubscriptionGroupDocumentTemplateDomainServiceServer

	createCalls      int
	createErr        error
	createResponseFn func(*pb.SubscriptionGroupDocumentTemplate) *pb.CreateSubscriptionGroupDocumentTemplateResponse
	order            *[]string
	publishCalls     int
	deleteCalls      int
	lastCreateData   *pb.SubscriptionGroupDocumentTemplate
}

func (r *fakeBindingRepo) CreateSubscriptionGroupDocumentTemplate(_ context.Context, req *pb.CreateSubscriptionGroupDocumentTemplateRequest) (*pb.CreateSubscriptionGroupDocumentTemplateResponse, error) {
	r.createCalls++
	if r.order != nil {
		*r.order = append(*r.order, "binding")
	}
	r.lastCreateData = req.GetData()
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.createResponseFn != nil {
		return r.createResponseFn(req.GetData()), nil
	}
	return &pb.CreateSubscriptionGroupDocumentTemplateResponse{Success: true, Data: []*pb.SubscriptionGroupDocumentTemplate{req.GetData()}}, nil
}

func (r *fakeBindingRepo) PublishSubscriptionGroupDocumentTemplate(_ context.Context, _ *pb.PublishSubscriptionGroupDocumentTemplateRequest) (*pb.PublishSubscriptionGroupDocumentTemplateResponse, error) {
	r.publishCalls++
	return &pb.PublishSubscriptionGroupDocumentTemplateResponse{Success: true}, nil
}

func (r *fakeBindingRepo) DeleteSubscriptionGroupDocumentTemplate(_ context.Context, _ *pb.DeleteSubscriptionGroupDocumentTemplateRequest) (*pb.DeleteSubscriptionGroupDocumentTemplateResponse, error) {
	r.deleteCalls++
	return &pb.DeleteSubscriptionGroupDocumentTemplateResponse{Success: true}, nil
}

func newContext() context.Context {
	return contextutil.WithUserID(context.Background(), "user-1")
}

func allowedAuthorizer() *testAuthorizer {
	return &testAuthorizer{
		enabled: false,
		perms: map[string]bool{
			permittedDocumentTemplateCreate: true,
			permittedSGDTCreate:             true,
		},
	}
}

func deniedAuthorizer(missing string) *testAuthorizer {
	authz := &testAuthorizer{
		enabled: true,
		perms: map[string]bool{
			permittedDocumentTemplateCreate: true,
			permittedSGDTCreate:             true,
		},
	}
	authz.perms[missing] = false
	return authz
}

func newUC(
	artifactRepo *fakeDocumentTemplateRepo,
	bindingRepo *fakeBindingRepo,
	tx *testTransactor,
	authz *testAuthorizer,
) *CreateUploadPairUseCase {
	return NewCreateUploadPairUseCase(
		CreateUploadPairRepositories{
			DocumentTemplate:                  artifactRepo,
			SubscriptionGroupDocumentTemplate: bindingRepo,
		},
		Services{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Transactor:       tx,
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
		},
	)
}

func validPairRequest() *CreateUploadPairRequest {
	artifactWorkspace := "artifact-workspace"
	storageContainer := "container"
	storageKey := "key"
	jobCategory := "job-category-1"
	return &CreateUploadPairRequest{
		Artifact: &documenttemplatepb.DocumentTemplate{
			Id:               "artifact-1",
			WorkspaceId:      &artifactWorkspace,
			TemplateType:     "docx",
			DocumentPurpose:  "subscription_group_outcome_summary",
			Status:           "active",
			Active:           true,
			StorageContainer: &storageContainer,
			StorageKey:       &storageKey,
		},
		Binding: &pb.SubscriptionGroupDocumentTemplate{
			Id:                 "binding-1",
			WorkspaceId:        "binding-workspace",
			DocumentTemplateId: "artifact-1",
			RenderProfile:      pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			JobCategoryId:      &jobCategory,
			VersionStatus:      enums.VersionStatus_VERSION_STATUS_PUBLISHED,
			Version:            5,
			Active:             false,
			DocumentTemplate:   &documenttemplatepb.DocumentTemplate{Id: "artifact-1"},
		},
	}
}

func TestCreateUploadPair_Success_CreatesArtifactThenBindingInTransaction(t *testing.T) {
	t.Parallel()

	order := []string{}
	docRepo := &fakeDocumentTemplateRepo{order: &order}
	bindingRepo := &fakeBindingRepo{order: &order}
	tx := &testTransactor{supports: true}
	uc := newUC(docRepo, bindingRepo, tx, allowedAuthorizer())
	req := validPairRequest()

	resp, err := uc.Execute(newContext(), req)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if tx.executeCalls != 1 {
		t.Fatalf("expected 1 transaction execute, got %d", tx.executeCalls)
	}
	if !reflect.DeepEqual(order, []string{"artifact", "binding"}) {
		t.Fatalf("expected call order artifact then binding, got %v", order)
	}
	if docRepo.createCalls != 1 {
		t.Fatalf("expected 1 artifact create call, got %d", docRepo.createCalls)
	}
	if bindingRepo.createCalls != 1 {
		t.Fatalf("expected 1 binding create call, got %d", bindingRepo.createCalls)
	}
	if resp.Artifact == nil || resp.Binding == nil {
		t.Fatalf("expected artifact and binding in response")
	}
	if req.Artifact.WorkspaceId != nil {
		t.Fatalf("artifact workspace_id must be cleared")
	}
	if req.Binding.WorkspaceId != "" {
		t.Fatalf("binding workspace_id must be cleared")
	}
	if req.Binding.VersionStatus != enums.VersionStatus_VERSION_STATUS_DRAFT {
		t.Fatalf("binding version_status must be draft")
	}
	if req.Binding.Version != 0 {
		t.Fatalf("binding version must be 0")
	}
	if req.Binding.SupersedesBindingId != nil {
		t.Fatalf("supersedes_binding_id must be cleared")
	}
	if req.Binding.PublishedAt != nil {
		t.Fatalf("published_at must be cleared")
	}
	if req.Binding.PublishedAtString != nil {
		t.Fatalf("published_at_string must be cleared")
	}
	if req.Binding.PublishedBy != nil {
		t.Fatalf("published_by must be cleared")
	}
	if req.Binding.CreatedBy != nil {
		t.Fatalf("created_by must be cleared")
	}
	if req.Binding.DocumentTemplate != nil {
		t.Fatalf("document_template hydrate should be cleared before persist")
	}
	if req.Binding.DateCreated == nil || req.Binding.DateModified == nil {
		t.Fatalf("binding date fields must be set")
	}
	if bindingRepo.publishCalls != 0 || bindingRepo.deleteCalls != 0 {
		t.Fatalf("publish/delete must not be called")
	}
}

func TestCreateUploadPair_MissingPermissions_LeaveNoRepoOrTxCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		authorizer          *testAuthorizer
		expectedPermission  string
		expectedPermission2 string
	}{
		{
			name:               "missing_document_template_create",
			authorizer:         deniedAuthorizer(permittedDocumentTemplateCreate),
			expectedPermission: permittedDocumentTemplateCreate,
		},
		{
			name:                "missing_subscription_group_document_template_create",
			authorizer:          deniedAuthorizer(permittedSGDTCreate),
			expectedPermission:  permittedDocumentTemplateCreate,
			expectedPermission2: permittedSGDTCreate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			docRepo := &fakeDocumentTemplateRepo{}
			bindingRepo := &fakeBindingRepo{}
			tx := &testTransactor{supports: true}
			uc := newUC(docRepo, bindingRepo, tx, tc.authorizer)
			_, err := uc.Execute(newContext(), validPairRequest())
			if err == nil {
				t.Fatalf("expected permission denial")
			}
			if docRepo.createCalls != 0 || bindingRepo.createCalls != 0 {
				t.Fatalf("permission denial must not call repositories")
			}
			if tx.executeCalls != 0 {
				t.Fatalf("permission denial must not execute transaction")
			}
			if tc.expectedPermission2 == "" {
				if !reflect.DeepEqual(tc.authorizer.asked, []string{tc.expectedPermission}) {
					t.Fatalf("expected checks %v, got %v", []string{tc.expectedPermission}, tc.authorizer.asked)
				}
			} else {
				if !reflect.DeepEqual(tc.authorizer.asked, []string{tc.expectedPermission, tc.expectedPermission2}) {
					t.Fatalf("expected checks %v, got %v", []string{tc.expectedPermission, tc.expectedPermission2}, tc.authorizer.asked)
				}
			}
		})
	}
}

func TestCreateUploadPair_TransactionUnavailable_FailsBeforeRepositoryCalls(t *testing.T) {
	t.Parallel()

	docRepo := &fakeDocumentTemplateRepo{}
	bindingRepo := &fakeBindingRepo{}
	tx := &testTransactor{supports: false}
	uc := newUC(docRepo, bindingRepo, tx, allowedAuthorizer())

	_, err := uc.Execute(newContext(), validPairRequest())
	if err == nil {
		t.Fatalf("expected transaction required failure")
	}
	if tx.executeCalls != 0 {
		t.Fatalf("expected zero transaction executes, got %d", tx.executeCalls)
	}
	if docRepo.createCalls != 0 || bindingRepo.createCalls != 0 {
		t.Fatalf("repositories must not be called without transaction support")
	}
}

func TestCreateUploadPair_ArtifactFailureBlocksBinding(t *testing.T) {
	t.Parallel()

	docRepo := &fakeDocumentTemplateRepo{
		createResponseFn: func(*documenttemplatepb.DocumentTemplate) *documenttemplatepb.CreateDocumentTemplateResponse {
			return &documenttemplatepb.CreateDocumentTemplateResponse{Success: false}
		},
	}
	bindingRepo := &fakeBindingRepo{}
	tx := &testTransactor{supports: true}
	uc := newUC(docRepo, bindingRepo, tx, allowedAuthorizer())

	_, err := uc.Execute(newContext(), validPairRequest())
	if err == nil {
		t.Fatalf("expected artifact failure to be rejected")
	}
	if docRepo.createCalls != 1 {
		t.Fatalf("artifact create should be attempted once")
	}
	if bindingRepo.createCalls != 0 {
		t.Fatalf("binding create must not run when artifact fails")
	}
	if tx.executeCalls != 1 {
		t.Fatalf("expected transaction wrapper to run for artifact failure")
	}
}

func TestCreateUploadPair_BindingFailureTriggersRollbackSignal(t *testing.T) {
	t.Parallel()

	docRepo := &fakeDocumentTemplateRepo{}
	bindingRepo := &fakeBindingRepo{
		createResponseFn: func(*pb.SubscriptionGroupDocumentTemplate) *pb.CreateSubscriptionGroupDocumentTemplateResponse {
			return &pb.CreateSubscriptionGroupDocumentTemplateResponse{Success: false}
		},
	}
	tx := &testTransactor{supports: true, rollbackExpected: true}
	uc := newUC(docRepo, bindingRepo, tx, allowedAuthorizer())

	_, err := uc.Execute(newContext(), validPairRequest())
	if err == nil {
		t.Fatalf("expected binding failure")
	}
	if !tx.rolledBack {
		t.Fatalf("expected rollback signal on binding failure")
	}
	if docRepo.createCalls != 1 {
		t.Fatalf("artifact create should still run before binding failure")
	}
	if bindingRepo.createCalls != 1 {
		t.Fatalf("binding create should run and fail")
	}
	if tx.executeCalls != 1 {
		t.Fatalf("expected one transaction execute")
	}
}

func TestCreateUploadPair_InvalidInputsRejected_AndNoCalls(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		tweak func(*documenttemplatepb.DocumentTemplate, *pb.SubscriptionGroupDocumentTemplate)
	}{
		{
			name: "mismatch_document_template_id",
			tweak: func(_ *documenttemplatepb.DocumentTemplate, binding *pb.SubscriptionGroupDocumentTemplate) {
				binding.DocumentTemplateId = "different-artifact"
			},
		},
		{
			name: "wrong_purpose",
			tweak: func(artifact *documenttemplatepb.DocumentTemplate, _ *pb.SubscriptionGroupDocumentTemplate) {
				artifact.DocumentPurpose = "student_outcome_summary"
			},
		},
		{
			name: "unsupported_render_profile",
			tweak: func(_ *documenttemplatepb.DocumentTemplate, binding *pb.SubscriptionGroupDocumentTemplate) {
				binding.RenderProfile = pb.RenderProfile_RENDER_PROFILE_UNSPECIFIED
			},
		},
		{
			name: "missing_category",
			tweak: func(_ *documenttemplatepb.DocumentTemplate, binding *pb.SubscriptionGroupDocumentTemplate) {
				binding.JobCategoryId = nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			docRepo := &fakeDocumentTemplateRepo{}
			bindingRepo := &fakeBindingRepo{}
			tx := &testTransactor{supports: true}
			uc := newUC(docRepo, bindingRepo, tx, allowedAuthorizer())

			req := validPairRequest()
			tc.tweak(req.Artifact, req.Binding)

			_, err := uc.Execute(newContext(), req)
			if err == nil {
				t.Fatalf("expected input %s to be rejected", tc.name)
			}
			if docRepo.createCalls != 0 || bindingRepo.createCalls != 0 {
				t.Fatalf("invalid %s must not call repositories", tc.name)
			}
			if tx.executeCalls != 0 {
				t.Fatalf("invalid %s must not open a transaction", tc.name)
			}
		})
	}
}
