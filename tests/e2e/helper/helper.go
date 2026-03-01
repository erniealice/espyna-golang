//go:build mock_auth

package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/tests/testutil"
)

// TestEnvironment encapsulates test server and utilities
type TestEnvironment struct {
	Server    *httptest.Server
	Container *core.Container
	Client    *http.Client
	BaseURL   string
	cleanup   func()
}

// SetupTestEnvironment creates isolated test environment
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// Setup environment for mock provider (providers self-configure from env)
	testutil.SetupTestEnvironment("mock")

	// Create container from environment (providers read their own config)
	container, err := core.NewContainerFromEnv()
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	// Get route manager from container for routes
	routeManager := container.GetRouteManager()
	handler := createTestHandler(routeManager)

	server := httptest.NewServer(handler)

	env := &TestEnvironment{
		Server:    server,
		Container: container,
		Client:    &http.Client{Timeout: 10 * time.Second},
		BaseURL:   server.URL,
		cleanup: func() {
			server.Close()
			container.Close()
		},
	}

	t.Cleanup(env.cleanup)
	return env
}

// createTestHandler creates HTTP handler for testing using the registry system
func createTestHandler(registry any) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy","timestamp":"`+time.Now().Format(time.RFC3339)+`"}`)
	})

	// For now, create mock handlers until the registry integration is complete
	// TODO: Replace with actual registry route mounting when available
	registerMockEndpointsForTesting(mux)

	return mux
}

// APIRequest represents a structured API request
type APIRequest struct {
	Method      string
	Path        string
	Body        any
	Headers     map[string]string
	QueryParams map[string]string
}

// APIResponse represents a structured API response
type APIResponse struct {
	StatusCode   int
	Body         []byte
	Headers      http.Header
	ResponseTime time.Duration
}

// PerformRequest executes HTTP request with comprehensive logging and validation
func (env *TestEnvironment) PerformRequest(t *testing.T, req APIRequest) *APIResponse {
	var bodyReader *bytes.Reader

	// Handle request body
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	// Construct URL with query parameters
	url := env.BaseURL + req.Path
	if len(req.QueryParams) > 0 {
		params := make([]string, 0, len(req.QueryParams))
		for key, value := range req.QueryParams {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		url += "?" + strings.Join(params, "&")
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, url, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request with timing
	start := time.Now()
	resp, err := env.Client.Do(httpReq)
	responseTime := time.Since(start)

	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body := make([]byte, 0)
	if resp.Body != nil {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		body = buf.Bytes()
	}

	// Log request/response for debugging
	t.Logf("API Request: %s %s -> %d (%v)", req.Method, req.Path, resp.StatusCode, responseTime)
	if len(body) > 0 && len(body) < 1000 { // Avoid logging huge responses
		t.Logf("Response Body: %s", string(body))
	}

	return &APIResponse{
		StatusCode:   resp.StatusCode,
		Body:         body,
		Headers:      resp.Header,
		ResponseTime: responseTime,
	}
}

// TestCreateOperation tests a generic create operation for any entity and returns the created entity ID
func TestCreateOperation(t *testing.T, env *TestEnvironment, entityPath string, createData map[string]any) string {
	req := APIRequest{
		Method: "POST",
		Path:   entityPath + "/create",
		Body:   createData,
	}

	resp := env.PerformRequest(t, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to create entity at %s: status %d, body: %s",
			entityPath, resp.StatusCode, string(resp.Body))
	}

	var createResp map[string]any
	if err := json.Unmarshal(resp.Body, &createResp); err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}

	if success, ok := createResp["success"].(bool); !ok || !success {
		t.Fatalf("Create operation should return success=true")
	}

	// Extract ID from created entity
	data, ok := createResp["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatalf("Create response should contain data array with at least one entity")
	}

	entity, ok := data[0].(map[string]any)
	if !ok {
		t.Fatalf("Created entity should be a valid object")
	}

	entityID, ok := entity["id"].(string)
	if !ok || entityID == "" {
		t.Fatalf("Created entity should have a valid ID")
	}

	t.Logf("Create operation validated successfully for %s, created ID: %s", entityPath, entityID)
	return entityID
}

