package listdata

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// FilterUtils provides utilities for evaluating filters against data
type FilterUtils struct{}

// NewFilterUtils creates a new filter utility instance
func NewFilterUtils() *FilterUtils {
	return &FilterUtils{}
}

// EvaluateFilters evaluates all filters against a single item
func (f *FilterUtils) EvaluateFilters(item interface{}, filters *commonpb.FilterRequest) bool {
	if filters == nil || len(filters.Filters) == 0 {
		return true // No filters means include all
	}

	results := make([]bool, len(filters.Filters))
	for i, filter := range filters.Filters {
		results[i] = f.evaluateTypedFilter(item, filter)
	}

	// Apply logic (AND/OR)
	if filters.Logic == commonpb.FilterLogic_OR {
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	} else { // AND logic (default)
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	}
}

// evaluateTypedFilter evaluates a single typed filter against an item
func (f *FilterUtils) evaluateTypedFilter(item interface{}, filter *commonpb.TypedFilter) bool {
	fieldValue := f.getFieldValue(item, filter.Field)

	switch filterType := filter.FilterType.(type) {
	case *commonpb.TypedFilter_StringFilter:
		return f.evaluateStringFilter(fieldValue, filterType.StringFilter)
	case *commonpb.TypedFilter_NumberFilter:
		return f.evaluateNumberFilter(fieldValue, filterType.NumberFilter)
	case *commonpb.TypedFilter_DateFilter:
		return f.evaluateDateFilter(fieldValue, filterType.DateFilter)
	case *commonpb.TypedFilter_ListFilter:
		return f.evaluateListFilter(fieldValue, filterType.ListFilter)
	case *commonpb.TypedFilter_RangeFilter:
		return f.evaluateRangeFilter(fieldValue, filterType.RangeFilter)
	case *commonpb.TypedFilter_BooleanFilter:
		return f.evaluateBooleanFilter(fieldValue, filterType.BooleanFilter)
	default:
		return true // Unknown filter type, include by default
	}
}

