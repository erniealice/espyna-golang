package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/shared/database/model"
	"github.com/erniealice/pyeza-golang/view"
)

func TestHandleError_DirectDatabaseErrorSanitized(t *testing.T) {
	t.Parallel()

	a := newTestViewAdapter(nil)
	dbErr := model.NewDatabaseError(
		`raw pg error: duplicate key value violates unique constraint "secret_table_pk"`,
		"conflict",
		http.StatusConflict,
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)

	a.handleError(rec, req, viewResultWithError(dbErr, http.StatusConflict))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusConflict)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != http.StatusText(http.StatusConflict) {
		t.Fatalf("body: got %q want %q", body, http.StatusText(http.StatusConflict))
	}
	if strings.Contains(body, "secret_table") {
		t.Fatalf("response body must not leak db detail (table name)")
	}
}

func TestHandleError_WrappedDatabaseErrorSanitized(t *testing.T) {
	t.Parallel()

	a := newTestViewAdapter(nil)
	dbErr := model.NewDatabaseError(
		`raw pg error: duplicate key value violates unique constraint "secret_table_pk"`,
		"conflict",
		http.StatusConflict,
	)
	wrapped := fmt.Errorf("while processing request: %w", dbErr)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)

	a.handleError(rec, req, viewResultWithError(wrapped, http.StatusConflict))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusConflict)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != http.StatusText(http.StatusConflict) {
		t.Fatalf("body: got %q want %q", body, http.StatusText(http.StatusConflict))
	}
	if strings.Contains(body, "secret_table") {
		t.Fatalf("response body must not leak db detail (constraint name)")
	}
}

func TestHandleError_NonDatabaseInternalErrorSanitized(t *testing.T) {
	t.Parallel()

	a := newTestViewAdapter(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)

	a.handleError(rec, req, viewResultWithError(errors.New("secret_table=private.connection_pool_timeout"), http.StatusInternalServerError))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusInternalServerError)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("body: got %q want %q", body, http.StatusText(http.StatusInternalServerError))
	}
	if strings.Contains(body, "secret_table") {
		t.Fatalf("response body must not leak internals")
	}
}

func TestHandleError_RegularClientErrorPreserved(t *testing.T) {
	t.Parallel()

	a := newTestViewAdapter(nil)
	const clientErr = "validation failed for customer_id"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/demo", nil)

	a.handleError(rec, req, viewResultWithError(errors.New(clientErr), http.StatusUnprocessableEntity))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != clientErr {
		t.Fatalf("body: got %q want %q", body, clientErr)
	}
}

func viewResultWithError(err error, status int) view.ViewResult {
	return view.ViewResult{
		Error:      err,
		StatusCode: status,
	}
}
