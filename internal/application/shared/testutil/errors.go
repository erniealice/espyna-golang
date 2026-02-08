package testutil

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// AssertTranslatedError validates that an error matches the expected translated message
// This ensures tests validate specific error messages rather than just boolean error presence
//
// Parameters:
//   - t: testing instance
//   - err: the actual error to validate
//   - errorKey: the translation key for the expected error message
//   - translationService: service to get the translated message
//   - ctx: context for translation
func AssertTranslatedError(t *testing.T, err error, errorKey string, translationService ports.TranslationService, ctx context.Context) {
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	expectedError := contextutil.GetTranslatedMessageWithContext(
		ctx,
		translationService,
		errorKey,
		"",
	)

	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
	}
}

// AssertTranslatedErrorWithContext validates an error with context substitution using JSON string
// DEPRECATED: Use AssertTranslatedErrorWithTags for better type safety
// Useful for errors that include dynamic data like IDs
//
// Parameters:
//   - t: testing instance
//   - err: the actual error to validate
//   - errorKey: the translation key for the expected error message
//   - contextData: JSON string for context substitution in the error message
//   - translationService: service to get the translated message
//   - ctx: context for translation
//
// Example: AssertTranslatedErrorWithContext(t, err, "admin.errors.not_found", "{\"id\": \"123\"}", service, ctx)
func AssertTranslatedErrorWithContext(t *testing.T, err error, errorKey, contextData string, translationService ports.TranslationService, ctx context.Context) {
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	// Get translated message (ignoring contextData since GetTranslatedMessageWithContext doesn't support it)
	expectedError := contextutil.GetTranslatedMessageWithContext(
		ctx,
		translationService,
		errorKey,
		"", // Use empty fallback, since we're providing contextData externally
	)

	// Parse contextData JSON and perform placeholder substitution
	if contextData != "" {
		// Simple JSON parsing for common patterns like {"clientId": "value", "delegateId": "value"}
		// This handles the most common use cases in the tests
		contextData = strings.Trim(contextData, "{}")
		if strings.Contains(contextData, ":") {
			pairs := strings.Split(contextData, ",")
			for _, pair := range pairs {
				keyValue := strings.Split(pair, ":")
				if len(keyValue) == 2 {
					key := strings.Trim(strings.TrimSpace(keyValue[0]), "\"")
					value := strings.Trim(strings.TrimSpace(keyValue[1]), "\"")
					placeholder := "{" + key + "}"
					expectedError = strings.ReplaceAll(expectedError, placeholder, value)
				}
			}
		}
	}

	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
	}
}

// AssertTranslatedErrorWithTags validates an error with context substitution using type-safe map
// Preferred over AssertTranslatedErrorWithContext for better type safety and cleaner API
// Useful for errors that include dynamic data like IDs
//
// Parameters:
//   - t: testing instance
//   - err: the actual error to validate
//   - errorKey: the translation key for the expected error message
//   - contextData: map for context substitution in the error message
//   - translationService: service to get the translated message
//   - ctx: context for translation
//
// Example: AssertTranslatedErrorWithTags(t, err, "admin.errors.not_found", map[string]any{"id": "123"}, service, ctx)
func AssertTranslatedErrorWithTags(t *testing.T, err error, errorKey string, contextData map[string]any, translationService ports.TranslationService, ctx context.Context) {
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	// Get translated message
	expectedError := contextutil.GetTranslatedMessageWithContext(
		ctx,
		translationService,
		errorKey,
		"", // Use empty fallback, since we're providing contextData externally
	)

	// Perform placeholder substitution using the provided context data
	if contextData != nil {
		for key, value := range contextData {
			placeholder := "{" + key + "}"
			// Convert value to string representation
			var valueStr string
			switch v := value.(type) {
			case string:
				valueStr = v
			case int, int32, int64:
				valueStr = fmt.Sprintf("%d", v)
			case float32, float64:
				valueStr = fmt.Sprintf("%g", v)
			case bool:
				valueStr = fmt.Sprintf("%t", v)
			default:
				valueStr = fmt.Sprintf("%v", v)
			}
			expectedError = strings.ReplaceAll(expectedError, placeholder, valueStr)
		}
	}

	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
	}
}

