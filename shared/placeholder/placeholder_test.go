package placeholder

import (
	"errors"
	"reflect"
	"testing"
)

func TestAllowlist_ExactlyClientUserFirstName(t *testing.T) {
	if got, want := Allowed(), []string{"client.user.first_name"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Allowed() = %v, want %v", got, want)
	}
}

func TestTags(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"none", "Plain text.", nil},
		{"one", "{client.user.first_name} works well.", []string{"client.user.first_name"}},
		{"distinct first-appearance", "{b.c} {a.b} {b.c}", []string{"b.c", "a.b"}},
		{"single segment is literal", "{name} and {x}", nil},
		{"uppercase is literal", "{Client.User.First_Name}", nil},
		{"spaces are literal", "{client. user}", nil},
		{"leading digit segment is literal", "{1a.b}", nil},
		{"empty braces literal", "{} {{}}", nil},
		{"trailing dot literal", "{client.}", nil},
		{"nested outer braces", "{{client.user.first_name}}", []string{"client.user.first_name"}},
		{"unknown but grammatical", "{client.user.last_name}", []string{"client.user.last_name"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Tags(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Tags(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestValidateAllowed(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		unknown []string
	}{
		{"plain", "Plain {text} with {braces}.", nil},
		{"allowed", "{client.user.first_name} does well.", nil},
		{"unknown", "{client.user.last_name} and {client.user.first_name}", []string{"client.user.last_name"}},
		{"two unknown", "{a.b} {c.d}", []string{"a.b", "c.d"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAllowed(tc.text)
			if tc.unknown == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var pe *Error
			if !errors.As(err, &pe) || pe.Code != CodeUnknownPlaceholder || !reflect.DeepEqual(pe.Tags, tc.unknown) {
				t.Fatalf("err = %#v, want UNKNOWN_PLACEHOLDER %v", err, tc.unknown)
			}
			if !errors.Is(err, ErrUnknownPlaceholder) || errors.Is(err, ErrMissingValue) {
				t.Fatalf("errors.Is mismatch for %v", err)
			}
		})
	}
}

func TestRender(t *testing.T) {
	name := map[string]string{TagClientUserFirstName: "Ana"}
	cases := []struct {
		name    string
		text    string
		values  map[string]string
		want    string
		code    string
		errTags []string
	}{
		{"no tag unchanged", "Works well.", nil, "Works well.", "", nil},
		{"non-grammar braces stay literal", "{name} and {Client.X} {}", name, "{name} and {Client.X} {}", "", nil},
		{"single tag", "{client.user.first_name} works well.", name, "Ana works well.", "", nil},
		{"multiple occurrences", "{client.user.first_name}: {client.user.first_name}!", name, "Ana: Ana!", "", nil},
		{"nested braces keep outer", "{{client.user.first_name}}", name, "{Ana}", "", nil},
		{"value containing a tag stays inert",
			"{client.user.first_name} works.",
			map[string]string{TagClientUserFirstName: "{client.user.first_name}"},
			"{client.user.first_name} works.", "", nil},
		{"value with braces and other tag inert",
			"Hi {client.user.first_name}",
			map[string]string{TagClientUserFirstName: "{a.b}"},
			"Hi {a.b}", "", nil},
		{"missing value", "{client.user.first_name} works.", nil, "", CodeMissingValue, []string{"client.user.first_name"}},
		{"blank value", "{client.user.first_name} works.", map[string]string{TagClientUserFirstName: "  "}, "", CodeMissingValue, []string{"client.user.first_name"}},
		{"unknown tag", "{client.user.first_name} {client.user.last_name}", name, "", CodeUnknownPlaceholder, []string{"client.user.last_name"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(tc.text, tc.values)
			if tc.code == "" {
				if err != nil || got != tc.want {
					t.Fatalf("Render = %q, %v; want %q", got, err, tc.want)
				}
				return
			}
			var pe *Error
			if !errors.As(err, &pe) || pe.Code != tc.code || !reflect.DeepEqual(pe.Tags, tc.errTags) {
				t.Fatalf("err = %#v, want %s %v", err, tc.code, tc.errTags)
			}
			if got != "" {
				t.Fatalf("half-rendered output on error: %q", got)
			}
		})
	}
}

func TestRoot(t *testing.T) {
	if Root("client.user.first_name") != "client" || Root("x") != "x" {
		t.Fatal("Root mismatch")
	}
}
