package compose

// unitif.go — conditional-mount support (the mantra-review R-gate).
//
// A flat []Unit cannot express the pervasive gated-wiring patterns in the
// codebase without imperative code wrapping the list, e.g. the conversation
// portal gate (entydad/block/block.go:491):
//
//	if authzEnforce && portalPhase4Ready && convMod.PortalReady() {
//	    // register REAL portal routes
//	} else {
//	    // register a 503 stub
//	}
//
// UnitIf makes that gate a first-class, declarative element of the curated
// list: it produces the REAL unit when the condition holds, or a stub unit
// otherwise. Either way the engine sees exactly one Unit, so the route table,
// sidebar, and templates stay derivable from the list — the imperative gate
// does not leak back into composition.

// UnitIf returns whenTrue when cond is true, otherwise whenFalse.
//
// Both branches are plain Units, so a gated entity can:
//   - mount real routes vs a 503 stub (pass StubUnit as whenFalse),
//   - mount one feature variant vs another,
//   - swap a fully-wired Mount for an empty data-only contribution.
//
// The condition is evaluated by the APP at curation time (it owns the env /
// capability checks: AUTHZ_ENFORCE, SupplierPortalReady, convMod.PortalReady()).
// The engine never sees cond — it only ever sees the selected Unit, preserving
// the fail-closed, reflection-free contract.
//
// Because the chosen Unit still flows through the engine's phase-3 checks, a
// stub that contributes nav items must still register routes for them (or omit
// the items) — the gate cannot smuggle a dangling reference past fail-closed
// resolution.
func UnitIf(cond bool, whenTrue, whenFalse Unit) Unit {
	if cond {
		return whenTrue
	}
	return whenFalse
}

// StubUnit builds a minimal "disabled" Unit for a gated mount whose real
// surface is unavailable (gate failed). It carries the same mount Key and
// EntityKey (so cross-unit lookups and curation references still resolve to a
// known mount) but NO routes, NO labels, NO templates, and NO nav items — so
// it adds nothing to the route table and contributes no sidebar entries.
//
// Mount is the optional stub handler installer: it is where the caller
// registers the 503 / "feature unavailable" responder for the routes the real
// unit would have served. Pass nil for a pure no-op (the surface simply does
// not exist). When Mount registers stub routes, do so WITHOUT contributing nav
// items, so phase-3 fail-closed resolution stays satisfied.
//
// Usage (the conversation portal gate):
//
//	portalReady := authzEnforce && portalPhase4Ready && convMod.PortalReady()
//	units = append(units, compose.UnitIf(portalReady,
//	    entydadblock.ConversationPortalUnit(uc, infra),        // real routes
//	    compose.StubUnit("conversation.portal", "conversation.portal", mountPortal503),
//	))
func StubUnit(key, entityKey string, mount func(mc *MountContext) error) Unit {
	return Unit{
		Key:       key,
		EntityKey: entityKey,
		Mount:     mount,
		// No Routes / Labels / Templates / Nav — the stub is inert in the
		// derived artifacts. Any 503 handlers are installed imperatively by
		// `mount` and intentionally do NOT appear in the route table.
	}
}

// Enabled is a convenience for the common "mount this unit only if enabled,
// else contribute nothing" case (the enableAll gating that is everywhere).
// When enabled it returns u unchanged; when disabled it returns a StubUnit
// with the same Key/EntityKey and no Mount — the mount is entirely absent
// from the route table and sidebar, the same way deleting its line from the
// curated list would behave, but keeping the line for readability.
func Enabled(enabled bool, u Unit) Unit {
	if enabled {
		return u
	}
	return StubUnit(u.Key, u.entityKeyOrDefault(), nil)
}
