// Package httperror classifies execution errors into public HTTP responses.
package httperror

import (
	"net/http"

	"github.com/erniealice/espyna-golang/shared/database/model"
)

// Classify returns the safe HTTP status and public message for err. Database
// errors may expose their explicit client status; every other error fails
// closed as an internal server error.
func Classify(err error) (int, string) {
	const fallbackStatus = http.StatusInternalServerError

	dbErr, ok := model.GetDatabaseError(err)
	if ok && dbErr.HTTPStatus >= http.StatusBadRequest && dbErr.HTTPStatus <= 599 {
		if message := http.StatusText(dbErr.HTTPStatus); message != "" {
			return dbErr.HTTPStatus, message
		}
	}

	return fallbackStatus, http.StatusText(fallbackStatus)
}
