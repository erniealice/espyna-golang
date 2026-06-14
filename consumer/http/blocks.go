package http

// blocks.go — the espyna-side block-assembly KIT (A.1 #10).
//
// These are the framework-owned twins of service-admin's adapters_shared.go
// requireUseCases / assembleEngineBlock helpers. Hoisting them into espyna lets
// every domain block (pyeza.AppOption) be assembled the same way without each
// consumer app re-deriving the compose-engine boilerplate.
//
// R6 (carried VERBATIM): AssembleBlock swallows an Engine.Assemble failure with
// log.Printf + return nil — a block that fails to assemble degrades to empty
// state rather than aborting the whole composition. Tightening this swallow is a
// separate follow-on; the behaviour is preserved byte-for-byte from
// assembleEngineBlock so the migration is non-breaking.

import (
	"fmt"
	"log"
	"strings"

	"github.com/erniealice/espyna-golang/consumer"
	lynguaV1 "github.com/erniealice/lyngua/golang/v1"
	"github.com/erniealice/pyeza-golang"
	"github.com/erniealice/pyeza-golang/compose"
)

// RequireUseCases asserts ctx.UseCases to *consumer.UseCases. Returns
// (uc, true) on success or (nil, false) when the context carries the wrong type
// or a nil aggregate. Every block calls this first. (A.1 #10 — the public twin
// of adapters_shared.go requireUseCases, returning a bool instead of an error so
// the block can decide its own degradation.)
func RequireUseCases(ctx *pyeza.AppContext, block string) (*consumer.UseCases, bool) {
	uc, ok := ctx.UseCases.(*consumer.UseCases)
	if !ok || uc == nil {
		log.Printf("%s: ctx.UseCases must be *consumer.UseCases (got %T)", block, ctx.UseCases)
		return nil, false
	}
	return uc, true
}

// AssembleBlock returns a pyeza.AppOption that runs the common tail of every
// engine block: build a compose.Engine with the standard lyngua overlay,
// Assemble the units into ctx.Routes, log results + label warnings, and merge
// the result into ctx.ComposeResult. (A.1 #10 — the public twin of
// adapters_shared.go assembleEngineBlock.)
//
// R6: an Assemble failure is logged and SWALLOWED (return nil) — verbatim from
// assembleEngineBlock. A missing/typed-wrong Translations provider returns an
// error (the block cannot run its overlay) — also verbatim.
func AssembleBlock(name string, units []compose.Unit, ctx *pyeza.AppContext) pyeza.AppOption {
	return func(_ *pyeza.AppContext) error {
		translations, ok := ctx.Translations.(*lynguaV1.TranslationProvider)
		if !ok || translations == nil {
			return fmt.Errorf("%s: ctx.Translations must be *lynguaV1.TranslationProvider", name)
		}

		eng := compose.Engine{
			BusinessType: ctx.BusinessType,
			Common:       ctx.Common,
			Table:        ctx.Table,
			Overlay: func(b compose.JSONBinding, target any) error {
				return translations.LoadPathIfExists("en", ctx.BusinessType, b.File, b.Key, target)
			},
			Validate: true,
		}

		res, err := eng.Assemble(units, ctx.Routes)
		if err != nil {
			// R6 — log + return nil (verbatim). Degrade to empty state; do NOT
			// abort the whole composition.
			log.Printf("compose: %s engine assembly failed: %v", name, err)
			return nil
		}

		log.Printf("  compose: %s engine — %d units, %d route-map keys, %d templates",
			name, len(units), len(res.RouteMap), len(res.Templates))
		if len(res.LabelWarnings) > 0 {
			log.Printf("  compose: label warnings: %s", strings.Join(res.LabelWarnings, "; "))
		}
		if cr, ok := ctx.ComposeResult.(*compose.Result); ok && cr != nil {
			if err := cr.MergeFrom(res); err != nil {
				return fmt.Errorf("compose merge %s: %w", name, err)
			}
		}
		return nil
	}
}
