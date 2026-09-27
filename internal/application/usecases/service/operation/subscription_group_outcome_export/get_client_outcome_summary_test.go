package subscription_group_outcome_export

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	joblinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_line"
	jobsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

type clientOutcomeSummaryFakeQuery struct {
	called   bool
	gotReq   *exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest
	gotScope ports.SubscriptionGroupOutcomeExportScope
	response *exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse
	err      error
}

func (f *clientOutcomeSummaryFakeQuery) GetSubscriptionGroupClientOutcomeSummaryScoped(_ context.Context, req *exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest, scope ports.SubscriptionGroupOutcomeExportScope) (*exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse, error) {
	f.called = true
	f.gotReq = req
	f.gotScope = scope
	return f.response, f.err
}

func newClientOutcomeSummaryUseCase(query ports.SubscriptionGroupClientOutcomeSummaryQueryService, permissions ...string) *GetClientOutcomeSummaryUseCase {
	allowed := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		allowed[permission] = true
	}
	return NewUseCases(
		Repositories{ClientOutcomeSummaryQuery: query},
		Services{ActionGatekeeper: actiongate.NewActionGatekeeper(&getFakeAuthorizer{allowed: allowed}, nil)},
	).GetSubscriptionGroupClientOutcomeSummary
}

func validClientOutcomeSummaryResponse() *exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse {
	return &exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse{
		Success: true,
		OutcomeSummary: &exportpb.ClientOutcomeSummaryProjection{
			Context:                              &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "group-1"},
			RenderGateAppliedSubscriptionGroupId: "group-1",
			ClientSubscriptionIds:                []string{"subscription-1"},
			Client:                               &exportpb.ClientOutcomeSummaryClient{ClientId: "client-1", Name: "Student"},
		},
	}
}

func TestValidateClientOutcomeSummaryResponseReturnsTypedNotFoundForMissingProjection(t *testing.T) {
	cases := []struct {
		name          string
		response      *exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse
		wantNotFound  bool
		wantRealError bool
	}{
		{name: "nil response", wantNotFound: true},
		{name: "successful empty membership", response: &exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse{Success: true}, wantNotFound: true},
		{name: "unsuccessful empty response", response: &exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse{Success: false}, wantRealError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateClientOutcomeSummaryResponse(
				&exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
				tc.response,
				"workspace-1",
			)
			if gotNotFound := errors.Is(err, ports.ErrClientReportNotFound); gotNotFound != tc.wantNotFound {
				t.Fatalf("validateClientOutcomeSummaryResponse() typed not-found = %v, error = %v; want %v", gotNotFound, err, tc.wantNotFound)
			}
			if tc.wantRealError && err == nil {
				t.Fatal("validateClientOutcomeSummaryResponse() error = nil, want real unsuccessful-response error")
			}
		})
	}
}

func TestValidateRenderGateSheetsCoversDistinctTemplateAndSingletonKeys(t *testing.T) {
	workspaceID := "workspace-1"
	card := validClientOutcomeSummaryResponse().GetOutcomeSummary()
	card.Jobs = []*jobpb.Job{{Id: "job-1", WorkspaceId: &workspaceID}}
	card.JobPhases = []*jobphasepb.JobPhase{
		{Id: "phase-1", JobId: "job-1", WorkspaceId: &workspaceID, TemplatePhaseId: stringPtr("template-phase-1"), Active: true},
		{Id: "phase-2", JobId: "job-1", WorkspaceId: &workspaceID, TemplatePhaseId: stringPtr("template-phase-1"), Active: true},
		{Id: "phase-singleton", JobId: "job-1", WorkspaceId: &workspaceID, Active: true},
	}
	card.RenderGateSheets = []*exportpb.ClientOutcomeSummaryRenderGateSheet{
		{JobTemplatePhaseId: stringPtr("template-phase-1"), AppliedSubscriptionGroupId: "group-1", TargetCount: 4},
		{JobPhaseId: stringPtr("phase-singleton"), AppliedSubscriptionGroupId: "group-1", TargetCount: 1},
	}
	if err := validateRenderGateSheets("group-1", workspaceID, card); err != nil {
		t.Fatalf("validateRenderGateSheets() rejected complete distinct-sheet coverage: %v", err)
	}
	card.RenderGateSheets = append(card.RenderGateSheets, card.RenderGateSheets[0])
	if err := validateRenderGateSheets("group-1", workspaceID, card); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate sheet error = %v, want duplicate rejection", err)
	}
}

