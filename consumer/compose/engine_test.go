package compose

import (
	"strings"
	"testing"

	"github.com/erniealice/pyeza-golang/view"
)

// --- test fixtures ---------------------------------------------------------

// fakeRoutes is a minimal RouteSet mirroring a real entity Routes struct
// (e.g. job.Routes): it carries an ActiveNav anchor + a couple of routes and
// implements RouteMap() with entity-local dot-keys. The `base` field is the
// dot-prefix the entity uses ("job", "product").
type fakeRoutes struct {
	base      string
	ActiveNav string `json:"active_nav"`
	ListURL   string `json:"list_url"`
	DetailURL string `json:"detail_url"`
}

func (r *fakeRoutes) RouteMap() map[string]string {
	return map[string]string{
		r.base + ".list":   r.ListURL,
		r.base + ".detail": r.DetailURL,
	}
}

type fakeLabels struct {
	Title string `json:"title"`
}

// recordingRegistrar records GET/POST registrations so a unit's Mount can be
// observed without a real HTTP server.
type recordingRegistrar struct {
	gets  []string
	posts []string
}

func (r *recordingRegistrar) GET(path string, v view.View, mw ...string)  { r.gets = append(r.gets, path) }
func (r *recordingRegistrar) POST(path string, v view.View, mw ...string) { r.posts = append(r.posts, path) }

// jobUnit / eventUnit build descriptor-shaped units for two distinct entities.
func jobUnit() Unit {
	r := &fakeRoutes{base: "job", ActiveNav: "job", ListURL: "/jobs/list/{status}", DetailURL: "/jobs/detail/{id}"}
	l := &fakeLabels{Title: "Jobs"}
	return Unit{
		Key:       "operation.job",
		EntityKey: "operation.job",
		Routes:    r,
		Labels:    l,
		LabelName: "JobLabels",
		Nav: NavContrib{
			Permission: "job:list",
			Items: []NavItem{
				{Key: "jobs-draft", Route: "job.list", Params: map[string]string{"status": "draft"}, Label: "Draft"},
				{Key: "jobs-active", Route: "job.list", Params: map[string]string{"status": "active"}, Label: "Active"},
			},
		},
	}
}

func eventUnit() Unit {
	r := &fakeRoutes{base: "event", ActiveNav: "schedule", ListURL: "/events/list", DetailURL: "/events/detail/{id}"}
	l := &fakeLabels{Title: "Events"}
	return Unit{
		Key:       "schedule.event",
		EntityKey: "schedule.event",
		Routes:    r,
		Labels:    l,
		LabelName: "EventLabels",
		Nav: NavContrib{
			Permission: "event:list",
			Items:      []NavItem{{Key: "events-all", Route: "event.list", Label: "All Events"}},
		},
	}
}

// --- tests -----------------------------------------------------------------

// TestAssemble_DerivesRouteTableFromTwoDescriptors covers the core promise:
// the engine derives a merged route table, label list, nav map, and template
// list from >=2 descriptors with no collisions.
func TestAssemble_DerivesRouteTableFromTwoDescriptors(t *testing.T) {
	eng := &Engine{BusinessType: "professional"}
	reg := &recordingRegistrar{}

	res, err := eng.Assemble([]Unit{jobUnit(), eventUnit()}, reg)
	if err != nil {
		t.Fatalf("Assemble: unexpected error: %v", err)
	}

	wantRoutes := map[string]string{
		"job.list":     "/jobs/list/{status}",
		"job.detail":   "/jobs/detail/{id}",
		"event.list":   "/events/list",
		"event.detail": "/events/detail/{id}",
	}
	if len(res.RouteMap) != len(wantRoutes) {
		t.Fatalf("RouteMap size = %d, want %d (%v)", len(res.RouteMap), len(wantRoutes), res.RouteMap)
	}
	for k, v := range wantRoutes {
		if got := res.RouteMap[k]; got != v {
			t.Errorf("RouteMap[%q] = %q, want %q", k, got, v)
		}
	}

	if len(res.Labels) != 2 {
		t.Errorf("Labels len = %d, want 2", len(res.Labels))
	}
	if _, ok := res.Nav["operation.job"]; !ok {
		t.Errorf("Nav missing operation.job")
	}
	if _, ok := res.Nav["schedule.event"]; !ok {
		t.Errorf("Nav missing schedule.event")
	}

	// Nav hrefs resolve with params substituted.
	href, ok := res.ResolveNavHref("", NavItem{Route: "job.list", Params: map[string]string{"status": "draft"}})
	if !ok || href != "/jobs/list/draft" {
		t.Errorf("ResolveNavHref(job.list, status=draft) = %q,%v want /jobs/list/draft,true", href, ok)
	}
}

