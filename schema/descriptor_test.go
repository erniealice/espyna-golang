package schema

import (
	"testing"

	eventv1 "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_recurrence"
	fulfillmentv1 "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
	operationv1 "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// colByName is a small test helper: look up a classified column by name.
func colByName(cols []ColumnInfo, name string) (ColumnInfo, bool) {
	for _, c := range cols {
		if c.Name == name {
			return c, true
		}
	}
	return ColumnInfo{}, false
}

func names(cols []ColumnInfo) map[string]bool {
	m := make(map[string]bool, len(cols))
	for _, c := range cols {
		m[c.Name] = true
	}
	return m
}

// TestClassifyFulfillment exercises all four column-kind edge cases in a single
// message: int64 audit millis, google.protobuf.Timestamp business timestamps,
// google.protobuf.Struct -> jsonb metadata, and the bool active flag. Fulfillment
// was authored under the new convention with zero *_string mirrors, so it is the
// cleanest anchor.
func TestClassifyFulfillment(t *testing.T) {
	md := (&fulfillmentv1.Fulfillment{}).ProtoReflect().Descriptor()
	cols := Classify(md)

	// Scalar/enum columns present.
	for _, want := range []string{"id", "workspace_id", "revenue_id", "delivery_mode", "status", "delivery_cost", "currency", "notes", "created_by"} {
		if _, ok := colByName(cols, want); !ok {
			t.Errorf("expected scalar column %q to be classified, got columns %v", want, names(cols))
		}
	}

	// google.protobuf.Struct metadata -> jsonb column.
	if c, ok := colByName(cols, "metadata"); !ok {
		t.Errorf("metadata (Struct) must be a jsonb column, missing")
	} else if !c.IsMetadata {
		t.Errorf("metadata.IsMetadata = false, want true")
	} else if c.IsTimestamp || c.IsBigintMillis {
		t.Errorf("metadata wrongly flagged timestamp/bigint: %+v", c)
	}

	// google.protobuf.Timestamp -> IsTimestamp column (not a relation skip).
	for _, want := range []string{"scheduled_at", "delivered_at"} {
		c, ok := colByName(cols, want)
		if !ok {
			t.Errorf("Timestamp column %q missing", want)
			continue
		}
		if !c.IsTimestamp {
			t.Errorf("%s.IsTimestamp = false, want true", want)
		}
		if c.IsBigintMillis {
			t.Errorf("%s wrongly flagged bigint-millis", want)
		}
	}

	// int64 audit columns -> bigint-millis.
	for _, want := range []string{"date_created", "date_modified"} {
		c, ok := colByName(cols, want)
		if !ok {
			t.Errorf("audit column %q missing", want)
			continue
		}
		if !c.IsBigintMillis {
			t.Errorf("%s.IsBigintMillis = false, want true", want)
		}
		if c.IsTimestamp {
			t.Errorf("%s wrongly flagged Timestamp", want)
		}
		if c.ProtoKind != protoreflect.Int64Kind {
			t.Errorf("%s.ProtoKind = %v, want Int64Kind", want, c.ProtoKind)
		}
	}

	// bool active flag.
	if c, ok := colByName(cols, "active"); !ok {
		t.Errorf("active column missing")
	} else if !c.IsActive {
		t.Errorf("active.IsActive = false, want true")
	} else if c.ProtoKind != protoreflect.BoolKind {
		t.Errorf("active.ProtoKind = %v, want BoolKind", c.ProtoKind)
	}
}

// TestClassifyJobRelationsAndIgnore verifies the two skip paths: nested-message
// relations (client, location) are dropped, repeated scalar (predecessor_job_ids)
// is dropped, and (db).ignore *_string mirrors are dropped — while the underlying
// numeric/audit columns survive.
func TestClassifyJobRelationsAndIgnore(t *testing.T) {
	md := (&operationv1.Job{}).ProtoReflect().Descriptor()
	cols := Classify(md)
	have := names(cols)

	// Nested-message relations must be skipped.
	for _, skip := range []string{"client", "location"} {
		if have[skip] {
			t.Errorf("nested-message relation %q must be skipped, but was classified", skip)
		}
	}

	// Repeated scalar must be skipped.
	if have["predecessor_job_ids"] {
		t.Errorf("repeated scalar predecessor_job_ids must be skipped")
	}

	// (db).ignore *_string mirrors must be skipped.
	for _, mirror := range []string{
		"date_created_string", "date_modified_string", "due_date_string",
		"planned_start_string", "actual_end_string",
	} {
		if have[mirror] {
			t.Errorf("(db).ignore mirror %q must be skipped", mirror)
		}
	}

	// FK scalar columns and the underlying numeric columns survive.
	for _, keep := range []string{"id", "name", "client_id", "location_id", "due_date", "planned_start", "actual_end", "priority", "parent_job_id"} {
		if !have[keep] {
			t.Errorf("column %q must be classified, got %v", keep, have)
		}
	}

	// The audit numerics survive AND are flagged bigint-millis even though their
	// *_string mirrors are ignored.
	if c, ok := colByName(cols, "date_created"); !ok || !c.IsBigintMillis {
		t.Errorf("date_created must survive as bigint-millis, got %+v ok=%v", c, ok)
	}

	// A non-audit int64 date (due_date) is a plain bigint column, NOT bigint-millis.
	if c, ok := colByName(cols, "due_date"); !ok {
		t.Errorf("due_date column missing")
	} else if c.IsBigintMillis {
		t.Errorf("due_date must NOT be flagged bigint-millis (only date_created/date_modified are auto-stamped)")
	}
}

// TestClassifyEventRecurrenceStringTrap is the critical regression: rrule_string
// (field 4) and exdate_string (field 17) are REAL stored columns with NO
// (db).ignore annotation. A naive suffix-based ignore rule would drop them and
// destroy data. Classify must preserve them (annotation-only ignore) while still
// dropping the genuinely-annotated date_created_string / date_modified_string.
func TestClassifyEventRecurrenceStringTrap(t *testing.T) {
	md := (&eventv1.EventRecurrence{}).ProtoReflect().Descriptor()
	cols := Classify(md)
	have := names(cols)

	for _, keep := range []string{"rrule_string", "exdate_string"} {
		if !have[keep] {
			t.Errorf("REAL stored column %q was dropped — ignore must be annotation-only, NOT suffix-based", keep)
		}
	}

	for _, drop := range []string{"date_created_string", "date_modified_string"} {
		if have[drop] {
			t.Errorf("annotated (db).ignore mirror %q must be dropped", drop)
		}
	}
}
