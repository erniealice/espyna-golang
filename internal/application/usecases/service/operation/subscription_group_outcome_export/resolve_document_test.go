package subscription_group_outcome_export

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	domainpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

type resolveFakeAuthorizer struct {
	allowed map[string]bool
}

func (f *resolveFakeAuthorizer) IsEnabled() bool { return true }

func (f *resolveFakeAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.allowed[permission], nil
}

type resolveFakePort struct {
	called    bool
	gotReq    *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest
	gotScope  ports.SubscriptionGroupOutcomeExportScope
	resp      *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse
	err       error
	returnNil bool
}

func (p *resolveFakePort) GetSubscriptionGroupOutcomeExportScoped(
	context.Context,
	*exportpb.GetSubscriptionGroupOutcomeExportRequest,
	ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.GetSubscriptionGroupOutcomeExportResponse, error) {
	return nil, errors.New("unexpected scoped get call in resolve tests")
}

func (p *resolveFakePort) ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(
	_ context.Context,
	req *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse, error) {
	p.called = true
	p.gotReq = req
	p.gotScope = scope
	if p.err != nil {
		return nil, p.err
	}
	if p.returnNil {
		return nil, nil
	}
	if p.resp != nil {
		return p.resp, nil
	}
	return &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{}, nil
}

func resolveUseCase(query ports.SubscriptionGroupOutcomeExportQueryService, allowed ...string) *ResolveDocumentUseCase {
	permissions := make(map[string]bool, len(allowed))
	for _, permission := range allowed {
		permissions[permission] = true
	}
	return NewUseCases(
		Repositories{Query: query},
		Services{ActionGatekeeper: actiongate.NewActionGatekeeper(&resolveFakeAuthorizer{allowed: permissions}, nil)},
	).ResolveSubscriptionGroupOutcomeDocumentForRender
}

func resolveContext(kinds ...int32) context.Context {
	return getTestCtx(kinds...)
}

