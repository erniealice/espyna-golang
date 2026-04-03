package context

import (
	"context"
	"testing"
)

// mockTranslationService implements ports.TranslationService for testing
type mockTranslationService struct {
	translations map[string]string
}

func (m *mockTranslationService) GetWithDefault(_ context.Context, businessType, key, fallback string) string {
	if v, ok := m.translations[businessType+"."+key]; ok {
		return v
	}
	if v, ok := m.translations[key]; ok {
		return v
	}
	return fallback
}

func newMockTranslationService(translations map[string]string) *mockTranslationService {
	return &mockTranslationService{translations: translations}
}

func TestGetTranslatedMessage_WithService(t *testing.T) {
	t.Parallel()

	svc := newMockTranslationService(map[string]string{
		"education.role.name": "Student Role",
	})

	ctx := context.Background()
	got := GetTranslatedMessage(ctx, svc, "education", "role.name", "Default Role")
	if got != "Student Role" {
		t.Errorf("GetTranslatedMessage() = %q, want %q", got, "Student Role")
	}
}

func TestGetTranslatedMessage_NilService_ReturnsFallback(t *testing.T) {
	t.Parallel()

	got := GetTranslatedMessage(context.Background(), nil, "education", "role.name", "Fallback Value")
	if got != "Fallback Value" {
		t.Errorf("GetTranslatedMessage(nil) = %q, want %q", got, "Fallback Value")
	}
}

func TestGetTranslatedMessage_MissingKey_ReturnsFallback(t *testing.T) {
	t.Parallel()

	svc := newMockTranslationService(map[string]string{})
	got := GetTranslatedMessage(context.Background(), svc, "education", "missing.key", "Default")
	if got != "Default" {
		t.Errorf("GetTranslatedMessage(missing) = %q, want %q", got, "Default")
	}
}

func TestGetTranslatedMessageWithContext(t *testing.T) {
	t.Parallel()

	svc := newMockTranslationService(map[string]string{
		"fitness_center.role.name": "Trainer Role",
	})

	ctx := WithBusinessType(context.Background(), "fitness_center")
	got := GetTranslatedMessageWithContext(ctx, svc, "role.name", "Default Role")
	if got != "Trainer Role" {
		t.Errorf("GetTranslatedMessageWithContext() = %q, want %q", got, "Trainer Role")
	}
}

func TestGetTranslatedMessageWithContext_DefaultBusinessType(t *testing.T) {
	t.Parallel()

	svc := newMockTranslationService(map[string]string{
		"education.role.name": "Student Role",
	})

	// No business type set in context, should fall back to "education"
	got := GetTranslatedMessageWithContext(context.Background(), svc, "role.name", "Default Role")
	if got != "Student Role" {
		t.Errorf("GetTranslatedMessageWithContext(default) = %q, want %q", got, "Student Role")
	}
}

func TestGetTranslatedMessageWithContextAndTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		tags     map[string]interface{}
		fallback string
		want     string
	}{
		{
			name:     "single_tag",
			key:      "error.not_found",
			tags:     map[string]interface{}{"id": "123"},
			fallback: "Entity {id} not found",
			want:     "Entity 123 not found",
		},
		{
			name:     "multiple_tags",
			key:      "error.conflict",
			tags:     map[string]interface{}{"entity": "role", "name": "Admin"},
			fallback: "The {entity} named {name} already exists",
			want:     "The role named Admin already exists",
		},
		{
			name:     "no_tags",
			key:      "error.generic",
			tags:     nil,
			fallback: "Something went wrong",
			want:     "Something went wrong",
		},
		{
			name:     "empty_tags",
			key:      "error.generic",
			tags:     map[string]interface{}{},
			fallback: "Something went wrong",
			want:     "Something went wrong",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Use nil service so fallback is used and we can test tag substitution
			got := GetTranslatedMessageWithContextAndTags(context.Background(), nil, tc.key, tc.tags, tc.fallback)
			if got != tc.want {
				t.Errorf("GetTranslatedMessageWithContextAndTags() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		str    string
		substr string
		want   bool
	}{
		{name: "exact_match", str: "Hello", substr: "Hello", want: true},
		{name: "case_insensitive", str: "Hello World", substr: "hello", want: true},
		{name: "substring", str: "Hello World", substr: "world", want: true},
		{name: "no_match", str: "Hello", substr: "xyz", want: false},
		{name: "empty_substr", str: "Hello", substr: "", want: true},
		{name: "empty_str", str: "", substr: "hello", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Contains(tc.str, tc.substr)
			if got != tc.want {
				t.Errorf("Contains(%q, %q) = %v, want %v", tc.str, tc.substr, got, tc.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "nil_value", value: nil, want: ""},
		{name: "string_value", value: "hello", want: "hello"},
		{name: "int_value", value: 42, want: ""},
		{name: "bool_value", value: true, want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := toString(tc.value)
			if got != tc.want {
				t.Errorf("toString(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestNewTranslationHelper(t *testing.T) {
	t.Parallel()

	svc := newMockTranslationService(map[string]string{
		"education.greeting": "Hello Student",
	})

	helper := NewTranslationHelper(svc)
	if helper == nil {
		t.Fatal("NewTranslationHelper returned nil")
	}

	ctx := context.Background()
	got := helper.GetTranslatedMessage(ctx, "education", "greeting", "Hi")
	if got != "Hello Student" {
		t.Errorf("helper.GetTranslatedMessage() = %q, want %q", got, "Hello Student")
	}
}

func TestTranslationHelper_GetTranslatedMessageWithContext(t *testing.T) {
	t.Parallel()

	svc := newMockTranslationService(map[string]string{
		"education.greeting": "Hello Student",
	})

	helper := NewTranslationHelper(svc)
	ctx := WithBusinessType(context.Background(), "education")
	got := helper.GetTranslatedMessageWithContext(ctx, "greeting", "Hi")
	if got != "Hello Student" {
		t.Errorf("helper.GetTranslatedMessageWithContext() = %q, want %q", got, "Hello Student")
	}
}

func TestTranslationHelper_NilService(t *testing.T) {
	t.Parallel()

	helper := NewTranslationHelper(nil)
	got := helper.GetTranslatedMessage(context.Background(), "education", "greeting", "Fallback")
	if got != "Fallback" {
		t.Errorf("helper.GetTranslatedMessage(nil svc) = %q, want %q", got, "Fallback")
	}
}