func TestValidateRenderGateSheetsFailsClosedForMissingForeignAndIncompleteEvidence(t *testing.T) {
	workspaceID := "workspace-1"
	base := func() *exportpb.ClientOutcomeSummaryProjection {
		card := validClientOutcomeSummaryResponse().GetOutcomeSummary()
		card.Jobs = []*jobpb.Job{{Id: "job-1", WorkspaceId: &workspaceID}}
		card.JobPhases = []*jobphasepb.JobPhase{{Id: "phase-1", JobId: "job-1", WorkspaceId: &workspaceID, TemplatePhaseId: stringPtr("template-phase-1"), Active: true}}
		card.RenderGateSheets = []*exportpb.ClientOutcomeSummaryRenderGateSheet{{JobTemplatePhaseId: stringPtr("template-phase-1"), AppliedSubscriptionGroupId: "group-1", TargetCount: 1}}
		return card
	}
	cases := []struct {
		name   string
		mutate func(*exportpb.ClientOutcomeSummaryProjection)
	}{
		{name: "missing", mutate: func(card *exportpb.ClientOutcomeSummaryProjection) { card.RenderGateSheets = nil }},
		{name: "foreign key", mutate: func(card *exportpb.ClientOutcomeSummaryProjection) {
			card.RenderGateSheets[0].JobTemplatePhaseId = stringPtr("foreign-phase")
		}},
		{name: "group mismatch", mutate: func(card *exportpb.ClientOutcomeSummaryProjection) {
			card.RenderGateSheets[0].AppliedSubscriptionGroupId = "other-group"
		}},
		{name: "undersized target count", mutate: func(card *exportpb.ClientOutcomeSummaryProjection) { card.RenderGateSheets[0].TargetCount = 0 }},
		{name: "top-level group mismatch", mutate: func(card *exportpb.ClientOutcomeSummaryProjection) {
			card.RenderGateAppliedSubscriptionGroupId = "other-group"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			card := base()
			tc.mutate(card)
			if err := validateRenderGateSheets("group-1", workspaceID, card); err == nil {
				t.Fatal("expected fail-closed render-gate validation error")
			}
		})
	}
}

func TestGetClientOutcomeSummaryUseCase(t *testing.T) {
	cases := []struct {
		name        string
		permissions []string
		req         *exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest
		response    *exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse
		wantErr     string
		wantCall    bool
	}{
		{
			name:        "scoped member projection is returned",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			response:    validClientOutcomeSummaryResponse(),
			wantCall:    true,
		},
		{
			name:    "missing permission makes zero query calls",
			req:     &exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			wantErr: "permission",
		},
		{
			name:        "noncanonical client id makes zero query calls",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: " client-1 "},
			wantErr:     "client_id",
		},
		{
			name:        "response cannot substitute another client",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			response: &exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse{
				Success: true,
				OutcomeSummary: &exportpb.ClientOutcomeSummaryProjection{
					Context:               &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "group-1"},
					ClientSubscriptionIds: []string{"subscription-1"},
					Client:                &exportpb.ClientOutcomeSummaryClient{ClientId: "client-2"},
				},
			},
			wantErr:  "exact group membership",
			wantCall: true,
		},
		{
			name:        "unrequested profile attributes are rejected",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			response: func() *exportpb.GetSubscriptionGroupClientOutcomeSummaryResponse {
				result := validClientOutcomeSummaryResponse()
				result.OutcomeSummary.Attributes = []*exportpb.ClientOutcomeSummaryAttribute{{Code: "lrn", Value: "x"}}
				return result
			}(),
			wantErr:  "unrequested attribute",
			wantCall: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query := &clientOutcomeSummaryFakeQuery{response: tc.response}
			useCase := newClientOutcomeSummaryUseCase(query, tc.permissions...)
			got, err := useCase.Execute(getTestCtx(), tc.req)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.wantErr))) {
				t.Fatalf("Execute() error = %v, want substring %q", err, tc.wantErr)
			}
			if query.called != tc.wantCall {
				t.Fatalf("query called = %v, want %v", query.called, tc.wantCall)
			}
			if tc.wantErr == "" && got != tc.response {
				t.Fatal("Execute() did not return the validated projection response")
			}
		})
	}
}

