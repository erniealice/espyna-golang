package contracts

// WorkflowEngineMode determines when the workflow engine is initialized
type WorkflowEngineMode string

const (
	// ModeLate initializes engine after all domains are ready (default)
	// Uses SetEngine() pattern for late binding
	ModeLate WorkflowEngineMode = "late"

	// ModeEager initializes engine during container.Initialize()
	// Uses deferred registry that resolves after aggregate is ready
	ModeEager WorkflowEngineMode = "eager"

	// ModeLazy initializes engine on first workflow execution
	// Minimizes startup time when workflows aren't used
	ModeLazy WorkflowEngineMode = "lazy"
)

// IsValid returns true if the mode is a recognized value
func (m WorkflowEngineMode) IsValid() bool {
	switch m {
	case ModeLate, ModeEager, ModeLazy:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (m WorkflowEngineMode) String() string {
	return string(m)
}
