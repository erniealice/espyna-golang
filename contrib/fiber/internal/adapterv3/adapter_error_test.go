//go:build fiber_v3

package adapterv3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/composition/routing"
	"github.com/erniealice/espyna-golang/shared/database/model"
)

type errorRouteHandler struct{ err error }

func (h errorRouteHandler) Execute(context.Context, proto.Message) (proto.Message, error) {
	return nil, h.err
}

func TestCreateFiberV3Handler_ExecutionErrorsAreSanitized(t *testing.T) {
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
			a := NewFiberV3Adapter()
			app := fiber.New()
			route := &routing.Route{Method: http.MethodGet, Path: "/demo", Metadata: routing.RouteMetadata{Name: "demo"}, Handler: errorRouteHandler{err: test.err}}
			app.Get("/demo", a.createFiberHandler(route))
			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/demo", nil))
			if err != nil {
				t.Fatalf("serve request: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status: got %d want %d", response.StatusCode, test.wantStatus)
			}
			var body map[string]any
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["error"] != test.wantError || body["route_name"] != "demo" {
				t.Fatalf("body: got %#v", body)
			}
			encoded, _ := json.Marshal(body)
			if _, ok := body["details"]; ok || strings.Contains(string(encoded), secret) {
				t.Fatalf("response leaked execution details: %s", encoded)
			}
		})
	}
}
