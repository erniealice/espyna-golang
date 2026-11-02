//go:build http

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/erniealice/pyeza-golang/render"
)

// switchHarness holds a fake session store, so a real switch changes what the next
// request sees, the way the session table does.
type switchHarness struct {
	t             *testing.T
	token         string
	workspace     string
	actingAs      string
	bindingKind   PrincipalType
	targets       []ActingAsTarget
	applySwitch   bool // false: the switch reports success and changes nothing
	switches      int
	downstream    int
	seenPath      string
	seenRawQuery  string
	seenWorkspace string
	seenActingAs  string
	seenBanner    *render.PostRotationBannerData
	cookies       map[string]string
	handler       http.Handler
}

func newSwitchHarness(t *testing.T) *switchHarness {
	t.Helper()
	h := &switchHarness{
		t:           t,
		token:       "tok-a",
		workspace:   "ws-a",
		bindingKind: PrincipalTypeOperatorStaff,
		applySwitch: true,
		cookies:     map[string]string{"ichizen_session": "tok-a"},
	}
	mw := NewWorkspacePathMiddleware(WorkspacePathConfig{
		SlugLookup: func(_ context.Context, slug string) (string, error) {
			switch slug {
			case "alpha":
				return "ws-a", nil
			case "bravo":
				return "ws-b", nil
			}
			return "", nil
		},
		SessionLookup: func(r *http.Request) (string, string, string, bool) {
			c, err := r.Cookie("ichizen_session")
			if err != nil || c.Value != h.token {
				return "", "", "", false
			}
			return "user-1", h.workspace, h.token, true
		},
		PrincipalLookup: func(*http.Request) (PrincipalType, string) { return h.bindingKind, "principal-1" },
		ActingAsLookup:  func(*http.Request) (string, string) { return h.actingAs, "" },
		BindingResolver: func(_ context.Context, _, workspaceID string, _ PrincipalType, _ string) (*Principal, error) {
			return &Principal{Type: h.bindingKind, ID: "principal-1", WorkspaceID: workspaceID, ActingAsTargets: h.targets}, nil
		},
		ExecuteSwitch: func(_ context.Context, _, token string, target *Principal, urlActingAs string, _, _, _, _ string) (*WorkspaceSwitchResult, error) {
			h.switches++
			if !h.applySwitch {
				return &WorkspaceSwitchResult{NewToken: token}, nil
			}
			h.actingAs = urlActingAs
			if target.WorkspaceID == h.workspace {
				return &WorkspaceSwitchResult{NewToken: token}, nil // in place
			}
			h.workspace = target.WorkspaceID
			h.token = "tok-" + target.WorkspaceID
			return &WorkspaceSwitchResult{NewToken: h.token}, nil
		},
		RotationRateLimitPerMin: 1000,
	})
	h.handler = mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.downstream++
		h.seenPath = r.URL.Path
		h.seenRawQuery = r.URL.RawQuery
		h.seenWorkspace = GetURLWorkspaceIDFromContext(r.Context())
		h.seenActingAs = h.actingAs
		h.seenBanner = render.PostRotationBannerFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	return h
}

// do sends one request with the jar and stores the cookies the response set.
func (h *switchHarness) do(method, target string, headers ...string) *httptest.ResponseRecorder {
	h.t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	for name, value := range h.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge < 0 {
			delete(h.cookies, c.Name)
		} else {
			h.cookies[c.Name] = c.Value
		}
	}
	return rec
}

func TestSwitch_GETRedirectsToSameURLThenRunsUnderTheNewIdentity(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			h := newSwitchHarness(t)
			target := "/w/bravo/clients/list?page=2&q=x%20y"

			first := h.do(method, target)
			if first.Code != http.StatusSeeOther || first.Header().Get("Location") != target {
				t.Fatalf("first: code=%d location=%q", first.Code, first.Header().Get("Location"))
			}
			if h.downstream != 0 {
				t.Fatalf("the handler ran on the request that switched")
			}
			if h.cookies["ichizen_session"] != "tok-ws-b" {
				t.Fatalf("the redirect did not carry the new session cookie")
			}
			if first.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("the redirect may be cached")
			}

			second := h.do(method, target)
			if second.Code != http.StatusOK || h.downstream != 1 || h.switches != 1 {
				t.Fatalf("second: code=%d downstream=%d switches=%d", second.Code, h.downstream, h.switches)
			}
			if h.seenPath != "/clients/list" || h.seenRawQuery != "page=2&q=x%20y" || h.seenWorkspace != "ws-b" {
				t.Errorf("second ran as path=%q query=%q ws=%q", h.seenPath, h.seenRawQuery, h.seenWorkspace)
			}
			if _, stays := h.cookies[switchGuardCookieName]; stays {
				t.Errorf("the guard was not cleared on arrival")
			}
		})
	}
}

