package omnisearch

import "github.com/erniealice/espyna-golang/registry/entityid"

// Category is one compile-time omni-search category descriptor. It binds a
// stable category key (shared verbatim with the postgres adapter's SQL specs and
// with the hybra presentation layer) to the entity whose ":list" gate governs
// the category's visibility in the palette (Q-OMNI-4: no new permission code —
// palette visibility == list-page visibility, structurally).
//
// The registry is the SINGLE source of the wave-1 category set + gate mapping.
// The adapter owns the per-category SQL (column allowlists); this registry owns
// the gate. A registry-agreement test (P2) asserts the two agree and that each
// GateEntity equals the entity's existing list-page gate.
type Category struct {
	// Key is the stable category key (client, subscription, ...). It is the
	// OmniSearchCategoryResults.category value and the adapter spec key.
	Key string
	// GateEntity is the entityid.* constant whose "<entity>:list" gate the use
	// case checks (via ActionGatekeeper) before including the category. A denied
	// gate silently drops the category (never enumerated).
	GateEntity string
}

// wave1Categories is the omni-search registry (wave-1 six + wave-2 staff). The
// slice order is the fixed presentation order (registry order, not ranked).
// Adding a category is an additive registry append + a matching adapter SQL spec
// + lyngua labels — no schema, no new permission.
var wave1Categories = []Category{
	{Key: "client", GateEntity: entityid.Client},
	{Key: "subscription", GateEntity: entityid.Subscription},
	{Key: "subscription_group", GateEntity: entityid.SubscriptionGroup},
	{Key: "plan", GateEntity: entityid.Plan},
	{Key: "price_schedule", GateEntity: entityid.PriceSchedule},
	{Key: "product", GateEntity: entityid.Product},
	// wave-2: staff (Teachers). Gated on staff:list — palette visibility ==
	// staff list-page visibility (Q-OMNI-4, no new permission code). The adapter
	// resolves each staff row's name from the joined user and links to the
	// workspace_user detail page (staff has no detail page of its own).
	{Key: "staff", GateEntity: entityid.Staff},
}

// Registry returns a defensive copy of the compile-time category registry, in
// presentation order. Callers must not mutate the returned slice's backing
// array beyond the copy.
func Registry() []Category {
	out := make([]Category, len(wave1Categories))
	copy(out, wave1Categories)
	return out
}

// categoryByKey resolves a registry entry by key. ok is false for any key not in
// the compile-time registry (an unknown category is a client request error the
// use case rejects; it never reaches the adapter).
func categoryByKey(key string) (Category, bool) {
	for _, c := range wave1Categories {
		if c.Key == key {
			return c, true
		}
	}
	return Category{}, false
}