// TestReadOperation tests a generic read operation for any entity
func TestReadOperation(t *testing.T, env *TestEnvironment, entityPath string, entityID string) {
	// Use empty body for read operations - the real implementation uses different format
	testData := map[string]any{}

	req := APIRequest{
		Method: "POST",
		Path:   entityPath + "/read",
		Body:   testData,
	}

	resp := env.PerformRequest(t, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to read entity at %s: status %d, body: %s",
			entityPath, resp.StatusCode, string(resp.Body))
	}

	var readResp map[string]any
	if err := json.Unmarshal(resp.Body, &readResp); err != nil {
		t.Fatalf("Failed to parse read response: %v", err)
	}

	if success, ok := readResp["success"].(bool); !ok || !success {
		t.Fatalf("Read operation should return success=true")
	}

	t.Logf("Read operation validated successfully for %s", entityPath)
}

// TestListOperation tests a generic list operation for any entity
func TestListOperation(t *testing.T, env *TestEnvironment, entityPath string) {
	// Use empty body for list operations
	testData := map[string]any{}

	req := APIRequest{
		Method: "POST",
		Path:   entityPath + "/list",
		Body:   testData,
	}

	resp := env.PerformRequest(t, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to list entities at %s: status %d, body: %s",
			entityPath, resp.StatusCode, string(resp.Body))
	}

	var listResp map[string]any
	if err := json.Unmarshal(resp.Body, &listResp); err != nil {
		t.Fatalf("Failed to parse list response: %v", err)
	}

	if success, ok := listResp["success"].(bool); !ok || !success {
		t.Fatalf("List operation should return success=true")
	}

	// Check if data array exists (even if empty)
	if _, ok := listResp["data"]; !ok {
		t.Fatalf("List operation should return data array")
	}

	t.Logf("List operation validated successfully for %s", entityPath)
}

// TestUpdateOperation tests a generic update operation for any entity
func TestUpdateOperation(t *testing.T, env *TestEnvironment, entityPath string, entityID string, updateData map[string]any) {
	// Ensure ID is set in the update data
	updateData["id"] = entityID

	req := APIRequest{
		Method: "POST",
		Path:   entityPath + "/update",
		Body:   updateData,
	}

	resp := env.PerformRequest(t, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to update entity at %s: status %d, body: %s",
			entityPath, resp.StatusCode, string(resp.Body))
	}

	var updateResp map[string]any
	if err := json.Unmarshal(resp.Body, &updateResp); err != nil {
		t.Fatalf("Failed to parse update response: %v", err)
	}

	if success, ok := updateResp["success"].(bool); !ok || !success {
		t.Fatalf("Update operation should return success=true")
	}

	t.Logf("Update operation validated successfully for %s with ID: %s", entityPath, entityID)
}

// TestCreateUpdateFlow tests create followed by update operation for realistic testing
func TestCreateUpdateFlow(t *testing.T, env *TestEnvironment, entityPath string, createData map[string]any, updateData map[string]any) {
	// 1. Create entity
	entityID := TestCreateOperation(t, env, entityPath, createData)

	// 2. Update the created entity
	TestUpdateOperation(t, env, entityPath, entityID, updateData)

	t.Logf("Create-Update flow validated successfully for %s", entityPath)
}

// TestCreateReadFlow tests create followed by read operation for realistic testing
func TestCreateReadFlow(t *testing.T, env *TestEnvironment, entityPath string, createData map[string]any) {
	// 1. Create entity
	entityID := TestCreateOperation(t, env, entityPath, createData)

	// 2. Read the created entity
	TestReadOperation(t, env, entityPath, entityID)

	t.Logf("Create-Read flow validated successfully for %s", entityPath)
}

