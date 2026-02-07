package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// SchemaField defines how a single field is resolved
type SchemaField struct {
	Source   string `json:"source"`
	Type     string `json:"type"`
	Default  any    `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// SchemaProcessor handles resolution of input and output schemas
type SchemaProcessor struct{}

// NewSchemaProcessor creates a new schema processor
func NewSchemaProcessor() *SchemaProcessor {
	return &SchemaProcessor{}
}

// arrayNotationRegex matches [n] patterns for array/map index access
var arrayNotationRegex = regexp.MustCompile(`\[(\d+)\]`)

// templateExprRegex matches ${...} patterns for template interpolation
var templateExprRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// convertArrayNotation converts JSONPath array notation to dot notation
// e.g., "$.stage[0].activity[1].output" -> "stage.0.activity.1.output"
func convertArrayNotation(path string) string {
	result := path

	// Strip "$." prefix if present
	result = strings.TrimPrefix(result, "$.")

	// Replace [n] with .n
	result = arrayNotationRegex.ReplaceAllString(result, ".$1")

	// Clean up any double dots that might result
	result = strings.ReplaceAll(result, "..", ".")

	// Remove leading dot if present
	result = strings.TrimPrefix(result, ".")

	return result
}

// Resolve resolves the mapping against the provided context
func (p *SchemaProcessor) Resolve(workflowContext map[string]any, mappingJson string) (map[string]any, error) {
	if mappingJson == "" || mappingJson == "{}" {
		return make(map[string]any), nil
	}

	var mapping map[string]SchemaField
	if err := json.Unmarshal([]byte(mappingJson), &mapping); err != nil {
		// Fallback for simple key-value mapping if the structured one fails
		var simpleMapping map[string]string
		if err2 := json.Unmarshal([]byte(mappingJson), &simpleMapping); err2 == nil {
			return p.resolveSimple(workflowContext, simpleMapping), nil
		}
		return nil, fmt.Errorf("failed to unmarshal schema mapping: %w", err)
	}

	result := make(map[string]any)
	for targetField, fieldDef := range mapping {
		// Support nested source paths with array notation (e.g., "$.stage[0].activity[1].output.field")
		cleanSource := convertArrayNotation(fieldDef.Source)
		value := p.getNestedValue(workflowContext, cleanSource)
		if value == nil {
			if fieldDef.Default != nil {
				value = fieldDef.Default
			} else {
				// Field not found and no default
				continue
			}
		}

		// Type coercion (simplified for now)
		coercedValue, err := p.coerce(value, fieldDef.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to coerce field %s: %w", targetField, err)
		}

		// Handle dot notation for nested structures (e.g., "user.first_name")
		p.setNestedValue(result, targetField, coercedValue)
	}

	return result, nil
}

// arrayKeyRegex matches array notation at the end of a key segment
// e.g., "to[0]" matches with base="to", index=0
var arrayKeyRegex = regexp.MustCompile(`^([^\[]+)\[(\d+)\]$`)

// setNestedValue sets a value in a nested map structure using dot notation
// Supports both dot notation and array notation:
// - "user.first_name" → result["user"]["first_name"] = value
// - "to[0].address" → result["to"][0]["address"] = value
func (p *SchemaProcessor) setNestedValue(result map[string]any, key string, value any) {
	// Check if the key contains a dot (nested structure)
	dotIndex := -1
	for i, r := range key {
		if r == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex == -1 {
		// No dot found - check for array notation in the final key
		if matches := arrayKeyRegex.FindStringSubmatch(key); matches != nil {
			// Key is like "to[0]" - create array and set value at index
			baseName := matches[1]
			var index int
			fmt.Sscanf(matches[2], "%d", &index)

			// Ensure the array exists and has enough capacity
			arr, ok := result[baseName].([]any)
			if !ok {
				arr = make([]any, index+1)
			}
			for len(arr) <= index {
				arr = append(arr, nil)
			}
			arr[index] = value
			result[baseName] = arr
		} else {
			// Simple key
			result[key] = value
		}
		return
	}

	// Split on the first dot
	prefix := key[:dotIndex]
	rest := key[dotIndex+1:]

	// Check if prefix has array notation (e.g., "to[0]")
	if matches := arrayKeyRegex.FindStringSubmatch(prefix); matches != nil {
		baseName := matches[1]
		var index int
		fmt.Sscanf(matches[2], "%d", &index)

		// Ensure the array exists and has enough capacity
		arr, ok := result[baseName].([]any)
		if !ok {
			arr = make([]any, index+1)
		}
		for len(arr) <= index {
			arr = append(arr, nil)
		}

		// Ensure the element at index is a map
		elemMap, ok := arr[index].(map[string]any)
		if !ok || elemMap == nil {
			elemMap = make(map[string]any)
		}
		arr[index] = elemMap
		result[baseName] = arr

		// Recursively set the nested value in the element map
		p.setNestedValue(elemMap, rest, value)
		return
	}

	// Regular dot notation (no array)
	// Ensure the prefix exists as a map
	if result[prefix] == nil {
		result[prefix] = make(map[string]any)
	}

	// Recursively set the nested value
	nestedMap, ok := result[prefix].(map[string]any)
	if !ok {
		// Prefix exists but is not a map, replace it
		nestedMap = make(map[string]any)
		result[prefix] = nestedMap
	}

	p.setNestedValue(nestedMap, rest, value)
}

func (p *SchemaProcessor) resolveSimple(workflowContext map[string]any, mapping map[string]string) map[string]any {
	result := make(map[string]any)
	for target, source := range mapping {
		// Check if source is a template expression (contains ${...})
		if templateExprRegex.MatchString(source) {
			val := p.resolveTemplate(workflowContext, source)
			if val != "" {
				p.setNestedValue(result, target, val)
			}
			continue
		}

		// Convert array notation and strip "$." prefix
		// e.g., "$.stage[0].activity[1].output.client_id" -> "stage.0.activity.1.output.client_id"
		cleanSource := convertArrayNotation(source)

		// Try nested path lookup (e.g., "stage.0.activity.1.output.client_id")
		val := p.getNestedValue(workflowContext, cleanSource)
		if val != nil {
			p.setNestedValue(result, target, val)
		}
	}
	return result
}

// resolveTemplate resolves a template string containing ${...} expressions
// e.g., "${$.input.user.first_name} ${$.input.user.last_name} [${$.stage[0].activity[0].output.plan_name}]"
// Each ${path} is replaced with the value at that path in the workflow context.
func (p *SchemaProcessor) resolveTemplate(workflowContext map[string]any, template string) string {
	result := templateExprRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Extract the path from ${path}
		path := match[2 : len(match)-1] // Remove ${ and }

		// Resolve the path
		cleanPath := convertArrayNotation(path)
		val := p.getNestedValue(workflowContext, cleanPath)
		if val == nil {
			return "" // Return empty string for missing values
		}
		return fmt.Sprintf("%v", val)
	})
	return result
}

// deepConvertMap recursively converts map[string]interface{} to map[string]any
// This handles nested maps that result from JSON unmarshaling
func deepConvertMap(m map[string]interface{}) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			// Recursively convert nested maps
			result[k] = deepConvertMap(val)
		case []interface{}:
			// Also handle slices that may contain maps
			result[k] = deepConvertSlice(val)
		default:
			result[k] = v
		}
	}
	return result
}

// deepConvertSlice recursively converts []interface{} elements that may contain maps
func deepConvertSlice(s []interface{}) []any {
	result := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = deepConvertMap(val)
		case []interface{}:
			result[i] = deepConvertSlice(val)
		default:
			result[i] = v
		}
	}
	return result
}

// getNestedValue retrieves a value from a nested map structure using dot notation
// For example: getNestedValue(ctx, "user.email_address") returns ctx["user"]["email_address"]
func (p *SchemaProcessor) getNestedValue(m map[string]any, key string) any {
	// Check if the key contains a dot (nested structure)
	dotIndex := -1
	for i, r := range key {
		if r == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex == -1 {
		// No dot found, simple key lookup
		return m[key]
	}

	// Split on the first dot
	prefix := key[:dotIndex]
	rest := key[dotIndex+1:]

	// Get the nested map
	if nestedMap, ok := m[prefix].(map[string]any); ok {
		return p.getNestedValue(nestedMap, rest)
	}
	// Also try map[string]interface{} (common from JSON unmarshaling)
	if nestedMap, ok := m[prefix].(map[string]interface{}); ok {
		// Deep convert to map[string]any and recurse
		converted := deepConvertMap(nestedMap)
		return p.getNestedValue(converted, rest)
	}

	return nil
}

func (p *SchemaProcessor) coerce(value any, targetType string) (any, error) {
	if targetType == "" {
		return value, nil
	}

	switch targetType {
	case "string":
		return fmt.Sprintf("%v", value), nil
	case "int":
		// Basic int coercion
		switch v := value.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case string:
			var i int
			_, err := fmt.Sscanf(v, "%d", &i)
			return i, err
		}
	case "bool":
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			return v == "true", nil
		case int:
			return v != 0, nil
		}
	}

	return value, nil
}

// JSONSchemaObject represents a JSON Schema object with type, properties, and required fields
type JSONSchemaObject struct {
	Type       string                     `json:"type"`
	Properties map[string]JSONSchemaField `json:"properties"`
	Required   []string                   `json:"required"`
}

// JSONSchemaField represents a field definition in JSON Schema format
type JSONSchemaField struct {
	Type                 string                     `json:"type"`
	Description          string                     `json:"description"`
	Default              any                        `json:"default,omitempty"`
	Properties           map[string]JSONSchemaField `json:"properties,omitempty"`
	AdditionalProperties *JSONSchemaField           `json:"additionalProperties,omitempty"`
}

// ValidateInput validates input JSON against a schema definition and applies defaults.
// Supports two schema formats:
// 1. JSON Schema: { "type": "object", "properties": {...}, "required": [...] }
// 2. Simple format: { "field_name": { "type": "string", "required": true, "default": "value" } }
// Returns the validated and enriched input map with defaults applied.
func (p *SchemaProcessor) ValidateInput(inputJson string, schemaJson string) (map[string]any, error) {
	// If no schema, just parse and return the input
	if schemaJson == "" || schemaJson == "{}" {
		var input map[string]any
		if inputJson == "" {
			return make(map[string]any), nil
		}
		if err := json.Unmarshal([]byte(inputJson), &input); err != nil {
			return nil, fmt.Errorf("failed to parse input JSON: %w", err)
		}
		return input, nil
	}

	// Parse input
	var input map[string]any
	if inputJson == "" {
		input = make(map[string]any)
	} else {
		if err := json.Unmarshal([]byte(inputJson), &input); err != nil {
			return nil, fmt.Errorf("failed to parse input JSON: %w", err)
		}
	}

	// Try JSON Schema format first (has "type": "object" and "properties")
	var jsonSchema JSONSchemaObject
	if err := json.Unmarshal([]byte(schemaJson), &jsonSchema); err == nil && jsonSchema.Type == "object" && jsonSchema.Properties != nil {
		return p.validateWithJSONSchema(input, &jsonSchema)
	}

	// Fall back to simple format
	var schema map[string]SchemaField
	if err := json.Unmarshal([]byte(schemaJson), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Validate and enrich
	result := make(map[string]any)
	var validationErrors []string

	for fieldName, fieldDef := range schema {
		value, exists := input[fieldName]

		if !exists || value == nil {
			// Field not provided
			if fieldDef.Required && fieldDef.Default == nil {
				validationErrors = append(validationErrors, fmt.Sprintf("required field '%s' is missing", fieldName))
				continue
			}
			if fieldDef.Default != nil {
				result[fieldName] = fieldDef.Default
			}
			continue
		}

		// Type coercion
		coercedValue, err := p.coerce(value, fieldDef.Type)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("field '%s' type coercion failed: %v", fieldName, err))
			continue
		}

		result[fieldName] = coercedValue
	}

	// Include any extra fields from input that weren't in schema (pass-through)
	for fieldName, value := range input {
		if _, exists := result[fieldName]; !exists {
			if _, inSchema := schema[fieldName]; !inSchema {
				result[fieldName] = value
			}
		}
	}

	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	return result, nil
}

// ValidateInputToJson validates input and returns it as a JSON string for storage
func (p *SchemaProcessor) ValidateInputToJson(inputJson string, schemaJson string) (string, error) {
	validated, err := p.ValidateInput(inputJson, schemaJson)
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(validated)
	if err != nil {
		return "", fmt.Errorf("failed to marshal validated input: %w", err)
	}

	return string(result), nil
}

// validateWithJSONSchema validates input against a JSON Schema object
func (p *SchemaProcessor) validateWithJSONSchema(input map[string]any, schema *JSONSchemaObject) (map[string]any, error) {
	result := make(map[string]any)
	var validationErrors []string

	// Build required field set for quick lookup
	requiredFields := make(map[string]bool)
	for _, field := range schema.Required {
		requiredFields[field] = true
	}

	// Validate each property defined in schema
	for fieldName, fieldDef := range schema.Properties {
		value, exists := input[fieldName]

		if !exists || value == nil {
			// Field not provided
			if requiredFields[fieldName] && fieldDef.Default == nil {
				validationErrors = append(validationErrors, fmt.Sprintf("required field '%s' is missing", fieldName))
				continue
			}
			if fieldDef.Default != nil {
				result[fieldName] = fieldDef.Default
			}
			continue
		}

		// Type coercion
		coercedValue, err := p.coerce(value, fieldDef.Type)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("field '%s' type coercion failed: %v", fieldName, err))
			continue
		}

		result[fieldName] = coercedValue
	}

	// Include any extra fields from input that weren't in schema (pass-through)
	for fieldName, value := range input {
		if _, exists := result[fieldName]; !exists {
			if _, inSchema := schema.Properties[fieldName]; !inSchema {
				result[fieldName] = value
			}
		}
	}

	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	return result, nil
}
