package subscription_group_outcome_export

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/shared/identity"
	jobdoctmplpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

type reportCardTemplateFakeAuthorizer struct {
	allowed map[string]bool
}

func (f *reportCardTemplateFakeAuthorizer) IsEnabled() bool { return true }

func (f *reportCardTemplateFakeAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.allowed[permission], nil
}

// reportCardTemplateFakeRepo satisfies jobdoctmplpb.
// JobOutcomeSummaryDocumentTemplateDomainServiceServer by embedding the
// Unimplemented stub, overriding only the resolver method exercised here.
type reportCardTemplateFakeRepo struct {
	jobdoctmplpb.UnimplementedJobOutcomeSummaryDocumentTemplateDomainServiceServer
	called bool
	gotReq *jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest
	resp   *jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse
	err    error
}

func (r *reportCardTemplateFakeRepo) FindApplicableJobOutcomeSummaryDocumentTemplate(
	_ context.Context,
	req *jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest,
) (*jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse, error) {
	r.called = true
	r.gotReq = req
	if r.err != nil {
		return nil, r.err
	}
	if r.resp != nil {
		return r.resp, nil
	}
	return &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse{Success: true, Found: false}, nil
}

func reportCardTemplateUseCase(repo jobdoctmplpb.JobOutcomeSummaryDocumentTemplateDomainServiceServer, allowed ...string) *ResolvePublishedReportCardTemplateUseCase {
	permissions := make(map[string]bool, len(allowed))
	for _, permission := range allowed {
		permissions[permission] = true
	}
	return NewUseCases(
		Repositories{JobOutcomeSummaryDocumentTemplate: repo},
		Services{ActionGatekeeper: actiongate.NewActionGatekeeper(&reportCardTemplateFakeAuthorizer{allowed: permissions}, nil)},
	).ResolvePublishedReportCardTemplate
}

func reportCardTemplateTestCtx(kind int32) context.Context {
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	return identity.WithRequestIdentity(ctx, &identity.RequestIdentity{
		UserID: "user-1", WorkspaceID: "workspace-1", PrincipalType: kind, PrincipalID: "principal-1",
	})
}

func TestResolvePublishedReportCardTemplate_DeniedWithoutExportRead(t *testing.T) {
	repo := &reportCardTemplateFakeRepo{}
	uc := reportCardTemplateUseCase(repo)
	ctx := reportCardTemplateTestCtx(principalTypeStaff)

	_, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{})
	if err == nil {
		t.Fatal("expected denial without subscription_group_outcome_export:read")
	}
	if repo.called {
		t.Fatal("expected zero repository calls on denial")
	}
}

func TestResolvePublishedReportCardTemplate_DeniedForDisallowedPrincipalType(t *testing.T) {
	repo := &reportCardTemplateFakeRepo{}
	uc := reportCardTemplateUseCase(repo, exportPerm())
	// principalType 3 is not in the {1,2,7} allowlist reportScope enforces.
	ctx := reportCardTemplateTestCtx(int32(3))

	_, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{})
	if err == nil {
		t.Fatal("expected denial for a disallowed principal type")
	}
	if repo.called {
		t.Fatal("expected zero repository calls on denial")
	}
}

func TestResolvePublishedReportCardTemplate_AllowedForStaffWithExportRead(t *testing.T) {
	scheduleID := "ps-1"
	phaseCode := "sem_1"
	repo := &reportCardTemplateFakeRepo{
		resp: &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse{
			Success: true,
			Found:   true,
			Binding: &jobdoctmplpb.JobOutcomeSummaryDocumentTemplate{
				Id:                   "binding-1",
				WorkspaceId:          "workspace-1",
				DocumentTemplateId:   "doctmpl-1",
				PriceScheduleId:      &scheduleID,
				JobTemplatePhaseCode: &phaseCode,
				CreatedBy:            strPtr("operator-1"),
				PublishedBy:          strPtr("operator-1"),
			},
		},
	}
	uc := reportCardTemplateUseCase(repo, exportPerm())
	ctx := reportCardTemplateTestCtx(principalTypeStaff)

	resp, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{
		PriceScheduleId:      &scheduleID,
		JobTemplatePhaseCode: &phaseCode,
	})
	if err != nil {
		t.Fatalf("expected STAFF with export:read to resolve, got %v", err)
	}
	if !repo.called {
		t.Fatal("expected the repository resolver to be called")
	}
	if resp == nil || !resp.GetFound() || resp.GetBinding() == nil {
		t.Fatalf("expected a found binding, got %+v", resp)
	}
}