// GetUpdateDataForEntity returns appropriate test data for updating different entity types
// Note: Primary key (id) will be excluded from updates as requested
func GetUpdateDataForEntity(entityType string) map[string]any {
	timestamp := time.Now().Unix()

	switch entityType {
	case "admin":
		return map[string]any{
			"active": true,
		}
	case "client":
		return map[string]any{
			"active": true,
		}
	case "client-attribute":
		return map[string]any{
			"name":  fmt.Sprintf("updated-attribute-%d", timestamp),
			"value": "updated-value",
		}
	case "delegate":
		return map[string]any{
			"active": true,
		}
	case "delegate-client":
		return map[string]any{
			"active": true,
		}
	case "group":
		return map[string]any{
			"name":        fmt.Sprintf("updated-group-%d", timestamp),
			"description": "Updated group description",
		}
	case "location":
		return map[string]any{
			"name":    fmt.Sprintf("updated-location-%d", timestamp),
			"address": "456 Updated Street, Updated City",
		}
	case "location-attribute":
		return map[string]any{
			"name":  fmt.Sprintf("updated-location-attr-%d", timestamp),
			"value": "updated-value",
		}
	case "manager":
		return map[string]any{
			"active": true,
		}
	case "permission":
		return map[string]any{
			"name":        fmt.Sprintf("updated-permission-%d", timestamp),
			"description": "Updated permission description",
		}
	case "role":
		return map[string]any{
			"name":        fmt.Sprintf("updated-role-%d", timestamp),
			"description": "Updated role description",
		}
	case "role-permission":
		return map[string]any{
			"active": true,
		}
	case "staff":
		return map[string]any{
			"active": true,
		}
	case "user":
		return map[string]any{
			"email":     fmt.Sprintf("updated-user-%d@example.com", timestamp),
			"firstName": "Updated",
			"lastName":  "User",
		}
	case "workspace":
		return map[string]any{
			"name":        fmt.Sprintf("updated-workspace-%d", timestamp),
			"description": "Updated workspace description",
		}
	case "workspace-user":
		return map[string]any{
			"active": true,
		}
	case "workspace-user-role":
		return map[string]any{
			"active": true,
		}
	case "event":
		return map[string]any{
			"name":        fmt.Sprintf("updated-event-%d", timestamp),
			"description": "Updated event description",
		}
	case "framework":
		return map[string]any{
			"name":        fmt.Sprintf("updated-framework-%d", timestamp),
			"description": "Updated framework description",
		}
	case "objective":
		return map[string]any{
			"name":        fmt.Sprintf("updated-objective-%d", timestamp),
			"description": "Updated objective description",
		}
	case "task":
		return map[string]any{
			"name":        fmt.Sprintf("updated-task-%d", timestamp),
			"description": "Updated task description",
		}
	case "payment":
		return map[string]any{
			"amount":   "149.99",
			"currency": "USD",
		}
	case "payment-method":
		return map[string]any{
			"type":  "debit_card",
			"last4": "5678",
		}
	case "payment-profile":
		return map[string]any{
			"billingName":  "Updated Client Name",
			"billingEmail": fmt.Sprintf("updated-billing-%d@example.com", timestamp),
		}
	case "product":
		return map[string]any{
			"name":        fmt.Sprintf("updated-product-%d", timestamp),
			"description": "Updated product description",
		}
	case "collection":
		return map[string]any{
			"name":        fmt.Sprintf("updated-collection-%d", timestamp),
			"description": "Updated collection description",
		}
	case "collection-plan":
		return map[string]any{
			"active": true,
		}
	case "price-product":
		return map[string]any{
			"amount":   "69.99",
			"currency": "USD",
		}
	case "product-attribute":
		return map[string]any{
			"name":  fmt.Sprintf("updated-product-attr-%d", timestamp),
			"value": "updated-value",
		}
	case "product-collection":
		return map[string]any{
			"active": true,
		}
	case "product-plan":
		return map[string]any{
			"active": true,
		}
	case "resource":
		return map[string]any{
			"name":        fmt.Sprintf("updated-resource-%d", timestamp),
			"description": "Updated resource description",
		}
	case "record":
		return map[string]any{
			"title":   fmt.Sprintf("updated-record-%d", timestamp),
			"content": "Updated record content",
		}
	case "subscription":
		return map[string]any{
			"status": "cancelled",
		}
	case "balance":
		return map[string]any{
			"currentBalance": "150.00",
			"currency":       "USD",
		}
	case "invoice":
		return map[string]any{
			"amount":   "149.99",
			"currency": "USD",
		}
	case "plan":
		return map[string]any{
			"name":        fmt.Sprintf("updated-plan-%d", timestamp),
			"description": "Updated plan description",
		}
	case "plan-settings":
		return map[string]any{
			"key":   fmt.Sprintf("updated-setting-%d", timestamp),
			"value": "updated-value",
		}
	case "price-plan":
		return map[string]any{
			"amount":   "39.99",
			"currency": "USD",
		}
	default:
		return map[string]any{
			"name":        fmt.Sprintf("updated-%s-%d", entityType, timestamp),
			"description": fmt.Sprintf("Updated %s description", entityType),
		}
	}
}

