// Package gradecompute holds the pure (DB-free, side-effect-free) algorithmic
// core of the education job-grading roll-up: scoring-scheme resolution,
// within-criterion aggregation, composite roll-up, and score_scale band
// transmutation. Everything here is unit-testable in isolation; the use-case
// layer wires these functions to repositories and persists the results onto
// phase_outcome_summary / job_outcome_summary / job_outcome_line.
//
// Generic-first: the transform is the IB-MYP lead worked example (best-fit MAX
// within a criterion -> SUM of criteria into a /32 composite -> transmute to a
// 1-7 grade band) but every primitive is vertical-neutral (clinical outcome
// scoring, QC dispositions, asset-condition ratings all reuse it).
package gradecompute

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	scorescalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	scorescalebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

// ErrNoScheme is returned by ResolveScoringScheme when every rung of the
// precedence ladder is nil/empty — a gradable phase MUST resolve to a scheme.
var ErrNoScheme = errors.New("gradecompute: no scoring scheme resolved on the precedence ladder (all rungs nil/empty)")

// ResolveScoringScheme walks the most-specific-first precedence ladder and
// returns the first non-empty scheme id (NULL/empty = inherit -> fall through to
// the next rung). It fails loud (ErrNoScheme) when nothing resolves.
//
// The proto currently carries scoring_scheme_id only on job_phase and
// job_template_phase, so callers pass those two pointers most-specific-first:
//
//	ResolveScoringScheme(jobPhase.ScoringSchemeId, jobTemplatePhase.ScoringSchemeId)
//
// The full 5-rung ladder (… > job > job_template > evaluation_template) is the
// design intent; add those anchors to proto to extend the ladder — this function
// already accepts an arbitrary-length ladder, so no signature change is needed.
func ResolveScoringScheme(ladder ...*string) (string, error) {
	for _, rung := range ladder {
		if rung != nil && *rung != "" {
			return *rung, nil
		}
	}
	return "", ErrNoScheme
}

// MaxWithinCriterion is the within-criterion best-fit aggregator
// (AGGREGATION_METHOD_MAXIMUM): the highest observed value across the assessments
// that target one criterion. ok is false when there are no values (a criterion
// with zero recorded outcomes contributes nothing — the caller decides whether
// that is a zero or a suppression).
func MaxWithinCriterion(values []float64) (max float64, ok bool) {
	for i, v := range values {
		if i == 0 || v > max {
			max = v
		}
	}
	return max, len(values) > 0
}

// SumComposite is the composite roll-up for composite_method = SCORING_METHOD_SUM:
// the criteria best-fit values summed into the scheme total (the IB-MYP /32).
func SumComposite(values []float64) float64 {
	var total float64
	for _, v := range values {
		total += v
	}
	return total
}

// IsSumMethod / IsMaxAggregation are convenience guards so callers branch on the
// proto enums without importing the enums package directly.
func IsSumMethod(m enumspb.ScoringMethod) bool {
	return m == enumspb.ScoringMethod_SCORING_METHOD_SUM
}

func IsMaxAggregation(a enumspb.AggregationMethod) bool {
	return a == enumspb.AggregationMethod_AGGREGATION_METHOD_MAXIMUM
}

// CriterionInput is the per-criterion bundle the roll-up consumes: every numeric
// task_outcome value recorded against one criterion (across the phase's
// assessments). Empty Values = the criterion had no recorded outcome.
type CriterionInput struct {
	CriterionID string
	Values      []float64
}

// RollUp is the result of RollUpCriteria: the per-criterion best-fit values, the
// summed composite (the IB-MYP /32), and the count of criteria that contributed.
type RollUp struct {
	PerCriterion map[string]float64 // criterion_id -> best-fit (MAX) value
	Composite    float64            // SUM of the per-criterion best-fits
	Contributing int                // criteria that had at least one value
}

