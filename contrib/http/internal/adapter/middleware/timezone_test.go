//go:build http

package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTimezoneUsesCurrentWorkspaceBeforeUser(t *testing.T) {
	zone := "Asia/Manila"
	var lookupErr error
	m := NewTimezoneMiddleware(func(context.Context) string { return "same-user" }, func(context.Context, string) (string, error) { return "Europe/London", nil })
	m.LookupWorkspaceTZ = func(context.Context) (string, error) { return zone, lookupErr }
	handler := m.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(TimezoneLocationFromContext(r.Context()).String()))
	}))
	for _, want := range []string{"Asia/Manila", "Australia/Sydney", "America/New_York"} {
		zone = want
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/action/subscription/add", nil))
		if w.Body.String() != want {
			t.Fatalf("got %s want %s", w.Body, want)
		}
	}
	lookupErr = errors.New("failed")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/action/subscription/add", nil))
	if w.Code != 500 {
		t.Fatalf("lookup failure status=%d", w.Code)
	}
}
