package consumer

import (
	"testing"

	compose "github.com/erniealice/espyna-golang/consumer/compose"
)

// --- fixtures --------------------------------------------------------------

// testResult builds a compose.Result by hand (no Engine needed) so the
// NavResolver has a funding unit (query-variant source), an app.home unit
// (AppEntry + fragment source), and a deliberately dangling item to exercise
// the fail-closed skip. RouteMap + Nav are the only fields a NavResolver reads.
func testResult() *compose.Result {
	res := compose.NewResult()
	res.RouteMap["funding.list"] = "/funding"
	res.RouteMap["home.dashboard_url"] = "/home"

	res.Nav["funding.funding"] = compose.NavContrib{
		Permission: "fund:list",
		Items: []compose.NavItem{
			// no LabelKey/IconKey -> cascade key derives from Key ("sources-all")
			{Key: "sources-all", Route: "funding.list", Label: "All Sources", Icon: "icon-wallet"},
			// explicit LabelKey/IconKey + own permission
			{Key: "sources-draft", Route: "funding.list", LabelKey: "draft_label", IconKey: "draft_icon", Label: "Draft", Icon: "icon-edit", Permission: "fund:draft"},
			// key that misses the resolver map -> resolver returns fallback
			{Key: "sources-mobile", Route: "funding.list", Label: "Mobile", Icon: "icon-phone"},
			// route absent from RouteMap -> Href "" -> Pick skips
			{Key: "dangling", Route: "funding.missing", Label: "Dangling"},
		},
	}
	res.Nav["app.home"] = compose.NavContrib{
		AppEntry: &compose.AppEntry{Key: "home", Route: "home.dashboard_url", Label: "Home", Icon: "icon-home"},
		Items: []compose.NavItem{
			{Key: "overview", Route: "home.dashboard_url", Label: "Overview", Icon: "icon-layout"},
		},
	}
	return res
}

// cascadeResolvers returns a label + icon resolver pair that overrides a few
// keys and returns the fallback for everything else (the contract every
// resolver must honor so a miss stays byte-identical to the default).
func cascadeResolvers() (LabelResolver, IconResolver) {
	labels := map[string]string{
		"draft_label": "Unposted",  // explicit LabelKey hit
		"sources-all": "All Funds", // key-derived hit
	}
	icons := map[string]string{
		"draft_icon":  "icon-pencil",
		"sources-all": "icon-coins",
	}
	label := func(key, fallback string) string {
		if v, ok := labels[key]; ok {
			return v
		}
		return fallback
	}
	icon := func(key, fallback string) string {
		if v, ok := icons[key]; ok {
			return v
		}
		return fallback
	}
	return label, icon
}

// --- 1. cascade resolution (resolver set vs unset) -------------------------

func TestPick_NoResolver_EmitsDescriptorDefaults(t *testing.T) {
	nav := NewNavResolver(testResult())
	got := nav.Pick("funding.funding", "sources-all", "sources-draft", "dangling")

	if len(got) != 2 {
		t.Fatalf("Pick len = %d, want 2 (dangling must be skipped): %+v", len(got), got)
	}
	if got[0].Label != "All Sources" || got[0].Icon != "icon-wallet" {
		t.Errorf("sources-all default label/icon = %q/%q, want All Sources/icon-wallet", got[0].Label, got[0].Icon)
	}
	if got[0].Href != "/funding" {
		t.Errorf("sources-all href = %q, want /funding", got[0].Href)
	}
	if got[0].Permission != "fund:list" {
		t.Errorf("sources-all permission = %q, want inherited fund:list", got[0].Permission)
	}
	if got[1].Label != "Draft" || got[1].Permission != "fund:draft" {
		t.Errorf("sources-draft = %q/%q, want Draft/fund:draft", got[1].Label, got[1].Permission)
	}
}

