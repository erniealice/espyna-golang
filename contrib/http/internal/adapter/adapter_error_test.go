//go:build http

package vanilla

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/composition/routing"
	"github.com/erniealice/espyna-golang/shared/database/model"
)

type stubRouteHandler struct {
	err error
}

func (s stubRouteHandler) Execute(_ context.Context, _ proto.Message) (proto.Message, error) {
	return nil, s.err
}

func TestCreateHTTPHandler_DirectDatabaseError_SanitizedResponse(t *testing.T) {
	t.Parallel()

	secret := "secret_table_pk"
	dbErr := model.NewDatabaseError(
		fmt.Sprintf("insert failed on %s", secret),
		"conflict",
		http.StatusConflict,
	)

	a := NewVanillaAdapter()
	route := &routing.Route{
		Method: http.MethodGet,
		Path:   "/demo",
		Handler: stubRouteHandler{
			err: dbErr,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)
	rec := httptest.NewRecorder()
	a.createHTTPHandler(route)(rec, req)

	if got := rec.Code; got != http.StatusConflict {
		t.Fatalf("status: got %d want %d", got, http.StatusConflict)
	}

	var body map[string]interface{}
	if err := json.UnmarshalDecode(jsontext.NewDecoder(rec.Body), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if got := body["error"]; got != http.StatusText(http.StatusConflict) {
		t.Fatalf("error body: got %v want %q", got, http.StatusText(http.StatusConflict))
	}

	text := strings.TrimSpace(rec.Body.String())
	if strings.Contains(text, secret) {
		t.Fatalf("response must not contain database secret token")
	}
	if _, ok := body["details"]; ok {
		t.Fatalf("response details must not include database internals")
	}
}

func TestCreateHTTPHandler_WrappedDatabaseError_SanitizedResponse(t *testing.T) {
	t.Parallel()

	secret := "secret_table_pk"
	baseErr := model.NewDatabaseError(
		fmt.Sprintf("insert failed on %s", secret),
		"conflict",
		http.StatusConflict,
	)
	wrapped := fmt.Errorf("while handling request: %w", baseErr)

	a := NewVanillaAdapter()
	route := &routing.Route{
		Method: http.MethodGet,
		Path:   "/demo",
		Handler: stubRouteHandler{
			err: wrapped,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)
	rec := httptest.NewRecorder()
	a.createHTTPHandler(route)(rec, req)

	if got := rec.Code; got != http.StatusConflict {
		t.Fatalf("status: got %d want %d", got, http.StatusConflict)
	}

	var body map[string]interface{}
	if err := json.UnmarshalDecode(jsontext.NewDecoder(rec.Body), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if got := body["error"]; got != http.StatusText(http.StatusConflict) {
		t.Fatalf("error body: got %v want %q", got, http.StatusText(http.StatusConflict))
	}
	text := strings.TrimSpace(rec.Body.String())
	if strings.Contains(text, secret) {
		t.Fatalf("response must not contain database secret token")
	}
}

func TestCreateHTTPHandler_GenericInternalError_SanitizedResponse(t *testing.T) {
	t.Parallel()

	secret := "secret.connection_pool_timeout"
	a := NewVanillaAdapter()
	route := &routing.Route{
		Method: http.MethodGet,
		Path:   "/demo",
		Handler: stubRouteHandler{
			err: errors.New(secret),
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)
	rec := httptest.NewRecorder()
	a.createHTTPHandler(route)(rec, req)

	if got := rec.Code; got != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", got, http.StatusInternalServerError)
	}

	var body map[string]interface{}
	if err := json.UnmarshalDecode(jsontext.NewDecoder(rec.Body), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	want := http.StatusText(http.StatusInternalServerError)
	if got := body["error"]; got != want {
		t.Fatalf("error body: got %v want %q", got, want)
	}

	text := strings.TrimSpace(rec.Body.String())
	if strings.Contains(text, secret) {
		t.Fatalf("response must not contain internal error secret token")
	}
	if _, ok := body["details"]; ok {
		t.Fatalf("response details must not include internal internals")
	}
}
