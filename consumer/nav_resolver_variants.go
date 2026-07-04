// Package consumer — nav_resolver_variants.go
//
// Additive Pick siblings that carry what a plain Pick cannot: query/fragment
// item variants (one base route projected into several sidebar rows that differ
// only by "?k=v" or "#anchor"), and the mobile bottom-nav return shapes
// (types.BottomNavTab / types.AppGridItem). Every entry point here is a pure
// projection: with no label/icon resolver configured it emits the caller's
// literal label/icon and a href assembled the same way the hand-built rows
// assemble it today, so a byte-identical replacement is a resolver-less call.
//
// pyeza return types are already this package's dependency (Pick returns
// types.SidebarItem), so BottomNavTab / AppGridItem are in-direction — no new
// dependency edge is introduced by the tab/grid siblings.

package consumer

import (
	"strings"

	"github.com/erniealice/pyeza-golang/types"
)

// appendQueryFragment appends an optional query then fragment onto a base href.
// A leading "?" on query and a leading "#" on fragment are optional and are
// stripped if present, so callers may pass either the raw token ("kind=cash")
// or the delimited form ("?kind=cash"). The query delimiter is "?" when the
// base has none yet, otherwise "&", preserving any query the route already
// carries. base is assumed non-empty (callers skip empty-base entries).
func appendQueryFragment(base, query, fragment string) string {
	href := base
	if q := strings.TrimPrefix(query, "?"); q != "" {
		if strings.Contains(href, "?") {
			href += "&" + q
		} else {
			href += "?" + q
		}
	}
	if f := strings.TrimPrefix(fragment, "#"); f != "" {
		href += "#" + f
	}
	return href
}

// HrefWithVariant resolves a NavItem's route and appends an optional query and
// fragment, workspace-prefixing applied by the underlying Href. Returns "" when
// the route does not resolve (fail-closed, matching Href). This is the
// single-href analogue of the per-item variants PickWithVariants produces.
func (nav *NavResolver) HrefWithVariant(unitKey, itemKey, query, fragment string) string {
	base := nav.Href(unitKey, itemKey)
	if base == "" {
		return ""
	}
	return appendQueryFragment(base, query, fragment)
}

// NavVariant declares one query/fragment variant of a base route as a sidebar
// row. The base href is resolved by precedence: a literal Href wins; else an
// AppKey resolves via AppURL (the app-switcher entry route); else UnitKey +
// ItemKey resolves via Href. Query and Fragment are then appended. This covers
// the two hand-built variant families: funding "?kind=…" rows (UnitKey +
// ItemKey sharing one base NavItem) and Home "#anchor" tabs (AppKey sharing one
// app-entry URL).
type NavVariant struct {
	Key      string // produced SidebarItem.Key (distinct per variant)
	UnitKey  string // resolve base via Href(UnitKey, ItemKey)
	ItemKey  string // base NavItem key under UnitKey
	AppKey   string // OR resolve base via AppURL(AppKey)
	Href     string // OR a literal base href, used verbatim (no workspace prefix)
	Query    string // appended as "?k=v" / "&k=v"; leading "?" optional
	Fragment string // appended as "#anchor"; leading "#" optional
	LabelKey string // cascade key; empty derives from Key
	IconKey  string // cascade key; empty derives from Key
	Label    string // fallback label (emitted verbatim when no resolver)
	Icon     string // fallback icon (emitted verbatim when no resolver)

	// Permission overrides the produced row's permission. Empty inherits the
	// base NavItem's permission (then the unit's NavContrib permission) when the
	// base was resolved through UnitKey/ItemKey; literal/AppKey bases with no
	// explicit Permission produce an empty permission.
	Permission string
}

// variantBase resolves a variant's base href and reports whether it resolved.
// A literal Href is always "resolved" (verbatim); AppKey/UnitKey bases are
// resolved-and-non-empty or the variant is dropped (fail-closed, matching Pick).
func (nav *NavResolver) variantBase(v NavVariant) (string, bool) {
	switch {
	case v.Href != "":
		return v.Href, true
	case v.AppKey != "":
		b := nav.AppURL(v.AppKey)
		return b, b != ""
	case v.UnitKey != "":
		b := nav.Href(v.UnitKey, v.ItemKey)
		return b, b != ""
	}
	return "", false
}

// variantPermission applies the same permission inheritance Pick uses: an
// explicit override wins; otherwise, for UnitKey/ItemKey bases, the base
// NavItem's permission, then the unit NavContrib's permission.
func (nav *NavResolver) variantPermission(v NavVariant) string {
	if v.Permission != "" {
		return v.Permission
	}
	if v.UnitKey == "" || nav.result == nil {
		return ""
	}
	nc, ok := nav.result.Nav[v.UnitKey]
	if !ok {
		return ""
	}
	for _, it := range nc.Items {
		if it.Key == v.ItemKey && it.Permission != "" {
			return it.Permission
		}
	}
	return nc.Permission
}

