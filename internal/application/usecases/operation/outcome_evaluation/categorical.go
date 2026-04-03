package outcome_evaluation

import (
	"fmt"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// categoricalEvaluator evaluates CRITERIA_TYPE_CATEGORICAL outcomes.
// It looks up the selected option by OptionKey and returns the option's MapsToDetermination.
type categoricalEvaluator struct{}

func (e *categoricalEvaluator) Evaluate(
	outcome *outcomepb.TaskOutcome,
	ctx *EvaluationContext,
) (*portsdomain.EvaluationResult, error) {
	if outcome.CategoricalValue == nil {
		return nil, fmt.Errorf("categorical evaluation requires a categorical_value")
	}
	key := *outcome.CategoricalValue

	for _, opt := range ctx.Options {
		if opt.OptionKey != key {
			continue
		}
		if opt.MapsToDetermination == nil {
			// Option exists but has no mapped determination — treat as not evaluated
			return &portsdomain.EvaluationResult{
				Determination: enums.Determination_DETERMINATION_NOT_EVALUATED,
				Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
				Notes:         fmt.Sprintf("option %q has no mapped determination", key),
			}, nil
		}
		det, err := parseDetermination(*opt.MapsToDetermination)
		if err != nil {
			return nil, fmt.Errorf("option %q: %w", key, err)
		}
		return &portsdomain.EvaluationResult{
			Determination: det,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
		}, nil
	}

	return nil, fmt.Errorf("categorical_value %q does not match any configured option", key)
}

// parseDetermination converts the string stored in MapsToDetermination to an enum value.
// The stored string is the proto enum name, e.g. "DETERMINATION_PASS".
func parseDetermination(s string) (enums.Determination, error) {
	if v, ok := enums.Determination_value[s]; ok {
		return enums.Determination(v), nil
	}
	return enums.Determination_DETERMINATION_UNSPECIFIED,
		fmt.Errorf("unknown determination value %q", s)
}