// TestAssemble_DualMount is the dual-mount finding: the SAME entity (product)
// mounts twice under two mount-ids with distinct RouteKeyPrefix + distinct
// ActiveNav. The engine must namespace each mount's keys so they coexist
// without collision.
func TestAssemble_DualMount(t *testing.T) {
	// Two mounts of the same Product entity, each its own Routes pointer with
	// a different ActiveNav and a re-anchored URL set (as route_config.go does).
	inv := &fakeRoutes{base: "product", ActiveNav: "inventory",
		ListURL: "/inventory/list", DetailURL: "/inventory/detail/{id}"}
	sup := &fakeRoutes{base: "product", ActiveNav: "supplies",
		ListURL: "/supplies/list", DetailURL: "/supplies/detail/{id}"}

	inventoryUnit := Unit{
		Key:            "commerce.product_inventory",
		EntityKey:      "commerce.product",
		Routes:         inv,
		RouteKeyPrefix: "product_inventory",
		Nav: NavContrib{Permission: "product:list",
			Items: []NavItem{{Key: "inv-list", Route: "product.list", Label: "Inventory"}}},
	}
	suppliesUnit := Unit{
		Key:            "commerce.product_supplies",
		EntityKey:      "commerce.product",
		Routes:         sup,
		RouteKeyPrefix: "product_supplies",
		Nav: NavContrib{Permission: "product:list",
			Items: []NavItem{{Key: "sup-list", Route: "product.list", Label: "Supplies"}}},
	}

	eng := &Engine{}
	res, err := eng.Assemble([]Unit{inventoryUnit, suppliesUnit}, &recordingRegistrar{})
	if err != nil {
		t.Fatalf("dual-mount Assemble: unexpected error: %v", err)
	}

	want := map[string]string{
		"product_inventory.list":   "/inventory/list",
		"product_inventory.detail": "/inventory/detail/{id}",
		"product_supplies.list":    "/supplies/list",
		"product_supplies.detail":  "/supplies/detail/{id}",
	}
	if len(res.RouteMap) != len(want) {
		t.Fatalf("dual-mount RouteMap = %v, want %d keys", res.RouteMap, len(want))
	}
	for k, v := range want {
		if got := res.RouteMap[k]; got != v {
			t.Errorf("RouteMap[%q] = %q, want %q", k, got, v)
		}
	}

	// The two mounts must NOT have collided onto a shared "product.*" key.
	if _, leaked := res.RouteMap["product.list"]; leaked {
		t.Errorf("un-prefixed product.list leaked into route table: %v", res.RouteMap)
	}

	// Each mount's nav item resolves to its OWN prefixed route (distinct URLs).
	invHref, ok := res.ResolveNavHref("product_inventory", NavItem{Route: "product.list"})
	if !ok || invHref != "/inventory/list" {
		t.Errorf("inventory nav href = %q,%v want /inventory/list,true", invHref, ok)
	}
	supHref, ok := res.ResolveNavHref("product_supplies", NavItem{Route: "product.list"})
	if !ok || supHref != "/supplies/list" {
		t.Errorf("supplies nav href = %q,%v want /supplies/list,true", supHref, ok)
	}

	// Distinct ActiveNav preserved per mount (mount-context decision).
	if inv.ActiveNav == sup.ActiveNav {
		t.Errorf("dual-mount ActiveNav not distinct: both %q", inv.ActiveNav)
	}
}

