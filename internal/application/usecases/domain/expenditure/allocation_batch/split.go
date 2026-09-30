package allocation_batch

import (
	"math/big"
	"sort"
)

// SplitInput is one participant of the split, in stable sequence_order.
type SplitInput struct {
	SequenceOrder int32
	Numerator     int64
}

// LargestRemainderSplit splits `total` centavos over the participants proportionally to
// numerator/denominator using the largest-remainder method (build-spec §6.3, N5):
//
//   - every participant first gets floor(total*num/den);
//   - the leftover centavos go one each to the largest fractional remainders;
//   - ties are broken by ascending sequence_order (stable, replay-safe);
//   - the amounts add up to `total` exactly whenever Σ numerators == denominator.
//
// It returns amounts in the input order and ok=false when the inputs cannot produce an exact
// split (denominator <= 0, negative numerator/total, or Σ numerators != denominator). All
// arithmetic is exact integer (math/big); there are no floats.
func LargestRemainderSplit(total, denominator int64, in []SplitInput) (amounts []int64, ok bool) {
	if total < 0 || denominator <= 0 || len(in) == 0 {
		return nil, false
	}
	var sum int64
	for _, p := range in {
		if p.Numerator < 0 {
			return nil, false
		}
		sum += p.Numerator
	}
	if sum != denominator {
		return nil, false
	}
	den := big.NewInt(denominator)
	amounts = make([]int64, len(in))
	rems := make([]int64, len(in))
	var floorSum int64
	for i, p := range in {
		prod := new(big.Int).Mul(big.NewInt(total), big.NewInt(p.Numerator))
		q, r := new(big.Int).QuoRem(prod, den, new(big.Int))
		amounts[i] = q.Int64() // <= total, fits
		rems[i] = r.Int64()    // < denominator, fits
		floorSum += amounts[i]
	}
	leftover := total - floorSum // 0 <= leftover < len(in)
	idx := make([]int, len(in))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ia, ib := idx[a], idx[b]
		if rems[ia] != rems[ib] {
			return rems[ia] > rems[ib]
		}
		return in[ia].SequenceOrder < in[ib].SequenceOrder
	})
	for k := int64(0); k < leftover; k++ {
		amounts[idx[k]]++
	}
	return amounts, true
}
