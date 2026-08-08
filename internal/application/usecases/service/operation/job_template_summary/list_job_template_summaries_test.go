package job_template_summary

import (
	"context"
	"testing"

	summarypb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/job_template_summary"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

type summaryAuthorizer struct{ allowed map[string]bool }

func (a *summaryAuthorizer) IsEnabled() bool { return true }
func (a *summaryAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return a.allowed[permission], nil
}

type summaryQuery struct {
	summarypb.UnimplementedJobTemplateSummaryServiceServer
	called bool
	gotReq *summarypb.ListJobTemplateSummariesRequest
}

func (q *summaryQuery) ListJobTemplateSummaries(_ context.Context, req *summarypb.ListJobTemplateSummariesRequest) (*summarypb.ListJobTemplateSummariesResponse, error) {
	q.called = true
	q.gotReq = req
	return &summarypb.ListJobTemplateSummariesResponse{Success: true}, nil
}

func summaryGate(allowed ...string) *actiongate.ActionGatekeeper {
	permissions := make(map[string]bool, len(allowed))
	for _, permission := range allowed {
		permissions[permission] = true
	}
	return actiongate.NewActionGatekeeper(&summaryAuthorizer{allowed: permissions}, nil)
}

func summaryPermission(entity string) string {
	return entityid.EntityPermission(entity, entityid.ActionList)
}

func summaryContext() context.Context {
	return contextutil.WithUserID(context.Background(), "user-1")
}

func summaryUC(queryPort query, gate *actiongate.ActionGatekeeper) *ListJobTemplateSummariesUseCase {
	return NewListJobTemplateSummariesUseCase(
		ListJobTemplateSummariesRepositories{Query: queryPort},
		ListJobTemplateSummariesServices{ActionGatekeeper: gate},
	)
}

func TestExecuteJobListDeniedDoesNotQuery(t *testing.T) {
	port := &summaryQuery{}
	_, err := summaryUC(port, summaryGate()).Execute(summaryContext(), &summarypb.ListJobTemplateSummariesRequest{})
	if err == nil {
		t.Fatal("job:list denial must return an error")
	}
	if port.called {
		t.Fatal("job:list denial must not query")
	}
}

func TestExecuteRequestedFallbackTemplateGrantedForwardsTrue(t *testing.T) {
	port := &summaryQuery{}
	req := &summarypb.ListJobTemplateSummariesRequest{IncludeTemplateFallback: boolPtr(true)}
	_, err := summaryUC(port, summaryGate(summaryPermission(entityid.Job), summaryPermission(entityid.JobTemplate))).Execute(summaryContext(), req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !port.called || !port.gotReq.GetIncludeTemplateFallback() {
		t.Fatalf("granted fallback must reach the query, got %+v", port.gotReq)
	}
}

func TestExecuteRequestedFallbackTemplateDeniedSanitizesWithoutMutatingCaller(t *testing.T) {
	port := &summaryQuery{}
	req := &summarypb.ListJobTemplateSummariesRequest{IncludeTemplateFallback: boolPtr(true)}
	_, err := summaryUC(port, summaryGate(summaryPermission(entityid.Job))).Execute(summaryContext(), req)
	if err != nil {
		t.Fatalf("template fallback denial must not fail the summary: %v", err)
	}
	if !port.called {
		t.Fatal("template fallback denial must still query the aggregate")
	}
	if port.gotReq.GetIncludeTemplateFallback() {
		t.Fatal("denied template fallback must be forwarded as false")
	}
	if !req.GetIncludeTemplateFallback() {
		t.Fatal("caller-owned request must remain true after sanitization")
	}
}

func TestExecuteAbsentOrFalseFallbackDoesNotRequireTemplatePermission(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  *summarypb.ListJobTemplateSummariesRequest
	}{
		{name: "absent", req: &summarypb.ListJobTemplateSummariesRequest{}},
		{name: "false", req: &summarypb.ListJobTemplateSummariesRequest{IncludeTemplateFallback: boolPtr(false)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			port := &summaryQuery{}
			_, err := summaryUC(port, summaryGate(summaryPermission(entityid.Job))).Execute(summaryContext(), tc.req)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !port.called || port.gotReq.GetIncludeTemplateFallback() {
				t.Fatalf("unrequested fallback must proceed without templates, got %+v", port.gotReq)
			}
			if (tc.req.IncludeTemplateFallback == nil) != (port.gotReq.IncludeTemplateFallback == nil) {
				t.Fatalf("unrequested fallback presence must be preserved, got %+v", port.gotReq)
			}
		})
	}
}

func boolPtr(value bool) *bool { return &value }
