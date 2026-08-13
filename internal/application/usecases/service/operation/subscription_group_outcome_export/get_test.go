package subscription_group_outcome_export

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

type getFakeAuthorizer struct {
	allowed map[string]bool
}

func (f *getFakeAuthorizer) IsEnabled() bool { return true }

func (f *getFakeAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.allowed[permission], nil
}

type getResolveFakePort struct {
	getCalled bool
	getReq    *exportpb.GetSubscriptionGroupOutcomeExportRequest
	getScope  ports.SubscriptionGroupOutcomeExportScope
	getResp   *exportpb.GetSubscriptionGroupOutcomeExportResponse
	getErr    error
}

func (p *getResolveFakePort) GetSubscriptionGroupOutcomeExportScoped(
	_ context.Context,
	req *exportpb.GetSubscriptionGroupOutcomeExportRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.GetSubscriptionGroupOutcomeExportResponse, error) {
	p.getCalled = true
	p.getReq = req
	p.getScope = scope
	if p.getErr != nil {
		return nil, p.getErr
	}
	if p.getResp != nil {
		return p.getResp, nil
	}
	return &exportpb.GetSubscriptionGroupOutcomeExportResponse{}, nil
}

func (p *getResolveFakePort) ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(
	context.Context,
	*exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest,
	ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse, error) {
	return nil, errors.New("unexpected resolver call in get tests")
}

func getUseCase(query ports.SubscriptionGroupOutcomeExportQueryService, allowed ...string) *GetUseCase {
	permissions := make(map[string]bool, len(allowed))
	for _, permission := range allowed {
		permissions[permission] = true
	}
	return NewUseCases(
		Repositories{Query: query},
		Services{ActionGatekeeper: actiongate.NewActionGatekeeper(&getFakeAuthorizer{allowed: permissions}, nil)},
	).GetSubscriptionGroupOutcomeExport
}

