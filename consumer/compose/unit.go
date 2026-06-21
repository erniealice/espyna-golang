// Package compose is the composition-v2 "modular home" engine.
//
// It turns a curated list of self-describing entity blocks (Unit) plus
// per-app curation (order, route-key prefix, enable/disable, conditional
// gates) into the derived artifacts an app needs: a merged route table, a
// translation/label bundle, a sidebar contribution map, and the entity
// template filesystems.
//
// Design constraints (from docs/orchestrate/20260612-composition-model/):
//
//   - ZERO domain imports. This package imports only pyeza-internal types
//     (view.RouteRegistrar, pyeza.CommonLabels, types.TableLabels) and the
//     Go standard library (io/fs, encoding/json via the injected overlay).
//     No esqyma / lyngua / espyna / fayna types ever leak in. The unit's
//     typed payloads ride behind RouteSet / any pointers that the OWNING
//     package created and that the package's own Mount closure reads back —
//     type assertions only ever happen inside the owning package, where they
//     are statically sensible.
//
//   - Reflection-free wiring. The engine never reflects to discover or inject
//     dependencies (Q-WIRE-1 removed exactly that). The only reflection in
//     play is encoding/json (in the injected JSON overlay) and the existing
//     pyeza.ValidateLabels label-completeness walk — both pre-existing and
//     opt-in. Deps binding stays a typed package concern (block/catalog.go).
//
//   - FAIL-CLOSED at boot. A missing / unresolvable reference (a NavItem
//     pointing at a route key the route table does not contain, a duplicate
//     mount-id, a route-key collision between two mounts, an overlay parse
//     error) is a BOOT ERROR, never a silent default. This mirrors the
//     AUTHZ_ENFORCE boot-guard posture: the app refuses to start rather than
//     serve a half-wired surface.
//
//   - Mount-id, not entity-id. Unit.Key is the MOUNT identity, not the entity
//     identity, so one entity can mount twice (the dual-mount finding: Product
//     mounts as "inventory" and "supplies", each a re-prefixed / re-anchored
//     variant of the same entity). RouteKeyPrefix namespaces each mount's
//     route-map keys so the two mounts coexist without collision.
package compose

import (
	"io/fs"

	pyeza "github.com/erniealice/pyeza-golang"
	"github.com/erniealice/pyeza-golang/types"
	"github.com/erniealice/pyeza-golang/view"
)

// RouteSet is satisfied by every entity Routes struct today — job.Routes,
// event.Routes, product.Routes, … all already implement RouteMap() (verified:
// packages/fayna-golang/domain/operation/job/routes.go). The engine reads a
// unit's routes purely as data through this interface; it never sees the
// concrete type.
type RouteSet interface {
	// RouteMap returns dot-notation keys ("job.list") to route paths
	// ("/jobs/list/{status}"). Keys are entity-local (prefixed "job."); the
	// engine applies Unit.RouteKeyPrefix to namespace multi-mounts.
	RouteMap() map[string]string
}

// JSONBinding names a lyngua overlay target: a file plus the root key within
// it. The engine applies the override IN PLACE into the descriptor's Routes /
// Labels pointer via the injected Overlay func (see Engine.Overlay).
//
// File is the bundle filename ("route.json", "job.json"); Key is the
// dot-path root ("job", "product_inventory"). An empty Key overlays the whole
// file. An empty File means "no overlay for this binding" (skipped).
type JSONBinding struct {
	File string
	Key  string
}

// NavItem is one sidebar entry the entity can contribute. Route is a
// route-map KEY (entity-local, e.g. "job.list") — the engine resolves it
// AFTER the JSON overlay and AFTER the mount's RouteKeyPrefix is applied, so
// a tier URL rewrite (or a dual-mount namespace shift) flows into the sidebar
// href for free.
type NavItem struct {
	Key        string            // stable item id, e.g. "jobs-draft"
	Route      string            // route-map key, entity-local, e.g. "job.list"
	Params     map[string]string // chi-style {name} substitutions, applied via route.ResolveURL semantics
	Query      string            // optional raw query suffix, e.g. "?status=draft"
	LabelKey   string            // sidebar.json key for the label, e.g. "draft_label"
	IconKey    string            // sidebar.json key for the icon, e.g. "jobs_draft_icon"
	Label      string            // default label fallback ("Draft")
	Icon       string            // default icon fallback ("icon-edit")
	Permission string            // "job:list"; empty inherits NavContrib.Permission
}

// NavContrib is an entity's complete sidebar self-description: a default
// permission plus the items it can place. The app's sidebar declaration
// curates WHICH items go WHERE (NavOf / PickNav) — that is a Phase-1 concern;
// Phase-0 only resolves and validates the contributions.
type NavContrib struct {
	Permission string    // unit default permission, e.g. "job:list"
	Items      []NavItem // every item this entity offers
	AppEntry   *AppEntry // optional top-level app registration for this unit
}

// AppEntry describes how a unit registers as a top-level app in the sidebar
// app switcher. Only the "primary" unit for each app carries an AppEntry;
// supporting units (job_activity, outcome_criteria, etc.) contribute only
// Items, not an AppEntry.
type AppEntry struct {
	Key        string
	Route      string
	Params     map[string]string
	LabelKey   string
	IconKey    string
	Label      string
	Icon       string
	Permission string
}

