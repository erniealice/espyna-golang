package gross_profit

// CalculateMargin computes gross profit margin percentage.
// Returns 0 if netRevenue is zero to avoid division by zero.
func CalculateMargin(grossProfit, netRevenue int64) float64 {
	if netRevenue == 0 {
		return 0
	}
	return (float64(grossProfit) / float64(netRevenue)) * 100
}