func TestSwitch_BannerSurvivesTheRedirectOnce(t *testing.T) {
	h := newSwitchHarness(t)
	h.do(http.MethodGet, "/w/alpha/home") // caches the slug of the session's workspace
	h.do(http.MethodGet, "/w/bravo/home")
	h.do(http.MethodGet, "/w/bravo/home")
	if h.seenBanner == nil || h.seenBanner.TargetSlug != "bravo" || h.seenBanner.PreviousSlug != "alpha" {
		t.Fatalf("banner on arrival: %+v", h.seenBanner)
	}
	h.do(http.MethodGet, "/w/bravo/home")
	if h.seenBanner != nil {
		t.Errorf("the banner showed twice")
	}
}

func TestSwitch_HTMXGETFollowsTheRedirectInsteadOfNavigating(t *testing.T) {
	h := newSwitchHarness(t)
	rec := h.do(http.MethodGet, "/w/bravo/drawer", "HX-Request", "true")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("HX-Redirect") != "" {
		t.Fatalf("code=%d hx-redirect=%q", rec.Code, rec.Header().Get("HX-Redirect"))
	}
}

func TestSwitch_NonGETIsRefusedWithNoSwitchAndNoWrite(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			h := newSwitchHarness(t)
			rec := h.do(method, "/w/bravo/action/save")
			if rec.Code != http.StatusConflict {
				t.Fatalf("code=%d", rec.Code)
			}
			if h.switches != 0 || h.downstream != 0 {
				t.Fatalf("switches=%d downstream=%d", h.switches, h.downstream)
			}
			if h.workspace != "ws-a" || len(rec.Result().Cookies()) != 0 {
				t.Fatalf("session changed: ws=%s cookies=%d", h.workspace, len(rec.Result().Cookies()))
			}
		})
	}
	t.Run("htmx reloads the stale page", func(t *testing.T) {
		h := newSwitchHarness(t)
		rec := h.do(http.MethodPost, "/w/bravo/action/save", "HX-Request", "true")
		if rec.Code != http.StatusConflict || rec.Header().Get("HX-Refresh") != "true" {
			t.Fatalf("code=%d hx-refresh=%q", rec.Code, rec.Header().Get("HX-Refresh"))
		}
	})
	t.Run("same workspace is untouched", func(t *testing.T) {
		h := newSwitchHarness(t)
		rec := h.do(http.MethodPost, "/w/alpha/action/save")
		if rec.Code != http.StatusOK || h.downstream != 1 || h.switches != 0 {
			t.Fatalf("code=%d downstream=%d switches=%d", rec.Code, h.downstream, h.switches)
		}
	})
}

