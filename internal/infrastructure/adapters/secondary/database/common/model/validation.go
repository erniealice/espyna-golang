package model

// ValidationError represents validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return "validation failed"
	}
	return v[0].Message
}

// ValidateRequired checks if required fields are present
func ValidateRequired(data map[string]any, requiredFields ...string) ValidationErrors {
	var errors ValidationErrors

	for _, field := range requiredFields {
		if value, exists := data[field]; !exists || value == nil || value == "" {
			errors = append(errors, ValidationError{
				Field:   field,
				Message: field + " is required",
			})
		}
	}

	return errors
}