// AssertNoError validates that no error occurred, providing a consistent error message
// Use this instead of manually checking err != nil with t.Fatalf
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate (should be nil)
//
// Example: AssertNoError(t, err)
func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// AssertError validates that an error occurred, providing a consistent error message
// Use this instead of manually checking err == nil with t.Fatal
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate (should not be nil)
//
// Example: AssertError(t, err)
func AssertError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Expected error but got none")
	}
}

// AssertErrorForNilRequest validates that an error occurred for a nil request
// Use this when testing nil request validation
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate (should not be nil)
//
// Example: AssertErrorForNilRequest(t, err)
func AssertErrorForNilRequest(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Expected error for nil request")
	}
}

// AssertErrorForNilData validates that an error occurred for nil data
// Use this when testing nil data validation
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate (should not be nil)
//
// Example: AssertErrorForNilData(t, err)
func AssertErrorForNilData(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Expected error for nil data")
	}
}

// AssertAuthorizationError validates that an authorization error occurred
// Use this when testing authorization failures
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate (should not be nil)
//
// Example: AssertAuthorizationError(t, err)
func AssertAuthorizationError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Expected authorization error")
	}
}

// AssertTransactionError validates that a transaction error occurred
// Use this when testing transaction failures
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate (should not be nil)
//
// Example: AssertTransactionError(t, err)
func AssertTransactionError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Expected error due to transaction failure")
	}
}

// AssertValidationError validates that a validation error occurred
// Use this when testing validation failures
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate (should not be nil)
//   - fieldName: the field that failed validation
//
// Example: AssertValidationError(t, err, "name too short")
func AssertValidationError(t *testing.T, err error, fieldName string) {
	if err == nil {
		t.Fatalf("Expected error for %s", fieldName)
	}
}

// AssertNotNil validates that a value is not nil, providing a consistent error message
// Use this for checking responses, data structures, etc.
//
// Parameters:
//   - t: testing instance
//   - value: the value to check (should not be nil)
//   - valueName: name of the value for error message clarity
//
// Example: AssertNotNil(t, response, "response")
func AssertNotNil(t *testing.T, value interface{}, valueName string) {
	if value == nil {
		t.Fatalf("Expected %s, got nil", valueName)
	}
}

// AssertNotFoundError validates that a not found error occurred
// Use this for testing scenarios where an entity was not found
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate
//
// Example: AssertNotFoundError(t, err)
func AssertNotFoundError(t *testing.T, err error) {
	if err == nil {
		t.Fatalf("Expected not found error, got nil")
	}
	// Check if the error message contains "not found" or similar patterns
	errStr := err.Error()
	lowerErrStr := strings.ToLower(errStr)
	if !strings.Contains(lowerErrStr, "not found") &&
		!strings.Contains(lowerErrStr, "not_found") &&
		!strings.Contains(lowerErrStr, "notfound") {
		t.Fatalf("Expected not found error, got: %s", errStr)
	}
}

// AssertAlreadyDeletedError validates that an already deleted error occurred
// Use this for testing scenarios where an entity was already deleted
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate
//
// Example: AssertAlreadyDeletedError(t, err)
func AssertAlreadyDeletedError(t *testing.T, err error) {
	if err == nil {
		t.Fatalf("Expected already deleted error, got nil")
	}
	// Check if the error message contains "already deleted" or similar patterns
	errStr := err.Error()
	lowerErrStr := strings.ToLower(errStr)
	if !strings.Contains(lowerErrStr, "already deleted") &&
		!strings.Contains(lowerErrStr, "already_deleted") &&
		!strings.Contains(lowerErrStr, "alreadydeleted") {
		t.Fatalf("Expected already deleted error, got: %s", errStr)
	}
}

// AssertHasDependenciesError validates that a has dependencies error occurred
// Use this for testing scenarios where an entity cannot be deleted due to dependencies
//
// Parameters:
//   - t: testing instance
//   - err: the error to validate
//
// Example: AssertHasDependenciesError(t, err)
func AssertHasDependenciesError(t *testing.T, err error) {
	if err == nil {
		t.Fatalf("Expected has dependencies error, got nil")
	}
	// Check if the error message contains "has dependencies" or similar patterns
	errStr := err.Error()
	lowerErrStr := strings.ToLower(errStr)
	if !strings.Contains(lowerErrStr, "has dependencies") &&
		!strings.Contains(lowerErrStr, "has_dependencies") &&
		!strings.Contains(lowerErrStr, "hasdependencies") {
		t.Fatalf("Expected has dependencies error, got: %s", errStr)
	}
}

