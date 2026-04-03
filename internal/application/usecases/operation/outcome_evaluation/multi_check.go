package outcome_evaluation

import (
	"fmt"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	outcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// multiCheckEvaluator evaluates CRITERIA_TYPE_MULTI_CHECK outcomes.
//
// It joins the TaskOutcomeCheck rows against the CriteriaOption list by CriteriaOptionId,
// then applies the configured PassRule:
//
//   - ALL_REQUIRED: every option with Required=true must have Checked=true → PASS; else FAIL
//   - ALL_ITEMS:    every option (required or not) must have Checked=true  → PASS; else FAIL
//   - MIN_COUNT:    count(checked=true) >= criteria.MinPassCount           → PASS; else FAIL
type multiCheckEvaluator struct{}

func (e *multiCheckEvaluator) Evaluate(
	_ *outcomepb.TaskOutcome,
	ctx *EvaluationContext,
) (*portsdomain.EvaluationResult, error) {
	// Resolve pass rule — default to ALL_REQUIRED when absent
	passRule := enums.PassRule_PASS_RULE_ALL_REQUIRED
	if ctx.Criteria.PassRule != nil {
		passRule = *ctx.Criteria.PassRule
	}

	// Build an index of option ID → option for quick lookup
	optByID := make(map[string]struct {
		required bool
	}, len(ctx.Options))
	for _, opt := range ctx.Options {
		optByID[opt.Id] = struct{ required bool }{required: opt.Required}
	}

	// Build a set of checked option IDs
	checkedIDs := make(map[string]bool, len(ctx.Checks))
	for _, ch := range ctx.Checks {
		if ch.Checked {
			checkedIDs[ch.CriteriaOptionId] = true
		}
	}

	switch passRule {
	case enums.PassRule_PASS_RULE_ALL_REQUIRED:
		for _, opt := range ctx.Options {
			if !opt.Required {
				continue
			}
			if !checkedIDs[opt.Id] {
				return &portsdomain.EvaluationResult{
					Determination: enums.Determination_DETERMINATION_FAIL,
					Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
					Notes:         fmt.Sprintf("required item %q (%s) was not checked", opt.OptionLabel, opt.OptionKey),
				}, nil
			}
		}
		return &portsdomain.EvaluationResult{
			Determination: enums.Determination_DETERMINATION_PASS,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
		}, nil

	case enums.PassRule_PASS_RULE_ALL_ITEMS:
		for _, opt := range ctx.Options {
			if !checkedIDs[opt.Id] {
				return &portsdomain.EvaluationResult{
					Determination: enums.Determination_DETERMINATION_FAIL,
					Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
					Notes:         fmt.Sprintf("item %q (%s) was not checked", opt.OptionLabel, opt.OptionKey),
				}, nil
			}
		}
		return &portsdomain.EvaluationResult{
			Determination: enums.Determination_DETERMINATION_PASS,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
		}, nil

	case enums.PassRule_PASS_RULE_MIN_COUNT:
		minCount := 0
		if ctx.Criteria.MinPassCount != nil {
			minCount = int(*ctx.Criteria.MinPassCount)
		}
		checkedCount := len(checkedIDs)
		if checkedCount >= minCount {
			return &portsdomain.EvaluationResult{
				Determination: enums.Determination_DETERMINATION_PASS,
				Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
				Notes:         fmt.Sprintf("%d of %d items checked (required: %d)", checkedCount, len(ctx.Options), minCount),
			}, nil
		}
		return &portsdomain.EvaluationResult{
			Determination: enums.Determination_DETERMINATION_FAIL,
			Source:        enums.DeterminationSource_DETERMINATION_SOURCE_AUTO_COMPUTED,
			Notes:         fmt.Sprintf("%d items checked but %d required", checkedCount, minCount),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported pass rule: %v", passRule)
	}
}