// PickWithVariants projects query/fragment variants into sidebar rows. Variants
// whose base does not resolve are skipped (fail-closed, matching Pick). Labels
// and icons are cascade-resolved when a resolver is configured (keyed by
// LabelKey/IconKey, deriving from the variant Key when those are empty),
// otherwise the literal Label/Icon is emitted.
func (nav *NavResolver) PickWithVariants(variants ...NavVariant) []types.SidebarItem {
	out := make([]types.SidebarItem, 0, len(variants))
	for _, v := range variants {
		base, ok := nav.variantBase(v)
		if !ok {
			continue
		}
		out = append(out, types.SidebarItem{
			Key:        v.Key,
			Label:      nav.resolveLabelKey(v.LabelKey, v.Key, v.Label),
			Icon:       nav.resolveIconKey(v.IconKey, v.Key, v.Icon),
			Href:       appendQueryFragment(base, v.Query, v.Fragment),
			Permission: nav.variantPermission(v),
		})
	}
	return out
}

// specHref resolves a tab/grid spec's href by the same precedence as variants:
// literal wins, then AppKey via AppURL, then UnitKey/ItemKey via Href, then the
// optional query/fragment is appended. Returns "" when nothing resolved.
func (nav *NavResolver) specHref(literal, appKey, unitKey, itemKey, query, fragment string) string {
	base := literal
	if base == "" {
		switch {
		case appKey != "":
			base = nav.AppURL(appKey)
		case unitKey != "":
			base = nav.Href(unitKey, itemKey)
		}
	}
	if base == "" {
		return ""
	}
	return appendQueryFragment(base, query, fragment)
}

// NavTabSpec projects into a types.BottomNavTab. Href precedence mirrors
// NavVariant (literal > AppKey > UnitKey+ItemKey). Unlike the sidebar Pick
// paths, tab labels are NOT key-derived: bottom-nav labels come from a separate
// message bundle, so the cascade applies only when an explicit LabelKey/IconKey
// is set; otherwise the literal Label/Icon is emitted (byte-identical default).
type NavTabSpec struct {
	Key      string
	UnitKey  string
	ItemKey  string
	AppKey   string
	Href     string // literal href (e.g. "#more"), used verbatim
	Query    string
	Fragment string
	LabelKey string // explicit cascade key; no key-derivation for tabs
	IconKey  string
	Label    string
	Icon     string
	Badge    string
	Active   bool
	IsFAB    bool
	FABIcon  string
}

// PickTabs projects bottom-nav tab specs into types.BottomNavTab rows. It is a
// pure projection: it never drops a spec (a tab keeps its slot even if its href
// fails to resolve, matching the hand-built bottom-nav which emits whatever
// Href/AppURL returns). Only the href is resolver-assembled; FAB/active/badge
// flags pass through unchanged.
func (nav *NavResolver) PickTabs(specs ...NavTabSpec) []types.BottomNavTab {
	out := make([]types.BottomNavTab, 0, len(specs))
	for _, s := range specs {
		out = append(out, types.BottomNavTab{
			Key:     s.Key,
			Label:   nav.resolveLabelKey(s.LabelKey, "", s.Label),
			Icon:    nav.resolveIconKey(s.IconKey, "", s.Icon),
			Href:    nav.specHref(s.Href, s.AppKey, s.UnitKey, s.ItemKey, s.Query, s.Fragment),
			Badge:   s.Badge,
			Active:  s.Active,
			IsFAB:   s.IsFAB,
			FABIcon: s.FABIcon,
		})
	}
	return out
}

// NavGridSpec projects into a types.AppGridItem. Same href precedence and same
// explicit-only cascade semantics as NavTabSpec (grid labels also come from the
// app label bundle, so no key-derivation).
type NavGridSpec struct {
	Key        string
	UnitKey    string
	ItemKey    string
	AppKey     string
	Href       string
	Query      string
	Fragment   string
	LabelKey   string
	IconKey    string
	Label      string
	Icon       string
	Group      string
	Permission string
}

// PickGrid projects app-grid specs into types.AppGridItem rows. Pure
// projection (no dropping), href resolver-assembled, Group/Permission passed
// through.
func (nav *NavResolver) PickGrid(specs ...NavGridSpec) []types.AppGridItem {
	out := make([]types.AppGridItem, 0, len(specs))
	for _, s := range specs {
		out = append(out, types.AppGridItem{
			Key:        s.Key,
			Label:      nav.resolveLabelKey(s.LabelKey, "", s.Label),
			Icon:       nav.resolveIconKey(s.IconKey, "", s.Icon),
			Href:       nav.specHref(s.Href, s.AppKey, s.UnitKey, s.ItemKey, s.Query, s.Fragment),
			Group:      s.Group,
			Permission: s.Permission,
		})
	}
	return out
}
