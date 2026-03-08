//go:build !cel

package engine

import "log"

// NewCELEvaluator returns nil when built without the cel tag.
// The caller (execute_activity.go) already handles nil gracefully.
func NewCELEvaluator() (ConditionEvaluator, error) {
	log.Println("[INFO] CEL evaluator not available (built without cel tag)")
	return nil, nil
}
