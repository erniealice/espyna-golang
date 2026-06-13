package engine

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestSetNestedValue_SimpleKey(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	p.setNestedValue(result, "name", "John")

	if result["name"] != "John" {
		t.Errorf("Expected result[name] = 'John', got %v", result["name"])
	}
}

func TestSetNestedValue_DotNotation(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	p.setNestedValue(result, "user.first_name", "John")
	p.setNestedValue(result, "user.last_name", "Doe")

	user, ok := result["user"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result[user] to be a map, got %T", result["user"])
	}

	if user["first_name"] != "John" {
		t.Errorf("Expected user[first_name] = 'John', got %v", user["first_name"])
	}
	if user["last_name"] != "Doe" {
		t.Errorf("Expected user[last_name] = 'Doe', got %v", user["last_name"])
	}
}

func TestSetNestedValue_ArrayNotation_SingleElement(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	// Test: to[0].address should create an array with a map at index 0
	p.setNestedValue(result, "to[0].address", "john@example.com")
	p.setNestedValue(result, "to[0].name", "John Doe")

	// Verify structure
	toArray, ok := result["to"].([]any)
	if !ok {
		t.Fatalf("Expected result[to] to be an array, got %T", result["to"])
	}

	if len(toArray) != 1 {
		t.Errorf("Expected array length 1, got %d", len(toArray))
	}

	firstElem, ok := toArray[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected toArray[0] to be a map, got %T", toArray[0])
	}

	if firstElem["address"] != "john@example.com" {
		t.Errorf("Expected address = 'john@example.com', got %v", firstElem["address"])
	}
	if firstElem["name"] != "John Doe" {
		t.Errorf("Expected name = 'John Doe', got %v", firstElem["name"])
	}
}

func TestSetNestedValue_ArrayNotation_MultipleElements(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	// Build array with multiple elements
	p.setNestedValue(result, "to[0].address", "john@example.com")
	p.setNestedValue(result, "to[0].name", "John")
	p.setNestedValue(result, "to[1].address", "jane@example.com")
	p.setNestedValue(result, "to[1].name", "Jane")
	p.setNestedValue(result, "to[2].address", "admin@example.com")

	// Verify structure
	toArray, ok := result["to"].([]any)
	if !ok {
		t.Fatalf("Expected result[to] to be an array, got %T", result["to"])
	}

	if len(toArray) != 3 {
		t.Errorf("Expected array length 3, got %d", len(toArray))
	}

	// Verify first element
	elem0, ok := toArray[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected toArray[0] to be a map, got %T", toArray[0])
	}
	if elem0["address"] != "john@example.com" || elem0["name"] != "John" {
		t.Errorf("Element 0 mismatch: %v", elem0)
	}

	// Verify second element
	elem1, ok := toArray[1].(map[string]any)
	if !ok {
		t.Fatalf("Expected toArray[1] to be a map, got %T", toArray[1])
	}
	if elem1["address"] != "jane@example.com" || elem1["name"] != "Jane" {
		t.Errorf("Element 1 mismatch: %v", elem1)
	}

	// Verify third element (only address, no name)
	elem2, ok := toArray[2].(map[string]any)
	if !ok {
		t.Fatalf("Expected toArray[2] to be a map, got %T", toArray[2])
	}
	if elem2["address"] != "admin@example.com" {
		t.Errorf("Element 2 address mismatch: %v", elem2["address"])
	}
}

func TestSetNestedValue_ArrayNotation_OutOfOrder(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	// Set elements out of order - should still work
	p.setNestedValue(result, "items[2].value", "third")
	p.setNestedValue(result, "items[0].value", "first")
	p.setNestedValue(result, "items[1].value", "second")

	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("Expected result[items] to be an array, got %T", result["items"])
	}

	if len(items) != 3 {
		t.Errorf("Expected array length 3, got %d", len(items))
	}

	// Verify all elements
	for i, expected := range []string{"first", "second", "third"} {
		elem, ok := items[i].(map[string]any)
		if !ok {
			t.Fatalf("Expected items[%d] to be a map, got %T", i, items[i])
		}
		if elem["value"] != expected {
			t.Errorf("Expected items[%d].value = '%s', got %v", i, expected, elem["value"])
		}
	}
}

