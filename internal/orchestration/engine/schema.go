package engine

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaField defines how a single field is resolved
type SchemaField struct {
	Source   string `json:"source"`
	Type     string `json:"type"`
	Default  any    `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// SchemaProcessor handles resolution of input and output schemas.
// The schemaCache field (sync.Map) caches compiled JSON Schema validators keyed
// by the SHA-256 hash of the raw schema string. The zero value of sync.Map is
// usable, so NewSchemaProcessor() remains a zero-value factory.
type SchemaProcessor struct {
	schemaCache sync.Map // map[string]*jsonschema.Schema — keyed on sha256(schemaJson)
}

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
	case "int", "integer", "number":
		// Basic numeric coercion. "integer" is the JSON Schema keyword; "int" is
		// the legacy espyna simple-format keyword; "number" covers JSON Schema floats.
		// For "integer" and "int" we coerce to int; for "number" we coerce to float64.
		switch v := value.(type) {
		case int:
			if targetType == "number" {
				return float64(v), nil
			}
			return v, nil
		case float64:
			if targetType == "number" {
				return v, nil
			}
			return int(v), nil
		case string:
			if targetType == "number" {
				var f float64
				_, err := fmt.Sscanf(v, "%g", &f)
				return f, err
			}
			var i int
			_, err := fmt.Sscanf(v, "%d", &i)
			return i, err
		}
	case "bool", "boolean":
		// "boolean" is the JSON Schema keyword; "bool" is the legacy keyword.
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

	// Try JSON Schema format first (has "type": "object" and "properties").
	// We parse the struct only to DETECT the JSON Schema shape; the raw schemaJson
	// string is fed to santhosh-tekuri so that keywords the struct does not model
	// (enum, minimum, maximum, minLength, maxLength, pattern, additionalProperties)
	// are NOT dropped. See SEC-1/STRUCT-1.
	var jsonSchemaDetect JSONSchemaObject
	if err := json.Unmarshal([]byte(schemaJson), &jsonSchemaDetect); err == nil && jsonSchemaDetect.Type == "object" && jsonSchemaDetect.Properties != nil {
		return p.validateWithJSONSchema(input, schemaJson, &jsonSchemaDetect)
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

// validateWithJSONSchema validates input against a JSON Schema document using
// santhosh-tekuri/jsonschema v6 (draft 2020-12).
//
// The raw schemaJson string is fed to the library so that every keyword
// (enum, minimum, maximum, minLength, maxLength, pattern, additionalProperties)
// is enforced. The parsed JSONSchemaObject is used ONLY for type-aware coercion
// of the input map (Q-VAL-1: coerce-first — a portal submitting "42" for an
// integer field still works, because coercion normalises the type BEFORE the
// strict validator runs).
//
// Reject-by-default (Q-VAL-2): if the schema does NOT declare
// additionalProperties, an implicit "additionalProperties":false is applied so
// that undeclared fields are rejected at the validation boundary. A schema may
// opt out by setting "additionalProperties":true explicitly.
//
// The compiled schema is cached in a sync.Map keyed on sha256(schemaJson)
// (Q-VAL-4) so that repeated validations of the same schema do not re-parse.
func (p *SchemaProcessor) validateWithJSONSchema(input map[string]any, schemaJson string, schema *JSONSchemaObject) (map[string]any, error) {
	// --- Step 1: Coerce-first (Q-VAL-1) ---
	// Apply the existing coerce() helper to produce a type-correct map BEFORE
	// handing it to the strict validator. This is intentional, not a bug: it
	// preserves the lenient-input contract (portals may submit string-typed
	// values for integer/boolean fields). Coercion normalises the TYPE but
	// never relaxes a constraint (bounds, enum, pattern are checked on the
	// coerced value).
	coerced := make(map[string]any, len(input))
	for k, v := range input {
		if fieldDef, ok := schema.Properties[k]; ok && fieldDef.Type != "" {
			cv, err := p.coerce(v, fieldDef.Type)
			if err != nil {
				return nil, fmt.Errorf("validation failed: field '%s' type coercion failed: %v", k, err)
			}
			coerced[k] = cv
		} else {
			coerced[k] = v
		}
	}

	// Apply defaults for missing fields that have them declared in the schema.
	for fieldName, fieldDef := range schema.Properties {
		if _, exists := coerced[fieldName]; !exists && fieldDef.Default != nil {
			coerced[fieldName] = fieldDef.Default
		}
	}

	// --- Step 2: Compile the schema (cached, Q-VAL-4) ---
	compiled, err := p.compileSchema(schemaJson)
	if err != nil {
		// Malformed / uncompilable schema = deny (SEC-3). Never degrade to
		// pass-through.
		return nil, fmt.Errorf("validation failed: schema compilation error: %w", err)
	}

	// --- Step 3: Validate the coerced input against the compiled schema ---
	if err := compiled.Validate(coerced); err != nil {
		return nil, fmt.Errorf("validation failed: %s", err)
	}

	return coerced, nil
}

// compileSchema compiles a raw JSON Schema string into a *jsonschema.Schema,
// applying the reject-by-default policy (Q-VAL-2: implicit additionalProperties:false)
// and caching the result.
func (p *SchemaProcessor) compileSchema(schemaJson string) (*jsonschema.Schema, error) {
	// Cache key: SHA-256 of the schema string.
	h := sha256.Sum256([]byte(schemaJson))
	key := string(h[:])

	if cached, ok := p.schemaCache.Load(key); ok {
		return cached.(*jsonschema.Schema), nil
	}

	// Apply implicit additionalProperties:false (Q-VAL-2) if the schema does not
	// already declare it. We parse, inject, and re-serialize. This is safe because
	// compilation only happens once per unique schema (cached).
	effectiveSchema := applyAdditionalPropertiesDefault(schemaJson)

	// Parse the schema string into a JSON value using santhosh-tekuri's decoder,
	// which preserves number precision via json.Number. AddResource expects a
	// parsed JSON value (any), NOT a Reader.
	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(effectiveSchema))
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Compile with santhosh-tekuri, draft 2020-12.
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}
	compiled, err := c.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	p.schemaCache.Store(key, compiled)
	return compiled, nil
}

// applyAdditionalPropertiesDefault injects "additionalProperties":false into a
// JSON Schema document that does not already declare the key, enforcing the
// reject-by-default policy (Q-VAL-2). If the schema already sets
// additionalProperties to any value (true, false, or an object), it is left
// unchanged. This ensures that reject-by-default is the IMPLICIT stance and a
// schema must EXPLICITLY opt out.
func applyAdditionalPropertiesDefault(schemaJson string) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(schemaJson), &raw); err != nil {
		// If we cannot parse, return as-is — compilation will fail and deny.
		return schemaJson
	}

	if _, exists := raw["additionalProperties"]; exists {
		// Schema explicitly declares additionalProperties — do not override.
		return schemaJson
	}

	// Inject additionalProperties:false.
	raw["additionalProperties"] = json.RawMessage(`false`)
	out, err := json.Marshal(raw)
	if err != nil {
		return schemaJson
	}
	return string(out)
}
