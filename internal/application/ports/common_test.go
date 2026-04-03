package ports

import (
	"errors"
	"strings"
	"testing"
)

func TestApplicationError_Error_WithWrappedError(t *testing.T) {
	t.Parallel()

	inner := errors.New("database connection failed")
	appErr := NewApplicationError("DB_CONN", "Connection error", inner, nil)

	got := appErr.Error()
	if !strings.Contains(got, "DB_CONN") {
		t.Errorf("Error() should contain code 'DB_CONN', got %q", got)
	}
	if !strings.Contains(got, "Connection error") {
		t.Errorf("Error() should contain message 'Connection error', got %q", got)
	}
	if !strings.Contains(got, "database connection failed") {
		t.Errorf("Error() should contain inner error details, got %q", got)
	}
}

func TestApplicationError_Error_WithoutWrappedError(t *testing.T) {
	t.Parallel()

	appErr := NewApplicationError("VALIDATION", "Name is required", nil, nil)

	got := appErr.Error()
	want := "VALIDATION: Name is required"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestApplicationError_Error_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		message string
		err     error
		ctx     map[string]any
		check   func(t *testing.T, got string)
	}{
		{
			name:    "code_and_message_only",
			code:    "NOT_FOUND",
			message: "Entity not found",
			err:     nil,
			ctx:     nil,
			check: func(t *testing.T, got string) {
				if got != "NOT_FOUND: Entity not found" {
					t.Errorf("got %q", got)
				}
			},
		},
		{
			name:    "with_wrapped_error",
			code:    "INTERNAL",
			message: "Processing failed",
			err:     errors.New("timeout"),
			ctx:     nil,
			check: func(t *testing.T, got string) {
				if !strings.Contains(got, "INTERNAL") {
					t.Errorf("missing code, got %q", got)
				}
				if !strings.Contains(got, "timeout") {
					t.Errorf("missing inner error, got %q", got)
				}
			},
		},
		{
			name:    "with_context",
			code:    "CONFLICT",
			message: "Duplicate entry",
			err:     nil,
			ctx:     map[string]any{"entity": "role", "id": "123"},
			check: func(t *testing.T, got string) {
				if got != "CONFLICT: Duplicate entry" {
					t.Errorf("got %q", got)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			appErr := NewApplicationError(tc.code, tc.message, tc.err, tc.ctx)
			tc.check(t, appErr.Error())
		})
	}
}

func TestNewApplicationError_FieldsSet(t *testing.T) {
	t.Parallel()

	inner := errors.New("inner")
	ctx := map[string]any{"key": "value"}

	appErr := NewApplicationError("CODE", "msg", inner, ctx)

	if appErr.Code != "CODE" {
		t.Errorf("Code = %q, want %q", appErr.Code, "CODE")
	}
	if appErr.Message != "msg" {
		t.Errorf("Message = %q, want %q", appErr.Message, "msg")
	}
	if appErr.Err != inner {
		t.Errorf("Err = %v, want %v", appErr.Err, inner)
	}
	if appErr.Context["key"] != "value" {
		t.Errorf("Context[key] = %v, want %q", appErr.Context["key"], "value")
	}
}

func TestApplicationError_ImplementsErrorInterface(t *testing.T) {
	t.Parallel()

	var err error = NewApplicationError("CODE", "msg", nil, nil)
	if err == nil {
		t.Fatal("ApplicationError should implement error interface")
	}
}
