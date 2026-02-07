package context

import "context"

// ExtractBusinessTypeFromContext extracts business type from context.
// Returns "education" as default fallback if no business type is found.
func ExtractBusinessTypeFromContext(ctx context.Context) string {
	if businessType, ok := ctx.Value("businessType").(string); ok {
		return businessType
	}
	return "education" // Default fallback
}

// ExtractBusinessTypeFromContextWithFallback extracts business type with custom fallback.
func ExtractBusinessTypeFromContextWithFallback(ctx context.Context, fallback string) string {
	if businessType, ok := ctx.Value("businessType").(string); ok {
		return businessType
	}
	return fallback
}

// WithBusinessType creates a new context with the specified business type.
// This is useful for testing scenarios where we need to simulate HTTP middleware behavior.
func WithBusinessType(ctx context.Context, businessType string) context.Context {
	return context.WithValue(ctx, "businessType", businessType)
}
