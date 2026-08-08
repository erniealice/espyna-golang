//go:build gin

package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/composition/routing"
	"github.com/erniealice/espyna-golang/shared/database/model"
)

type errorRouteHandler struct{ err error }

func (h errorRouteHandler) Execute(context.Context, proto.Message) (proto.Message, error) {
	return nil, h.err
}

func TestCreateGinHandler_ExecutionErrorsAreSanitized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "secret_table_primary_key"
	databaseErr := model.NewDatabaseError("insert failed on "+secret, "conflict", http.StatusConflict)

	for _, test := range []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{"direct database error", databaseErr, http.StatusConflict, http.StatusText(http.StatusConflict)},
		{"wrapped database error", fmt.Errorf("request: %w", databaseErr), http.StatusConflict, http.StatusText(http.StatusConflict)},
		{"generic error", errors.New(secret), http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := NewGinAdapter()
			route := &routing.Route{Method: http.MethodGet, Path: "/demo", Metadata: routing.RouteMetadata{Name: "demo"}, Handler: errorRouteHandler{err: test.err}}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/demo", nil)
			a.createGinHandler(route)(ctx)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status: got %d want %d", recorder.Code, test.wantStatus)
			}
			var body map[string]any
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["error"] != test.wantError || body["route_name"] != "demo" {
				t.Fatalf("body: got %#v", body)
			}
			if _, ok := body["details"]; ok || strings.Contains(recorder.Body.String(), secret) {
				t.Fatalf("response leaked execution details: %s", recorder.Body.String())
			}
		})
	}
}
