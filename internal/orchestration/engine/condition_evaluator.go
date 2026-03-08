package engine

// ConditionEvaluator evaluates workflow condition expressions.
type ConditionEvaluator interface {
	EvaluateCondition(expression string, context map[string]any) (bool, error)
}