func TestSwitch_NoLoopWhenTheSwitchDoesNotTakeEffect(t *testing.T) {
	h := newSwitchHarness(t)
	h.applySwitch = false
	target := "/w/bravo/home"
	if rec := h.do(http.MethodGet, target); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != target {
		t.Fatalf("first: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	rec := h.do(http.MethodGet, target)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != pickerRedirectURL {
		t.Fatalf("second: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if h.switches != 1 || h.downstream != 0 {
		t.Errorf("switches=%d downstream=%d", h.switches, h.downstream)
	}
	if _, stays := h.cookies[switchGuardCookieName]; stays {
		t.Errorf("the guard stayed, so a later visit would go straight to the picker")
	}
}

func TestSwitch_GuardForAnotherURLDoesNotBlock(t *testing.T) {
	h := newSwitchHarness(t)
	h.do(http.MethodGet, "/w/bravo/one") // guard is for /one; a parallel request lands elsewhere
	h.workspace, h.token = "ws-a", "tok-a"
	h.cookies["ichizen_session"] = "tok-a"
	rec := h.do(http.MethodGet, "/w/bravo/two")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/w/bravo/two" {
		t.Fatalf("code=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestSwitch_ForgedGuardValuesGrantNothing(t *testing.T) {
	for _, value := range []string{"", "garbage", "a.b", switchGuardPathKey("/w/bravo/other") + ".r.alpha"} {
		h := newSwitchHarness(t)
		h.cookies[switchGuardCookieName] = value
		rec := h.do(http.MethodGet, "/w/bravo/home")
		if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/w/bravo/home") || h.switches != 1 {
			t.Errorf("value %q: code=%d location=%q switches=%d", value, rec.Code, rec.Header().Get("Location"), h.switches)
		}
	}
	// A guard with a bad slug on the right URL shows a banner without that slug.
	h := newSwitchHarness(t)
	h.workspace = "ws-b"
	h.cookies[switchGuardCookieName] = switchGuardPathKey("/w/bravo/home") + ".r.<script>"
	h.do(http.MethodGet, "/w/bravo/home")
	if h.seenBanner == nil || h.seenBanner.PreviousSlug != "" {
		t.Errorf("banner: %+v", h.seenBanner)
	}
}

func delegateHarness(t *testing.T) *switchHarness {
	h := newSwitchHarness(t)
	h.bindingKind = PrincipalTypeClientDelegate
	h.targets = []ActingAsTarget{{ID: "client-1", WorkspaceID: "ws-a"}, {ID: "client-2", WorkspaceID: "ws-a"}}
	return h
}

func TestSwitch_ActingAs(t *testing.T) {
	t.Run("matching session does not switch", func(t *testing.T) {
		h := delegateHarness(t)
		h.actingAs = "client-1"
		for i := 0; i < 5; i++ {
			if rec := h.do(http.MethodGet, "/w/alpha/as/client-1/home"); rec.Code != http.StatusOK {
				t.Fatalf("code=%d", rec.Code)
			}
		}
		if rec := h.do(http.MethodPost, "/w/alpha/as/client-1/action/save"); rec.Code != http.StatusOK {
			t.Fatalf("POST code=%d", rec.Code)
		}
		if h.switches != 0 || h.downstream != 6 {
			t.Errorf("switches=%d downstream=%d", h.switches, h.downstream)
		}
	})
	t.Run("a new target switches in place, redirects, and then settles", func(t *testing.T) {
		h := delegateHarness(t)
		h.actingAs = "client-1"
		target := "/w/alpha/as/client-2/home"
		rec := h.do(http.MethodGet, target)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != target || h.downstream != 0 {
			t.Fatalf("code=%d location=%q downstream=%d", rec.Code, rec.Header().Get("Location"), h.downstream)
		}
		if h.cookies["ichizen_session"] != "tok-a" {
			t.Errorf("an in-place switch changed the token")
		}
		if rec := h.do(http.MethodGet, target); rec.Code != http.StatusOK {
			t.Fatalf("second code=%d", rec.Code)
		}
		if h.seenActingAs != "client-2" || h.switches != 1 {
			t.Errorf("actingAs=%q switches=%d", h.seenActingAs, h.switches)
		}
		if h.seenBanner != nil {
			t.Errorf("an in-place switch showed the rotation banner")
		}
	})
	t.Run("POST to a target the session does not hold is refused", func(t *testing.T) {
		h := delegateHarness(t)
		h.actingAs = "client-1"
		rec := h.do(http.MethodPost, "/w/alpha/as/client-2/action/save")
		if rec.Code != http.StatusConflict || h.switches != 0 || h.downstream != 0 || h.actingAs != "client-1" {
			t.Fatalf("code=%d switches=%d downstream=%d actingAs=%s", rec.Code, h.switches, h.downstream, h.actingAs)
		}
	})
	t.Run("a revoked target still goes through the switch, which re-checks the grant", func(t *testing.T) {
		h := delegateHarness(t)
		h.actingAs = "client-1"
		h.targets = []ActingAsTarget{{ID: "client-2", WorkspaceID: "ws-a"}}
		h.do(http.MethodGet, "/w/alpha/as/client-1/home")
		if h.switches != 1 {
			t.Errorf("switches=%d", h.switches)
		}
	})
	t.Run("no lookup wired switches as before", func(t *testing.T) {
		m := &WorkspacePathMiddleware{}
		if m.sessionHoldsActingAs(httptest.NewRequest(http.MethodGet, "/", nil), "ws", "ws", "client-1",
			&Principal{Type: PrincipalTypeClientDelegate, ID: "p", ActingAsTargets: []ActingAsTarget{{ID: "client-1"}}},
			PrincipalTypeClientDelegate, "p") {
			t.Errorf("held without a lookup")
		}
	})
	t.Run("another principal does not count as holding the target", func(t *testing.T) {
		m := &WorkspacePathMiddleware{cfg: WorkspacePathConfig{ActingAsLookup: func(*http.Request) (string, string) { return "client-1", "" }}}
		if m.sessionHoldsActingAs(httptest.NewRequest(http.MethodGet, "/", nil), "ws", "ws", "client-1",
			&Principal{Type: PrincipalTypeClientDelegate, ID: "p", ActingAsTargets: []ActingAsTarget{{ID: "client-1"}}},
			PrincipalTypeClientDelegate, "other") {
			t.Errorf("held under another principal")
		}
	})
}