// AssertTrue validates that a boolean value is true
// Use this instead of manually checking with t.Fatal
//
// Parameters:
//   - t: testing instance
//   - value: the boolean value to check
//   - fieldName: name of the field being checked
//
// Example: AssertTrue(t, response.Success, "success")
func AssertTrue(t *testing.T, value bool, fieldName string) {
	if !value {
		t.Fatalf("Expected %s to be true", fieldName)
	}
}

// AssertFalse validates that a boolean value is false
// Use this instead of manually checking with t.Fatal
//
// Parameters:
//   - t: testing instance
//   - value: the boolean value to check
//   - fieldName: name of the field being checked
//
// Example: AssertFalse(t, response.HasError, "HasError")
func AssertFalse(t *testing.T, value bool, fieldName string) {
	if value {
		t.Fatalf("Expected %s to be false", fieldName)
	}
}

// AssertEqual validates that two values are equal
// Use this for checking specific values or states
//
// Parameters:
//   - t: testing instance
//   - expected: the expected value
//   - actual: the actual value
//   - valueName: name of the value for error message clarity
//
// Example: AssertEqual(t, 1, len(response.Data), "response data length")
func AssertEqual(t *testing.T, expected, actual interface{}, valueName string) {
	if expected != actual {
		t.Fatalf("Expected %s to be %v, got %v", valueName, expected, actual)
	}
}

// AssertNotEqual validates that two values are not equal
// Use this for checking values that should be different
//
// Parameters:
//   - t: testing instance
//   - notExpected: the value that should not match
//   - actual: the actual value
//   - valueName: name of the value for error message clarity
//
// Example: AssertNotEqual(t, "", user.ID, "user ID")
func AssertNotEqual(t *testing.T, notExpected, actual interface{}, valueName string) {
	if notExpected == actual {
		t.Fatalf("Expected %s to not be %v, but it was", valueName, notExpected)
	}
}

// AssertGreaterThan validates that a value is greater than expected
// Use this for checking counts, lengths, etc.
//
// Parameters:
//   - t: testing instance
//   - actual: the actual numeric value
//   - threshold: the minimum value (exclusive)
//   - valueName: name of the value for error message clarity
//
// Example: AssertGreaterThan(t, len(response.Data), 0, "response data count")
func AssertGreaterThan(t *testing.T, actual, threshold int, valueName string) {
	if actual <= threshold {
		t.Fatalf("Expected %s to be greater than %d, got %d", valueName, threshold, actual)
	}
}

// AssertResponseValid validates common response patterns
// Use this for standard response validation
//
// Parameters:
//   - t: testing instance
//   - response: the response object to validate (should not be nil)
//   - err: the error that should be nil
//   - responseName: name of the response for error messages
//
// Example: AssertResponseValid(t, response, err, "create user response")
func AssertResponseValid(t *testing.T, response interface{}, err error, responseName string) {
	AssertNoError(t, err)
	AssertNotNil(t, response, responseName)
}

// AssertOperationSuccess validates that an operation completed successfully
// Use this for operations that should succeed without detailed response checking
//
// Parameters:
//   - t: testing instance
//   - err: the error that should be nil
//   - operationName: name of the operation for error messages
//
// Example: AssertOperationSuccess(t, err, "delete user operation")
func AssertOperationSuccess(t *testing.T, err error, operationName string) {
	if err != nil {
		t.Fatalf("Expected %s to succeed, but got error: %v", operationName, err)
	}
}

// AssertStringEqual validates that two strings are equal
// Use this for checking string values with consistent error messages
//
// Parameters:
//   - t: testing instance
//   - expected: the expected string value
//   - actual: the actual string value
//   - fieldName: name of the field for error message
//
// Example: AssertStringEqual(t, "expected-name", user.Name, "user name")
func AssertStringEqual(t *testing.T, expected, actual, fieldName string) {
	if expected != actual {
		t.Errorf("Expected %s '%s', got '%s'", fieldName, expected, actual)
	}
}

