// Package compose provides the shared engine-block assembly helpers every
// domain block package (entydad, centymo, fycha, fayna, cyta) calls from its
// exported EngineBlock(opts...) pyeza.AppOption.
//
// These helpers were app-side (service-admin/internal/composition/
// adapters_shared.go: requireUseCases / assembleEngineBlock) until Wave B D2a.
// They are pure compose-engine glue — they assert ctx.UseCases /
// ctx.Translations, build a compose.Engine, Assemble, log, and MergeFrom into
// ctx.ComposeResult — so they take no app/domain-view import and can live in
// ONE shared location all five block packages import (espyna imports
// pyeza/compose + lyngua + its own consumer; no cycle since espyna/consumer
// imports none of the domain block packages).
package compose

import (
	"fmt"
	"log"
	"strings"

	"github.com/erniealice/espyna-golang/consumer"
	lynguaV1 "github.com/erniealice/lyngua/golang/v1"
	"github.com/erniealice/pyeza-golang"
	"github.com/erniealice/pyeza-golang/compose"
)

// RequireUseCases asserts ctx.UseCases to *consumer.UseCases and returns a
// descriptive error on failure. Every engine block calls this first.
func RequireUseCases(ctx *pyeza.AppContext, block string) (*consumer.UseCases, error) {
	uc, ok := ctx.UseCases.(*consumer.UseCases)
	if !ok || uc == nil {
		return nil, fmt.Errorf("%s: ctx.UseCases must be *consumer.UseCases", block)
	}
	return uc, nil
}

// AssembleEngineBlock is the common tail of every engine block. It:
//  1. asserts ctx.Translations to *lynguaV1.TranslationProvider
//  2. builds a compose.Engine with the standard overlay
//  3. assembles the units
//  4. logs results + label warnings
//  5. merges the result into ctx.ComposeResult
//
// All five engine blocks (fayna, centymo, entydad, fycha, cyta) delegate to
// this after preparing their block-specific UseCases, Infra, and units.
//
// #18 FAIL-LOUD (Wave B D2a, relocated from app-side adapters_shared.go where
// the Assemble error was SWALLOWED with `return nil`): a failed
// eng.Assemble now `return err` so an ENTITY-overlay failure is boot-FATAL via
// the opt loop (the host's WithBlocks loop propagates the error to
// MustBuild → log.Fatalf). This does NOT lock out /auth/login under D2-β
// because the entydad EngineBlock registers auth DIRECTLY and INDEPENDENTLY of
// AllUnits → AssembleEngineBlock (the auth path is decoupled from the entity
// overlay; preserve that decoupling — do not conflate).
func AssembleEngineBlock(name string, units []compose.Unit, ctx *pyeza.AppContext) error {
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
		// #18 FAIL-LOUD: propagate the assembly error so the host's opt loop
		// boot-FATALs instead of silently dark-launching an entity overlay.
		return fmt.Errorf("compose: %s engine assembly failed: %w", name, err)
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
