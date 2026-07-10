package subscription

import (
	"testing"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

func TestFormatCode(t *testing.T) {
	tokens := map[string]string{
		"first_name":     "Maria",
		"last_name":      "Santos",
		"client_name":    "Maria Santos",
		"grade":          "Grade 7",
		"price_schedule": "AY 2025-2026",
	}

	tests := []struct {
		name     string
		template string
		tokens   map[string]string
		want     string
	}{
		{
			name:     "default format via empty template",
			template: "",
			tokens:   tokens,
			want:     "Santos, Maria (AY 2025-2026)",
		},
		{
			name:     "default format via auto sentinel",
			template: "auto",
			tokens:   tokens,
			want:     "Santos, Maria (AY 2025-2026)",
		},
		{
			name:     "default format via AUTO uppercase sentinel",
			template: "AUTO",
			tokens:   tokens,
			want:     "Santos, Maria (AY 2025-2026)",
		},
		{
			name:     "explicit default format string",
			template: DefaultCodeFormat,
			tokens:   tokens,
			want:     "Santos, Maria (AY 2025-2026)",
		},
		{
			name:     "custom format with grade + client_name",
			template: "{grade} - {client_name}",
			tokens:   tokens,
			want:     "Grade 7 - Maria Santos",
		},
		{
			name:     "unknown token resolves to empty, never a path walk",
			template: "{last_name} [{client.user.email}]",
			tokens:   tokens,
			want:     "Santos []",
		},
		{
			name:     "missing token value resolves to empty",
			template: "{last_name}, {first_name} ({price_schedule})",
			tokens:   map[string]string{"last_name": "Santos", "first_name": "Maria"},
			want:     "Santos, Maria ()",
		},
		{
			name:     "unclosed brace is left literal",
			template: "{last_name",
			tokens:   tokens,
			want:     "{last_name",
		},
		{
			name:     "no tokens at all",
			template: "STATIC-CODE",
			tokens:   tokens,
			want:     "STATIC-CODE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCode(tt.template, tt.tokens)
			if got != tt.want {
				t.Errorf("FormatCode(%q) = %q, want %q", tt.template, got, tt.want)
			}
		})
	}
}

func TestResolveCodeTokens(t *testing.T) {
	first := "Maria"
	last := "Santos"
	name := "Maria Santos"
	client := &clientpb.Client{FirstName: &first, LastName: &last, Name: &name}
	plan := &planpb.Plan{Name: "Grade 7"}
	schedule := &priceschedulepb.PriceSchedule{Name: "AY 2025-2026"}

	tokens := ResolveCodeTokens(client, plan, schedule)
	if got := FormatCode("auto", tokens); got != "Santos, Maria (AY 2025-2026)" {
		t.Errorf("default format from resolved tokens = %q", got)
	}
	if tokens["grade"] != "Grade 7" {
		t.Errorf("grade token = %q, want %q", tokens["grade"], "Grade 7")
	}
	if tokens["client_name"] != "Maria Santos" {
		t.Errorf("client_name token = %q, want %q", tokens["client_name"], "Maria Santos")
	}
}

func TestResolveCodeTokens_ClientNameFallback(t *testing.T) {
	first := "Maria"
	last := "Santos"
	// No explicit name -> fall back to "first last".
	client := &clientpb.Client{FirstName: &first, LastName: &last}
	tokens := ResolveCodeTokens(client, nil, nil)
	if tokens["client_name"] != "Maria Santos" {
		t.Errorf("client_name fallback = %q, want %q", tokens["client_name"], "Maria Santos")
	}
	if tokens["grade"] != "" || tokens["price_schedule"] != "" {
		t.Errorf("nil plan/schedule should yield empty tokens, got grade=%q schedule=%q", tokens["grade"], tokens["price_schedule"])
	}
}

func TestResolveCodeTokens_AllNil(t *testing.T) {
	tokens := ResolveCodeTokens(nil, nil, nil)
	if got := FormatCode("auto", tokens); got != ",  ()" {
		t.Errorf("all-nil default format = %q, want %q", got, ",  ()")
	}
}