// RollUpCriteria runs the within-criterion MAX (AGGREGATION_METHOD_MAXIMUM) then
// the SUM composite (SCORING_METHOD_SUM): the canonical IB-MYP transform. A
// criterion with no values contributes nothing (not a zero) — the §8 all-zero
// suppression / scaffold-synthesis policy is the caller's, not the math's.
func RollUpCriteria(inputs []CriterionInput) RollUp {
	per := make(map[string]float64, len(inputs))
	for _, in := range inputs {
		if m, ok := MaxWithinCriterion(in.Values); ok {
			per[in.CriterionID] = m
		}
	}
	var composite float64
	for _, v := range per {
		composite += v
	}
	return RollUp{PerCriterion: per, Composite: composite, Contributing: len(per)}
}

// Transmute maps a raw composite score to its score_scale band per the scale's
// kind. RANGE_MAP uses half-open intervals [input_min, input_max) (a nil bound
// is unbounded: nil min = -inf, nil max = +inf — author the top band with a nil
// max so a perfect score still lands). EXACT_MAP matches input_match parsed as a
// number equal to raw. UNSPECIFIED defaults to RANGE_MAP. Returns an error when
// no band matches (fail-loud — a gap in the scale is a config bug, not a 0).
func Transmute(scale *scorescalepb.ScoreScale, bands []*scorescalebandpb.ScoreScaleBand, raw float64) (*scorescalebandpb.ScoreScaleBand, error) {
	if scale == nil {
		return nil, errors.New("gradecompute: nil score_scale")
	}
	switch scale.ScaleKind {
	case enumspb.ScaleKind_SCALE_KIND_EXACT_MAP:
		return transmuteExact(bands, raw)
	case enumspb.ScaleKind_SCALE_KIND_RANGE_MAP, enumspb.ScaleKind_SCALE_KIND_UNSPECIFIED:
		return transmuteRange(bands, raw)
	default:
		return nil, fmt.Errorf("gradecompute: unsupported scale_kind %v", scale.ScaleKind)
	}
}

// transmuteRange returns the first active band whose half-open interval contains
// raw. Bands are expected non-overlapping; the first containing band wins.
func transmuteRange(bands []*scorescalebandpb.ScoreScaleBand, raw float64) (*scorescalebandpb.ScoreScaleBand, error) {
	for _, b := range bands {
		if b == nil || !b.Active {
			continue
		}
		lo := math.Inf(-1)
		if b.InputMin != nil {
			lo = *b.InputMin
		}
		hi := math.Inf(1)
		if b.InputMax != nil {
			hi = *b.InputMax
		}
		if raw >= lo && raw < hi {
			return b, nil
		}
	}
	return nil, fmt.Errorf("gradecompute: no RANGE_MAP band contains raw=%v", raw)
}

// transmuteExact returns the active band whose input_match parses to exactly raw.
func transmuteExact(bands []*scorescalebandpb.ScoreScaleBand, raw float64) (*scorescalebandpb.ScoreScaleBand, error) {
	for _, b := range bands {
		if b == nil || !b.Active || b.InputMatch == nil {
			continue
		}
		if f, err := strconv.ParseFloat(*b.InputMatch, 64); err == nil && f == raw {
			return b, nil
		}
	}
	return nil, fmt.Errorf("gradecompute: no EXACT_MAP band matches raw=%v", raw)
}

// TransmuteExactKey is the string-keyed EXACT_MAP variant for discrete code
// scales (e.g. a disposition/letter code -> label) where the input is not numeric.
func TransmuteExactKey(bands []*scorescalebandpb.ScoreScaleBand, key string) (*scorescalebandpb.ScoreScaleBand, error) {
	for _, b := range bands {
		if b == nil || !b.Active || b.InputMatch == nil {
			continue
		}
		if *b.InputMatch == key {
			return b, nil
		}
	}
	return nil, fmt.Errorf("gradecompute: no EXACT_MAP band matches key=%q", key)
}

// BandOutput is the (scaled_score, scaled_label) pair a transmuted band yields,
// ready to stamp onto a summary. score is 0 when the band has no numeric output
// (a label-only band).
func BandOutput(b *scorescalebandpb.ScoreScaleBand) (score float64, label string) {
	if b == nil {
		return 0, ""
	}
	if b.OutputValue != nil {
		score = *b.OutputValue
	}
	return score, b.OutputLabel
}
