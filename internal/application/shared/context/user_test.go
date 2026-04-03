package context

import (
	"context"
	"testing"
)

func TestWithUserID_and_ExtractUserIDFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
		want   string
	}{
		{name: "normal_user_id", userID: "user-123", want: "user-123"},
		{name: "uuid_user_id", userID: "550e8400-e29b-41d4-a716-446655440000", want: "550e8400-e29b-41d4-a716-446655440000"},
		{name: "empty_string", userID: "", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := WithUserID(context.Background(), tc.userID)
			got := ExtractUserIDFromContext(ctx)
			if got != tc.want {
				t.Errorf("ExtractUserIDFromContext() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractUserIDFromContext_EmptyContext(t *testing.T) {
	t.Parallel()
	got := ExtractUserIDFromContext(context.Background())
	if got != "" {
		t.Errorf("ExtractUserIDFromContext(empty) = %q, want empty string", got)
	}
}

func TestExtractUserIDFromContext_LegacyKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "legacy_uid_key", key: "uid", want: "legacy-user-1"},
		{name: "legacy_user_id_key", key: "user_id", want: "legacy-user-2"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(context.Background(), tc.key, tc.want)
			got := ExtractUserIDFromContext(ctx)
			if got != tc.want {
				t.Errorf("ExtractUserIDFromContext(legacy %s) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestWithWorkspaceID_and_ExtractWorkspaceIDFromContext(t *testing.T) {
	t.Parallel()

	ctx := WithWorkspaceID(context.Background(), "ws-456")
	got := ExtractWorkspaceIDFromContext(ctx)
	if got != "ws-456" {
		t.Errorf("ExtractWorkspaceIDFromContext() = %q, want %q", got, "ws-456")
	}
}

func TestExtractWorkspaceIDFromContext_EmptyContext(t *testing.T) {
	t.Parallel()
	got := ExtractWorkspaceIDFromContext(context.Background())
	if got != "" {
		t.Errorf("ExtractWorkspaceIDFromContext(empty) = %q, want empty string", got)
	}
}

func TestWithWorkspaceUserID_and_ExtractWorkspaceUserIDFromContext(t *testing.T) {
	t.Parallel()

	ctx := WithWorkspaceUserID(context.Background(), "wsu-789")
	got := ExtractWorkspaceUserIDFromContext(ctx)
	if got != "wsu-789" {
		t.Errorf("ExtractWorkspaceUserIDFromContext() = %q, want %q", got, "wsu-789")
	}
}

func TestWithEmail_and_ExtractEmailFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "normal_email", email: "test@example.com", want: "test@example.com"},
		{name: "empty_email", email: "", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := WithEmail(context.Background(), tc.email)
			got := ExtractEmailFromContext(ctx)
			if got != tc.want {
				t.Errorf("ExtractEmailFromContext() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractEmailFromContext_LegacyKey(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), "email", "legacy@example.com")
	got := ExtractEmailFromContext(ctx)
	if got != "legacy@example.com" {
		t.Errorf("ExtractEmailFromContext(legacy) = %q, want %q", got, "legacy@example.com")
	}
}

func TestWithSessionIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		userID          string
		workspaceID     string
		workspaceUserID string
		email           string
		expectWsID      string
		expectWsUserID  string
		expectEmail     string
	}{
		{
			name:            "all_fields_set",
			userID:          "u1",
			workspaceID:     "ws1",
			workspaceUserID: "wsu1",
			email:           "u1@test.com",
			expectWsID:      "ws1",
			expectWsUserID:  "wsu1",
			expectEmail:     "u1@test.com",
		},
		{
			name:            "optional_fields_empty",
			userID:          "u2",
			workspaceID:     "",
			workspaceUserID: "",
			email:           "",
			expectWsID:      "",
			expectWsUserID:  "",
			expectEmail:     "",
		},
		{
			name:            "partial_fields",
			userID:          "u3",
			workspaceID:     "ws3",
			workspaceUserID: "",
			email:           "u3@test.com",
			expectWsID:      "ws3",
			expectWsUserID:  "",
			expectEmail:     "u3@test.com",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := WithSessionIdentity(context.Background(), tc.userID, tc.workspaceID, tc.workspaceUserID, tc.email)

			if got := ExtractUserIDFromContext(ctx); got != tc.userID {
				t.Errorf("userID = %q, want %q", got, tc.userID)
			}
			if got := ExtractWorkspaceIDFromContext(ctx); got != tc.expectWsID {
				t.Errorf("workspaceID = %q, want %q", got, tc.expectWsID)
			}
			if got := ExtractWorkspaceUserIDFromContext(ctx); got != tc.expectWsUserID {
				t.Errorf("workspaceUserID = %q, want %q", got, tc.expectWsUserID)
			}
			if got := ExtractEmailFromContext(ctx); got != tc.expectEmail {
				t.Errorf("email = %q, want %q", got, tc.expectEmail)
			}
		})
	}
}

func TestRequireUserIDFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ctx       context.Context
		wantID    string
		wantError bool
	}{
		{
			name:      "user_present",
			ctx:       WithUserID(context.Background(), "user-1"),
			wantID:    "user-1",
			wantError: false,
		},
		{
			name:      "user_missing",
			ctx:       context.Background(),
			wantID:    "",
			wantError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := RequireUserIDFromContext(tc.ctx)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err != ErrUserNotFoundInContext {
					t.Errorf("expected ErrUserNotFoundInContext, got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.wantID {
					t.Errorf("got %q, want %q", got, tc.wantID)
				}
			}
		})
	}
}

func TestHasUserInContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "user_present", ctx: WithUserID(context.Background(), "user-1"), want: true},
		{name: "user_missing", ctx: context.Background(), want: false},
		{name: "empty_user_id", ctx: WithUserID(context.Background(), ""), want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := HasUserInContext(tc.ctx)
			if got != tc.want {
				t.Errorf("HasUserInContext() = %v, want %v", got, tc.want)
			}
		})
	}
}
