//go:build uuidv7

package uuidv7

import (
	"strings"
	"testing"
	"uuid"
)

func TestServiceGeneratesUUIDv7(t *testing.T) {
	svc := NewService()
	got := svc.GenerateID()
	parsed, err := uuid.Parse(got)
	if err != nil {
		t.Fatalf("Parse(%q): %v", got, err)
	}
	if version := parsed[6] >> 4; version != 7 {
		t.Fatalf("UUID version = %d, want 7", version)
	}
	if variant := parsed[8] >> 6; variant != 2 {
		t.Fatalf("UUID variant bits = %02b, want 10", variant)
	}
}

func TestServicePrefix(t *testing.T) {
	got := NewService().GenerateIDWithPrefix("job")
	if !strings.HasPrefix(got, "job_") {
		t.Fatalf("GenerateIDWithPrefix = %q, want job_ prefix", got)
	}
	if _, err := uuid.Parse(strings.TrimPrefix(got, "job_")); err != nil {
		t.Fatalf("prefixed UUID: %v", err)
	}
}
