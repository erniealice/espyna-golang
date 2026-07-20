//go:build http

package middleware

// audit_context_test.go — clientIP/normalizeIP: every value that reaches the
// audit sink's inet column must be a canonical bare IP or "" (NULL), never a
// bracketed/port-bearing/garbage string — audit inserts are tx-fatal, so a
// bad IP string bricks the whole audited write (the "[::1]" login outage,
// 2026-07-20).

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeIP(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"127.0.0.1", "127.0.0.1"},
		{"::1", "::1"},
		{"[::1]", "::1"},
		{"10.0.0.1:443", "10.0.0.1"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"fe80::1%eth0", "fe80::1"},         // zone stripped — inet rejects zones
		{"[fe80::1%eth0]:8080", "fe80::1"},  // bracketed + port + zone
		{" 192.168.1.9 ", "192.168.1.9"},    // padding
		{"not-an-ip", ""},
		{"example.com:443", ""},             // hostname is not an IP
		{"=cmd|calc", ""},                   // header garbage
	}
	for _, c := range cases {
		if got := normalizeIP(c.in); got != c.want {
			t.Errorf("normalizeIP(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestClientIP(t *testing.T) {
	req := func(remote, xff string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
		return r
	}
	cases := []struct {
		name   string
		remote string
		xff    string
		want   string
	}{
		{"ipv4 socket", "203.0.113.7:52011", "", "203.0.113.7"},
		{"ipv6 socket keeps no brackets", "[::1]:52011", "", "::1"},
		{"portless socket", "10.1.2.3", "", "10.1.2.3"},
		{"xff first entry wins", "10.0.0.2:80", "198.51.100.9, 10.0.0.2", "198.51.100.9"},
		{"port-bearing xff parses", "10.0.0.2:80", "[2001:db8::1]:443", "2001:db8::1"},
		{"malformed xff falls back to socket", "203.0.113.7:52011", "garbage-value", "203.0.113.7"},
		{"malformed xff and socket degrade to empty", "bogus", "garbage", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := clientIP(req(c.remote, c.xff)); got != c.want {
				t.Errorf("clientIP(remote=%q, xff=%q) = %q, want %q", c.remote, c.xff, got, c.want)
			}
		})
	}
}