func TestSetNestedValue_ArrayNotation_FinalKey(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	// Test array notation without nested property (direct value assignment)
	p.setNestedValue(result, "tags[0]", "important")
	p.setNestedValue(result, "tags[1]", "urgent")
	p.setNestedValue(result, "tags[2]", "review")

	tags, ok := result["tags"].([]any)
	if !ok {
		t.Fatalf("Expected result[tags] to be an array, got %T", result["tags"])
	}

	if len(tags) != 3 {
		t.Errorf("Expected array length 3, got %d", len(tags))
	}

	expected := []string{"important", "urgent", "review"}
	for i, exp := range expected {
		if tags[i] != exp {
			t.Errorf("Expected tags[%d] = '%s', got %v", i, exp, tags[i])
		}
	}
}

func TestSetNestedValue_DeeplyNested(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	// Test deeply nested structure: data[0].items[1].options[0].value
	p.setNestedValue(result, "data[0].items[0].options[0].key", "color")
	p.setNestedValue(result, "data[0].items[0].options[0].value", "blue")
	p.setNestedValue(result, "data[0].items[0].options[1].key", "size")
	p.setNestedValue(result, "data[0].items[0].options[1].value", "large")
	p.setNestedValue(result, "data[0].items[0].name", "T-Shirt")

	// Navigate and verify
	dataArr, ok := result["data"].([]any)
	if !ok {
		t.Fatalf("Expected data to be array, got %T", result["data"])
	}

	data0, ok := dataArr[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected data[0] to be map, got %T", dataArr[0])
	}

	itemsArr, ok := data0["items"].([]any)
	if !ok {
		t.Fatalf("Expected items to be array, got %T", data0["items"])
	}

	item0, ok := itemsArr[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected items[0] to be map, got %T", itemsArr[0])
	}

	if item0["name"] != "T-Shirt" {
		t.Errorf("Expected name = 'T-Shirt', got %v", item0["name"])
	}

	optionsArr, ok := item0["options"].([]any)
	if !ok {
		t.Fatalf("Expected options to be array, got %T", item0["options"])
	}

	if len(optionsArr) != 2 {
		t.Errorf("Expected 2 options, got %d", len(optionsArr))
	}

	opt0, ok := optionsArr[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected options[0] to be map, got %T", optionsArr[0])
	}
	if opt0["key"] != "color" || opt0["value"] != "blue" {
		t.Errorf("Option 0 mismatch: %v", opt0)
	}

	opt1, ok := optionsArr[1].(map[string]any)
	if !ok {
		t.Fatalf("Expected options[1] to be map, got %T", optionsArr[1])
	}
	if opt1["key"] != "size" || opt1["value"] != "large" {
		t.Errorf("Option 1 mismatch: %v", opt1)
	}
}

func TestSetNestedValue_MixedNotation(t *testing.T) {
	p := NewSchemaProcessor()
	result := make(map[string]any)

	// Mix of array notation and dot notation
	p.setNestedValue(result, "order.items[0].product.name", "Widget")
	p.setNestedValue(result, "order.items[0].product.price", 19.99)
	p.setNestedValue(result, "order.items[0].quantity", 2)
	p.setNestedValue(result, "order.customer.name", "John")

	// Verify structure
	order, ok := result["order"].(map[string]any)
	if !ok {
		t.Fatalf("Expected order to be map, got %T", result["order"])
	}

	items, ok := order["items"].([]any)
	if !ok {
		t.Fatalf("Expected items to be array, got %T", order["items"])
	}

	item0, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected items[0] to be map, got %T", items[0])
	}

	product, ok := item0["product"].(map[string]any)
	if !ok {
		t.Fatalf("Expected product to be map, got %T", item0["product"])
	}

	if product["name"] != "Widget" {
		t.Errorf("Expected product.name = 'Widget', got %v", product["name"])
	}
	if product["price"] != 19.99 {
		t.Errorf("Expected product.price = 19.99, got %v", product["price"])
	}

	if item0["quantity"] != 2 {
		t.Errorf("Expected quantity = 2, got %v", item0["quantity"])
	}

	customer, ok := order["customer"].(map[string]any)
	if !ok {
		t.Fatalf("Expected customer to be map, got %T", order["customer"])
	}
	if customer["name"] != "John" {
		t.Errorf("Expected customer.name = 'John', got %v", customer["name"])
	}
}

