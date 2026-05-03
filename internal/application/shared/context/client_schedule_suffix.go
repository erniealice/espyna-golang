package context

import "context"

// ExtractClientScheduleSuffixFromContext returns the lyngua-resolved
// `customClientPriceScheduleLabelSuffix` stuffed by the centymo handler
// before invoking the price-plan create/update use case, so the auto-created
// PriceSchedule's name shares the drawer preview's tier-correct shape
// ("{client} - Rate Cards" on professional, "{client} - Price Schedule" on
// general). Falls back to "Price Schedule" — the lyngua general-tier default
// — when unset. The string key matches the literal used by the centymo
// writer (mirrors the existing `businessType` plumbing in business.go).
func ExtractClientScheduleSuffixFromContext(ctx context.Context) string {
	if v, ok := ctx.Value("clientScheduleSuffix").(string); ok && v != "" {
		return v
	}
	return "Price Schedule"
}
