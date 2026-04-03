package outcome_evaluation

import (
	"fmt"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// numericRangeEvaluator evaluates CRITERIA_TYPE_NUMERIC_RANGE outcomes.
//
// Zone logic:
//  1. If value is outside RECORDABLE_MIN..RECORDABLE_MAX → error (unrecordable)
//  2. CRITICAL zone (value < CRITICAL_MIN or value > CRITICAL_MAX) → FAIL
//  3. PASS zone (PASS_MIN ≤ value ≤ PASS_MAX) → PASS
//  4. WARN zone (WARN_MIN ≤ value ≤ WARN_MAX) → PASS_WITH_CONDITION
//  5. Else → FAIL
//
// When NOMINAL + TOLERANCE exist, PASS_MIN and PASS_MAX are derived as:
//
//	PASS_MIN = NOMINAL − TOLERANCE_MINUS
//	PASS_MAX = NOMINAL + TOLERANCE_PLUS
type numericRangeEvaluator struct{}

func (e *numericRangeEvaluator) Evaluate(
	outcome *outcomepb.TaskOutcome,
	ctx *EvaluationContext,
) (*portsdomain.EvaluationResult, error) {
	if outcome.NumericValue == nil {
		return nil, fmt.Errorf("numeric_range evaluation requires a numeric_value")
	}
	v := *outcome.NumericValue

	// Validate recordable range
	if recMin, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_RECORDABLE_MIN); ok {
		if v < recMin {
			return nil, fmt.Errorf("value %.4g is below recordable minimum %.4g", v, recMin)
		}
	}
	if recMax, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_RECORDABLE_MAX); ok {
		if v > recMax {
			return nil, fmt.Errorf("value %.4g is above recordable maximum %.4g", v, recMax)
		}
	}

	// Derive PASS_MIN / PASS_MAX from NOMINAL + TOLERANCE if explicit PASS_MIN/MAX absent
	passMin, hasPassMin := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_PASS_MIN)
	passMax, hasPassMax := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_PASS_MAX)

	if nominal, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_NOMINAL); ok {
		if tolPlus, ok2 := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_TOLERANCE_PLUS); ok2 && !hasPassMax {
			passMax = nominal + tolPlus
			hasPassMax = true
		}
		if tolMinus, ok2 := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_TOLERANCE_MINUS); ok2 && !hasPassMin {
			passMin = nominal - tolMinus
			hasPassMin = true
		}
	}

	// CRITICAL zone check → FAIL immediately
	if critMin, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_CRITICAL_MIN); ok {
		if v < critMin {
			return &portsdomain.EvaluationResult{
				Determination: enums.Determination_DETERMINATION_FAIL,
				Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
				Notes:         fmt.Sprintf("value %.4g is in critical low zone (< %.4g)", v, critMin),
			}, nil
		}
	}
	if critMax, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_CRITICAL_MAX); ok {
		if v > critMax {
			return &portsdomain.EvaluationResult{
				Determination: enums.Determination_DETERMINATION_FAIL,
				Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
				Notes:         fmt.Sprintf("value %.4g is in critical high zone (> %.4g)", v, critMax),
			}, nil
		}
	}

	// PASS zone check
	inPass := true
	if hasPassMin && v < passMin {
		inPass = false
	}
	if hasPassMax && v > passMax {
		inPass = false
	}
	if inPass && (hasPassMin || hasPassMax) {
		return &portsdomain.EvaluationResult{
			Determination: enums.Determination_DETERMINATION_PASS,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
		}, nil
	}

	// WARN zone check — derives warn bounds from warn tolerance if needed
	warnMin, hasWarnMin := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_WARN_MIN)
	warnMax, hasWarnMax := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_WARN_MAX)

	if nominal, ok := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_NOMINAL); ok {
		if wtp, ok2 := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_WARN_TOLERANCE_PLUS); ok2 && !hasWarnMax {
			warnMax = nominal + wtp
			hasWarnMax = true
		}
		if wtm, ok2 := threshold(ctx.Thresholds, enums.ThresholdRole_THRESHOLD_ROLE_WARN_TOLERANCE_MINUS); ok2 && !hasWarnMin {
			warnMin = nominal - wtm
			hasWarnMin = true
		}
	}

	inWarn := true
	if hasWarnMin && v < warnMin {
		inWarn = false
	}
	if hasWarnMax && v > warnMax {
		inWarn = false
	}
	if inWarn && (hasWarnMin || hasWarnMax) {
		return &portsdomain.EvaluationResult{
			Determination: enums.Determination_DETERMINATION_PASS_WITH_CONDITION,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
			Notes:         fmt.Sprintf("value %.4g is within warning zone", v),
		}, nil
	}

	// Default: outside all defined pass zones → FAIL
	return &portsdomain.EvaluationResult{
		Determination: enums.Determination_DETERMINATION_FAIL,
		Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
		Notes:         fmt.Sprintf("value %.4g is outside acceptable ranges", v),
	}, nil
}