func getTestCtx(kinds ...int32) context.Context {
	kind := principalTypeOperatorOwner
	if len(kinds) > 0 {
		kind = kinds[0]
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	return identity.WithRequestIdentity(ctx, &identity.RequestIdentity{
		UserID: "user-1", WorkspaceID: "workspace-1", PrincipalType: kind, PrincipalID: "principal-1",
	})
}

func perm(entity string) string {
	return entityid.EntityPermission(entity, entityid.ActionList)
}

func exportPerm() string {
	return entityid.EntityPermission(subscriptionGroupOutcomeExportPermissionEntity, entityid.ActionRead)
}

func stringPtr(value string) *string {
	return &value
}

func TestGetSubscriptionGroupOutcomeExport_TableDriven(t *testing.T) {
	workspaceWide := true
	workspaceNarrow := false

	cases := []struct {
		name                string
		allowed             []string
		req                 *exportpb.GetSubscriptionGroupOutcomeExportRequest
		nilQuery            bool
		portResponse        *exportpb.GetSubscriptionGroupOutcomeExportResponse
		portErr             error
		expectErr           bool
		expectErrContains   string
		expectGetCalled     bool
		expectWorkspaceWide *bool
		principalKind       int32
	}{
		{
			name:            "export permission denial makes zero query calls",
			allowed:         []string{},
			req:             &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			expectErr:       true,
			expectGetCalled: false,
		},
		{
			name:            "legacy list permission alone makes zero query calls",
			allowed:         []string{perm(entityid.JobOutcomeSummary)},
			req:             &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			expectErr:       true,
			expectGetCalled: false,
		},
		{
			name:    "operator staff with workspace list is workspace wide",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req:     &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "grp-A"},
				Success: true,
			},
			expectGetCalled:     true,
			expectWorkspaceWide: &workspaceWide,
			principalKind:       principalTypeOperatorStaff,
		},
		{
			name:    "operator staff without workspace list remains servicing-grant scoped",
			allowed: []string{exportPerm()},
			req:     &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "grp-A"},
				Success: true,
			},
			expectGetCalled:     true,
			expectWorkspaceWide: &workspaceNarrow,
			principalKind:       principalTypeOperatorStaff,
		},
		{
			name: "staff binding remains narrow despite workspace list permission",
			allowed: []string{
				exportPerm(),
				perm(entityid.Workspace),
			},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "grp-A"},
				Success: true,
			},
			expectGetCalled:     true,
			expectWorkspaceWide: &workspaceNarrow,
			principalKind:       principalTypeStaff,
		},
		{
			name: "workspace:list sets WorkspaceWide=true while report succeeds",
			allowed: []string{
				exportPerm(),
				perm(entityid.Workspace),
			},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
			},
			expectGetCalled:     true,
			expectWorkspaceWide: &workspaceWide,
		},
		{
			name: "workspace:list denied leaves false but still reads",
			allowed: []string{
				exportPerm(),
			},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
			},
			expectGetCalled:     true,
			expectWorkspaceWide: &workspaceNarrow,
		},
		{
			name:              "nil query fails closed",
			nilQuery:          true,
			allowed:           []string{exportPerm(), perm(entityid.Workspace)},
			req:               &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			expectErr:         true,
			expectErrContains: "subscription group outcome export is unavailable",
		},
		{
			name:    "options response with nested categories succeeds",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req:     &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: "grp-A"},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a"},
						},
					},
					{
						JobCategoryId: "cat-b",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_b"},
							{Code: "phase_c"},
						},
					},
				},
			},
			expectGetCalled: true,
		},
		{
			name:    "options refresh rejects a stale requested category",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-stale"),
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "grp-A"},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{JobCategoryId: "cat-a", JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{{Code: "phase_a"}}},
				},
			},
			expectErr:         true,
			expectErrContains: "requested job category is not present in the scoped options",
			expectGetCalled:   true,
		},
		{
			name:    "category/phase selector mismatch rejects",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
					JobTemplatePhaseCode: "phase_x",
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a"},
						},
					},
				},
			},
			expectErr:         true,
			expectErrContains: "selected job template phase is not available for the category",
			expectGetCalled:   true,
		},
		{
			name:    "category/phase selector ambiguous rejects",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
					JobTemplatePhaseCode: "phase_a",
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a", Ambiguous: true},
						},
					},
				},
			},
			expectErr:         true,
			expectErrContains: "selected job template phase is ambiguous",
			expectGetCalled:   true,
		},
		{
			name:    "final unavailable rejects",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome{
					FinalOutcome: true,
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId:         "cat-a",
						FinalOutcomeAvailable: false,
					},
				},
			},
			expectErr:         true,
			expectErrContains: "final outcome is not available for the category",
			expectGetCalled:   true,
		},
		{
			name:    "missing enrollment evidence rejects",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
					JobTemplatePhaseCode: "phase_a",
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a"},
						},
					},
				},
				JobTemplateColumns: []*exportpb.JobTemplateColumn{
					{JobTemplateId: "tp-a"},
				},
				ClientRows: []*exportpb.SubscriptionGroupOutcomeClientRow{
					{
						ClientId: "client-a",
						Cells: []*exportpb.SubscriptionGroupOutcomeCell{
							{JobTemplateId: "tp-a"},
						},
					},
				},
			},
			expectErr:         true,
			expectErrContains: "contains a cell without enrollment evidence",
			expectGetCalled:   true,
		},
		{
			name:    "duplicate cell identity rejects",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
					JobTemplatePhaseCode: "phase_a",
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a"},
						},
					},
				},
				JobTemplateColumns: []*exportpb.JobTemplateColumn{
					{JobTemplateId: "tp-a"},
					{JobTemplateId: "tp-b"},
				},
				ClientRows: []*exportpb.SubscriptionGroupOutcomeClientRow{
					{
						ClientId: "client-a",
						Cells: []*exportpb.SubscriptionGroupOutcomeCell{
							{
								JobTemplateId:      "tp-a",
								EnrollmentEvidence: &exportpb.EnrollmentEvidence{},
							},
							{
								JobTemplateId:      "tp-a",
								EnrollmentEvidence: &exportpb.EnrollmentEvidence{},
							},
						},
					},
				},
			},
			expectErr:         true,
			expectErrContains: "contains a duplicate job template cell",
			expectGetCalled:   true,
		},
		{
			name:    "missing job-template identity in cell rejects",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
					JobTemplatePhaseCode: "phase_a",
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a"},
						},
					},
				},
				JobTemplateColumns: []*exportpb.JobTemplateColumn{
					{JobTemplateId: "tp-a"},
				},
				ClientRows: []*exportpb.SubscriptionGroupOutcomeClientRow{
					{
						ClientId: "client-a",
						Cells: []*exportpb.SubscriptionGroupOutcomeCell{
							{
								EnrollmentEvidence: &exportpb.EnrollmentEvidence{},
							},
						},
					},
				},
			},
			expectErr:         true,
			expectErrContains: "unknown job template cell",
			expectGetCalled:   true,
		},
		{
			name:    "unknown cell identity rejects",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
					JobTemplatePhaseCode: "phase_a",
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a"},
						},
					},
				},
				JobTemplateColumns: []*exportpb.JobTemplateColumn{
					{JobTemplateId: "tp-a"},
				},
				ClientRows: []*exportpb.SubscriptionGroupOutcomeClientRow{
					{
						ClientId: "client-a",
						Cells: []*exportpb.SubscriptionGroupOutcomeCell{
							{
								JobTemplateId:      "tp-z",
								EnrollmentEvidence: &exportpb.EnrollmentEvidence{},
							},
						},
					},
				},
			},
			expectErr:         true,
			expectErrContains: "unknown job template cell",
			expectGetCalled:   true,
		},
		{
			name:    "valid complete keyed rectangle succeeds",
			allowed: []string{exportPerm(), perm(entityid.Workspace)},
			req: &exportpb.GetSubscriptionGroupOutcomeExportRequest{
				SubscriptionGroupId: "grp-A",
				JobCategoryId:       stringPtr("cat-a"),
				OutcomeSelector: &exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode{
					JobTemplatePhaseCode: "phase_a",
				},
			},
			portResponse: &exportpb.GetSubscriptionGroupOutcomeExportResponse{
				Context: &exportpb.SubscriptionGroupOutcomeExportContext{
					SubscriptionGroupId: "grp-A",
				},
				Success: true,
				JobCategories: []*exportpb.JobCategoryOption{
					{
						JobCategoryId: "cat-a",
						JobTemplatePhases: []*exportpb.JobTemplatePhaseOption{
							{Code: "phase_a"},
						},
					},
				},
				JobTemplateColumns: []*exportpb.JobTemplateColumn{
					{JobTemplateId: "tp-a"},
					{JobTemplateId: "tp-b"},
				},
				ClientRows: []*exportpb.SubscriptionGroupOutcomeClientRow{
					{
						ClientId: "client-a",
						Cells: []*exportpb.SubscriptionGroupOutcomeCell{
							{
								JobTemplateId:      "tp-a",
								EnrollmentEvidence: &exportpb.EnrollmentEvidence{},
							},
							{
								JobTemplateId:      "tp-b",
								EnrollmentEvidence: &exportpb.EnrollmentEvidence{},
							},
						},
					},
				},
			},
			expectGetCalled: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var query ports.SubscriptionGroupOutcomeExportQueryService
			if !tc.nilQuery {
				query = &getResolveFakePort{
					getResp: tc.portResponse,
					getErr:  tc.portErr,
				}
			}

			uc := getUseCase(query, tc.allowed...)
			ctx := getTestCtx()
			if tc.principalKind != 0 {
				ctx = getTestCtx(tc.principalKind)
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

			port, _ := query.(*getResolveFakePort)
			if tc.expectGetCalled != (port != nil && port.getCalled) {
				t.Fatalf("get call = %v, want %v", port != nil && port.getCalled, tc.expectGetCalled)
			}
			if tc.expectWorkspaceWide != nil {
				if port == nil {
					t.Fatalf("expected query call to capture workspace scope")
				}
				if port.getScope.WorkspaceWide != *tc.expectWorkspaceWide {
					t.Fatalf("scope.WorkspaceWide=%v, want %v", port.getScope.WorkspaceWide, *tc.expectWorkspaceWide)
				}
			}
			if tc.portErr != nil && port != nil && !errors.Is(err, tc.portErr) && err == nil {
				t.Fatalf("expected query error %v, got %v", tc.portErr, err)
			}
			if port != nil && port.getCalled && port.getReq.GetSubscriptionGroupId() != tc.req.GetSubscriptionGroupId() {
				t.Fatalf("request was rewritten: %q", port.getReq.GetSubscriptionGroupId())
			}
		})
	}
}
