package http

import (
	"context"
	"encoding/json/v2"
	"testing"

	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
)

func TestWorkspaceNavURL(t *testing.T) {
	ctx := consumermw.WithURLWorkspaceSlug(context.Background(), "mmis")
	cases := map[string]string{
		"/report-cards/templates":           "/w/mmis/report-cards/templates",
		"/grade-sheet/t1/section/g1":        "/w/mmis/grade-sheet/t1/section/g1",
		"/w/mmis/report-cards/templates":    "/w/mmis/report-cards/templates",
		"/action/report-cards/templates/up": "/action/report-cards/templates/up",
		"/app/profile":                      "/app/profile",
		"":                                  "",
	}
	for in, want := range cases {
		if got := workspaceNavURL(ctx, in); got != want {
			t.Errorf("workspaceNavURL(%q) = %q, want %q", in, got, want)
		}
	}
	if got := workspaceNavURL(context.Background(), "/report-cards/templates"); got != "/report-cards/templates" {
		t.Errorf("outside the /w lane the URL must stay bare, got %q", got)
	}
}

func TestWorkspaceNavHeader(t *testing.T) {
	ctx := consumermw.WithURLWorkspaceSlug(context.Background(), "mmis")
	if got := workspaceNavHeader(ctx, "/grade-sheet/t1"); got != "/w/mmis/grade-sheet/t1" {
		t.Fatalf("bare header = %q", got)
	}
	got := workspaceNavHeader(ctx, `{"path":"/grade-sheet/t1/section/g1","target":"#main"}`)
	var loc map[string]string
	if err := json.Unmarshal([]byte(got), &loc); err != nil {
		t.Fatalf("HX-Location must stay valid JSON: %v (%q)", err, got)
	}
	if loc["path"] != "/w/mmis/grade-sheet/t1/section/g1" || loc["target"] != "#main" {
		t.Fatalf("HX-Location rewritten wrongly: %v", loc)
	}
	if got := workspaceNavHeader(ctx, `{not json`); got != `{not json` {
		t.Fatalf("unparseable JSON must pass through, got %q", got)
	}
}

func TestIsNavigationHeader(t *testing.T) {
	for _, k := range []string{"HX-Redirect", "hx-location", "Location"} {
		if !isNavigationHeader(k) {
			t.Errorf("%s should be a navigation header", k)
		}
	}
	if isNavigationHeader("HX-Trigger") {
		t.Error("HX-Trigger is not a navigation header")
	}
}
