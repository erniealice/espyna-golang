package context

import (
	"context"
	"testing"
)

func TestExtractBusinessTypeFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "business_type_set",
			ctx:  WithBusinessType(context.Background(), "fitness_center"),
			want: "fitness_center",
		},
		{
			name: "empty_context_returns_default",
			ctx:  context.Background(),
			want: "education",
		},
		{
			name: "service_type",
			ctx:  WithBusinessType(context.Background(), "service"),
			want: "service",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractBusinessTypeFromContext(tc.ctx)
			if got != tc.want {
				t.Errorf("ExtractBusinessTypeFromContext() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractBusinessTypeFromContextWithFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      context.Context
		fallback string
		want     string
	}{
		{
			name:     "business_type_set_ignores_fallback",
			ctx:      WithBusinessType(context.Background(), "fitness_center"),
			fallback: "retail",
			want:     "fitness_center",
		},
		{
			name:     "empty_context_uses_fallback",
			ctx:      context.Background(),
			fallback: "retail",
			want:     "retail",
		},
		{
			name:     "empty_context_with_empty_fallback",
			ctx:      context.Background(),
			fallback: "",
			want:     "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractBusinessTypeFromContextWithFallback(tc.ctx, tc.fallback)
			if got != tc.want {
				t.Errorf("ExtractBusinessTypeFromContextWithFallback() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWithBusinessType(t *testing.T) {
	t.Parallel()

	ctx := WithBusinessType(context.Background(), "retail")
	got := ExtractBusinessTypeFromContext(ctx)
	if got != "retail" {
		t.Errorf("WithBusinessType round-trip failed: got %q, want %q", got, "retail")
	}
}
