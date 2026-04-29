//go:build postgresql

package entity

import (
	"fmt"
	"time"
)

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey, _ string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}
