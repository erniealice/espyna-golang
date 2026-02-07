package contracts

// ActivityType defines the type of activity in a workflow
type ActivityType string

const (
	// ActivityTypeUseCase executes a domain use case
	// Format: "{domain}.{resource}.{operation}"
	// Example: "entity.client.create"
	ActivityTypeUseCase ActivityType = "use_case"

	// ActivityTypeIntegration executes an integration provider operation
	// Format: "integration.{provider}.{operation}"
	// Example: "integration.payment.checkout"
	ActivityTypeIntegration ActivityType = "integration"

	// Future types (documented, not yet implemented):
	// ActivityTypeHTTP ActivityType = "http"
	// ActivityTypeCondition ActivityType = "condition"
	// ActivityTypeTimer ActivityType = "timer"
	// ActivityTypeHumanTask ActivityType = "human_task"
)

// IsCurrent returns true if the activity type is currently implemented
func (t ActivityType) IsCurrent() bool {
	switch t {
	case ActivityTypeUseCase, ActivityTypeIntegration:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (t ActivityType) String() string {
	return string(t)
}
