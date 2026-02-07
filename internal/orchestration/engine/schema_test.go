package engine

import (
	"encoding/json"
	"reflect"
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
			"primary_email":   "john@example.com",
			"primary_name":    "John",
			"cc_email":        "manager@example.com",
			"cc_name":         "Manager",
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
			"email_subject":      "Payment Confirmation",
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
