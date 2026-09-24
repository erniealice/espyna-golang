package gradecompute

import (
	"testing"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
)

func TestRoundWhole(t *testing.T) {
	cases := []struct {
		in   float64
		mode enumspb.RoundingMode
		want float64
	}{
		{88.5, enumspb.RoundingMode_ROUNDING_MODE_HALF_UP, 89},
		{88.49, enumspb.RoundingMode_ROUNDING_MODE_HALF_UP, 88},
		{88.5, enumspb.RoundingMode_ROUNDING_MODE_HALF_DOWN, 88},
		{88.51, enumspb.RoundingMode_ROUNDING_MODE_HALF_DOWN, 89},
		{88.5, enumspb.RoundingMode_ROUNDING_MODE_HALF_EVEN, 88},
		{89.5, enumspb.RoundingMode_ROUNDING_MODE_HALF_EVEN, 90},
		{88.5, enumspb.RoundingMode_ROUNDING_MODE_UNSPECIFIED, 88.5},
	}
	for _, c := range cases {
		if got := RoundWhole(c.in, c.mode); got != c.want {
			t.Errorf("RoundWhole(%v, %v) = %v, want %v", c.in, c.mode, got, c.want)
		}
	}
}

// Owner 2026-09-24: deportment = plain average of the RECORDED activities,
// rounded to a whole number with the scheme's rounding mode.
func TestRollUpCriteriaRounded_AverageRecordedOnly(t *testing.T) {
	avg := enumspb.AggregationMethod_AGGREGATION_METHOD_AVERAGE
	got := RollUpCriteriaRounded([]CriterionInput{
		{CriterionID: "rating", Values: []float64{90, 85, 88}, Aggregation: avg}, // 87.67 -> 88
	}, enumspb.RoundingMode_ROUNDING_MODE_HALF_UP)
	if got.PerCriterion["rating"] != 88 || got.Composite != 88 || got.Contributing != 1 {
		t.Fatalf("rollup = %+v, want rating 88 / composite 88 / 1 contributing", got)
	}
	none := RollUpCriteriaRounded([]CriterionInput{{CriterionID: "rating", Aggregation: avg}}, enumspb.RoundingMode_ROUNDING_MODE_HALF_UP)
	if none.Contributing != 0 || len(none.PerCriterion) != 0 {
		t.Fatalf("no recorded values must not contribute: %+v", none)
	}
}

// Existing MAX/SUM schemes are untouched by the rounding mode.
func TestRollUpCriteriaRounded_MaxUnchanged(t *testing.T) {
	inputs := []CriterionInput{
		{CriterionID: "a", Values: []float64{5, 7.5}},
		{CriterionID: "b", Values: []float64{6}, Aggregation: enumspb.AggregationMethod_AGGREGATION_METHOD_MAXIMUM},
	}
	got := RollUpCriteriaRounded(inputs, enumspb.RoundingMode_ROUNDING_MODE_HALF_UP)
	want := RollUpCriteria(inputs)
	if got.Composite != 13.5 || want.Composite != 13.5 || got.PerCriterion["a"] != 7.5 {
		t.Fatalf("MAX/SUM changed: got %+v want %+v", got, want)
	}
}