// AssertDataLength validates that a slice has the expected length
// Use this for checking response data arrays, lists, etc.
//
// Parameters:
//   - t: testing instance
//   - expected: the expected length
//   - actual: the actual slice length
//   - dataName: name of the data for error message
//
// Example: AssertDataLength(t, 1, len(response.Data), "response data")
func AssertDataLength(t *testing.T, expected, actual int, dataName string) {
	if expected != actual {
		t.Errorf("Expected %d %s, got %d", expected, dataName, actual)
	}
}

// AssertPositiveValue validates that a numeric value is positive
// Use this for checking IDs, counts, timestamps, etc.
//
// Parameters:
//   - t: testing instance
//   - value: the numeric value to check
//   - fieldName: name of the field for error message
//
// Example: AssertPositiveValue(t, user.DateCreated, "DateCreated")
func AssertPositiveValue(t *testing.T, value int64, fieldName string) {
	if value <= 0 {
		t.Errorf("Expected %s to be positive, got %d", fieldName, value)
	}
}

// AssertNonEmptyString validates that a string is not empty
// Use this for checking required string fields
//
// Parameters:
//   - t: testing instance
//   - value: the string value to check
//   - fieldName: name of the field for error message
//
// Example: AssertNonEmptyString(t, user.ID, "user ID")
func AssertNonEmptyString(t *testing.T, value, fieldName string) {
	if value == "" {
		t.Errorf("Expected non-empty %s", fieldName)
	}
}

// AssertTestCaseLoad validates that a test case loaded successfully
// Use this when loading test cases to ensure consistent error messaging
//
// Parameters:
//   - t: testing instance
//   - err: the error from test case loading (should be nil)
//   - testCaseName: name of the test case for error message
//
// Example: AssertTestCaseLoad(t, err, "CreateFramework_Success")
func AssertTestCaseLoad(t *testing.T, err error, testCaseName string) {
	if err != nil {
		t.Fatalf("Failed to load test case '%s': %v", testCaseName, err)
	}
}

// AssertFieldSet validates that a field is set (not nil)
// Use this for checking that optional fields are properly set
//
// Parameters:
//   - t: testing instance
//   - value: the pointer value to check (should not be nil)
//   - fieldName: name of the field for error message
//
// Example: AssertFieldSet(t, user.DateCreated, "DateCreated")
func AssertFieldSet(t *testing.T, value interface{}, fieldName string) {
	if value == nil {
		t.Errorf("Expected %s to be set", fieldName)
	}
}

// AssertFieldLength validates that a field has the expected length
// Use this for checking string lengths, array sizes, etc.
//
// Parameters:
//   - t: testing instance
//   - expected: the expected length
//   - actual: the actual length
//   - fieldName: name of the field for error message
//
// Example: AssertFieldLength(t, 100, len(framework.Name), "framework name")
func AssertFieldLength(t *testing.T, expected, actual int, fieldName string) {
	if expected != actual {
		t.Errorf("Expected %s length %d, got %d", fieldName, expected, actual)
	}
}

// AssertTimestampPositive validates that a timestamp is positive (greater than 0)
// Use this for checking DateCreated, DateModified, etc.
//
// Parameters:
//   - t: testing instance
//   - timestamp: the timestamp value to check
//   - fieldName: name of the field for error message
//
// Example: AssertTimestampPositive(t, *framework.DateCreated, "DateCreated")
func AssertTimestampPositive(t *testing.T, timestamp int64, fieldName string) {
	if timestamp <= 0 {
		t.Errorf("Expected %s to be positive, got %d", fieldName, timestamp)
	}
}

// AssertTimestampInMilliseconds validates that a timestamp appears to be in milliseconds
// Use this for checking that timestamps are properly formatted
//
// Parameters:
//   - t: testing instance
//   - timestamp: the timestamp value to check
//   - fieldName: name of the field for error message
//
// Example: AssertTimestampInMilliseconds(t, *framework.DateCreated, "DateCreated")
func AssertTimestampInMilliseconds(t *testing.T, timestamp int64, fieldName string) {
	// Year 2001 in milliseconds - reasonable minimum for modern timestamps
	const minMillisecondTimestamp = 1000000000000
	if timestamp < minMillisecondTimestamp {
		t.Errorf("Expected %s to be in milliseconds, got %d", fieldName, timestamp)
	}
}

