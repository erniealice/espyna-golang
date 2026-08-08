package httperror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/erniealice/espyna-golang/shared/database/model"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	direct := model.NewDatabaseError("secret database detail", "conflict", http.StatusConflict)
	for _, test := range []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"direct database error", direct, http.StatusConflict, http.StatusText(http.StatusConflict)},
		{"wrapped database error", fmt.Errorf("request failed: %w", direct), http.StatusConflict, http.StatusText(http.StatusConflict)},
		{"generic error", errors.New("secret database detail"), http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)},
		{"invalid database status", model.NewDatabaseError("secret database detail", "bad", 200), http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)},
		{"unknown database status", model.NewDatabaseError("secret database detail", "custom", 499), http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, body := Classify(test.err)
			if status != test.wantStatus || body != test.wantBody {
				t.Fatalf("Classify(%v) = (%d, %q), want (%d, %q)", test.err, status, body, test.wantStatus, test.wantBody)
			}
		})
	}
}