func TestResolveSimple_WithArrayNotation(t *testing.T) {
	p := NewSchemaProcessor()

	// Simulate workflow context
	workflowContext := map[string]any{
		"input": map[string]any{
			"recipient_email": "john@example.com",
			"recipient_name":  "John Doe",
			"email_subject":   "Payment Confirmation",
		},
	}

	// Simulate input_mapping JSON (from YAML workflow)
	mappingJson := `{
		"to[0].address": "$.input.recipient_email",
		"to[0].name": "$.input.recipient_name",
		"subject": "$.input.email_subject"
	}`

	result, err := p.Resolve(workflowContext, mappingJson)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify 'to' is an array
	toArray, ok := result["to"].([]any)
	if !ok {
		t.Fatalf("Expected to be array, got %T", result["to"])
	}

	if len(toArray) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(toArray))
	}

	recipient, ok := toArray[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected to[0] to be map, got %T", toArray[0])
	}

	if recipient["address"] != "john@example.com" {
		t.Errorf("Expected address = 'john@example.com', got %v", recipient["address"])
	}
	if recipient["name"] != "John Doe" {
		t.Errorf("Expected name = 'John Doe', got %v", recipient["name"])
	}

	// Verify subject
	if result["subject"] != "Payment Confirmation" {
		t.Errorf("Expected subject = 'Payment Confirmation', got %v", result["subject"])
	}
}

func TestResolveSimple_WithMultipleRecipients(t *testing.T) {
	p := NewSchemaProcessor()

	workflowContext := map[string]any{
		"input": map[string]any{
			"primary_email": "john@example.com",
			"primary_name":  "John",
			"cc_email":      "manager@example.com",
			"cc_name":       "Manager",
		},
	}

	mappingJson := `{
		"to[0].address": "$.input.primary_email",
		"to[0].name": "$.input.primary_name",
		"to[1].address": "$.input.cc_email",
		"to[1].name": "$.input.cc_name"
	}`

	result, err := p.Resolve(workflowContext, mappingJson)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	toArray, ok := result["to"].([]any)
	if !ok {
		t.Fatalf("Expected to be array, got %T", result["to"])
	}

	if len(toArray) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(toArray))
	}

	// Verify first recipient
	r0, ok := toArray[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected to[0] to be map, got %T", toArray[0])
	}
	if r0["address"] != "john@example.com" || r0["name"] != "John" {
		t.Errorf("Recipient 0 mismatch: %v", r0)
	}

	// Verify second recipient
	r1, ok := toArray[1].(map[string]any)
	if !ok {
		t.Fatalf("Expected to[1] to be map, got %T", toArray[1])
	}
	if r1["address"] != "manager@example.com" || r1["name"] != "Manager" {
		t.Errorf("Recipient 1 mismatch: %v", r1)
	}
}

func TestResolveSimple_WithTemplateValues(t *testing.T) {
	p := NewSchemaProcessor()

	workflowContext := map[string]any{
		"input": map[string]any{
			"client_first_name": "John",
			"client_last_name":  "Doe",
			"plan_name":         "Premium Plan",
		},
	}

	// This matches the actual YAML workflow structure
	mappingJson := `{
		"template_values.client_first_name": "$.input.client_first_name",
		"template_values.client_last_name": "$.input.client_last_name",
		"template_values.plan_name": "$.input.plan_name"
	}`

	result, err := p.Resolve(workflowContext, mappingJson)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	templateValues, ok := result["template_values"].(map[string]any)
	if !ok {
		t.Fatalf("Expected template_values to be map, got %T", result["template_values"])
	}

	if templateValues["client_first_name"] != "John" {
		t.Errorf("Expected client_first_name = 'John', got %v", templateValues["client_first_name"])
	}
	if templateValues["client_last_name"] != "Doe" {
		t.Errorf("Expected client_last_name = 'Doe', got %v", templateValues["client_last_name"])
	}
	if templateValues["plan_name"] != "Premium Plan" {
		t.Errorf("Expected plan_name = 'Premium Plan', got %v", templateValues["plan_name"])
	}
}

