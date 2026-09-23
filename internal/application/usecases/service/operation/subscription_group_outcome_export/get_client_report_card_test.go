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

type clientReportCardFakeQuery struct {
	called   bool
	gotReq   *exportpb.GetSubscriptionGroupClientReportCardRequest
	gotScope ports.SubscriptionGroupOutcomeExportScope
	response *exportpb.GetSubscriptionGroupClientReportCardResponse
	err      error
}

func (f *clientReportCardFakeQuery) GetSubscriptionGroupClientReportCardScoped(_ context.Context, req *exportpb.GetSubscriptionGroupClientReportCardRequest, scope ports.SubscriptionGroupOutcomeExportScope) (*exportpb.GetSubscriptionGroupClientReportCardResponse, error) {
	f.called = true
	f.gotReq = req
	f.gotScope = scope
	return f.response, f.err
}

func newClientReportCardUseCase(query ports.SubscriptionGroupClientReportCardQueryService, permissions ...string) *GetClientReportCardUseCase {
	allowed := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		allowed[permission] = true
	}
	return NewUseCases(
		Repositories{ClientReportCardQuery: query},
		Services{ActionGatekeeper: actiongate.NewActionGatekeeper(&getFakeAuthorizer{allowed: allowed}, nil)},
	).GetSubscriptionGroupClientReportCard
}

func validClientReportCardResponse() *exportpb.GetSubscriptionGroupClientReportCardResponse {
	return &exportpb.GetSubscriptionGroupClientReportCardResponse{
		Success: true,
		ReportCard: &exportpb.ClientReportCardProjection{
			Context:                              &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "group-1"},
			RenderGateAppliedSubscriptionGroupId: "group-1",
			ClientSubscriptionIds:                []string{"subscription-1"},
			Client:                               &exportpb.ClientReportCardClient{ClientId: "client-1", Name: "Student"},
		},
	}
}

func TestValidateClientReportCardResponseReturnsTypedNotFoundForMissingProjection(t *testing.T) {
	cases := []struct {
		name          string
		response      *exportpb.GetSubscriptionGroupClientReportCardResponse
		wantNotFound  bool
		wantRealError bool
	}{
		{name: "nil response", wantNotFound: true},
		{name: "successful empty membership", response: &exportpb.GetSubscriptionGroupClientReportCardResponse{Success: true}, wantNotFound: true},
		{name: "unsuccessful empty response", response: &exportpb.GetSubscriptionGroupClientReportCardResponse{Success: false}, wantRealError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateClientReportCardResponse(
				&exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
				tc.response,
				"workspace-1",
			)
			if gotNotFound := errors.Is(err, ports.ErrClientReportNotFound); gotNotFound != tc.wantNotFound {
				t.Fatalf("validateClientReportCardResponse() typed not-found = %v, error = %v; want %v", gotNotFound, err, tc.wantNotFound)
			}
			if tc.wantRealError && err == nil {
				t.Fatal("validateClientReportCardResponse() error = nil, want real unsuccessful-response error")
			}
		})
	}
}

