package operations

import (
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtobufTimestamp converts time.Time to protobuf timestamp pointers
type ProtobufTimestamp struct {
	Unix   *int64
	String *string
}

// NewProtobufTimestamp creates a protobuf timestamp from time.Time
func NewProtobufTimestamp(t time.Time) ProtobufTimestamp {
	timestamp := t.Unix()
	timestampStr := t.Format(time.RFC3339)
	return ProtobufTimestamp{
		Unix:   &timestamp,
		String: &timestampStr,
	}
}

// ConvertToProtobufMap converts a map[string]any to protobuf-compatible format
func ConvertToProtobufMap(data map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range data {
		switch v := value.(type) {
		case time.Time:
			// Convert time.Time to Unix timestamp for protobuf compatibility
			timestamp := v.Unix()
			timestampStr := v.Format(time.RFC3339)

			// Handle timestamp fields
			if key == "date_created" {
				result["date_created"] = &timestamp
				result["date_created_string"] = &timestampStr
			} else if key == "date_modified" {
				result["date_modified"] = &timestamp
				result["date_modified_string"] = &timestampStr
			} else {
				result[key] = &timestamp
			}
		default:
			result[key] = value
		}
	}

	return result
}

// generateUUID generates a simple UUID (simplified implementation)
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ProtobufMapper helps convert between protobuf models and database maps
type ProtobufMapper struct{}

// NewProtobufMapper creates a new protobuf mapper
func NewProtobufMapper() *ProtobufMapper {
	return &ProtobufMapper{}
}

// ConvertProtobufToMap converts a protobuf struct to a map[string]any using protojson
// This is the reverse operation of ConvertMapToProtobuf, providing consistent conversion
func (p *ProtobufMapper) ConvertProtobufToMap(pb proto.Message) (map[string]any, error) {
	if pb == nil {
		return nil, fmt.Errorf("protobuf object is nil")
	}

	// Use protojson to marshal protobuf to JSON bytes
	marshalOptions := protojson.MarshalOptions{
		UseProtoNames: true, // Use proto field names (snake_case) to match Firestore
	}

	jsonBytes, err := marshalOptions.Marshal(pb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	// Convert JSON bytes back to map[string]any
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	return result, nil
}

// ConvertMapToProtobufFields converts a map to protobuf-compatible field values
func (p *ProtobufMapper) ConvertMapToProtobufFields(data map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range data {
		switch v := value.(type) {
		case time.Time:
			// Handle timestamp conversions for protobuf
			timestamp := v.Unix()
			timestampStr := v.Format(time.RFC3339)

			if key == "date_created" {
				result["DateCreated"] = &timestamp
				result["DateCreatedString"] = &timestampStr
			} else if key == "date_modified" {
				result["DateModified"] = &timestamp
				result["DateModifiedString"] = &timestampStr
			} else {
				result[key] = &timestamp
			}
		default:
			// Keep original field names - SetProtobufFields will handle the mapping
			result[key] = value
		}
	}

	return result
}

// Helper functions

// field_name_comma_index finds the comma in a field name (for json tags with options)
func field_name_comma_index(s string) int {
	for i, c := range s {
		if c == ',' {
			return i
		}
	}
	return -1
}

// camelToSnake converts CamelCase to snake_case
func camelToSnake(s string) string {
	var result []rune

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r-'A'+'a')
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}

// TimestampFromTime converts a time.Time (from TIMESTAMPTZ) into Unix millis
// and an RFC3339 string suitable for protobuf fields.
// Returns (unixMillis, rfc3339String, ok). ok is false when t is zero.
func TimestampFromTime(t time.Time) (int64, string, bool) {
	if t.IsZero() {
		return 0, "", false
	}
	return t.UnixMilli(), t.Format(time.RFC3339), true
}

// ParseTimestamp converts string timestamp to Unix timestamp (milliseconds)
// This is a common utility function for parsing various timestamp formats
func ParseTimestamp(timestampStr string) (int64, error) {
	// Try parsing as RFC3339 format first (most common)
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// ConvertMapToProtobuf converts a map[string]any to any protobuf message using protojson
// This is a generic helper that can be used across all repositories
// It handles automatic field filtering (DiscardUnknown: true) and provides consistent conversion logic
func ConvertMapToProtobuf[T proto.Message](data map[string]any, target T) (T, error) {
	// Convert map to JSON bytes
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return target, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Use protojson to unmarshal directly into protobuf target
	unmarshalOptions := protojson.UnmarshalOptions{
		DiscardUnknown: true, // Automatically ignores fields not present in protobuf
	}

	if err := unmarshalOptions.Unmarshal(jsonBytes, target); err != nil {
		return target, fmt.Errorf("failed to unmarshal JSON to protobuf [ConvertMapToProtobuf]: %w", err)
	}

	return target, nil
}

// ConvertSliceToProtobuf converts a slice of map[string]any to a slice of protobuf messages
// This is a generic helper useful for list operations across all repositories
// It returns successful conversions and tracks conversion errors
func ConvertSliceToProtobuf[T proto.Message](dataSlice []map[string]any, targetFactory func() T) ([]T, []error) {
	// Initialize as empty slice instead of nil to ensure proper JSON marshaling
	results := make([]T, 0, len(dataSlice))
	var errors []error

	for i, data := range dataSlice {
		target := targetFactory()
		converted, err := ConvertMapToProtobuf(data, target)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to convert item %d: %w", i, err))
			continue
		}
		results = append(results, converted)
	}

	return results, errors
}