func TestPick_WithResolver_CascadeAndKeyDerivation(t *testing.T) {
	label, icon := cascadeResolvers()
	nav := NewNavResolver(testResult()).WithLabelResolver(label, icon)
	got := nav.Pick("funding.funding", "sources-all", "sources-draft", "sources-mobile")

	if len(got) != 3 {
		t.Fatalf("Pick len = %d, want 3", len(got))
	}
	// sources-all: empty LabelKey -> derived key "sources-all" hits the cascade.
	if got[0].Label != "All Funds" || got[0].Icon != "icon-coins" {
		t.Errorf("sources-all cascade = %q/%q, want All Funds/icon-coins", got[0].Label, got[0].Icon)
	}
	// sources-draft: explicit LabelKey/IconKey drive the lookup.
	if got[1].Label != "Unposted" || got[1].Icon != "icon-pencil" {
		t.Errorf("sources-draft cascade = %q/%q, want Unposted/icon-pencil", got[1].Label, got[1].Icon)
	}
	// sources-mobile: resolver SET but key misses -> fallback (byte-identical).
	if got[2].Label != "Mobile" || got[2].Icon != "icon-phone" {
		t.Errorf("sources-mobile miss = %q/%q, want Mobile/icon-phone (fallback)", got[2].Label, got[2].Icon)
	}
}

func TestWithLabelResolver_LabelOnly_IconFallsBack(t *testing.T) {
	label, _ := cascadeResolvers()
	nav := NewNavResolver(testResult()).WithLabelResolver(label, nil) // nil icon resolver
	got := nav.Pick("funding.funding", "sources-all")
	if got[0].Label != "All Funds" {
		t.Errorf("label = %q, want All Funds", got[0].Label)
	}
	if got[0].Icon != "icon-wallet" {
		t.Errorf("icon = %q, want default icon-wallet (nil icon resolver)", got[0].Icon)
	}
}

func TestWithWorkspace_PreservesResolver(t *testing.T) {
	label, icon := cascadeResolvers()
	nav := NewNavResolver(testResult()).WithLabelResolver(label, icon)
	wsNav := nav.WithWorkspace("acme", "")

	got := wsNav.Pick("funding.funding", "sources-all")
	if got[0].Href != "/w/acme/funding" {
		t.Errorf("href = %q, want /w/acme/funding", got[0].Href)
	}
	if got[0].Label != "All Funds" {
		t.Errorf("label = %q, want cascade All Funds preserved across WithWorkspace", got[0].Label)
	}
	// original resolver-bearing nav must be unmutated (still no workspace)
	if base := nav.Pick("funding.funding", "sources-all"); base[0].Href != "/funding" {
		t.Errorf("original nav mutated: href = %q, want /funding", base[0].Href)
	}
}

// --- 2. variant href assembly ---------------------------------------------

func TestPickWithVariants_QueryAndFragment(t *testing.T) {
	nav := NewNavResolver(testResult())
	got := nav.PickWithVariants(
		// funding "?kind=" family — shares base item sources-all, inherits perm
		NavVariant{Key: "src-cash", UnitKey: "funding.funding", ItemKey: "sources-all", Query: "kind=cash_on_hand", Label: "Cash"},
		// leading "?" is tolerated and normalized
		NavVariant{Key: "src-bank", UnitKey: "funding.funding", ItemKey: "sources-all", Query: "?kind=bank_account", Label: "Bank", Permission: "fund:list"},
		// home "#anchor" family — AppKey base, leading "#" tolerated
		NavVariant{Key: "overview", AppKey: "app.home", Fragment: "#overview", Label: "Overview"},
		// literal href base, used verbatim
		NavVariant{Key: "integrations", Href: "/integrations", Label: "Integrations"},
		// unresolvable base -> skipped
		NavVariant{Key: "bad", UnitKey: "funding.funding", ItemKey: "nope", Label: "Bad"},
	)

	if len(got) != 4 {
		t.Fatalf("variants len = %d, want 4 (bad base skipped): %+v", len(got), got)
	}
	if got[0].Href != "/funding?kind=cash_on_hand" {
		t.Errorf("cash href = %q, want /funding?kind=cash_on_hand", got[0].Href)
	}
	// permission inherited from the base NavContrib when unset
	if got[0].Permission != "fund:list" {
		t.Errorf("cash permission = %q, want inherited fund:list", got[0].Permission)
	}
	if got[1].Href != "/funding?kind=bank_account" {
		t.Errorf("bank href = %q, want /funding?kind=bank_account (leading ? stripped)", got[1].Href)
	}
	if got[2].Href != "/home#overview" {
		t.Errorf("overview href = %q, want /home#overview", got[2].Href)
	}
	if got[3].Href != "/integrations" {
		t.Errorf("integrations href = %q, want literal /integrations", got[3].Href)
	}
}