func TestValidateRenderGateSheetsCoversDistinctTemplateAndSingletonKeys(t *testing.T) {
	workspaceID := "workspace-1"
	card := validClientReportCardResponse().GetReportCard()
	card.Jobs = []*jobpb.Job{{Id: "job-1", WorkspaceId: &workspaceID}}
	card.JobPhases = []*jobphasepb.JobPhase{
		{Id: "phase-1", JobId: "job-1", WorkspaceId: &workspaceID, TemplatePhaseId: stringPtr("template-phase-1"), Active: true},
		{Id: "phase-2", JobId: "job-1", WorkspaceId: &workspaceID, TemplatePhaseId: stringPtr("template-phase-1"), Active: true},
		{Id: "phase-singleton", JobId: "job-1", WorkspaceId: &workspaceID, Active: true},
	}
	card.RenderGateSheets = []*exportpb.ClientReportCardRenderGateSheet{
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
	base := func() *exportpb.ClientReportCardProjection {
		card := validClientReportCardResponse().GetReportCard()
		card.Jobs = []*jobpb.Job{{Id: "job-1", WorkspaceId: &workspaceID}}
		card.JobPhases = []*jobphasepb.JobPhase{{Id: "phase-1", JobId: "job-1", WorkspaceId: &workspaceID, TemplatePhaseId: stringPtr("template-phase-1"), Active: true}}
		card.RenderGateSheets = []*exportpb.ClientReportCardRenderGateSheet{{JobTemplatePhaseId: stringPtr("template-phase-1"), AppliedSubscriptionGroupId: "group-1", TargetCount: 1}}
		return card
	}
	cases := []struct {
		name   string
		mutate func(*exportpb.ClientReportCardProjection)
	}{
		{name: "missing", mutate: func(card *exportpb.ClientReportCardProjection) { card.RenderGateSheets = nil }},
		{name: "foreign key", mutate: func(card *exportpb.ClientReportCardProjection) {
			card.RenderGateSheets[0].JobTemplatePhaseId = stringPtr("foreign-phase")
		}},
		{name: "group mismatch", mutate: func(card *exportpb.ClientReportCardProjection) {
			card.RenderGateSheets[0].AppliedSubscriptionGroupId = "other-group"
		}},
		{name: "undersized target count", mutate: func(card *exportpb.ClientReportCardProjection) { card.RenderGateSheets[0].TargetCount = 0 }},
		{name: "top-level group mismatch", mutate: func(card *exportpb.ClientReportCardProjection) {
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

func TestGetClientReportCardUseCase(t *testing.T) {
	cases := []struct {
		name        string
		permissions []string
		req         *exportpb.GetSubscriptionGroupClientReportCardRequest
		response    *exportpb.GetSubscriptionGroupClientReportCardResponse
		wantErr     string
		wantCall    bool
	}{
		{
			name:        "scoped member projection is returned",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			response:    validClientReportCardResponse(),
			wantCall:    true,
		},
		{
			name:    "missing permission makes zero query calls",
			req:     &exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			wantErr: "permission",
		},
		{
			name:        "noncanonical client id makes zero query calls",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: " client-1 "},
			wantErr:     "client_id",
		},
		{
			name:        "response cannot substitute another client",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			response: &exportpb.GetSubscriptionGroupClientReportCardResponse{
				Success: true,
				ReportCard: &exportpb.ClientReportCardProjection{
					Context:               &exportpb.SubscriptionGroupOutcomeExportContext{SubscriptionGroupId: "group-1"},
					ClientSubscriptionIds: []string{"subscription-1"},
					Client:                &exportpb.ClientReportCardClient{ClientId: "client-2"},
				},
			},
			wantErr:  "exact group membership",
			wantCall: true,
		},
		{
			name:        "unrequested profile attributes are rejected",
			permissions: []string{exportPerm()},
			req:         &exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"},
			response: func() *exportpb.GetSubscriptionGroupClientReportCardResponse {
				result := validClientReportCardResponse()
				result.ReportCard.Attributes = []*exportpb.ClientReportCardAttribute{{Code: "lrn", Value: "x"}}
				return result
			}(),
			wantErr:  "unrequested attribute",
			wantCall: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query := &clientReportCardFakeQuery{response: tc.response}
			useCase := newClientReportCardUseCase(query, tc.permissions...)
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

func TestValidateClientReportCardResponseRejectsCrossScopeJob(t *testing.T) {
	req := &exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"}
	response := validClientReportCardResponse()
	response.ReportCard.Jobs = nil
	response.ReportCard.RenderGateJobIds = []string{"job-other"}
	err := validateClientReportCardResponse(req, response, "workspace-1")
	if err == nil || !strings.Contains(err.Error(), "unprojected job") {
		t.Fatalf("validateClientReportCardResponse() error = %v, want unprojected render-gate job rejection", err)
	}
}

func TestValidateProjectedClientIDsAllowsOnlyNullableOrSelectedClientLinesForProjectedSummary(t *testing.T) {
	workspaceID := "workspace-1"
	subscriptionID := "subscription-1"
	selectedClientID := "client-1"
	otherClientID := "client-2"
	makeCard := func(lineClientID *string, summaryClientID *string, summaryID string) *exportpb.ClientReportCardProjection {
		return &exportpb.ClientReportCardProjection{
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

func TestClientReportCardUseCasePortErrorPropagates(t *testing.T) {
	want := errors.New("database unavailable")
	query := &clientReportCardFakeQuery{err: want}
	useCase := newClientReportCardUseCase(query, exportPerm())
	_, err := useCase.Execute(getTestCtx(), &exportpb.GetSubscriptionGroupClientReportCardRequest{SubscriptionGroupId: "group-1", ClientId: "client-1"})
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want %v", err, want)
	}
}

func TestClientReportCardPermissionUsesReportExportEntity(t *testing.T) {
	if got, want := exportPerm(), entityid.EntityPermission(subscriptionGroupOutcomeExportPermissionEntity, entityid.ActionRead); got != want {
		t.Fatalf("export permission = %q, want %q", got, want)
	}
}
