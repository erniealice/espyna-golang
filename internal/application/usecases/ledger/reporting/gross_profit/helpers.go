package gross_profit

// CalculateMargin computes gross profit margin percentage.
// Returns 0 if netRevenue is zero to avoid division by zero.
func CalculateMargin(grossProfit, netRevenue float64) float64 {
	if netRevenue == 0 {
		return 0
	}
	return (grossProfit / netRevenue) * 100
}