func TestResolveSimple_EmailWorkflowScenario(t *testing.T) {
	p := NewSchemaProcessor()

	// Simulate the actual workflow context structure from subscription_payment_webhook.yaml
	workflowContext := map[string]any{
		"input": map[string]any{
			"email_subject":       "Payment Confirmation",
			"email_template_html": "<h1>Hello {{client_first_name}}</h1>",
		},
		"stage": map[string]any{
			"1": map[string]any{
				"activity": map[string]any{
					"3": map[string]any{
						"output": map[string]any{
							"client_email":      "john@example.com",
							"client_first_name": "John",
							"client_last_name":  "Doe",
						},
					},
					"2": map[string]any{
						"output": map[string]any{
							"price_plan_name": "Premium Plan",
						},
					},
				},
			},
			"0": map[string]any{
				"activity": map[string]any{
					"0": map[string]any{
						"output": map[string]any{
							"payment_id": "PAY-12345",
							"transaction": map[string]any{
								"amount":   "100.00",
								"currency": "USD",
							},
						},
					},
				},
			},
		},
	}

	// This matches the actual input_mapping from subscription_payment_webhook.yaml
	mappingJson := `{
		"to[0].address": "$.stage.1.activity.3.output.client_email",
		"to[0].name": "$.stage.1.activity.3.output.client_first_name",
		"subject": "$.input.email_subject",
		"template_html": "$.input.email_template_html",
		"template_values.client_first_name": "$.stage.1.activity.3.output.client_first_name",
		"template_values.client_last_name": "$.stage.1.activity.3.output.client_last_name",
		"template_values.price_plan_name": "$.stage.1.activity.2.output.price_plan_name"
	}`

	result, err := p.Resolve(workflowContext, mappingJson)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify 'to' array structure (this was the bug!)
	toArray, ok := result["to"].([]any)
	if !ok {
		t.Fatalf("CRITICAL: Expected 'to' to be an array for protobuf mapping, got %T", result["to"])
	}

	if len(toArray) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(toArray))
	}

	recipient, ok := toArray[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected to[0] to be a map, got %T", toArray[0])
	}

	if recipient["address"] != "john@example.com" {
		t.Errorf("Expected to[0].address = 'john@example.com', got %v", recipient["address"])
	}
	if recipient["name"] != "John" {
		t.Errorf("Expected to[0].name = 'John', got %v", recipient["name"])
	}

	// Verify other fields
	if result["subject"] != "Payment Confirmation" {
		t.Errorf("Expected subject = 'Payment Confirmation', got %v", result["subject"])
	}

	templateValues, ok := result["template_values"].(map[string]any)
	if !ok {
		t.Fatalf("Expected template_values to be map, got %T", result["template_values"])
	}

	if templateValues["client_first_name"] != "John" {
		t.Errorf("Expected template_values.client_first_name = 'John', got %v", templateValues["client_first_name"])
	}
	if templateValues["price_plan_name"] != "Premium Plan" {
		t.Errorf("Expected template_values.price_plan_name = 'Premium Plan', got %v", templateValues["price_plan_name"])
	}

	// Log the final structure for debugging
	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Final resolved structure:\n%s", string(jsonBytes))
}

func TestConvertArrayNotation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"$.stage[0].activity[1].output", "stage.0.activity.1.output"},
		{"$.input.email", "input.email"},
		{"stage[0].output", "stage.0.output"},
		{"simple", "simple"},
		{"$.nested[0][1].value", "nested.0.1.value"}, // Double array notation
	}

	for _, tt := range tests {
		result := convertArrayNotation(tt.input)
		if result != tt.expected {
			t.Errorf("convertArrayNotation(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetNestedValue(t *testing.T) {
	p := NewSchemaProcessor()

	context := map[string]any{
		"input": map[string]any{
			"email": "test@example.com",
		},
		"stage": map[string]any{
			"0": map[string]any{
				"activity": map[string]any{
					"1": map[string]any{
						"output": map[string]any{
							"result": "success",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		key      string
		expected any
	}{
		{"input.email", "test@example.com"},
		{"stage.0.activity.1.output.result", "success"},
		{"nonexistent", nil},
		{"input.nonexistent", nil},
	}

	for _, tt := range tests {
		result := p.getNestedValue(context, tt.key)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("getNestedValue(%q) = %v, expected %v", tt.key, result, tt.expected)
		}
	}
}

// Benchmark for performance testing
func BenchmarkSetNestedValue_DeeplyNested(b *testing.B) {
	p := NewSchemaProcessor()

	for i := 0; i < b.N; i++ {
		result := make(map[string]any)
		p.setNestedValue(result, "data[0].items[0].options[0].attributes[0].value", "test")
	}
}

func BenchmarkResolveSimple_EmailScenario(b *testing.B) {
	p := NewSchemaProcessor()

	workflowContext := map[string]any{
		"input": map[string]any{
			"email":   "test@example.com",
			"name":    "Test User",
			"subject": "Test Subject",
		},
	}

	mappingJson := `{
		"to[0].address": "$.input.email",
		"to[0].name": "$.input.name",
		"subject": "$.input.subject"
	}`

	for i := 0; i < b.N; i++ {
		_, _ = p.Resolve(workflowContext, mappingJson)
	}
}

// =============================================================================
// Phase 4: Fail-closed test wave — JSON Schema validation via santhosh-tekuri
// =============================================================================
//
// These tests exercise the swapped validateWithJSONSchema (now backed by
// santhosh-tekuri/jsonschema/v6, draft 2020-12). Each FAIL-CLOSED case asserts
// a non-nil error; each regression case asserts success and the expected map.
// The tests use ValidateInput (the public seam) to confirm end-to-end behavior.

func TestValidateInput_JSONSchema_BadEnum_FailsClosed(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["active", "inactive"]
			}
		}
	}`

	input := `{"status": "unknown_value"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for bad enum value, got nil")
	}
	t.Logf("bad-enum error: %v", err)
}

func TestValidateInput_JSONSchema_OutOfBoundsMinimum_FailsClosed(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"new_monthly_rate": {
				"type": "integer",
				"minimum": 1
			}
		}
	}`

	input := `{"new_monthly_rate": 0}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for value below minimum (centavos floor), got nil")
	}
	t.Logf("minimum-violation error: %v", err)
}

