package usecaseerr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
)

type prefixTranslator struct{}

func (prefixTranslator) Get(context.Context, string, string, ...any) string { return "" }
func (prefixTranslator) GetWithDefault(_ context.Context, _, key, _ string, _ ...any) string {
	return "T(" + key + ")"
}

func TestErrorKeepsPrefixCodeAndMessage(t *testing.T) {
	e := New("collection_application: ", "not_found", "Application not found.")
	if e.Error() != "collection_application: Application not found." || e.ErrorCode() != "not_found" || e.Message() != "Application not found." {
		t.Fatalf("got %q / %q", e.Error(), e.ErrorCode())
	}
	if !IsCode(fmt.Errorf("wrapped: %w", e), "not_found") || IsCode(e, "other") || IsCode(nil, "not_found") {
		t.Fatal("IsCode must see through wrapping and match only the code")
	}
}

func TestLocalizeTranslatesByEntityKeyAndKeepsTheCode(t *testing.T) {
	e := New("x: ", "date_invalid", "fallback")
	got := Localize(context.Background(), prefixTranslator{}, "collection_application", fmt.Errorf("%w: detail", e))
	if !IsCode(got, "date_invalid") || got.Error() != "x: T(collection_application.errors.date_invalid)" {
		t.Fatalf("got %v", got)
	}
	if Localize(context.Background(), nil, "e", e).Error() != "x: fallback" {
		t.Fatal("a nil translator keeps the fallback message")
	}
	plain := errors.New("plain")
	if Localize(context.Background(), prefixTranslator{}, "e", plain) != plain || Localize(context.Background(), nil, "e", nil) != nil {
		t.Fatal("non-coded errors pass through")
	}
}

func TestIsNotFoundOnlyForAbsentRows(t *testing.T) {
	if !IsNotFound(fmt.Errorf("lock: %w", domainports.ErrLockedRowNotFound)) || !IsNotFound(errors.New("Record not found")) {
		t.Fatal("absent-row errors must be not found")
	}
	if IsNotFound(errors.New("connection reset")) || IsNotFound(nil) {
		t.Fatal("infrastructure errors are never not found (C9)")
	}
}

func TestRepoErrWrapsWithPackageAndOperation(t *testing.T) {
	boom := errors.New("boom")
	err := RepoErr("recovery_document", "lock revenue", boom, "r-1")
	if !errors.Is(err, boom) || err.Error() != "recovery_document: lock revenue: boom" || IsCode(err, "not_found") {
		t.Fatalf("got %v", err)
	}
}