func TestResolveSubscriptionGroupOutcomeDocumentForRender_TableDriven(t *testing.T) {
	workspaceWide := true
	workspaceNarrow := false

	cases := []struct {
		name                string
		allowed             []string
		req                 *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest
		nilQuery            bool
		resp                *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse
		returnNil           bool
		expectErr           bool
		expectErrContains   string
		expectCalled        bool
		expectWorkspaceWide *bool
		principalKind       int32
	}{
		{
			name: "export-only succeeds with successful miss and narrow workspace",
			allowed: []string{
				exportPerm(),
			},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   false,
				Success: true,
			},
			expectErr:           false,
			expectCalled:        true,
			expectWorkspaceWide: &workspaceNarrow,
		},
		{
			name: "workspace permission without export makes zero query calls",
			allowed: []string{
				perm(entityid.Workspace),
			},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			expectErr:    true,
			expectCalled: false,
		},
		{
			name: "operator staff resolver with workspace list is workspace wide",
			allowed: []string{
				exportPerm(),
				perm(entityid.Workspace),
			},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   false,
				Success: true,
			},
			expectCalled:        true,
			expectWorkspaceWide: &workspaceWide,
			principalKind:       principalTypeOperatorStaff,
		},
		{
			name: "operator staff resolver without workspace list remains servicing-grant scoped",
			allowed: []string{
				exportPerm(),
			},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   false,
				Success: true,
			},
			expectCalled:        true,
			expectWorkspaceWide: &workspaceNarrow,
			principalKind:       principalTypeOperatorStaff,
		},
		{
			name: "staff resolver remains narrow despite workspace list permission",
			allowed: []string{
				exportPerm(),
				perm(entityid.Workspace),
			},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   false,
				Success: true,
			},
			expectCalled:        true,
			expectWorkspaceWide: &workspaceNarrow,
			principalKind:       principalTypeStaff,
		},
		{
			name:    "empty supplied expected plan is rejected before query",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
				ExpectedPlanId:      stringPtr(""),
			},
			expectErr:         true,
			expectErrContains: "expected_plan_id must be nonempty and canonical when supplied",
			expectCalled:      false,
		},
		{
			name:    "empty supplied expected schedule is rejected before query",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId:     "grp-A",
				JobCategoryId:           "cat-a",
				RenderProfile:           domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
				ExpectedPriceScheduleId: stringPtr(""),
			},
			expectErr:         true,
			expectErrContains: "expected_price_schedule_id must be nonempty and canonical when supplied",
			expectCalled:      false,
		},
		{
			name: "workspace:list sets WorkspaceWide=true",
			allowed: []string{
				exportPerm(),
				perm(entityid.Workspace),
			},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   false,
				Success: true,
			},
			expectErr:           false,
			expectCalled:        true,
			expectWorkspaceWide: &workspaceWide,
		},
		{
			name:    "miss contains no locator",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   false,
				Success: true,
			},
			expectErr:    false,
			expectCalled: true,
		},
		{
			name:    "found exact profile/category/trimmed locator succeeds",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   true,
				Success: true,
				Document: &exportpb.ResolvedSubscriptionGroupOutcomeDocument{
					StorageContainer: " report ",
					StorageKey:       " blob ",
					RenderProfile:    domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
					JobCategoryId:    "cat-a",
				},
			},
			expectErr:    false,
			expectCalled: true,
		},
		{
			name:    "mismatched profile rejects",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   true,
				Success: true,
				Document: &exportpb.ResolvedSubscriptionGroupOutcomeDocument{
					StorageContainer: "report",
					StorageKey:       "blob",
					RenderProfile:    99,
					JobCategoryId:    "cat-a",
				},
			},
			expectErr:         true,
			expectErrContains: "does not match the trusted profile/category",
			expectCalled:      true,
		},
		{
			name:    "mismatched category rejects",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   true,
				Success: true,
				Document: &exportpb.ResolvedSubscriptionGroupOutcomeDocument{
					StorageContainer: "report",
					StorageKey:       "blob",
					RenderProfile:    domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
					JobCategoryId:    "cat-b",
				},
			},
			expectErr:         true,
			expectErrContains: "does not match the trusted profile/category",
			expectCalled:      true,
		},
		{
			name:    "blank locator rejects",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Found:   true,
				Success: true,
				Document: &exportpb.ResolvedSubscriptionGroupOutcomeDocument{
					StorageContainer: "  ",
					StorageKey:       " blob ",
					RenderProfile:    domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
					JobCategoryId:    "cat-a",
				},
			},
			expectErr:         true,
			expectErrContains: "contains an incomplete locator",
			expectCalled:      true,
		},
		{
			name:    "nil response rejects",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp:              nil,
			returnNil:         true,
			expectErr:         true,
			expectErrContains: "incomplete response",
			expectCalled:      true,
		},
		{
			name:    "incomplete response rejects",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			resp: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
				Success: false,
				Found:   true,
				Document: &exportpb.ResolvedSubscriptionGroupOutcomeDocument{
					StorageContainer: "report",
					StorageKey:       "blob",
					RenderProfile:    domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
					JobCategoryId:    "cat-a",
				},
			},
			expectErr:         true,
			expectErrContains: "incomplete response",
			expectCalled:      true,
		},
		{
			name:    "nil query fails closed",
			allowed: []string{exportPerm()},
			req: &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       "cat-a",
				RenderProfile:       domainpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1,
			},
			nilQuery:          true,
			expectErr:         true,
			expectErrContains: "document resolver is unavailable",
			expectCalled:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var query ports.SubscriptionGroupOutcomeExportQueryService
			if !tc.nilQuery {
				query = &resolveFakePort{
					resp:      tc.resp,
					returnNil: tc.returnNil,
				}
			}

			uc := resolveUseCase(query, tc.allowed...)
			ctx := resolveContext()
			if tc.principalKind != 0 {
				ctx = resolveContext(tc.principalKind)
			}
			_, err := uc.Execute(ctx, tc.req)

			if tc.expectErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectErrContains != "" && err != nil && !strings.Contains(err.Error(), tc.expectErrContains) {
				t.Fatalf("error %q, want %q", err, tc.expectErrContains)
			}
			port, _ := query.(*resolveFakePort)
			if tc.expectCalled != (port != nil && port.called) {
				t.Fatalf("resolve call = %v, want %v", port != nil && port.called, tc.expectCalled)
			}
			if tc.expectWorkspaceWide != nil {
				if port == nil {
					t.Fatalf("expected query call to capture workspace scope")
				}
				if port.gotScope.WorkspaceWide != *tc.expectWorkspaceWide {
					t.Fatalf("scope.WorkspaceWide=%v, want %v", port.gotScope.WorkspaceWide, *tc.expectWorkspaceWide)
				}
			}
			if tc.resp != nil && tc.resp.Success && !tc.resp.GetFound() && tc.resp.GetDocument() != nil {
				t.Fatalf("miss response should not include document locator")
			}
			if port != nil && port.called && tc.req.GetSubscriptionGroupId() != port.gotReq.GetSubscriptionGroupId() {
				t.Fatalf("request rewritten by usecase: %q", port.gotReq.GetSubscriptionGroupId())
			}
		})
	}

}
