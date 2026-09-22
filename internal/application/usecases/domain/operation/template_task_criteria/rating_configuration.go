package template_task_criteria

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

// validateRatingConfiguration keeps the binding-level UI instruction aligned
// with the criterion's value type. Legacy NULL/UNSPECIFIED/STANDARD rows stay
// untouched; only the new numeric-description mode requires the extra scale.
func validateRatingConfiguration(
	ctx context.Context,
	data *pb.TemplateTaskCriteria,
	outcomeCriteria outcomecriteriapb.OutcomeCriteriaDomainServiceServer,
	scoreScale scorescalepb.ScoreScaleDomainServiceServer,
	translator ports.Translator,
) error {
	if data == nil || data.GetRatingMode() != enums.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION {
		return nil
	}
	if data.GetRatingScaleId() == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"template_task_criteria.validation.rating_scale_required",
			"[ERR-DEFAULT] A rating scale is required for numeric descriptions"))
	}
	if outcomeCriteria == nil || scoreScale == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"template_task_criteria.validation.rating_dependencies_unavailable",
			"[ERR-DEFAULT] Rating configuration dependencies are unavailable"))
	}

	criteriaResp, err := outcomeCriteria.ReadOutcomeCriteria(ctx, &outcomecriteriapb.ReadOutcomeCriteriaRequest{
		Data: &outcomecriteriapb.OutcomeCriteria{Id: data.GetOutcomeCriteriaId()},
	})
	if err != nil || criteriaResp == nil || len(criteriaResp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"template_task_criteria.validation.numeric_criteria_required",
			"[ERR-DEFAULT] Numeric descriptions require a numeric outcome criterion"))
	}
	criteria := criteriaResp.GetData()[0]
	if criteria.GetCriteriaType() != enums.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE &&
		criteria.GetCriteriaType() != enums.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"template_task_criteria.validation.numeric_criteria_required",
			"[ERR-DEFAULT] Numeric descriptions require a numeric outcome criterion"))
	}

	scaleResp, err := scoreScale.ReadScoreScale(ctx, &scorescalepb.ReadScoreScaleRequest{
		Data: &scorescalepb.ScoreScale{Id: data.GetRatingScaleId()},
	})
	if err != nil || scaleResp == nil || len(scaleResp.GetData()) == 0 || !scaleResp.GetData()[0].GetActive() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"template_task_criteria.validation.rating_scale_not_found",
			"[ERR-DEFAULT] The selected rating scale is not active or was not found"))
	}
	scale := scaleResp.GetData()[0]
	if criteria.MinScore != nil && scale.InputMin != nil && scale.GetInputMin() > float64(criteria.GetMinScore()) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"template_task_criteria.validation.rating_scale_incompatible",
			"[ERR-DEFAULT] The selected rating scale does not cover the criterion's numeric range"))
	}
	if criteria.MaxScore != nil && scale.InputMax != nil && scale.GetInputMax() < float64(criteria.GetMaxScore()) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"template_task_criteria.validation.rating_scale_incompatible",
			"[ERR-DEFAULT] The selected rating scale does not cover the criterion's numeric range"))
	}
	return nil
}
