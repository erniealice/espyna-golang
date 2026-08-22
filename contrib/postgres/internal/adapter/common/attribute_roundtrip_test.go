//go:build postgresql

package common

// Wave-1 (grade-sheet edit mode, Q-GSE-10) round-trip proof for the additive
// typed-constraint columns on `attribute` (min_value/max_value/min_length/
// max_length/required) and the `label` column on `attribute_value`.
//
// These repositories persist entities GENERICALLY: protojson.Marshal(data) →
// json map → core.normalizeKeys (camelToSnake) → dbOps.Create/Update, and read
// back via json.Marshal(snakeMap) → protojson.Unmarshal{DiscardUnknown}. There
// is no hand-written per-column scan to test; the meaningful contract is that
// this marshal/normalize/unmarshal pipeline (a) maps every new field to its
// snake_case DB column, (b) preserves NULL-vs-zero through proto field presence,
// and (c) is permissive about constraint values (the DB CHECK — migration
// 20260717000000 — is the enforcement point). This test drives that exact
// pipeline with the same primitives the adapter uses.

import (
	"encoding/json/v2"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// persistMap mirrors the write half of the adapter: protojson → json map →
// normalizeKeys (camel→snake, exactly core.normalizeKeys). The returned map is
// keyed by the persisted snake_case column names.
func persistMap(t *testing.T, m proto.Message) map[string]any {
	t.Helper()
	jsonData, err := protojson.Marshal(m)
	if err != nil {
		t.Fatalf("protojson.Marshal: %v", err)
	}
	var camel map[string]any
	if err := json.Unmarshal(jsonData, &camel); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	snake := make(map[string]any, len(camel))
	for k, v := range camel {
		snake[postgresCore.CamelToSnake(k)] = v
	}
	return snake
}

// hydrateAttribute mirrors the read half: a snake_case column map → json →
// protojson.Unmarshal{DiscardUnknown} into a fresh Attribute.
func hydrateAttribute(t *testing.T, snake map[string]any) *commonpb.Attribute {
	t.Helper()
	resultJSON, err := json.Marshal(snake)
	if err != nil {
		t.Fatalf("json.Marshal(snake): %v", err)
	}
	out := &commonpb.Attribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, out); err != nil {
		t.Fatalf("protojson.Unmarshal: %v", err)
	}
	return out
}

func TestAttribute_ConstraintColumns_RoundTrip_AllSet(t *testing.T) {
	in := &commonpb.Attribute{
		Id:        "attr-lrn",
		Code:      "lrn",
		Name:      "LRN",
		DataType:  "free_number",
		Module:    "entity",
		Active:    true,
		MinValue:  proto.Float64(100000000000.0),
		MaxValue:  proto.Float64(999999999999.0),
		MinLength: proto.Int32(12),
		MaxLength: proto.Int32(12),
		Required:  proto.Bool(true),
	}

	snake := persistMap(t, in)

	// (a) every new field maps to its snake_case column.
	for _, col := range []string{"min_value", "max_value", "min_length", "max_length", "required"} {
		if _, ok := snake[col]; !ok {
			t.Errorf("expected column %q present in persisted map, keys=%v", col, snake)
		}
	}

	out := hydrateAttribute(t, snake)
	if out.GetMinValue() != in.GetMinValue() {
		t.Errorf("min_value: got %v want %v", out.GetMinValue(), in.GetMinValue())
	}
	if out.GetMaxValue() != in.GetMaxValue() {
		t.Errorf("max_value: got %v want %v", out.GetMaxValue(), in.GetMaxValue())
	}
	if out.GetMinLength() != in.GetMinLength() {
		t.Errorf("min_length: got %v want %v", out.GetMinLength(), in.GetMinLength())
	}
	if out.GetMaxLength() != in.GetMaxLength() {
		t.Errorf("max_length: got %v want %v", out.GetMaxLength(), in.GetMaxLength())
	}
	if out.GetRequired() != in.GetRequired() {
		t.Errorf("required: got %v want %v", out.GetRequired(), in.GetRequired())
	}
}

// NULL preservation: an unset optional field must NOT appear in the persisted
// map (so the column stays NULL) and must hydrate back to a nil pointer.
func TestAttribute_ConstraintColumns_UnsetStaysNull(t *testing.T) {
	in := &commonpb.Attribute{Id: "attr-gender", Code: "gender", Active: true}

	snake := persistMap(t, in)
	for _, col := range []string{"min_value", "max_value", "min_length", "max_length", "required"} {
		if _, ok := snake[col]; ok {
			t.Errorf("unset field %q must be absent from persisted map (stays NULL), got %v", col, snake[col])
		}
	}

	out := hydrateAttribute(t, snake)
	if out.MinValue != nil || out.MaxValue != nil || out.MinLength != nil || out.MaxLength != nil || out.Required != nil {
		t.Errorf("unset constraint fields must hydrate to nil pointers, got %+v", out)
	}
}