func TestPickWithVariants_CascadeApplies(t *testing.T) {
	label, icon := cascadeResolvers()
	nav := NewNavResolver(testResult()).WithLabelResolver(label, icon)
	got := nav.PickWithVariants(
		NavVariant{Key: "src-draft", UnitKey: "funding.funding", ItemKey: "sources-all", Query: "kind=x", LabelKey: "draft_label", IconKey: "draft_icon", Label: "X", Icon: "icon-x"},
	)
	if got[0].Label != "Unposted" || got[0].Icon != "icon-pencil" {
		t.Errorf("variant cascade = %q/%q, want Unposted/icon-pencil", got[0].Label, got[0].Icon)
	}
}

func TestHrefWithVariant(t *testing.T) {
	nav := NewNavResolver(testResult())
	cases := []struct {
		name                    string
		unit, item, query, frag string
		want                    string
	}{
		{"query", "funding.funding", "sources-all", "kind=cash", "", "/funding?kind=cash"},
		{"fragment", "funding.funding", "sources-all", "", "overview", "/funding#overview"},
		{"both", "funding.funding", "sources-all", "kind=cash", "top", "/funding?kind=cash#top"},
		{"prefixed", "funding.funding", "sources-all", "?kind=cash", "#top", "/funding?kind=cash#top"},
		{"missing", "funding.funding", "nope", "kind=cash", "", ""},
	}
	for _, c := range cases {
		if got := nav.HrefWithVariant(c.unit, c.item, c.query, c.frag); got != c.want {
			t.Errorf("%s: HrefWithVariant = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAppendQueryFragment(t *testing.T) {
	cases := []struct{ base, q, f, want string }{
		{"/x", "", "", "/x"},
		{"/x", "a=1", "", "/x?a=1"},
		{"/x", "", "f", "/x#f"},
		{"/x?a=1", "b=2", "", "/x?a=1&b=2"}, // existing query -> "&"
		{"/x?a=1", "b=2", "f", "/x?a=1&b=2#f"},
	}
	for _, c := range cases {
		if got := appendQueryFragment(c.base, c.q, c.f); got != c.want {
			t.Errorf("appendQueryFragment(%q,%q,%q) = %q, want %q", c.base, c.q, c.f, got, c.want)
		}
	}
}

// --- 3. tab / grid projection ---------------------------------------------

func TestPickTabs_Projection(t *testing.T) {
	nav := NewNavResolver(testResult())
	tabs := nav.PickTabs(
		NavTabSpec{Key: "home", AppKey: "app.home", Label: "Home", Icon: "icon-home", Active: true},
		NavTabSpec{Key: "cash", UnitKey: "funding.funding", ItemKey: "sources-all", Label: "Cash", Icon: "icon-cash"},
		NavTabSpec{Key: "book", UnitKey: "funding.funding", ItemKey: "sources-all", Label: "Book", Icon: "icon-plus", IsFAB: true, FABIcon: "icon-calendar-plus"},
		NavTabSpec{Key: "more", Href: "#more", Label: "More", Icon: "icon-menu"},
	)
	if len(tabs) != 4 {
		t.Fatalf("tabs len = %d, want 4", len(tabs))
	}
	if tabs[0].Href != "/home" || !tabs[0].Active {
		t.Errorf("home tab = %q/active=%v, want /home/active=true", tabs[0].Href, tabs[0].Active)
	}
	if tabs[1].Href != "/funding" {
		t.Errorf("cash tab href = %q, want /funding", tabs[1].Href)
	}
	if !tabs[2].IsFAB || tabs[2].FABIcon != "icon-calendar-plus" {
		t.Errorf("book tab FAB = %v/%q, want true/icon-calendar-plus", tabs[2].IsFAB, tabs[2].FABIcon)
	}
	if tabs[3].Href != "#more" {
		t.Errorf("more tab href = %q, want literal #more", tabs[3].Href)
	}
}

func TestPickTabs_NoKeyDerivation(t *testing.T) {
	label, icon := cascadeResolvers()
	nav := NewNavResolver(testResult()).WithLabelResolver(label, icon)
	// Key "sources-all" IS in the resolver map, but tabs must NOT derive a key
	// from Key — only an explicit LabelKey opts a tab into the cascade.
	tabs := nav.PickTabs(
		NavTabSpec{Key: "sources-all", UnitKey: "funding.funding", ItemKey: "sources-all", Label: "Literal"},
		NavTabSpec{Key: "x", UnitKey: "funding.funding", ItemKey: "sources-all", LabelKey: "draft_label", Label: "Fallback"},
	)
	if tabs[0].Label != "Literal" {
		t.Errorf("tab[0] label = %q, want Literal (no key-derivation for tabs)", tabs[0].Label)
	}
	if tabs[1].Label != "Unposted" {
		t.Errorf("tab[1] label = %q, want Unposted (explicit LabelKey)", tabs[1].Label)
	}
}

func TestPickGrid_Projection(t *testing.T) {
	nav := NewNavResolver(testResult())
	grid := nav.PickGrid(
		NavGridSpec{Key: "clients", UnitKey: "funding.funding", ItemKey: "sources-all", Label: "Clients", Icon: "icon-users", Group: "Manage", Permission: "client:list"},
		NavGridSpec{Key: "home", AppKey: "app.home", Label: "Home", Icon: "icon-home", Group: "Manage"},
	)
	if len(grid) != 2 {
		t.Fatalf("grid len = %d, want 2", len(grid))
	}
	if grid[0].Href != "/funding" || grid[0].Group != "Manage" || grid[0].Permission != "client:list" || grid[0].Key != "clients" {
		t.Errorf("grid[0] = %+v, want href /funding group Manage perm client:list key clients", grid[0])
	}
	if grid[1].Href != "/home" {
		t.Errorf("grid[1] href = %q, want /home", grid[1].Href)
	}
}

// --- 4. fail-safe default (no resolver == today's byte-behavior) -----------

func TestFailSafeDefault_LiteralPassthrough(t *testing.T) {
	nav := NewNavResolver(testResult()) // no resolver

	if v := nav.PickWithVariants(NavVariant{Key: "k", UnitKey: "funding.funding", ItemKey: "sources-all", Label: "LitLabel", Icon: "LitIcon"}); v[0].Label != "LitLabel" || v[0].Icon != "LitIcon" {
		t.Errorf("variant no-resolver = %q/%q, want LitLabel/LitIcon", v[0].Label, v[0].Icon)
	}
	if tb := nav.PickTabs(NavTabSpec{Key: "k", UnitKey: "funding.funding", ItemKey: "sources-all", Label: "TabLabel", Icon: "TabIcon"}); tb[0].Label != "TabLabel" || tb[0].Icon != "TabIcon" {
		t.Errorf("tab no-resolver = %q/%q, want TabLabel/TabIcon", tb[0].Label, tb[0].Icon)
	}
	if gr := nav.PickGrid(NavGridSpec{Key: "k", UnitKey: "funding.funding", ItemKey: "sources-all", Label: "GridLabel", Icon: "GridIcon"}); gr[0].Label != "GridLabel" || gr[0].Icon != "GridIcon" {
		t.Errorf("grid no-resolver = %q/%q, want GridLabel/GridIcon", gr[0].Label, gr[0].Icon)
	}
}