func TestValidateClientOutcomeSummaryResponseRejectsCrossScopeJob(t *testing.T) {
	req := &exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"}
	response := validClientOutcomeSummaryResponse()
	response.OutcomeSummary.Jobs = nil
	response.OutcomeSummary.RenderGateJobIds = []string{"job-other"}
	err := validateClientOutcomeSummaryResponse(req, response, "workspace-1")
	if err == nil || !strings.Contains(err.Error(), "unprojected job") {
		t.Fatalf("validateClientOutcomeSummaryResponse() error = %v, want unprojected render-gate job rejection", err)
	}
}

func TestValidateProjectedClientIDsAllowsOnlyNullableOrSelectedClientLinesForProjectedSummary(t *testing.T) {
	workspaceID := "workspace-1"
	subscriptionID := "subscription-1"
	selectedClientID := "client-1"
	otherClientID := "client-2"
	makeCard := func(lineClientID *string, summaryClientID *string, summaryID string) *exportpb.ClientOutcomeSummaryProjection {
		return &exportpb.ClientOutcomeSummaryProjection{
			Jobs: []*jobpb.Job{{Id: "job-1", WorkspaceId: &workspaceID, OriginId: &subscriptionID}},
			JobOutcomeSummaries: []*jobsumpb.JobOutcomeSummary{{
				Id: "summary-1", JobId: "job-1", WorkspaceId: workspaceID, ClientId: summaryClientID,
			}},
			JobOutcomeLines: []*joblinepb.JobOutcomeLine{{
				JobOutcomeSummaryId: summaryID, WorkspaceId: workspaceID, ClientId: lineClientID,
			}},
		}
	}
	cases := []struct {
		name            string
		lineClientID    *string
		summaryClientID *string
		summaryID       string
		wantErr         bool
	}{
		{name: "nullable line is scoped by projected parent", summaryID: "summary-1"},
		{name: "selected-client line is accepted", lineClientID: &selectedClientID, summaryID: "summary-1"},
		{name: "another client line is rejected", lineClientID: &otherClientID, summaryID: "summary-1", wantErr: true},
		{name: "another client summary is rejected", summaryClientID: &otherClientID, summaryID: "summary-1", wantErr: true},
		{name: "line without projected parent is rejected", summaryID: "summary-other", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProjectedClientIDs("client-1", workspaceID, makeCard(tc.lineClientID, tc.summaryClientID, tc.summaryID))
			if tc.wantErr && err == nil {
				t.Fatal("expected cross-scope projection to be rejected")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("valid projected outcome line was rejected: %v", err)
			}
		})
	}
}

func TestClientOutcomeSummaryUseCasePortErrorPropagates(t *testing.T) {
	want := errors.New("database unavailable")
	query := &clientOutcomeSummaryFakeQuery{err: want}
	useCase := newClientOutcomeSummaryUseCase(query, exportPerm())
	_, err := useCase.Execute(getTestCtx(), &exportpb.GetSubscriptionGroupClientOutcomeSummaryRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"})
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want %v", err, want)
	}
}

func TestClientOutcomeSummaryPermissionUsesReportExportEntity(t *testing.T) {
	if got, want := exportPerm(), entityid.EntityPermission(subscriptionGroupOutcomeExportPermissionEntity, entityid.ActionRead); got != want {
		t.Fatalf("export permission = %q, want %q", got, want)
	}
}
