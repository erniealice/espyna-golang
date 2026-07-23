package http

// view_adapter_history_restore_test.go — locks the full-vs-partial render
// decision for htmx history-restore requests (browser Back/Forward on an htmx
// history-cache MISS). Such a GET carries HX-Request:true (so a naive isHTMX
// gate would route it into the shell-less "-content"/partial branch) AND
// HX-History-Restore-Request:true. htmx swaps the response into hx-history-elt
// (the <body>) expecting a FULL document, so the adapter must render the full
// page template ("job-list"), never the "job-list-content" partial — otherwise
// the address bar shows a bare, unstyled fragment after Back.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/erniealice/pyeza-golang/types"
	"github.com/erniealice/pyeza-golang/view"
)

// recordingRenderer implements TemplateRenderer (NOT ContextRenderer, so the
// adapter takes the plain a.renderer.* path). RenderBuffered fails for any
// "*-partial" name so the non-target-non-restore fallback-to-full path is
// exercised realistically (no view defines a page-level "-partial" template).
type recordingRenderer struct {
	rendered []string // names passed to Render
	buffered []string // names passed to RenderBuffered
}

func (r *recordingRenderer) Render(w http.ResponseWriter, name string, _ interface{}) error {
	r.rendered = append(r.rendered, name)
	return nil
}

func (r *recordingRenderer) RenderBuffered(w http.ResponseWriter, name string, _ interface{}) error {
	r.buffered = append(r.buffered, name)
	if strings.HasSuffix(name, "-partial") {
		return http.ErrNotSupported // simulate "template not found" → fallback
	}
	return nil
}

func newTestViewAdapter(rr *recordingRenderer) *ViewAdapter {
	return NewViewAdapter(rr, "v-test", nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", "", nil)
}

func (r *recordingRenderer) has(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func TestHandleRender_HistoryRestore_RendersFullShell(t *testing.T) {
	cases := []struct {
		name        string
		headers     map[string]string
		wantFull    bool   // expect the full "job-list" template via Render
		wantPartial bool   // expect the "job-list-content" partial via RenderBuffered
		fullName    string
		partialName string
	}{
		{
			name:        "history-restore (no target) renders full shell",
			headers:     map[string]string{"HX-Request": "true", "HX-History-Restore-Request": "true"},
			wantFull:    true,
			wantPartial: false,
			fullName:    "job-list",
			partialName: "job-list-content",
		},
		{
			name:        "history-restore WITH main-content target still renders full shell",
			headers:     map[string]string{"HX-Request": "true", "HX-History-Restore-Request": "true", "HX-Target": "main-content"},
			wantFull:    true,
			wantPartial: false,
			fullName:    "job-list",
			partialName: "job-list-content",
		},
		{
			name:        "boosted partial (main-content target) renders the content partial",
			headers:     map[string]string{"HX-Request": "true", "HX-Target": "main-content"},
			wantFull:    false,
			wantPartial: true,
			fullName:    "job-list",
			partialName: "job-list-content",
		},
		{
			name:        "cold full-page load renders full shell",
			headers:     map[string]string{},
			wantFull:    true,
			wantPartial: false,
			fullName:    "job-list",
			partialName: "job-list-content",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := &recordingRenderer{}
			a := newTestViewAdapter(rr)

			req := httptest.NewRequest(http.MethodGet, "/courses/list/active", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			req = req.WithContext(context.Background())
			w := httptest.NewRecorder()

			a.handleRender(w, req, view.OK("job-list", &types.PageData{}))

			gotFull := rr.has(rr.rendered, tc.fullName)
			gotPartial := rr.has(rr.buffered, tc.partialName)

			if gotFull != tc.wantFull {
				t.Errorf("full shell render(%q): got %v want %v (rendered=%v buffered=%v)",
					tc.fullName, gotFull, tc.wantFull, rr.rendered, rr.buffered)
			}
			if gotPartial != tc.wantPartial {
				t.Errorf("content partial render(%q): got %v want %v (rendered=%v buffered=%v)",
					tc.partialName, gotPartial, tc.wantPartial, rr.rendered, rr.buffered)
			}
		})
	}
}
