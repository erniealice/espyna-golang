package engine

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// CELEvaluator provides CEL expression evaluation for workflow conditions.
// It supports expressions like:
//   - stage[0].activity[0].output.payment_status != "success"
//   - has(input.customer_email)
//   - input.amount > 100
type CELEvaluator struct {
	env *cel.Env
}

// NewCELEvaluator creates a new CEL evaluator with workflow-specific variables.
// The environment provides access to:
//   - input: workflow input data
//   - stage: stage outputs indexed by stage number (e.g., stage[0].activity[0].output)
//   - computed: computed values from workflow context
func NewCELEvaluator() (*CELEvaluator, error) {
	env, err := cel.NewEnv(
		// Declare workflow context variables as dynamic maps
		cel.Variable("input", cel.DynType),
		cel.Variable("stage", cel.DynType),
		cel.Variable("computed", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &CELEvaluator{env: env}, nil
}

// EvaluateCondition evaluates a CEL expression against the workflow context.
// Returns true if the condition is met, false otherwise.
// On error, returns false with the error for fail-open behavior at the caller's discretion.
func (e *CELEvaluator) EvaluateCondition(expression string, context map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}

	// Parse the expression
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	// Create the program
	prg, err := e.env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Build evaluation context with defaults for missing keys
	evalContext := map[string]any{
		"input":    getMapOrEmpty(context, "input"),
		"stage":    getMapOrEmpty(context, "stage"),
		"computed": getMapOrEmpty(context, "computed"),
	}

	// Evaluate the expression
	out, _, err := prg.Eval(evalContext)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	// Convert result to boolean
	return toBool(out)
}

// getMapOrEmpty returns the value for key as a map, or an empty map if not found or wrong type.
func getMapOrEmpty(m map[string]any, key string) map[string]any {
	if m == nil {
		return make(map[string]any)
	}
	if val, ok := m[key]; ok {
		if mapVal, ok := val.(map[string]any); ok {
			return mapVal
		}
	}
	return make(map[string]any)
}

// toBool converts a CEL value to a Go boolean.
func toBool(val ref.Val) (bool, error) {
	if val == nil {
		return false, fmt.Errorf("CEL evaluation returned nil")
	}

	// Check if it's a boolean type
	if val.Type() == types.BoolType {
		if b, ok := val.Value().(bool); ok {
			return b, nil
		}
	}

	// Try to convert to native Go value
	nativeVal, err := val.ConvertToNative(boolType)
	if err != nil {
		return false, fmt.Errorf("CEL result is not a boolean: %v (type: %s)", val.Value(), val.Type())
	}

	if b, ok := nativeVal.(bool); ok {
		return b, nil
	}

	return false, fmt.Errorf("CEL result cannot be converted to boolean: %v", val.Value())
}

// boolType is the reflect.Type for bool
var boolType = reflect.TypeOf(true)