// AssertDescriptionMatch validates that a description field matches expected value
// Use this for checking optional description fields with proper nil handling
//
// Parameters:
//   - t: testing instance
//   - expected: the expected description value
//   - actual: the actual description pointer (may be nil)
//   - fieldName: name of the field for error message
//
// Example: AssertDescriptionMatch(t, "expected desc", framework.Description, "description")
func AssertDescriptionMatch(t *testing.T, expected string, actual *string, fieldName string) {
	if actual == nil || *actual != expected {
		var actualValue string
		if actual == nil {
			actualValue = "<nil>"
		} else {
			actualValue = *actual
		}
		t.Errorf("Expected %s to match '%s', got '%s'", fieldName, expected, actualValue)
	}
}

// GenerateDefaultLongString generates a string of repeated 'a' characters for testing
// This is used to test validation rules like string length limits in a consistent way
//
// Parameters:
//   - length: the desired length of the generated string
//
// Returns:
//   - A string containing 'a' repeated length times
//
// Example: GenerateDefaultLongString(300) returns "aaaa..." (300 'a' characters)
func GenerateDefaultLongString(length int) string {
	return strings.Repeat("a", length)
}

// AssertNil validates that a value is nil, providing a consistent error message
// Use this for checking that responses are nil when errors occur
//
// Parameters:
//   - t: testing instance
//   - value: the value to check (should be nil)
//   - valueName: name of the value for error message clarity
//
// Example: AssertNil(t, response, "response for error case")
func AssertNil(t *testing.T, value interface{}, valueName string) {
	if value != nil {
		// Check if it's a nil pointer by trying to use it
		switch v := value.(type) {
		case nil:
			return // true nil
		default:
			// For typed nil pointers, Go's %v will show <nil>
			if fmt.Sprintf("%v", v) == "<nil>" {
				return // typed nil pointer
			}
			t.Fatalf("Expected %s to be nil, got %v", valueName, value)
		}
	}
}

// AssertGreaterThanOrEqual validates that a value is greater than or equal to expected
// Use this for checking counts, lengths, etc. where the value can be equal to the threshold
//
// Parameters:
//   - t: testing instance
//   - actual: the actual numeric value
//   - threshold: the minimum value (inclusive)
//   - valueName: name of the value for error message clarity
//
// Example: AssertGreaterThanOrEqual(t, len(response.Data), 1, "minimum response data count")
func AssertGreaterThanOrEqual(t *testing.T, actual, threshold int, valueName string) {
	if actual < threshold {
		t.Fatalf("Expected %s to be greater than or equal to %d, got %d", valueName, threshold, actual)
	}
}

// AssertMap validates that a value is a map[string]interface{}
// Use this for type assertions when working with JSON data
//
// Parameters:
//   - t: testing instance
//   - value: the value to check
//   - valueName: name of the value for error message
//
// Returns: the typed map if assertion succeeds, nil otherwise
//
// Example: myMap := AssertMap(t, rawData, "test data")
func AssertMap(t *testing.T, value interface{}, valueName string) map[string]interface{} {
	if mapValue, ok := value.(map[string]interface{}); ok {
		return mapValue
	}
	t.Fatalf("Expected %s to be a map[string]interface{}, got %T", valueName, value)
	return nil
}

// AssertArray validates that a value is a []interface{}
// Use this for type assertions when working with JSON arrays
//
// Parameters:
//   - t: testing instance
//   - value: the value to check
//   - valueName: name of the value for error message
//
// Returns: the typed array if assertion succeeds, nil otherwise
//
// Example: myArray := AssertArray(t, rawData, "test array")
func AssertArray(t *testing.T, value interface{}, valueName string) []interface{} {
	if arrayValue, ok := value.([]interface{}); ok {
		return arrayValue
	}
	t.Fatalf("Expected %s to be a []interface{}, got %T", valueName, value)
	return nil
}