func TestValidateInput_JSONSchema_OutOfBoundsMaximum_FailsClosed(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"hours_per_week": {
				"type": "integer",
				"maximum": 168
			}
		}
	}`

	input := `{"hours_per_week": 200}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for value above maximum, got nil")
	}
	t.Logf("maximum-violation error: %v", err)
}

func TestValidateInput_JSONSchema_ShortMinLength_FailsClosed(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"justification": {
				"type": "string",
				"minLength": 10
			}
		}
	}`

	input := `{"justification": "short"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for string shorter than minLength, got nil")
	}
	t.Logf("minLength-violation error: %v", err)
}

func TestValidateInput_JSONSchema_PatternMismatch_FailsClosed(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"code": {
				"type": "string",
				"pattern": "^[a-z_]+$"
			}
		}
	}`

	input := `{"code": "bad value!"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for pattern mismatch, got nil")
	}
	t.Logf("pattern-mismatch error: %v", err)
}

func TestValidateInput_JSONSchema_UnknownField_FailsClosed_ByDefault(t *testing.T) {
	p := NewSchemaProcessor()

	// Schema does NOT declare additionalProperties — reject-by-default applies
	// (implicit additionalProperties:false per Q-VAL-2).
	schema := `{
		"type": "object",
		"properties": {
			"known_field": {
				"type": "string"
			}
		}
	}`

	input := `{"known_field": "x", "rogue_field": "y"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for unknown field (reject-by-default), got nil")
	}
	t.Logf("unknown-field error: %v", err)
}

func TestValidateInput_JSONSchema_MissingRequired_FailsClosed(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"new_monthly_rate": {
				"type": "integer",
				"minimum": 1
			}
		},
		"required": ["new_monthly_rate"]
	}`

	input := `{}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
	t.Logf("missing-required error: %v", err)
}

func TestValidateInput_JSONSchema_ValidFullPayload_Succeeds(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"new_monthly_rate": {
				"type": "integer",
				"minimum": 1,
				"maximum": 10000000
			},
			"justification": {
				"type": "string",
				"minLength": 10,
				"maxLength": 5000
			},
			"status": {
				"type": "string",
				"enum": ["active", "inactive"]
			},
			"code": {
				"type": "string",
				"pattern": "^[a-z_]+$"
			}
		},
		"required": ["new_monthly_rate", "justification"]
	}`

	input := `{
		"new_monthly_rate": 500000,
		"justification": "Annual performance-based salary adjustment for senior developer role",
		"status": "active",
		"code": "salary_increase"
	}`

	result, err := p.ValidateInput(input, schema)
	if err != nil {
		t.Fatalf("expected success for valid full payload, got error: %v", err)
	}

	// Verify the enriched map contains the expected values
	if result["new_monthly_rate"] != 500000 {
		t.Errorf("expected new_monthly_rate=500000, got %v (type %T)", result["new_monthly_rate"], result["new_monthly_rate"])
	}
	if result["status"] != "active" {
		t.Errorf("expected status='active', got %v", result["status"])
	}
	if result["code"] != "salary_increase" {
		t.Errorf("expected code='salary_increase', got %v", result["code"])
	}
	t.Logf("valid payload result: %v", result)
}

func TestValidateInput_JSONSchema_EmptySchema_NoOp(t *testing.T) {
	p := NewSchemaProcessor()

	// Empty schema string
	input := `{"any_field": "any_value"}`

	result, err := p.ValidateInput(input, "")
	if err != nil {
		t.Fatalf("expected no error for empty schema string, got: %v", err)
	}
	if result["any_field"] != "any_value" {
		t.Errorf("expected pass-through of input with empty schema, got %v", result)
	}

	// Empty JSON object schema
	result2, err := p.ValidateInput(input, "{}")
	if err != nil {
		t.Fatalf("expected no error for empty JSON object schema, got: %v", err)
	}
	if result2["any_field"] != "any_value" {
		t.Errorf("expected pass-through of input with empty JSON object schema, got %v", result2)
	}
}