func TestResolvePublishedReportCardTemplate_PhaseCodePassesThroughToRepo(t *testing.T) {
	scheduleID := "ps-1"
	phaseCode := "sem_2"
	repo := &reportCardTemplateFakeRepo{}
	uc := reportCardTemplateUseCase(repo, exportPerm())
	ctx := reportCardTemplateTestCtx(principalTypeOperatorOwner)

	_, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{
		PriceScheduleId:      &scheduleID,
		JobTemplatePhaseCode: &phaseCode,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotReq == nil {
		t.Fatal("expected the repository to receive a request")
	}
	if repo.gotReq.GetPriceScheduleId() != scheduleID || repo.gotReq.GetJobTemplatePhaseCode() != phaseCode {
		t.Fatalf("expected schedule/phase to pass through unchanged, got %+v", repo.gotReq)
	}
}

func TestResolvePublishedReportCardTemplate_RejectsNonCanonicalPhaseCode(t *testing.T) {
	scheduleID := "ps-1"
	badPhaseCode := "Sem 1!"
	repo := &reportCardTemplateFakeRepo{}
	uc := reportCardTemplateUseCase(repo, exportPerm())
	ctx := reportCardTemplateTestCtx(principalTypeOperatorOwner)

	_, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{
		PriceScheduleId:      &scheduleID,
		JobTemplatePhaseCode: &badPhaseCode,
	})
	if err == nil {
		t.Fatal("expected a non-canonical phase code to be rejected")
	}
	if repo.called {
		t.Fatal("expected zero repository calls on validation failure")
	}
}

func TestResolvePublishedReportCardTemplate_ClearsAuditFields(t *testing.T) {
	repo := &reportCardTemplateFakeRepo{
		resp: &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse{
			Success: true,
			Found:   true,
			Binding: &jobdoctmplpb.JobOutcomeSummaryDocumentTemplate{
				Id:          "binding-1",
				WorkspaceId: "workspace-1",
				CreatedBy:   strPtr("operator-1"),
				PublishedBy: strPtr("operator-1"),
			},
		},
	}
	uc := reportCardTemplateUseCase(repo, exportPerm())
	ctx := reportCardTemplateTestCtx(principalTypeOperatorOwner)

	resp, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.GetBinding() == nil {
		t.Fatal("expected a binding in the response")
	}
	if resp.GetBinding().CreatedBy != nil {
		t.Fatalf("expected created_by to be cleared, got %v", resp.GetBinding().GetCreatedBy())
	}
	if resp.GetBinding().PublishedBy != nil {
		t.Fatalf("expected published_by to be cleared, got %v", resp.GetBinding().GetPublishedBy())
	}
}

func TestResolvePublishedReportCardTemplate_NilRepoReturnsError(t *testing.T) {
	uc := reportCardTemplateUseCase(nil, exportPerm())
	ctx := reportCardTemplateTestCtx(principalTypeOperatorOwner)

	_, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{})
	if err == nil {
		t.Fatal("expected an error when the repository is nil")
	}
}

func TestResolvePublishedReportCardTemplate_RepoErrorPropagates(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &reportCardTemplateFakeRepo{err: wantErr}
	uc := reportCardTemplateUseCase(repo, exportPerm())
	ctx := reportCardTemplateTestCtx(principalTypeOperatorOwner)

	_, err := uc.Execute(ctx, &jobdoctmplpb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest{})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected the repository error to propagate, got %v", err)
	}
}

func TestResolvePublishedReportCardTemplate_NilRequestRejected(t *testing.T) {
	repo := &reportCardTemplateFakeRepo{}
	uc := reportCardTemplateUseCase(repo, exportPerm())
	ctx := reportCardTemplateTestCtx(principalTypeOperatorOwner)

	_, err := uc.Execute(ctx, nil)
	if err == nil {
		t.Fatal("expected a nil request to be rejected")
	}
	if repo.called {
		t.Fatal("expected zero repository calls on validation failure")
	}
}

func strPtr(value string) *string {
	return &value
}
