package listdata

import (
	"reflect"
	"sort"
	"strings"
	"time"

	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// SortUtils provides utilities for sorting data based on protobuf sort requests
type SortUtils struct{}

// NewSortUtils creates a new sort utility instance
func NewSortUtils() *SortUtils {
	return &SortUtils{}
}

// SortItems sorts a slice of items based on the sort request
func (s *SortUtils) SortItems(items interface{}, sortRequest *commonpb.SortRequest) {
	if sortRequest == nil || len(sortRequest.Fields) == 0 {
		return // No sorting requested
	}

	// Use reflection to work with any slice type
	sliceValue := reflect.ValueOf(items)
	if sliceValue.Kind() != reflect.Slice {
		return // Not a slice, can't sort
	}

	// Create a sortable wrapper
	sortable := &sortableSlice{
		slice:  sliceValue,
		fields: sortRequest.Fields,
		utils:  s,
	}

	// Perform the sort
	sort.Sort(sortable)
}

// sortableSlice implements sort.Interface for generic slices
type sortableSlice struct {
	slice  reflect.Value
	fields []*commonpb.SortField
	utils  *SortUtils
}

func (s *sortableSlice) Len() int {
	return s.slice.Len()
}

func (s *sortableSlice) Swap(i, j int) {
	iValue := s.slice.Index(i)
	jValue := s.slice.Index(j)

	// Create temporary values for swapping
	temp := reflect.New(iValue.Type()).Elem()
	temp.Set(iValue)
	iValue.Set(jValue)
	jValue.Set(temp)
}

func (s *sortableSlice) Less(i, j int) bool {
	iItem := s.slice.Index(i).Interface()
	jItem := s.slice.Index(j).Interface()

	// Compare using all sort fields in order
	for _, field := range s.fields {
		result := s.utils.compareByField(iItem, jItem, field)
		if result != 0 {
			if field.Direction == commonpb.SortDirection_DESC {
				return result > 0
			}
			return result < 0
		}
	}

	return false // Items are equal
}

// compareByField compares two items by a specific field
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func (s *SortUtils) compareByField(a, b interface{}, field *commonpb.SortField) int {
	aValue := s.getFieldValue(a, field.Field)
	bValue := s.getFieldValue(b, field.Field)

	// Handle null values according to null_order
	if aValue == nil && bValue == nil {
		return 0
	}
	if aValue == nil {
		if field.NullOrder == commonpb.NullOrder_NULLS_FIRST {
			return -1
		}
		return 1
	}
	if bValue == nil {
		if field.NullOrder == commonpb.NullOrder_NULLS_FIRST {
			return 1
		}
		return -1
	}

	// Compare based on type
	return s.compareValues(aValue, bValue, field)
}

// getFieldValue extracts a field value using dot notation (similar to FilterUtils)
func (s *SortUtils) getFieldValue(item interface{}, fieldPath string) interface{} {
	if item == nil {
		return nil
	}

	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	parts := strings.Split(fieldPath, ".")

	for _, part := range parts {
		if val.Kind() == reflect.Struct {
			fieldName := s.toCamelCase(part)
			field := val.FieldByName(fieldName)
			if !field.IsValid() {
				return nil
			}

			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					return nil
				}
				val = field.Elem()
			} else {
				val = field
			}
		} else {
			return nil
		}
	}

	return val.Interface()
}

// toCamelCase converts snake_case to CamelCase
func (s *SortUtils) toCamelCase(str string) string {
	parts := strings.Split(str, "_")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	if len(result) > 0 {
		result = strings.ToUpper(result[:1]) + result[1:]
	}
	return result
}

// compareValues compares two values of the same type
func (s *SortUtils) compareValues(a, b interface{}, field *commonpb.SortField) int {
	// Try string comparison first (most common)
	if aStr, aOk := s.toString(a); aOk {
		if bStr, bOk := s.toString(b); bOk {
			return s.compareStrings(aStr, bStr, field.StringOptions)
		}
	}

	// Try numeric comparison
	if aNum, aOk := s.toFloat64(a); aOk {
		if bNum, bOk := s.toFloat64(b); bOk {
			return s.compareNumbers(aNum, bNum, field.NumberOptions)
		}
	}

	// Try time comparison
	if aTime, aOk := s.toTime(a); aOk {
		if bTime, bOk := s.toTime(b); bOk {
			return s.compareTimes(aTime, bTime)
		}
	}

	// Try bool comparison
	if aBool, aOk := s.toBool(a); aOk {
		if bBool, bOk := s.toBool(b); bOk {
			return s.compareBools(aBool, bBool)
		}
	}

	// Fallback to string representation
	aStr := s.toStringFallback(a)
	bStr := s.toStringFallback(b)
	return s.compareStrings(aStr, bStr, field.StringOptions)
}

// String comparison with options
func (s *SortUtils) compareStrings(a, b string, options *commonpb.StringSortOptions) int {
	if options != nil && !options.CaseSensitive {
		a = strings.ToLower(a)
		b = strings.ToLower(b)
	}

	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// Number comparison with options
func (s *SortUtils) compareNumbers(a, b float64, options *commonpb.NumberSortOptions) int {
	if options != nil && options.AbsoluteValue {
		if a < 0 {
			a = -a
		}
		if b < 0 {
			b = -b
		}
	}

	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// Time comparison
func (s *SortUtils) compareTimes(a, b time.Time) int {
	if a.Before(b) {
		return -1
	} else if a.After(b) {
		return 1
	}
	return 0
}

// Bool comparison
func (s *SortUtils) compareBools(a, b bool) int {
	if a == b {
		return 0
	}
	if a && !b {
		return 1
	}
	return -1
}

// Type conversion helpers with validation

func (s *SortUtils) toString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case *string:
		if v != nil {
			return *v, true
		}
	}
	return "", false
}

func (s *SortUtils) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case *float64:
		if v != nil {
			return *v, true
		}
	case *float32:
		if v != nil {
			return float64(*v), true
		}
	case *int:
		if v != nil {
			return float64(*v), true
		}
	case *int32:
		if v != nil {
			return float64(*v), true
		}
	case *int64:
		if v != nil {
			return float64(*v), true
		}
	}
	return 0, false
}

func (s *SortUtils) toTime(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case *time.Time:
		if v != nil {
			return *v, true
		}
	case int64:
		return time.UnixMilli(v), true
	case *int64:
		if v != nil {
			return time.UnixMilli(*v), true
		}
	}
	return time.Time{}, false
}

func (s *SortUtils) toBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case *bool:
		if v != nil {
			return *v, true
		}
	}
	return false, false
}

func (s *SortUtils) toStringFallback(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.ToLower(reflect.TypeOf(value).String())
}
