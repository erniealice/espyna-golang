package infrastructure

import "strconv"

// Helper functions for type-safe extraction from map[string]any
// These are used by config adapters (DatabaseConfigAdapter, AuthConfigAdapter, StorageConfigAdapter)

func getString(m map[string]any, key string, defaultValue string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getBool(m map[string]any, key string, defaultValue bool) bool {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "1" || v == "yes"
		case int:
			return v != 0
		}
	}
	return defaultValue
}

func getInt32(m map[string]any, key string, defaultValue int32) int32 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return int32(v)
		case int32:
			return v
		case int64:
			return int32(v)
		case float64:
			return int32(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 32); err == nil {
				return int32(i)
			}
		}
	}
	return defaultValue
}

func getStringSlice(m map[string]any, key string) []string {
	if val, ok := m[key]; ok {
		if slice, ok := val.([]string); ok {
			return slice
		}
		// Try to convert []interface{} to []string
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

func getStringMap(m map[string]any, key string) map[string]string {
	if val, ok := m[key]; ok {
		if strMap, ok := val.(map[string]string); ok {
			return strMap
		}
		// Try to convert map[string]interface{} to map[string]string
		if anyMap, ok := val.(map[string]interface{}); ok {
			result := make(map[string]string, len(anyMap))
			for k, v := range anyMap {
				if str, ok := v.(string); ok {
					result[k] = str
				}
			}
			return result
		}
	}
	return map[string]string{}
}
