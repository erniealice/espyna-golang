package template_task_criteria

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	ttcpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type ratingConfigurationCriteriaRepo struct {
	outcomecriteriapb.UnimplementedOutcomeCriteriaDomainServiceServer
	criteria *outcomecriteriapb.OutcomeCriteria
}

func (r *ratingConfigurationCriteriaRepo) ReadOutcomeCriteria(context.Context, *outcomecriteriapb.ReadOutcomeCriteriaRequest) (*outcomecriteriapb.ReadOutcomeCriteriaResponse, error) {
	return &outcomecriteriapb.ReadOutcomeCriteriaResponse{Success: true, Data: []*outcomecriteriapb.OutcomeCriteria{r.criteria}}, nil
}

type ratingConfigurationScaleRepo struct {
	scorescalepb.UnimplementedScoreScaleDomainServiceServer
	scale *scorescalepb.ScoreScale
}

func (r *ratingConfigurationScaleRepo) ReadScoreScale(context.Context, *scorescalepb.ReadScoreScaleRequest) (*scorescalepb.ReadScoreScaleResponse, error) {
	return &scorescalepb.ReadScoreScaleResponse{Success: true, Data: []*scorescalepb.ScoreScale{r.scale}}, nil
}

func numericRatingConfiguration(mode enumspb.RatingMode) *ttcpb.TemplateTaskCriteria {
	return &ttcpb.TemplateTaskCriteria{
		OutcomeCriteriaId: "criterion-1",
		RatingMode:        &mode,
		RatingScaleId:     ratingStringPtr("scale-1"),
	}
}

func ratingStringPtr(v string) *string { return &v }

func TestValidateRatingConfigurationAllowsLegacyModeWithoutDependencies(t *testing.T) {
	standard := enumspb.RatingMode_RATING_MODE_STANDARD
	if err := validateRatingConfiguration(context.Background(), &ttcpb.TemplateTaskCriteria{RatingMode: &standard}, nil, nil, ports.NewNoOpTranslator()); err != nil {
		t.Fatalf("legacy standard mode should not require rating dependencies: %v", err)
	}
}

func TestValidateRatingConfigurationRequiresScale(t *testing.T) {
	mode := enumspb.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION
	data := &ttcpb.TemplateTaskCriteria{RatingMode: &mode, OutcomeCriteriaId: "criterion-1"}
	if err := validateRatingConfiguration(context.Background(), data, nil, nil, ports.NewNoOpTranslator()); err == nil || !strings.Contains(err.Error(), "rating scale") {
		t.Fatalf("error = %v, want required-scale validation", err)
	}
}

func TestValidateRatingConfigurationRequiresNumericCriterionAndActiveScale(t *testing.T) {
	mode := enumspb.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION
	ctx := context.Background()
	textCriteria := &ratingConfigurationCriteriaRepo{criteria: &outcomecriteriapb.OutcomeCriteria{
		Id:           "criterion-1",
		CriteriaType: enumspb.CriteriaType_CRITERIA_TYPE_TEXT,
	}}
	activeScale := &ratingConfigurationScaleRepo{scale: &scorescalepb.ScoreScale{Id: "scale-1", Active: true}}
	if err := validateRatingConfiguration(ctx, numericRatingConfiguration(mode), textCriteria, activeScale, ports.NewNoOpTranslator()); err == nil || !strings.Contains(err.Error(), "numeric") {
		t.Fatalf("error = %v, want numeric-criterion validation", err)
	}

	numericCriteria := &ratingConfigurationCriteriaRepo{criteria: &outcomecriteriapb.OutcomeCriteria{
		Id:           "criterion-1",
		CriteriaType: enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE,
	}}
	inactiveScale := &ratingConfigurationScaleRepo{scale: &scorescalepb.ScoreScale{Id: "scale-1", Active: false}}
	if err := validateRatingConfiguration(ctx, numericRatingConfiguration(mode), numericCriteria, inactiveScale, ports.NewNoOpTranslator()); err == nil || !strings.Contains(err.Error(), "active") {
		t.Fatalf("error = %v, want inactive-scale validation", err)
	}
}

func TestValidateRatingConfigurationAcceptsNumericCriterionAndActiveScale(t *testing.T) {
	mode := enumspb.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION
	criteria := &ratingConfigurationCriteriaRepo{criteria: &outcomecriteriapb.OutcomeCriteria{
		Id:           "criterion-1",
		CriteriaType: enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE,
	}}
	scale := &ratingConfigurationScaleRepo{scale: &scorescalepb.ScoreScale{Id: "scale-1", Active: true}}
	if err := validateRatingConfiguration(context.Background(), numericRatingConfiguration(mode), criteria, scale, ports.NewNoOpTranslator()); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidateRatingConfigurationRejectsScaleOutsideCriterionRange(t *testing.T) {
	mode := enumspb.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION
	criterionMin, criterionMax := int32(0), int32(8)
	scaleMin, scaleMax := 1.0, 7.0
	criteria := &ratingConfigurationCriteriaRepo{criteria: &outcomecriteriapb.OutcomeCriteria{
		Id: "criterion-1", CriteriaType: enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE,
		MinScore: &criterionMin, MaxScore: &criterionMax,
	}}
	scale := &ratingConfigurationScaleRepo{scale: &scorescalepb.ScoreScale{
		Id: "scale-1", Active: true, InputMin: &scaleMin, InputMax: &scaleMax,
	}}
	if err := validateRatingConfiguration(context.Background(), numericRatingConfiguration(mode), criteria, scale, ports.NewNoOpTranslator()); err == nil || !strings.Contains(err.Error(), "cover") {
		t.Fatalf("error = %v, want scale range validation", err)
	}
}