func TestValidateInput_JSONSchema_PerSchemaOptOut_PassesThrough(t *testing.T) {
	p := NewSchemaProcessor()

	// Schema explicitly declares additionalProperties:true — opt-out from
	// the reject-by-default policy (Q-VAL-2).
	schema := `{
		"type": "object",
		"properties": {
			"known_field": {
				"type": "string"
			}
		},
		"additionalProperties": true
	}`

	input := `{"known_field": "x", "extra_field": "y", "another_extra": 42}`

	result, err := p.ValidateInput(input, schema)
	if err != nil {
		t.Fatalf("expected success when additionalProperties:true, got error: %v", err)
	}

	if result["known_field"] != "x" {
		t.Errorf("expected known_field='x', got %v", result["known_field"])
	}
	if result["extra_field"] != "y" {
		t.Errorf("expected extra_field='y' (pass-through), got %v", result["extra_field"])
	}
	t.Logf("opt-out result: %v", result)
}

func TestValidateInput_JSONSchema_TypeCoercionPreserved(t *testing.T) {
	p := NewSchemaProcessor()

	// Q-VAL-1: coerce-first. A portal submitting "42" for an integer field
	// should still work because coercion normalises the type before validation.
	schema := `{
		"type": "object",
		"properties": {
			"count": {
				"type": "integer",
				"minimum": 1,
				"maximum": 100
			}
		},
		"required": ["count"]
	}`

	input := `{"count": "42"}`

	result, err := p.ValidateInput(input, schema)
	if err != nil {
		t.Fatalf("expected success with string-to-integer coercion, got error: %v", err)
	}

	// After coercion, count should be an int
	count, ok := result["count"].(int)
	if !ok {
		t.Fatalf("expected count to be int after coercion, got %T", result["count"])
	}
	if count != 42 {
		t.Errorf("expected count=42, got %d", count)
	}
}

func TestValidateInput_JSONSchema_TypeCoercion_BooleanPreserved(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"active": {
				"type": "boolean"
			}
		}
	}`

	input := `{"active": "true"}`

	result, err := p.ValidateInput(input, schema)
	if err != nil {
		t.Fatalf("expected success with string-to-boolean coercion, got error: %v", err)
	}

	active, ok := result["active"].(bool)
	if !ok {
		t.Fatalf("expected active to be bool after coercion, got %T", result["active"])
	}
	if !active {
		t.Error("expected active=true, got false")
	}
}

func TestValidateInput_JSONSchema_CoercionDoesNotRelaxMinimum(t *testing.T) {
	p := NewSchemaProcessor()

	// SEC-4: Coerce-first must NOT relax a constraint. A string "0" for a
	// centavos field (minimum:1) must still fail after coercion to int(0).
	schema := `{
		"type": "object",
		"properties": {
			"new_monthly_rate": {
				"type": "integer",
				"minimum": 1
			}
		},
		"required": ["new_monthly_rate"]
	}`

	input := `{"new_monthly_rate": "0"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error: coercion of '0' to int(0) should still fail minimum:1")
	}
	t.Logf("coerce-does-not-relax error: %v", err)
}

func TestValidateInput_JSONSchema_NegativeIntegerFails(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"new_monthly_rate": {
				"type": "integer",
				"minimum": 1
			}
		}
	}`

	input := `{"new_monthly_rate": -100}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for negative integer with minimum:1, got nil")
	}
	t.Logf("negative-integer error: %v", err)
}

func TestValidateInput_JSONSchema_MaxLength_FailsClosed(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"code": {
				"type": "string",
				"maxLength": 5
			}
		}
	}`

	input := `{"code": "too_long_value"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for string exceeding maxLength, got nil")
	}
	t.Logf("maxLength-violation error: %v", err)
}

func TestValidateInput_JSONSchema_MalformedSchema_Denies(t *testing.T) {
	p := NewSchemaProcessor()

	// A malformed schema string that cannot be compiled must return an error
	// (SEC-3: deny, never degrade to pass-through).
	schema := `{
		"type": "object",
		"properties": {
			"field": {
				"type": "INVALID_TYPE_NOT_IN_SPEC"
			}
		}
	}`

	input := `{"field": "value"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error for malformed/uncompilable schema, got nil")
	}
	t.Logf("malformed-schema error: %v", err)
}

func TestValidateInput_JSONSchema_DefaultsApplied(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["active", "inactive"],
				"default": "active"
			},
			"name": {
				"type": "string"
			}
		},
		"required": ["name"]
	}`

	input := `{"name": "test"}`

	result, err := p.ValidateInput(input, schema)
	if err != nil {
		t.Fatalf("expected success with default applied, got error: %v", err)
	}

	if result["status"] != "active" {
		t.Errorf("expected status='active' (default), got %v", result["status"])
	}
	if result["name"] != "test" {
		t.Errorf("expected name='test', got %v", result["name"])
	}
}