// TestAssemble_DuplicateMountKeyFailsClosed: two units sharing a mount-id is a
// boot error (not silent last-write-wins).
func TestAssemble_DuplicateMountKeyFailsClosed(t *testing.T) {
	a := jobUnit()
	b := jobUnit() // same Key "operation.job"
	eng := &Engine{}
	_, err := eng.Assemble([]Unit{a, b}, &recordingRegistrar{})
	if err == nil {
		t.Fatal("duplicate mount key: expected boot error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate mount key") {
		t.Errorf("error = %v, want duplicate-mount-key boot error", err)
	}
}

// TestAssemble_RouteCollisionFailsClosed: two DIFFERENT mounts that produce the
// same route key with different values is a boot error (a dual-mount that
// forgot to set RouteKeyPrefix).
func TestAssemble_RouteCollisionFailsClosed(t *testing.T) {
	inv := &fakeRoutes{base: "product", ActiveNav: "inventory", ListURL: "/inventory/list", DetailURL: "/inventory/d/{id}"}
	sup := &fakeRoutes{base: "product", ActiveNav: "supplies", ListURL: "/supplies/list", DetailURL: "/supplies/d/{id}"}
	// Both omit RouteKeyPrefix => both emit "product.list" with different URLs.
	u1 := Unit{Key: "commerce.product_inventory", EntityKey: "commerce.product", Routes: inv}
	u2 := Unit{Key: "commerce.product_supplies", EntityKey: "commerce.product", Routes: sup}

	eng := &Engine{}
	_, err := eng.Assemble([]Unit{u1, u2}, &recordingRegistrar{})
	if err == nil {
		t.Fatal("route collision: expected boot error, got nil")
	}
	if !strings.Contains(err.Error(), "route key collision") {
		t.Errorf("error = %v, want route-key-collision boot error", err)
	}
}

// TestAssemble_MissingNavRefFailsClosed is the headline fail-closed check: a
// nav item pointing at a route key the table doesn't contain fails the boot.
func TestAssemble_MissingNavRefFailsClosed(t *testing.T) {
	u := jobUnit()
	// Add a nav item that references a non-existent route key.
	u.Nav.Items = append(u.Nav.Items, NavItem{Key: "jobs-ghost", Route: "job.does_not_exist", Label: "Ghost"})

	eng := &Engine{}
	_, err := eng.Assemble([]Unit{u}, &recordingRegistrar{})
	if err == nil {
		t.Fatal("missing nav ref: expected fail-closed boot error, got nil")
	}
	if !strings.Contains(err.Error(), "unresolvable nav reference") {
		t.Errorf("error = %v, want unresolvable-nav-reference boot error", err)
	}
	if !strings.Contains(err.Error(), "job.does_not_exist") {
		t.Errorf("error = %v, want it to name the dangling key job.does_not_exist", err)
	}
}

// TestAssemble_EmptyMountKeyFailsClosed: a unit with no Key is a boot error.
func TestAssemble_EmptyMountKeyFailsClosed(t *testing.T) {
	u := jobUnit()
	u.Key = ""
	eng := &Engine{}
	_, err := eng.Assemble([]Unit{u}, &recordingRegistrar{})
	if err == nil || !strings.Contains(err.Error(), "empty Key") {
		t.Fatalf("empty key: want empty-Key boot error, got %v", err)
	}
}

// TestAssemble_MountRunsAfterResolveWithSiblingLookup verifies phase ordering:
// Mount runs after phase-1 resolve, and a unit's Mount can read a sibling
// unit's typed Routes via RoutesOf — including post-overlay values.
func TestAssemble_MountRunsAfterResolveWithSiblingLookup(t *testing.T) {
	job := jobUnit()
	event := eventUnit()

	var sawSiblingList string
	var mountOrder []string

	job.Mount = func(mc *MountContext) error {
		mountOrder = append(mountOrder, "job")
		// Register a route to prove the registrar is wired.
		mc.Routes.GET("/jobs/list/{status}", nil)
		// Cross-unit typed lookup of the event sibling.
		if er, ok := RoutesOf[*fakeRoutes](mc, "schedule.event"); ok {
			sawSiblingList = er.ListURL
		}
		return nil
	}
	event.Mount = func(mc *MountContext) error {
		mountOrder = append(mountOrder, "event")
		return nil
	}

	reg := &recordingRegistrar{}
	eng := &Engine{}
	_, err := eng.Assemble([]Unit{job, event}, reg)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(mountOrder) != 2 || mountOrder[0] != "job" || mountOrder[1] != "event" {
		t.Errorf("mount order = %v, want [job event]", mountOrder)
	}
	if sawSiblingList != "/events/list" {
		t.Errorf("sibling lookup ListURL = %q, want /events/list", sawSiblingList)
	}
	if len(reg.gets) != 1 || reg.gets[0] != "/jobs/list/{status}" {
		t.Errorf("registrar GETs = %v, want [/jobs/list/{status}]", reg.gets)
	}
}

// TestRoutesOf_AbsentUnit: a lookup for a non-curated unit returns ok=false
// (graceful degrade), not a panic.
func TestRoutesOf_AbsentUnit(t *testing.T) {
	eng := &Engine{}
	var got bool
	probe := jobUnit()
	probe.Mount = func(mc *MountContext) error {
		_, got = RoutesOf[*fakeRoutes](mc, "does.not.exist")
		return nil
	}
	if _, err := eng.Assemble([]Unit{probe}, &recordingRegistrar{}); err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if got {
		t.Error("RoutesOf for absent unit returned ok=true, want false")
	}
}

// --- MergeFrom tests (F5) -------------------------------------------------

// TestMergeFrom_BasicMerge: two disjoint Results merge correctly.
func TestMergeFrom_BasicMerge(t *testing.T) {
	a := NewResult()
	a.RouteMap["job.list"] = "/jobs/list"
	a.RouteMap["job.detail"] = "/jobs/detail/{id}"
	a.Nav["operation.job"] = NavContrib{Permission: "job:list"}

	b := NewResult()
	b.RouteMap["event.list"] = "/events/list"
	b.Nav["schedule.event"] = NavContrib{Permission: "event:list"}

	if err := a.MergeFrom(b); err != nil {
		t.Fatalf("MergeFrom: unexpected error: %v", err)
	}

	if len(a.RouteMap) != 3 {
		t.Errorf("RouteMap size = %d, want 3; contents: %v", len(a.RouteMap), a.RouteMap)
	}
	if got := a.RouteMap["event.list"]; got != "/events/list" {
		t.Errorf("RouteMap[event.list] = %q, want /events/list", got)
	}
	if _, ok := a.Nav["schedule.event"]; !ok {
		t.Error("Nav missing schedule.event after merge")
	}
	if _, ok := a.Nav["operation.job"]; !ok {
		t.Error("Nav missing operation.job after merge (original entry lost)")
	}
}

// TestMergeFrom_NilOther: merging nil does not panic and returns no error.
func TestMergeFrom_NilOther(t *testing.T) {
	a := NewResult()
	a.RouteMap["job.list"] = "/jobs/list"

	if err := a.MergeFrom(nil); err != nil {
		t.Fatalf("MergeFrom(nil): unexpected error: %v", err)
	}
	if len(a.RouteMap) != 1 {
		t.Errorf("RouteMap size = %d after nil merge, want 1", len(a.RouteMap))
	}
}

// TestMergeFrom_SameValueIdempotent: merging a key with the same value is
// accepted (idempotent merge, not an error).
func TestMergeFrom_SameValueIdempotent(t *testing.T) {
	a := NewResult()
	a.RouteMap["job.list"] = "/jobs/list"

	b := NewResult()
	b.RouteMap["job.list"] = "/jobs/list" // same key, same value

	if err := a.MergeFrom(b); err != nil {
		t.Fatalf("MergeFrom same-value: unexpected error: %v", err)
	}
	if got := a.RouteMap["job.list"]; got != "/jobs/list" {
		t.Errorf("RouteMap[job.list] = %q, want /jobs/list", got)
	}
}

// TestMergeFrom_CollisionFailsClosed: merging a key with a DIFFERENT value
// is a collision error (fail-closed).
func TestMergeFrom_CollisionFailsClosed(t *testing.T) {
	a := NewResult()
	a.RouteMap["product.list"] = "/products/list"

	b := NewResult()
	b.RouteMap["product.list"] = "/inventory/list" // same key, different value

	err := a.MergeFrom(b)
	if err == nil {
		t.Fatal("MergeFrom collision: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("error = %v, want collision error", err)
	}
	if !strings.Contains(err.Error(), "product.list") {
		t.Errorf("error = %v, want it to name the colliding key", err)
	}
}

// TestMergeFrom_EmptyIntoEmpty: merging two empty results is fine.
func TestMergeFrom_EmptyIntoEmpty(t *testing.T) {
	a := NewResult()
	b := NewResult()
	if err := a.MergeFrom(b); err != nil {
		t.Fatalf("MergeFrom empty+empty: unexpected error: %v", err)
	}
	if len(a.RouteMap) != 0 {
		t.Errorf("RouteMap should be empty, got %v", a.RouteMap)
	}
}

// --- RequireRoute tests (F2) -----------------------------------------------

// TestRequireRoute_Present: key exists with non-empty value.
func TestRequireRoute_Present(t *testing.T) {
	r := NewResult()
	r.RouteMap["job.list"] = "/jobs/list"

	got, err := r.RequireRoute("job.list")
	if err != nil {
		t.Fatalf("RequireRoute: unexpected error: %v", err)
	}
	if got != "/jobs/list" {
		t.Errorf("RequireRoute = %q, want /jobs/list", got)
	}
}

// TestRequireRoute_Missing: absent key returns error.
func TestRequireRoute_Missing(t *testing.T) {
	r := NewResult()
	_, err := r.RequireRoute("does.not.exist")
	if err == nil {
		t.Fatal("RequireRoute for missing key: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want not-found error", err)
	}
}

// TestRequireRoute_EmptyValue: key exists but maps to "".
func TestRequireRoute_EmptyValue(t *testing.T) {
	r := NewResult()
	r.RouteMap["broken.key"] = ""
	_, err := r.RequireRoute("broken.key")
	if err == nil {
		t.Fatal("RequireRoute for empty value: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty string") {
		t.Errorf("error = %v, want empty-string error", err)
	}
}

// TestRouteOrEmpty_Absent: absent key returns "".
func TestRouteOrEmpty_Absent(t *testing.T) {
	r := NewResult()
	if got := r.RouteOrEmpty("absent.key"); got != "" {
		t.Errorf("RouteOrEmpty = %q, want empty string", got)
	}
}
