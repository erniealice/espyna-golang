package compose

import (
	"errors"
	"strings"
	"testing"
)

// TestAssemble_OverlayMutatesInPlace proves the load-bearing single-pointer
// trick: the engine overlays route.json into the descriptor's Routes pointer
// in phase 1, and the Mount closure (phase 2) reads the SAME post-overlay
// pointer. The route map and the value the module sees agree by construction —
// the double-load defect is impossible.
func TestAssemble_OverlayMutatesInPlace(t *testing.T) {
	job := jobUnit()
	job.RouteJSON = JSONBinding{File: "route.json", Key: "job"}

	var mountSawListURL string
	job.Mount = func(mc *MountContext) error {
		// The catalog asserts its own type back; here we read the same pointer.
		r := job.Routes.(*fakeRoutes)
		mountSawListURL = r.ListURL
		return nil
	}

	eng := &Engine{
		// A tier override that renames the list URL in place.
		Overlay: func(b JSONBinding, target any) error {
			if b.File == "route.json" && b.Key == "job" {
				if r, ok := target.(*fakeRoutes); ok {
					r.ListURL = "/practice/list/{status}" // professional-tier rewrite
				}
			}
			return nil
		},
	}

	res, err := eng.Assemble([]Unit{job}, &recordingRegistrar{})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Route table reflects the overridden value.
	if got := res.RouteMap["job.list"]; got != "/practice/list/{status}" {
		t.Errorf("RouteMap[job.list] = %q, want overridden /practice/list/{status}", got)
	}
	// The Mount closure saw the SAME overridden value (no double-load drift).
	if mountSawListURL != "/practice/list/{status}" {
		t.Errorf("Mount saw ListURL %q, want overridden /practice/list/{status}", mountSawListURL)
	}
	// And the nav href resolves against the overridden table.
	href, ok := res.ResolveNavHref("", NavItem{Route: "job.list", Params: map[string]string{"status": "active"}})
	if !ok || href != "/practice/list/active" {
		t.Errorf("nav href = %q,%v want /practice/list/active,true", href, ok)
	}
}

// TestAssemble_OverlayErrorFailsClosed: an overlay parse error aborts the boot
// (mirrors a malformed lyngua override file).
func TestAssemble_OverlayErrorFailsClosed(t *testing.T) {
	job := jobUnit()
	job.RouteJSON = JSONBinding{File: "route.json", Key: "job"}

	eng := &Engine{
		Overlay: func(b JSONBinding, target any) error {
			return errors.New("malformed JSON at line 3")
		},
	}
	_, err := eng.Assemble([]Unit{job}, &recordingRegistrar{})
	if err == nil || !strings.Contains(err.Error(), "route overlay") {
		t.Fatalf("overlay error: want fail-closed boot error, got %v", err)
	}
}

// TestAssemble_NilOverlaySkips: a nil Overlay leaves descriptor defaults
// intact (no panic, no override).
func TestAssemble_NilOverlaySkips(t *testing.T) {
	job := jobUnit()
	job.RouteJSON = JSONBinding{File: "route.json", Key: "job"}

	eng := &Engine{} // Overlay nil
	res, err := eng.Assemble([]Unit{job}, &recordingRegistrar{})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if got := res.RouteMap["job.list"]; got != "/jobs/list/{status}" {
		t.Errorf("RouteMap[job.list] = %q, want default /jobs/list/{status}", got)
	}
}