// Caller-level integration tests (TEST-3 replacement: ValidateInputToJson +
// ValidateInput are the two public callers; both must work end-to-end).

func TestValidateInputToJson_ValidPayload_ReturnsJSON(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"amount": {
				"type": "integer",
				"minimum": 1
			},
			"currency": {
				"type": "string",
				"enum": ["USD", "EUR", "GBP"]
			}
		},
		"required": ["amount", "currency"]
	}`

	input := `{"amount": 5000, "currency": "USD"}`

	result, err := p.ValidateInputToJson(input, schema)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Parse the JSON result and verify
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// JSON numbers unmarshal as float64
	if parsed["amount"] != float64(5000) {
		t.Errorf("expected amount=5000, got %v", parsed["amount"])
	}
	if parsed["currency"] != "USD" {
		t.Errorf("expected currency='USD', got %v", parsed["currency"])
	}
}

func TestValidateInputToJson_InvalidPayload_ReturnsError(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"amount": {
				"type": "integer",
				"minimum": 1
			}
		},
		"required": ["amount"]
	}`

	input := `{"amount": 0}`

	_, err := p.ValidateInputToJson(input, schema)
	if err == nil {
		t.Fatal("expected error for invalid payload through ValidateInputToJson, got nil")
	}
	t.Logf("ValidateInputToJson error: %v", err)
}

func TestValidateInput_JSONSchema_EmptyInput_WithRequired_Fails(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			}
		},
		"required": ["name"]
	}`

	// Empty input with a required field
	_, err := p.ValidateInput("", schema)
	if err == nil {
		t.Fatal("expected error for empty input with required field, got nil")
	}
	t.Logf("empty-input-with-required error: %v", err)
}

func TestValidateInput_JSONSchema_EmptyInput_NoRequired_Succeeds(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			}
		}
	}`

	// Empty input, no required fields
	result, err := p.ValidateInput("", schema)
	if err != nil {
		t.Fatalf("expected success for empty input with no required fields, got: %v", err)
	}
	t.Logf("empty-input-no-required result: %v", result)
}

func TestValidateInput_JSONSchema_SchemaCaching(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"minLength": 1
			}
		},
		"required": ["name"]
	}`

	// First call compiles and caches the schema
	result1, err := p.ValidateInput(`{"name": "first"}`, schema)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if result1["name"] != "first" {
		t.Errorf("first call: expected name='first', got %v", result1["name"])
	}

	// Second call should use the cached schema
	result2, err := p.ValidateInput(`{"name": "second"}`, schema)
	if err != nil {
		t.Fatalf("second call (cached) failed: %v", err)
	}
	if result2["name"] != "second" {
		t.Errorf("second call: expected name='second', got %v", result2["name"])
	}

	// Third call with invalid data should still fail (cache does not bypass validation)
	_, err = p.ValidateInput(`{}`, schema)
	if err == nil {
		t.Fatal("third call (missing required): expected error, got nil")
	}
}

func TestValidateInput_JSONSchema_MultipleConstraintsCombined(t *testing.T) {
	p := NewSchemaProcessor()

	// A realistic salary_increase request type schema
	schema := `{
		"type": "object",
		"properties": {
			"new_monthly_rate": {
				"type": "integer",
				"minimum": 1,
				"maximum": 10000000
			},
			"justification": {
				"type": "string",
				"minLength": 10,
				"maxLength": 5000
			},
			"effective_date": {
				"type": "string",
				"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
			}
		},
		"required": ["new_monthly_rate", "justification", "effective_date"]
	}`

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid salary increase request",
			input:     `{"new_monthly_rate": 500000, "justification": "Annual performance-based salary adjustment", "effective_date": "2026-07-01"}`,
			wantError: false,
		},
		{
			name:      "zero rate (centavos floor)",
			input:     `{"new_monthly_rate": 0, "justification": "Annual adjustment for role change", "effective_date": "2026-07-01"}`,
			wantError: true,
		},
		{
			name:      "short justification",
			input:     `{"new_monthly_rate": 500000, "justification": "too short", "effective_date": "2026-07-01"}`,
			wantError: true,
		},
		{
			name:      "bad date format",
			input:     `{"new_monthly_rate": 500000, "justification": "Annual performance-based salary adjustment", "effective_date": "July 1 2026"}`,
			wantError: true,
		},
		{
			name:      "missing required effective_date",
			input:     `{"new_monthly_rate": 500000, "justification": "Annual performance-based salary adjustment"}`,
			wantError: true,
		},
		{
			name:      "unknown field rejected by default",
			input:     `{"new_monthly_rate": 500000, "justification": "Annual performance-based salary adjustment", "effective_date": "2026-07-01", "rogue": "data"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ValidateInput(tt.input, schema)
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected success, got error: %v", err)
			}
		})
	}
}

