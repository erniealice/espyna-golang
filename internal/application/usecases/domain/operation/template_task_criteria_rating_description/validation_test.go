package template_task_criteria_rating_description

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	templatetaskcriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria_rating_description"
)

type permissionCaptureAuthorizer struct {
	permission string
}

func (a *permissionCaptureAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	a.permission = permission
	return true, nil
}

func (*permissionCaptureAuthorizer) IsEnabled() bool { return true }

type ratingDescriptionRepoMock struct {
	pb.UnimplementedTemplateTaskCriteriaRatingDescriptionDomainServiceServer
	createCalls int
	last        *pb.TemplateTaskCriteriaRatingDescription
}

func (m *ratingDescriptionRepoMock) CreateTemplateTaskCriteriaRatingDescription(_ context.Context, req *pb.CreateTemplateTaskCriteriaRatingDescriptionRequest) (*pb.CreateTemplateTaskCriteriaRatingDescriptionResponse, error) {
	m.createCalls++
	m.last = req.GetData()
	return &pb.CreateTemplateTaskCriteriaRatingDescriptionResponse{Success: true, Data: []*pb.TemplateTaskCriteriaRatingDescription{m.last}}, nil
}

type templateTaskCriteriaRepoMock struct {
	templatetaskcriteriapb.UnimplementedTemplateTaskCriteriaDomainServiceServer
	data *templatetaskcriteriapb.TemplateTaskCriteria
}

func (m *templateTaskCriteriaRepoMock) ReadTemplateTaskCriteria(_ context.Context, _ *templatetaskcriteriapb.ReadTemplateTaskCriteriaRequest) (*templatetaskcriteriapb.ReadTemplateTaskCriteriaResponse, error) {
	return &templatetaskcriteriapb.ReadTemplateTaskCriteriaResponse{Success: true, Data: []*templatetaskcriteriapb.TemplateTaskCriteria{m.data}}, nil
}

type outcomeCriteriaRepoMock struct {
	outcomecriteriapb.UnimplementedOutcomeCriteriaDomainServiceServer
	data *outcomecriteriapb.OutcomeCriteria
}

func (m *outcomeCriteriaRepoMock) ReadOutcomeCriteria(_ context.Context, _ *outcomecriteriapb.ReadOutcomeCriteriaRequest) (*outcomecriteriapb.ReadOutcomeCriteriaResponse, error) {
	return &outcomecriteriapb.ReadOutcomeCriteriaResponse{Success: true, Data: []*outcomecriteriapb.OutcomeCriteria{m.data}}, nil
}

type scoreScaleBandRepoMock struct {
	scorescalebandpb.UnimplementedScoreScaleBandDomainServiceServer
	data *scorescalebandpb.ScoreScaleBand
}

func (m *scoreScaleBandRepoMock) ReadScoreScaleBand(_ context.Context, _ *scorescalebandpb.ReadScoreScaleBandRequest) (*scorescalebandpb.ReadScoreScaleBandResponse, error) {
	return &scorescalebandpb.ReadScoreScaleBandResponse{Success: true, Data: []*scorescalebandpb.ScoreScaleBand{m.data}}, nil
}

func newRatingDescriptionUseCases(bandScaleID string) (*UseCases, *ratingDescriptionRepoMock) {
	mode := enumspb.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION
	ratingDescriptionRepo := &ratingDescriptionRepoMock{}
	return NewUseCases(
		Repositories{
			TemplateTaskCriteriaRatingDescription: ratingDescriptionRepo,
			TemplateTaskCriteria: &templateTaskCriteriaRepoMock{data: &templatetaskcriteriapb.TemplateTaskCriteria{
				Id:                "ttc-1",
				Active:            true,
				OutcomeCriteriaId: "criteria-1",
				RatingMode:        &mode,
				RatingScaleId:     stringPtr("scale-1"),
			}},
			OutcomeCriteria: &outcomeCriteriaRepoMock{data: &outcomecriteriapb.OutcomeCriteria{
				Id:           "criteria-1",
				Active:       true,
				CriteriaType: enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE,
			}},
			ScoreScaleBand: &scoreScaleBandRepoMock{data: &scorescalebandpb.ScoreScaleBand{
				Id:           "band-1",
				Active:       true,
				ScoreScaleId: bandScaleID,
			}},
		},
		Services{},
	), ratingDescriptionRepo
}

func stringPtr(v string) *string { return &v }

func ratingDescriptionRequest() *pb.CreateTemplateTaskCriteriaRatingDescriptionRequest {
	return &pb.CreateTemplateTaskCriteriaRatingDescriptionRequest{Data: &pb.TemplateTaskCriteriaRatingDescription{
		Id:                     "description-1",
		TemplateTaskCriteriaId: "ttc-1",
		ScoreScaleBandId:       "band-1",
		Description:            "Developing",
	}}
}

func TestCreateRatingDescriptionValidatesBindingAndCopiesWorkspace(t *testing.T) {
	useCases, repo := newRatingDescriptionUseCases("scale-1")
	ctx := appcontext.WithWorkspaceID(context.Background(), "workspace-1")

	if _, err := useCases.CreateTemplateTaskCriteriaRatingDescription.Execute(ctx, ratingDescriptionRequest()); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", repo.createCalls)
	}
	if repo.last == nil || repo.last.GetWorkspaceId() != "workspace-1" {
		t.Fatalf("workspace was not assigned from request context: %+v", repo.last)
	}
}

func TestCreateRatingDescriptionRejectsBandFromAnotherScale(t *testing.T) {
	useCases, repo := newRatingDescriptionUseCases("scale-foreign")
	ctx := appcontext.WithWorkspaceID(context.Background(), "workspace-1")

	_, err := useCases.CreateTemplateTaskCriteriaRatingDescription.Execute(ctx, ratingDescriptionRequest())
	if err == nil {
		t.Fatal("expected selected-scale mismatch error")
	}
	if !strings.Contains(err.Error(), "scale") {
		t.Fatalf("error = %q, want scale mismatch", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0 after validation failure", repo.createCalls)
	}
}

func TestEmbeddedDescriptionActionsReuseParentBindingPermissions(t *testing.T) {
	useCases, _ := newRatingDescriptionUseCases("scale-1")
	authorizer := &permissionCaptureAuthorizer{}
	gate := actiongate.NewActionGatekeeper(authorizer, nil)
	ctx := appcontext.WithUserID(context.Background(), "user-1")
	useCases.CreateTemplateTaskCriteriaRatingDescription.services.ActionGatekeeper = gate

	if err := useCases.CreateTemplateTaskCriteriaRatingDescription.check(ctx, entityid.ActionCreate); err != nil {
		t.Fatalf("unexpected create permission error: %v", err)
	}
	if authorizer.permission != "template_task_criteria:update" {
		t.Fatalf("create permission = %q, want parent update", authorizer.permission)
	}

	useCases.ListByTemplateTaskCriteria.services.ActionGatekeeper = gate
	if err := useCases.ListByTemplateTaskCriteria.check(ctx, entityid.ActionList); err != nil {
		t.Fatalf("unexpected list permission error: %v", err)
	}
	if authorizer.permission != "template_task_criteria:read" {
		t.Fatalf("list permission = %q, want parent read", authorizer.permission)
	}
}
