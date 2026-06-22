package gradecompute

import (
	"testing"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

func fp(v float64) *float64 { return &v }
func sp(v string) *string   { return &v }

func TestResolveScoringScheme(t *testing.T) {
	cases := []struct {
		name   string
		ladder []*string
		want   string
		wantOK bool
	}{
		{"most-specific wins", []*string{sp("phase-scheme"), sp("template-scheme")}, "phase-scheme", true},
		{"inherit past nil", []*string{nil, sp("template-scheme")}, "template-scheme", true},
		{"inherit past empty", []*string{sp(""), sp("template-scheme")}, "template-scheme", true},
		{"all nil -> error", []*string{nil, nil}, "", false},
		{"all empty -> error", []*string{sp(""), sp("")}, "", false},
		{"single rung", []*string{sp("only")}, "only", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveScoringScheme(c.ladder...)
			if c.wantOK && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !c.wantOK && err == nil {
				t.Fatalf("expected ErrNoScheme, got %q", got)
			}
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestMaxWithinCriterion(t *testing.T) {
	if m, ok := MaxWithinCriterion([]float64{3, 7, 5, 7, 2}); !ok || m != 7 {
		t.Fatalf("max got (%v,%v) want (7,true)", m, ok)
	}
	if m, ok := MaxWithinCriterion([]float64{-4, -1, -9}); !ok || m != -1 {
		t.Fatalf("negative max got (%v,%v) want (-1,true)", m, ok)
	}
	if _, ok := MaxWithinCriterion(nil); ok {
		t.Fatalf("empty should report ok=false")
	}
}

func TestSumComposite(t *testing.T) {
	// IB-MYP: four criteria best-fits summed into /32.
	if got := SumComposite([]float64{8, 7, 6, 7}); got != 28 {
		t.Fatalf("sum got %v want 28", got)
	}
	if got := SumComposite(nil); got != 0 {
		t.Fatalf("empty sum got %v want 0", got)
	}
}

func TestEnumGuards(t *testing.T) {
	if !IsSumMethod(enumspb.ScoringMethod_SCORING_METHOD_SUM) {
		t.Fatal("SUM guard failed")
	}
	if !IsMaxAggregation(enumspb.AggregationMethod_AGGREGATION_METHOD_MAXIMUM) {
		t.Fatal("MAX guard failed")
	}
	if IsSumMethod(enumspb.ScoringMethod_SCORING_METHOD_UNSPECIFIED) {
		t.Fatal("UNSPECIFIED should not be SUM")
	}
}

// ibMypBands models the IB-MYP /32 -> 1..7 transmutation. The top band has a nil
// InputMax (+inf) so a perfect 32 still lands on grade 7.
func ibMypBands() []*scorescalebandpb.ScoreScaleBand {
	return []*scorescalebandpb.ScoreScaleBand{
		{Active: true, InputMin: fp(0), InputMax: fp(6), OutputValue: fp(1), OutputLabel: "1"},
		{Active: true, InputMin: fp(6), InputMax: fp(10), OutputValue: fp(2), OutputLabel: "2"},
		{Active: true, InputMin: fp(10), InputMax: fp(14), OutputValue: fp(3), OutputLabel: "3"},
		{Active: true, InputMin: fp(14), InputMax: fp(19), OutputValue: fp(4), OutputLabel: "4"},
		{Active: true, InputMin: fp(19), InputMax: fp(24), OutputValue: fp(5), OutputLabel: "5"},
		{Active: true, InputMin: fp(24), InputMax: fp(28), OutputValue: fp(6), OutputLabel: "6"},
		{Active: true, InputMin: fp(28), InputMax: nil, OutputValue: fp(7), OutputLabel: "7"},
	}
}

func TestTransmuteRangeMap(t *testing.T) {
	scale := &scorescalepb.ScoreScale{ScaleKind: enumspb.ScaleKind_SCALE_KIND_RANGE_MAP}
	bands := ibMypBands()
	cases := []struct {
		raw   float64
		label string
	}{
		{0, "1"},
		{5.999, "1"},
		{6, "2"}, // half-open boundary: 6 belongs to the NEXT band
		{23, "5"},
		{27.5, "6"},
		{28, "7"}, // lower edge of top band
		{32, "7"}, // perfect score lands via nil (unbounded) max
		{100, "7"},
	}
	for _, c := range cases {
		b, err := Transmute(scale, bands, c.raw)
		if err != nil {
			t.Fatalf("raw=%v unexpected error: %v", c.raw, err)
		}
		if b.OutputLabel != c.label {
			t.Fatalf("raw=%v got label %q want %q", c.raw, b.OutputLabel, c.label)
		}
		score, label := BandOutput(b)
		if label != c.label || score == 0 && c.label != "" {
			t.Fatalf("raw=%v BandOutput got (%v,%q)", c.raw, score, label)
		}
	}
	// A gap below the scale is a fail-loud config error, not a silent 0.
	if _, err := Transmute(scale, bands, -1); err == nil {
		t.Fatal("expected error for raw below all bands")
	}
}

func TestTransmuteExactMap(t *testing.T) {
	scale := &scorescalepb.ScoreScale{ScaleKind: enumspb.ScaleKind_SCALE_KIND_EXACT_MAP}
	bands := []*scorescalebandpb.ScoreScaleBand{
		{Active: true, InputMatch: sp("0"), OutputLabel: "fail"},
		{Active: true, InputMatch: sp("1"), OutputLabel: "pass"},
	}
	b, err := Transmute(scale, bands, 1)
	if err != nil || b.OutputLabel != "pass" {
		t.Fatalf("exact numeric got (%v,%v) want pass", b, err)
	}
	if _, err := Transmute(scale, bands, 9); err == nil {
		t.Fatal("expected no-match error")
	}
	// string-keyed variant
	codeBands := []*scorescalebandpb.ScoreScaleBand{
		{Active: true, InputMatch: sp("A"), OutputLabel: "excellent"},
		{Active: true, InputMatch: sp("F"), OutputLabel: "fail"},
	}
	cb, err := TransmuteExactKey(codeBands, "A")
	if err != nil || cb.OutputLabel != "excellent" {
		t.Fatalf("exact key got (%v,%v) want excellent", cb, err)
	}
}

func TestRollUpCriteria(t *testing.T) {
	// IB-MYP: four criteria, each with multiple assessments; MAX within, SUM across.
	inputs := []CriterionInput{
		{CriterionID: "A", Values: []float64{6, 8, 7}}, // MAX 8
		{CriterionID: "B", Values: []float64{7, 5}},    // MAX 7
		{CriterionID: "C", Values: []float64{6}},       // MAX 6
		{CriterionID: "D", Values: []float64{7, 7, 4}}, // MAX 7
	}
	r := RollUpCriteria(inputs)
	if r.Composite != 28 {
		t.Fatalf("composite got %v want 28", r.Composite)
	}
	if r.Contributing != 4 {
		t.Fatalf("contributing got %d want 4", r.Contributing)
	}
	if r.PerCriterion["A"] != 8 || r.PerCriterion["D"] != 7 {
		t.Fatalf("per-criterion wrong: %v", r.PerCriterion)
	}

	// End-to-end: roll-up composite then transmute to the IB-MYP grade band.
	scale := &scorescalepb.ScoreScale{ScaleKind: enumspb.ScaleKind_SCALE_KIND_RANGE_MAP}
	band, err := Transmute(scale, ibMypBands(), r.Composite)
	if err != nil || band.OutputLabel != "7" {
		t.Fatalf("28 should transmute to grade 7, got (%v,%v)", band, err)
	}

	// A criterion with no recorded values does not contribute (not a zero).
	sparse := RollUpCriteria([]CriterionInput{
		{CriterionID: "A", Values: []float64{5}},
		{CriterionID: "B", Values: nil},
	})
	if sparse.Contributing != 1 || sparse.Composite != 5 {
		t.Fatalf("sparse roll-up got contributing=%d composite=%v want 1/5", sparse.Contributing, sparse.Composite)
	}
}

func TestTransmuteSkipsInactiveBands(t *testing.T) {
	scale := &scorescalepb.ScoreScale{ScaleKind: enumspb.ScaleKind_SCALE_KIND_RANGE_MAP}
	bands := []*scorescalebandpb.ScoreScaleBand{
		{Active: false, InputMin: fp(0), InputMax: fp(10), OutputLabel: "stale"},
		{Active: true, InputMin: fp(0), InputMax: fp(10), OutputLabel: "live"},
	}
	b, err := Transmute(scale, bands, 5)
	if err != nil || b.OutputLabel != "live" {
		t.Fatalf("got (%v,%v) want live", b, err)
	}
}
