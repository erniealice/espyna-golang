package subscription

import (
	"time"
)

// CalculateEndDate calculates the subscription end date based on frequency.
// This is a utility function used across subscription-related operations.
func CalculateEndDate(startDate time.Time, frequency string) time.Time {
	switch frequency {
	case "month":
		return startDate.AddDate(0, 1, 0)
	case "semi_annual":
		return startDate.AddDate(0, 6, 0)
	case "year":
		return startDate.AddDate(1, 0, 0)
	default:
		return startDate.AddDate(0, 1, 0) // Default to 1 month
	}
}