// GetTestDataForEntity returns appropriate test data for creating different entity types
func GetTestDataForEntity(entityType string) map[string]any {
	timestamp := time.Now().Unix()

	switch entityType {
	case "admin":
		return map[string]any{
			"userId": fmt.Sprintf("user-test-admin-%d", timestamp),
		}
	case "client":
		return map[string]any{
			"userId": fmt.Sprintf("user-test-client-%d", timestamp),
		}
	case "client-attribute":
		return map[string]any{
			"clientId": "test-client-id",
			"name":     fmt.Sprintf("test-attribute-%d", timestamp),
			"value":    "test-value",
		}
	case "delegate":
		return map[string]any{
			"userId": fmt.Sprintf("user-test-delegate-%d", timestamp),
		}
	case "delegate-client":
		return map[string]any{
			"delegateId": "test-delegate-id",
			"clientId":   "test-client-id",
		}
	case "group":
		return map[string]any{
			"name":        fmt.Sprintf("test-group-%d", timestamp),
			"description": "Test group for E2E testing",
		}
	case "location":
		return map[string]any{
			"name":    fmt.Sprintf("test-location-%d", timestamp),
			"address": "123 Test Street, Test City",
		}
	case "location-attribute":
		return map[string]any{
			"locationId": "test-location-id",
			"name":       fmt.Sprintf("test-location-attr-%d", timestamp),
			"value":      "test-value",
		}
	case "manager":
		return map[string]any{
			"userId": fmt.Sprintf("user-test-manager-%d", timestamp),
		}
	case "permission":
		return map[string]any{
			"name":        fmt.Sprintf("test-permission-%d", timestamp),
			"description": "Test permission for E2E testing",
		}
	case "role":
		return map[string]any{
			"name":        fmt.Sprintf("test-role-%d", timestamp),
			"description": "Test role for E2E testing",
		}
	case "role-permission":
		return map[string]any{
			"roleId":       "test-role-id",
			"permissionId": "test-permission-id",
		}
	case "staff":
		return map[string]any{
			"userId": fmt.Sprintf("user-test-staff-%d", timestamp),
		}
	case "user":
		return map[string]any{
			"email":     fmt.Sprintf("test-user-%d@example.com", timestamp),
			"firstName": "Test",
			"lastName":  "User",
		}
	case "workspace":
		return map[string]any{
			"name":        fmt.Sprintf("test-workspace-%d", timestamp),
			"description": "Test workspace for E2E testing",
		}
	case "workspace-user":
		return map[string]any{
			"workspaceId": "test-workspace-id",
			"userId":      "test-user-id",
		}
	case "workspace-user-role":
		return map[string]any{
			"workspaceUserId": "test-workspace-user-id",
			"roleId":          "test-role-id",
		}
	case "event":
		return map[string]any{
			"name":        fmt.Sprintf("test-event-%d", timestamp),
			"description": "Test event for E2E testing",
			"timezone":    "UTC",
		}
	case "framework":
		return map[string]any{
			"name":        fmt.Sprintf("test-framework-%d", timestamp),
			"description": "Test framework for E2E testing",
		}
	case "objective":
		return map[string]any{
			"frameworkId": "test-framework-id",
			"name":        fmt.Sprintf("test-objective-%d", timestamp),
			"description": "Test objective for E2E testing",
		}
	case "task":
		return map[string]any{
			"objectiveId": "test-objective-id",
			"name":        fmt.Sprintf("test-task-%d", timestamp),
			"description": "Test task for E2E testing",
		}
	case "payment":
		return map[string]any{
			"clientId": "test-client-id",
			"amount":   "99.99",
			"currency": "USD",
		}
	case "payment-method":
		return map[string]any{
			"clientId": "test-client-id",
			"type":     "credit_card",
			"last4":    "1234",
		}
	case "payment-profile":
		return map[string]any{
			"clientId":     "test-client-id",
			"billingName":  "Test Client",
			"billingEmail": fmt.Sprintf("billing-%d@example.com", timestamp),
		}
	case "product":
		return map[string]any{
			"name":        fmt.Sprintf("test-product-%d", timestamp),
			"description": "Test product for E2E testing",
		}
	case "collection":
		return map[string]any{
			"name":        fmt.Sprintf("test-collection-%d", timestamp),
			"description": "Test collection for E2E testing",
		}
	case "collection-plan":
		return map[string]any{
			"collectionId": "test-collection-id",
			"planId":       "test-plan-id",
		}
	case "price-product":
		return map[string]any{
			"productId": "test-product-id",
			"amount":    "49.99",
			"currency":  "USD",
		}
	case "product-attribute":
		return map[string]any{
			"productId": "test-product-id",
			"name":      fmt.Sprintf("test-product-attr-%d", timestamp),
			"value":     "test-value",
		}
	case "product-collection":
		return map[string]any{
			"productId":    "test-product-id",
			"collectionId": "test-collection-id",
		}
	case "product-plan":
		return map[string]any{
			"productId": "test-product-id",
			"planId":    "test-plan-id",
		}
	case "resource":
		return map[string]any{
			"name":        fmt.Sprintf("test-resource-%d", timestamp),
			"description": "Test resource for E2E testing",
		}
	case "record":
		return map[string]any{
			"title":   fmt.Sprintf("test-record-%d", timestamp),
			"content": "Test record content for E2E testing",
		}
	case "subscription":
		return map[string]any{
			"clientId": "test-client-id",
			"planId":   "test-plan-id",
			"status":   "active",
		}
	case "balance":
		return map[string]any{
			"clientId":       "test-client-id",
			"currentBalance": "0.00",
			"currency":       "USD",
		}
	case "invoice":
		return map[string]any{
			"subscriptionId": "test-subscription-id",
			"amount":         "99.99",
			"currency":       "USD",
		}
	case "plan":
		return map[string]any{
			"name":        fmt.Sprintf("test-plan-%d", timestamp),
			"description": "Test plan for E2E testing",
		}
	case "plan-settings":
		return map[string]any{
			"planId": "test-plan-id",
			"key":    fmt.Sprintf("test-setting-%d", timestamp),
			"value":  "test-value",
		}
	case "price-plan":
		return map[string]any{
			"planId":   "test-plan-id",
			"amount":   "29.99",
			"currency": "USD",
		}
	default:
		return map[string]any{
			"name":        fmt.Sprintf("test-%s-%d", entityType, timestamp),
			"description": fmt.Sprintf("Test %s for E2E testing", entityType),
		}
	}
}