// Unit is ONE mountable entity block — the self-describing "prefab room" the
// app cherry-picks. Typed payloads (Routes, Labels) are held by pointer; the
// engine overlays JSON into them in phase 1, and the unit's own Mount closure
// (bound by the package catalog) reads the SAME post-overlay pointer in phase
// 2. This single-pointer contract makes the historical routes/labels
// double-load defect structurally impossible.
type Unit struct {
	// Key is the MOUNT id (not the entity id), e.g. "operation.job" or, for a
	// second mount of the same entity, "commerce.product_inventory". Must be
	// unique across the curated list — a duplicate is a boot error.
	Key string

	// EntityKey is the underlying esqyma domain.entity, e.g. "commerce.product".
	// Two units mounting the same entity share an EntityKey but differ in Key.
	// Optional; defaults to Key when empty.
	EntityKey string

	// Routes is a pointer to the package's Routes struct (defaults pre-filled
	// by the descriptor). The engine overlays RouteJSON into it in place.
	Routes RouteSet

	// RouteJSON binds the lyngua route override for this mount, e.g.
	// {"route.json", "product_inventory"} for the inventory mount.
	RouteJSON JSONBinding

	// RouteKeyPrefix namespaces this mount's route-map keys. Empty = use the
	// keys as-is (single mount). For a dual mount, the inventory unit sets
	// prefix "product_inventory" and the engine rewrites "product.list" →
	// "product_inventory.list" (mirroring route_config.go's existing
	// TrimPrefix rewrite), so both mounts coexist with no collision.
	RouteKeyPrefix string

	// Labels is a pointer to the package's Labels struct (defaults pre-filled).
	// Held as any because label structs have no common interface; the owning
	// package asserts its own type back inside Mount.
	Labels any

	// LabelJSON binds the lyngua label override, e.g. {"job.json", "job"}.
	LabelJSON JSONBinding

	// LabelName is the human-readable prefix for pyeza.ValidateLabels warnings,
	// e.g. "JobLabels". Empty disables label validation for this unit.
	LabelName string

	// Templates is the entity's TemplatesFS. The engine collects it for the
	// renderer; nil is allowed (a unit with no templates of its own).
	Templates fs.FS

	// Nav is the entity's sidebar self-description.
	Nav NavContrib

	// Mount builds the view module and registers HTTP routes. It runs in
	// PHASE 2, after every unit's Routes/Labels pointers have been overlaid,
	// so the closure sees final post-override values and can resolve sibling
	// units via the MountContext. Nil is allowed (a data-only unit that
	// contributes routes/labels/nav but registers no handlers itself).
	Mount func(mc *MountContext) error
}

// resolvedRouteMap returns this unit's route map with RouteKeyPrefix applied.
// The base prefix (the segment before the first ".") is replaced by
// RouteKeyPrefix, matching route_config.go's `prefix + TrimPrefix(k, base)`
// rewrite. Keys without a "." are prefixed wholesale.
func (u Unit) resolvedRouteMap() map[string]string {
	if u.Routes == nil {
		return map[string]string{}
	}
	raw := u.Routes.RouteMap()
	if u.RouteKeyPrefix == "" {
		// Return a copy so callers can't mutate the descriptor's view.
		out := make(map[string]string, len(raw))
		for k, v := range raw {
			out[k] = v
		}
		return out
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[u.RouteKeyPrefix+trimBaseSegment(k)] = v
	}
	return out
}

// trimBaseSegment drops the first dot-delimited segment, keeping the leading
// ".". "product.list" -> ".list"; "product" -> "". Used so RouteKeyPrefix
// fully replaces the entity-local base ("product.list" + prefix
// "product_inventory" -> "product_inventory.list").
func trimBaseSegment(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return key[i:]
		}
	}
	return ""
}

// entityKeyOrDefault returns EntityKey, falling back to Key.
func (u Unit) entityKeyOrDefault() string {
	if u.EntityKey != "" {
		return u.EntityKey
	}
	return u.Key
}

// MountContext is what the engine hands every unit's Mount closure in phase 2.
// It carries the shared composition targets plus typed cross-unit lookup.
type MountContext struct {
	// Routes is the registrar modules push HTTP handlers into.
	Routes view.RouteRegistrar

	// Common / Table are the shared label sets every module needs.
	Common pyeza.CommonLabels
	Table  types.TableLabels

	// BusinessType is the configured tier ("professional", "service", …).
	BusinessType string

	// units is the resolved-unit table for cross-unit RoutesOf/LabelsOf
	// lookups. Unexported so the only access is via the typed generic helpers.
	units map[string]*Unit
}

// RoutesOf returns another mounted unit's RouteSet, typed to R, looked up by
// mount key. The bool is false when the unit is not in the curated list OR
// its Routes is not assertable to R — callers degrade exactly the way today's
// optional deps do (CTA hidden, tab empty-state). The assertion is
// statically sensible because it is performed inside the OWNING package's
// Mount closure (e.g. RoutesOf[*job_activity.Routes] inside fayna's catalog).
func RoutesOf[R RouteSet](mc *MountContext, key string) (R, bool) {
	var zero R
	if mc == nil || mc.units == nil {
		return zero, false
	}
	u, ok := mc.units[key]
	if !ok || u.Routes == nil {
		return zero, false
	}
	r, ok := u.Routes.(R)
	if !ok {
		return zero, false
	}
	return r, true
}

// LabelsOf returns another mounted unit's Labels pointer, typed to L, looked
// up by mount key. Same absent/assert semantics as RoutesOf.
func LabelsOf[L any](mc *MountContext, key string) (L, bool) {
	var zero L
	if mc == nil || mc.units == nil {
		return zero, false
	}
	u, ok := mc.units[key]
	if !ok || u.Labels == nil {
		return zero, false
	}
	l, ok := u.Labels.(L)
	if !ok {
		return zero, false
	}
	return l, true
}
