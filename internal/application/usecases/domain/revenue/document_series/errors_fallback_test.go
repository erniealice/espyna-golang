package document_series

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"
)

// F7: every refusal carries a readable English fallback (the general Lyngua value), never its raw code.
func TestRefusalFallbacksAreReadableEnglish(t *testing.T) {
	for _, e := range []*usecaseerr.Error{errValidation, errNotFound, errCodeTaken, errRetired, errTransactionRequired} {
		if e.Message() == "" || e.Message() == e.ErrorCode() {
			t.Errorf("%s: fallback is %q, want readable English", e.ErrorCode(), e.Message())
		}
	}
}
