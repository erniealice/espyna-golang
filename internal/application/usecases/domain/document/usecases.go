// Package document is an umbrella wrapper exposing the two sub-aggregates
// (attachment + template) of the document domain as a single typed value.
//
// Per docs/plan/20260522-usecases-realignment/ Q-UR4 LOCK: the Aggregate
// surface gets a single Document field instead of two separate fields, so
// callers reach the sub-aggregates via Document.Attachment.X / .Template.X.
package document

import (
	attachment "github.com/erniealice/espyna-golang/internal/application/usecases/domain/document/attachment"
	template "github.com/erniealice/espyna-golang/internal/application/usecases/domain/document/template"
)

// UseCases is the umbrella sub-aggregate for the document domain.
// Sub-aggregate fields may be nil when the corresponding repositories
// are not registered (graceful degradation on non-postgres builds).
type UseCases struct {
	Attachment *attachment.UseCases
	Template   *template.UseCases
}