func TestValidateInput_JSONSchema_ExplicitAdditionalPropertiesFalse(t *testing.T) {
	p := NewSchemaProcessor()

	// Explicit additionalProperties:false should behave identically to the
	// implicit default.
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`

	// Known field only — should pass
	result, err := p.ValidateInput(`{"name": "valid"}`, schema)
	if err != nil {
		t.Fatalf("expected success for known-only payload, got: %v", err)
	}
	if result["name"] != "valid" {
		t.Errorf("expected name='valid', got %v", result["name"])
	}

	// Unknown field — should fail
	_, err = p.ValidateInput(`{"name": "valid", "extra": "nope"}`, schema)
	if err == nil {
		t.Fatal("expected error for unknown field with explicit additionalProperties:false")
	}
}

func TestValidateInput_JSONSchema_NumberType(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"rate": {
				"type": "number",
				"minimum": 0.01,
				"maximum": 1.0
			}
		}
	}`

	// Valid number
	result, err := p.ValidateInput(`{"rate": 0.5}`, schema)
	if err != nil {
		t.Fatalf("expected success for valid number, got: %v", err)
	}
	t.Logf("number result: %v (type %T)", result["rate"], result["rate"])

	// Below minimum
	_, err = p.ValidateInput(`{"rate": 0.001}`, schema)
	if err == nil {
		t.Fatal("expected error for number below minimum")
	}
}

func TestValidateInput_JSONSchema_StringCoercionToNumber(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"rate": {
				"type": "number",
				"minimum": 0.01
			}
		}
	}`

	// String "0.5" should coerce to float64(0.5) before validation
	result, err := p.ValidateInput(`{"rate": "0.5"}`, schema)
	if err != nil {
		t.Fatalf("expected success with string-to-number coercion, got: %v", err)
	}

	rate, ok := result["rate"].(float64)
	if !ok {
		t.Fatalf("expected rate to be float64 after coercion, got %T", result["rate"])
	}
	if rate != 0.5 {
		t.Errorf("expected rate=0.5, got %v", rate)
	}
}

func TestValidateInput_JSONSchema_ErrorMessageContainsContext(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["active", "inactive"]
			}
		}
	}`

	input := `{"status": "bad"}`

	_, err := p.ValidateInput(input, schema)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The error message should contain "validation failed" (our wrapper)
	errMsg := err.Error()
	if !strings.Contains(errMsg, "validation failed") {
		t.Errorf("expected error to contain 'validation failed', got: %s", errMsg)
	}
}

func TestValidateInput_SimpleFormat_StillWorks(t *testing.T) {
	p := NewSchemaProcessor()

	// The simple format (non-JSON-Schema) path should be untouched (Q-VAL-5).
	schema := `{
		"name": {
			"type": "string",
			"required": true
		},
		"age": {
			"type": "int",
			"required": false,
			"default": 25
		}
	}`

	result, err := p.ValidateInput(`{"name": "John"}`, schema)
	if err != nil {
		t.Fatalf("expected success for simple format, got: %v", err)
	}
	if result["name"] != "John" {
		t.Errorf("expected name='John', got %v", result["name"])
	}
	// JSON unmarshaling produces float64 for numbers; the simple format path
	// stores the default as-is (json.Unmarshal default behaviour).
	if result["age"] != float64(25) {
		t.Errorf("expected age=25.0 (default, json float64), got %v (type %T)", result["age"], result["age"])
	}
}

func TestValidateInput_SimpleFormat_MissingRequired_Fails(t *testing.T) {
	p := NewSchemaProcessor()

	schema := `{
		"name": {
			"type": "string",
			"required": true
		}
	}`

	_, err := p.ValidateInput(`{}`, schema)
	if err == nil {
		t.Fatal("expected error for missing required in simple format, got nil")
	}
}