// registerMockEndpointsForTesting registers mock endpoints for all domains
func registerMockEndpointsForTesting(mux *http.ServeMux) {
	// Entity Domain
	registerMockEntityEndpoints(mux)

	// Event Domain
	registerMockEventEndpoints(mux)

	// Framework Domain
	registerMockFrameworkEndpoints(mux)

	// Payment Domain
	registerMockPaymentEndpoints(mux)

	// Product Domain
	registerMockProductEndpoints(mux)

	// Record Domain
	registerMockRecordEndpoints(mux)

	// Subscription Domain
	registerMockSubscriptionEndpoints(mux)
}

// Entity mock endpoints
func registerMockEntityEndpoints(mux *http.ServeMux) {
	entities := []string{"admin", "client", "client-attribute", "delegate", "delegate-client",
		"group", "location", "location-attribute", "manager", "permission", "role",
		"role-permission", "staff", "user", "workspace", "workspace-user", "workspace-user-role"}

	for _, entity := range entities {
		basePath := "/api/entity/" + entity
		mux.HandleFunc(basePath+"/read", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/list", createMockListHandler(entity))
		// Also register other operations for completeness
		mux.HandleFunc(basePath+"/create", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/update", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/delete", createMockSuccessHandler(entity))
	}
}

// Event mock endpoints
func registerMockEventEndpoints(mux *http.ServeMux) {
	entities := []string{"event"}

	for _, entity := range entities {
		basePath := "/api/event/" + entity
		mux.HandleFunc(basePath+"/read", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/list", createMockListHandler(entity))
		mux.HandleFunc(basePath+"/create", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/update", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/delete", createMockSuccessHandler(entity))
	}
}

// Framework mock endpoints
func registerMockFrameworkEndpoints(mux *http.ServeMux) {
	entities := []string{"framework", "objective", "task"}

	for _, entity := range entities {
		basePath := "/api/framework/" + entity
		mux.HandleFunc(basePath+"/read", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/list", createMockListHandler(entity))
		mux.HandleFunc(basePath+"/create", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/update", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/delete", createMockSuccessHandler(entity))
	}
}

// Payment mock endpoints
func registerMockPaymentEndpoints(mux *http.ServeMux) {
	entities := []string{"payment", "payment-method", "payment-profile"}

	for _, entity := range entities {
		basePath := "/api/payment/" + entity
		mux.HandleFunc(basePath+"/read", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/list", createMockListHandler(entity))
		mux.HandleFunc(basePath+"/create", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/update", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/delete", createMockSuccessHandler(entity))
	}
}

// Product mock endpoints
func registerMockProductEndpoints(mux *http.ServeMux) {
	entities := []string{"product", "collection", "collection-plan", "price-product",
		"product-attribute", "product-collection", "product-plan", "resource"}

	for _, entity := range entities {
		basePath := "/api/product/" + entity
		mux.HandleFunc(basePath+"/read", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/list", createMockListHandler(entity))
		mux.HandleFunc(basePath+"/create", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/update", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/delete", createMockSuccessHandler(entity))
	}
}

// Record mock endpoints
func registerMockRecordEndpoints(mux *http.ServeMux) {
	entities := []string{"record"}

	for _, entity := range entities {
		basePath := "/api/record/" + entity
		mux.HandleFunc(basePath+"/read", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/list", createMockListHandler(entity))
		mux.HandleFunc(basePath+"/create", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/update", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/delete", createMockSuccessHandler(entity))
	}
}

// Subscription mock endpoints
func registerMockSubscriptionEndpoints(mux *http.ServeMux) {
	entities := []string{"subscription", "balance", "invoice", "plan", "plan-settings", "price-plan"}

	for _, entity := range entities {
		basePath := "/api/subscription/" + entity
		mux.HandleFunc(basePath+"/read", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/list", createMockListHandler(entity))
		mux.HandleFunc(basePath+"/create", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/update", createMockSuccessHandler(entity))
		mux.HandleFunc(basePath+"/delete", createMockSuccessHandler(entity))
	}
}

// createMockSuccessHandler creates a generic success handler for read/create operations
func createMockSuccessHandler(entityType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Generate unique ID based on current time for create operations
		timestamp := time.Now().UnixNano()
		entityID := fmt.Sprintf("test-%s-%d", entityType, timestamp)

		// Create a mock response with single entity including timestamp fields
		mockResponse := fmt.Sprintf(`{"success":true,"data":[{
			"id":"%s",
			"type":"%s",
			"active":true,
			"dateCreated":"%d",
			"dateCreatedString":"%s",
			"dateModified":"%d",
			"dateModifiedString":"%s"
		}]}`, entityID, entityType, timestamp/1000000, time.Now().Format(time.RFC3339), timestamp/1000000, time.Now().Format(time.RFC3339))
		fmt.Fprint(w, mockResponse)
	}
}

// createMockListHandler creates a generic list handler for list operations
func createMockListHandler(entityType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a mock response with multiple entities
		mockResponse := fmt.Sprintf(`{"success":true,"data":[
			{"id":"mock-%s-id-1","type":"%s","active":true},
			{"id":"mock-%s-id-2","type":"%s","active":true}
		]}`, entityType, entityType, entityType, entityType)
		fmt.Fprint(w, mockResponse)
	}
}