// Zero-vs-NULL: a zero value is distinct from unset — it is emitted (column
// stores 0) and hydrates back to a non-nil pointer to zero.
func TestAttribute_ConstraintColumns_ZeroIsNotNull(t *testing.T) {
	in := &commonpb.Attribute{
		Id:        "attr-zero",
		Active:    true,
		MinValue:  proto.Float64(0),
		MinLength: proto.Int32(0),
		Required:  proto.Bool(false),
	}

	snake := persistMap(t, in)
	for _, col := range []string{"min_value", "min_length", "required"} {
		if _, ok := snake[col]; !ok {
			t.Errorf("explicit-zero field %q must be present in persisted map (distinct from NULL)", col)
		}
	}

	out := hydrateAttribute(t, snake)
	if out.MinValue == nil || out.GetMinValue() != 0 {
		t.Errorf("min_value=0: want non-nil 0, got %v", out.MinValue)
	}
	if out.MinLength == nil || out.GetMinLength() != 0 {
		t.Errorf("min_length=0: want non-nil 0, got %v", out.MinLength)
	}
	if out.Required == nil || out.GetRequired() != false {
		t.Errorf("required=false: want non-nil false, got %v", out.Required)
	}
}

// Constraint boundaries: the proto/adapter layer is intentionally PERMISSIVE —
// both the valid boundary (min==max) and the DB-invalid combo (min>max) survive
// the marshal pipeline unchanged. Enforcement is the DB CHECK
// (ck_attribute_length_order / ck_attribute_value_order in migration
// 20260717000000) plus the W2 use-case validator, NOT this serialization layer.
func TestAttribute_ConstraintBoundaries_ProtoLayerPermissive(t *testing.T) {
	cases := []struct {
		name           string
		minLen, maxLen int32
		minVal, maxVal float64
		dbWouldReject  bool // documents intent; not asserted here (no DB)
	}{
		{name: "valid_equal_boundary", minLen: 5, maxLen: 5, minVal: 1, maxVal: 1, dbWouldReject: false},
		{name: "valid_ordered", minLen: 2, maxLen: 8, minVal: 0, maxVal: 100, dbWouldReject: false},
		{name: "invalid_inverted_len", minLen: 10, maxLen: 5, minVal: 0, maxVal: 1, dbWouldReject: true},
		{name: "invalid_inverted_val", minLen: 0, maxLen: 1, minVal: 100, maxVal: 1, dbWouldReject: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &commonpb.Attribute{
				Id:        "attr-" + tc.name,
				Active:    true,
				MinLength: proto.Int32(tc.minLen),
				MaxLength: proto.Int32(tc.maxLen),
				MinValue:  proto.Float64(tc.minVal),
				MaxValue:  proto.Float64(tc.maxVal),
			}
			out := hydrateAttribute(t, persistMap(t, in))
			if out.GetMinLength() != tc.minLen || out.GetMaxLength() != tc.maxLen ||
				out.GetMinValue() != tc.minVal || out.GetMaxValue() != tc.maxVal {
				t.Errorf("boundary values must survive the proto layer verbatim: got minLen=%d maxLen=%d minVal=%v maxVal=%v",
					out.GetMinLength(), out.GetMaxLength(), out.GetMinValue(), out.GetMaxValue())
			}
		})
	}
}

// attribute_value.label round-trips; a NULL label hydrates to unset (falls back
// to `value` at render time), distinct from an explicit "".
func TestAttributeValue_Label_RoundTrip(t *testing.T) {
	in := &commonpb.AttributeValue{
		Id:          "av-male",
		AttributeId: "attr-gender",
		Value:       "male",
		SortOrder:   1,
		Active:      true,
		Label:       proto.String("Male"),
	}

	jsonData, err := protojson.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var camel map[string]any
	if err := json.Unmarshal(jsonData, &camel); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	snake := make(map[string]any, len(camel))
	for k, v := range camel {
		snake[postgresCore.CamelToSnake(k)] = v
	}
	if _, ok := snake["label"]; !ok {
		t.Errorf("label column missing from persisted map, keys=%v", snake)
	}

	resultJSON, _ := json.Marshal(snake)
	out := &commonpb.AttributeValue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, out); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if out.GetLabel() != "Male" {
		t.Errorf("label round-trip: got %q want %q", out.GetLabel(), "Male")
	}

	// NULL label: unset stays unset (nil), not "".
	bare := &commonpb.AttributeValue{Id: "av-x", AttributeId: "attr-gender", Value: "x", Active: true}
	bareSnake := persistMapValue(t, bare)
	if _, ok := bareSnake["label"]; ok {
		t.Errorf("unset label must be absent from persisted map (stays NULL)")
	}
}

func persistMapValue(t *testing.T, m proto.Message) map[string]any {
	t.Helper()
	jsonData, err := protojson.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var camel map[string]any
	if err := json.Unmarshal(jsonData, &camel); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	snake := make(map[string]any, len(camel))
	for k, v := range camel {
		snake[postgresCore.CamelToSnake(k)] = v
	}
	return snake
}