// getFieldValue extracts a field value from an item using dot notation
func (f *FilterUtils) getFieldValue(item interface{}, fieldPath string) interface{} {
	if item == nil {
		return nil
	}

	// Handle pointer types
	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Split field path by dots
	parts := strings.Split(fieldPath, ".")

	for _, part := range parts {
		if val.Kind() == reflect.Struct {
			// Convert snake_case to CamelCase for Go struct fields
			fieldName := f.toCamelCase(part)
			field := val.FieldByName(fieldName)
			if !field.IsValid() {
				return nil
			}

			// Handle pointer fields
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
func (f *FilterUtils) toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	// Capitalize first letter
	if len(result) > 0 {
		result = strings.ToUpper(result[:1]) + result[1:]
	}
	return result
}

// evaluateStringFilter evaluates string filter operations
func (f *FilterUtils) evaluateStringFilter(value interface{}, filter *commonpb.StringFilter) bool {
	strValue := f.toString(value)
	filterValue := filter.Value

	if !filter.CaseSensitive {
		strValue = strings.ToLower(strValue)
		filterValue = strings.ToLower(filterValue)
	}

	switch filter.Operator {
	case commonpb.StringOperator_STRING_EQUALS:
		return strValue == filterValue
	case commonpb.StringOperator_STRING_NOT_EQUALS:
		return strValue != filterValue
	case commonpb.StringOperator_STRING_CONTAINS:
		return strings.Contains(strValue, filterValue)
	case commonpb.StringOperator_STRING_STARTS_WITH:
		return strings.HasPrefix(strValue, filterValue)
	case commonpb.StringOperator_STRING_ENDS_WITH:
		return strings.HasSuffix(strValue, filterValue)
	case commonpb.StringOperator_STRING_REGEX:
		if regex, err := regexp.Compile(filterValue); err == nil {
			return regex.MatchString(strValue)
		}
		return false
	default:
		return false
	}
}

// evaluateNumberFilter evaluates number filter operations
func (f *FilterUtils) evaluateNumberFilter(value interface{}, filter *commonpb.NumberFilter) bool {
	numValue := f.toFloat64(value)
	if numValue == nil {
		return false
	}

	switch filter.Operator {
	case commonpb.NumberOperator_NUMBER_EQUALS:
		return *numValue == filter.Value
	case commonpb.NumberOperator_NUMBER_NOT_EQUALS:
		return *numValue != filter.Value
	case commonpb.NumberOperator_NUMBER_GREATER_THAN:
		return *numValue > filter.Value
	case commonpb.NumberOperator_NUMBER_GREATER_THAN_OR_EQUAL:
		return *numValue >= filter.Value
	case commonpb.NumberOperator_NUMBER_LESS_THAN:
		return *numValue < filter.Value
	case commonpb.NumberOperator_NUMBER_LESS_THAN_OR_EQUAL:
		return *numValue <= filter.Value
	default:
		return false
	}
}

// evaluateDateFilter evaluates date filter operations
func (f *FilterUtils) evaluateDateFilter(value interface{}, filter *commonpb.DateFilter) bool {
	dateValue := f.toTime(value)
	if dateValue == nil {
		return false
	}

	filterDate, err := time.Parse(time.RFC3339, filter.Value)
	if err != nil {
		return false
	}

	switch filter.Operator {
	case commonpb.DateOperator_DATE_EQUALS:
		return dateValue.Format("2006-01-02") == filterDate.Format("2006-01-02")
	case commonpb.DateOperator_DATE_BEFORE:
		return dateValue.Before(filterDate)
	case commonpb.DateOperator_DATE_AFTER:
		return dateValue.After(filterDate)
	case commonpb.DateOperator_DATE_BETWEEN:
		if filter.RangeEnd == nil {
			return false
		}
		endDate, err := time.Parse(time.RFC3339, *filter.RangeEnd)
		if err != nil {
			return false
		}
		return (dateValue.After(filterDate) || dateValue.Equal(filterDate)) &&
			(dateValue.Before(endDate) || dateValue.Equal(endDate))
	default:
		return false
	}
}

// evaluateListFilter evaluates list filter operations (IN/NOT_IN)
func (f *FilterUtils) evaluateListFilter(value interface{}, filter *commonpb.ListFilter) bool {
	strValue := f.toString(value)

	contains := false
	for _, filterValue := range filter.Values {
		if strValue == filterValue {
			contains = true
			break
		}
	}

	switch filter.Operator {
	case commonpb.ListOperator_LIST_IN:
		return contains
	case commonpb.ListOperator_LIST_NOT_IN:
		return !contains
	default:
		return false
	}
}

// evaluateRangeFilter evaluates range filter operations
func (f *FilterUtils) evaluateRangeFilter(value interface{}, filter *commonpb.RangeFilter) bool {
	numValue := f.toFloat64(value)
	if numValue == nil {
		return false
	}

	minCheck := true
	maxCheck := true

	if filter.IncludeMin {
		minCheck = *numValue >= filter.Min
	} else {
		minCheck = *numValue > filter.Min
	}

	if filter.IncludeMax {
		maxCheck = *numValue <= filter.Max
	} else {
		maxCheck = *numValue < filter.Max
	}

	return minCheck && maxCheck
}

// evaluateBooleanFilter evaluates boolean filter operations
func (f *FilterUtils) evaluateBooleanFilter(value interface{}, filter *commonpb.BooleanFilter) bool {
	boolValue := f.toBool(value)
	return boolValue == filter.Value
}

// Helper functions for type conversion

func (f *FilterUtils) toString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case *string:
		if v == nil {
			return ""
		}
		return *v
	default:
		return fmt.Sprintf("%v", value)
	}
}

func (f *FilterUtils) toFloat64(value interface{}) *float64 {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case float64:
		return &v
	case float32:
		f64 := float64(v)
		return &f64
	case int:
		f64 := float64(v)
		return &f64
	case int32:
		f64 := float64(v)
		return &f64
	case int64:
		f64 := float64(v)
		return &f64
	case *float64:
		return v
	case *float32:
		if v == nil {
			return nil
		}
		f64 := float64(*v)
		return &f64
	case *int:
		if v == nil {
			return nil
		}
		f64 := float64(*v)
		return &f64
	case *int32:
		if v == nil {
			return nil
		}
		f64 := float64(*v)
		return &f64
	case *int64:
		if v == nil {
			return nil
		}
		f64 := float64(*v)
		return &f64
	case string:
		if f64, err := strconv.ParseFloat(v, 64); err == nil {
			return &f64
		}
	case *string:
		if v != nil {
			if f64, err := strconv.ParseFloat(*v, 64); err == nil {
				return &f64
			}
		}
	}
	return nil
}

func (f *FilterUtils) toTime(value interface{}) *time.Time {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case int64:
		// Assume Unix timestamp in milliseconds
		t := time.UnixMilli(v)
		return &t
	case *int64:
		if v == nil {
			return nil
		}
		t := time.UnixMilli(*v)
		return &t
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return &t
		}
	case *string:
		if v != nil {
			if t, err := time.Parse(time.RFC3339, *v); err == nil {
				return &t
			}
		}
	}
	return nil
}

func (f *FilterUtils) toBool(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case *bool:
		if v == nil {
			return false
		}
		return *v
	case string:
		return v == "true" || v == "1" || v == "yes"
	case *string:
		if v == nil {
			return false
		}
		return *v == "true" || *v == "1" || *v == "yes"
	case int:
		return v != 0
	case *int:
		if v == nil {
			return false
		}
		return *v != 0
	default:
		return false
	}
}
