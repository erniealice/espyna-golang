//go:build postgresql

package operation

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestScanTaskOutcomeRowWithTotal_TimestampAndNulls(t *testing.T) {
	stamp := time.Date(2026, 9, 27, 8, 9, 10, 123000000, time.UTC)
	values := []any{
		"to-1", "jt-1", "cv-1", "CRITERIA_TYPE_UNSPECIFIED", false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		stamp, nil, nil, nil, nil, int64(1), true,
		stamp, nil, int64(17),
	}
	outcome, total, err := scanTaskOutcomeWithTotal(func(dest ...any) error {
		if len(dest) != len(values) {
			return fmt.Errorf("scan destinations: got %d, want %d", len(dest), len(values))
		}
		for i, src := range values {
			switch d := dest[i].(type) {
			case sql.Scanner:
				if err := d.Scan(src); err != nil {
					return fmt.Errorf("column %d: %w", i, err)
				}
			case *string:
				*d = src.(string)
			case *bool:
				*d = src.(bool)
			case *int32:
				*d = int32(src.(int64))
			case *int64:
				*d = src.(int64)
			default:
				return fmt.Errorf("column %d: unsupported destination %T", i, d)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 17 || outcome.GetId() != "to-1" || outcome.GetRecordedDate() != stamp.UnixMilli() || outcome.GetDateCreated() != stamp.UnixMilli() {
		t.Fatalf("scan result: id=%q total=%d recorded=%d created=%d", outcome.GetId(), total, outcome.GetRecordedDate(), outcome.GetDateCreated())
	}
	if outcome.ReviewedDate != nil || outcome.DateModified != nil || outcome.GetRecordedBy() != "" || len(outcome.GetAttachmentIds()) != 0 {
		t.Fatalf("NULL fields were not preserved: %+v", outcome)
	}
}
