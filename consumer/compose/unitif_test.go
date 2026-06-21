package compose

import (
	"strings"
	"testing"
)

// portalRealUnit is the "gate passed" form: real portal routes + a nav item.
func portalRealUnit() Unit {
	r := &fakeRoutes{base: "portal", ActiveNav: "portal",
		ListURL: "/portal/list", DetailURL: "/portal/detail/{id}"}
	u := Unit{
		Key:       "conversation.portal",
		EntityKey: "conversation.portal",
		Routes:    r,
		Nav: NavContrib{Permission: "portal:list",
			Items: []NavItem{{Key: "portal-inbox", Route: "portal.list", Label: "Inbox"}}},
	}
	return u
}

// TestUnitIf_GateOn: condition true selects the real unit; its routes appear.
func TestUnitIf_GateOn(t *testing.T) {
	var stubInstalled bool
	stub := StubUnit("conversation.portal", "conversation.portal", func(mc *MountContext) error {
		stubInstalled = true
		return nil
	})

	unit := UnitIf(true, portalRealUnit(), stub)

	eng := &Engine{}
	res, err := eng.Assemble([]Unit{unit}, &recordingRegistrar{})
	if err != nil {
		t.Fatalf("gate-on Assemble: %v", err)
	}
	if _, ok := res.RouteMap["portal.list"]; !ok {
		t.Errorf("gate-on: expected real route portal.list in table, got %v", res.RouteMap)
	}
	if nav := res.Nav["conversation.portal"]; len(nav.Items) != 1 {
		t.Errorf("gate-on: expected 1 nav item, got %d", len(nav.Items))
	}
	if stubInstalled {
		t.Error("gate-on: stub Mount ran, but real unit should have been selected")
	}
}

// TestUnitIf_GateOff: condition false selects the stub; NO real routes, NO nav,
// and the stub's 503 Mount runs. The boot still succeeds (fail-closed nav
// resolution is satisfied because the stub contributes no nav items).
func TestUnitIf_GateOff(t *testing.T) {
	var stub503Routes []string
	stub := StubUnit("conversation.portal", "conversation.portal", func(mc *MountContext) error {
		// Install a 503 responder for the path the real unit would have served,
		// WITHOUT contributing a nav item.
		mc.Routes.GET("/portal/list", nil)
		stub503Routes = append(stub503Routes, "/portal/list")
		return nil
	})

	unit := UnitIf(false, portalRealUnit(), stub)

	reg := &recordingRegistrar{}
	eng := &Engine{}
	res, err := eng.Assemble([]Unit{unit}, reg)
	if err != nil {
		t.Fatalf("gate-off Assemble: %v", err)
	}
	if _, leaked := res.RouteMap["portal.list"]; leaked {
		t.Errorf("gate-off: real route portal.list leaked into table: %v", res.RouteMap)
	}
	if nav := res.Nav["conversation.portal"]; len(nav.Items) != 0 {
		t.Errorf("gate-off: expected 0 nav items from stub, got %d", len(nav.Items))
	}
	if len(stub503Routes) != 1 {
		t.Errorf("gate-off: expected stub 503 handler installed, got %v", stub503Routes)
	}
	// The mount key is still known (cross-unit references can resolve to it).
	if _, ok := res.Nav["conversation.portal"]; !ok {
		t.Error("gate-off: stub mount key missing from Nav map")
	}
}

// TestEnabled_DisabledContributesNothing: Enabled(false, u) yields an inert
// stub — no routes, no nav, no error.
func TestEnabled_DisabledContributesNothing(t *testing.T) {
	eng := &Engine{}
	res, err := eng.Assemble([]Unit{Enabled(false, jobUnit())}, &recordingRegistrar{})
	if err != nil {
		t.Fatalf("Enabled(false) Assemble: %v", err)
	}
	if len(res.RouteMap) != 0 {
		t.Errorf("disabled unit contributed routes: %v", res.RouteMap)
	}
	if items := res.Nav["operation.job"].Items; len(items) != 0 {
		t.Errorf("disabled unit contributed nav items: %v", items)
	}
}

// TestEnabled_EnabledIsPassthrough: Enabled(true, u) == u.
func TestEnabled_EnabledIsPassthrough(t *testing.T) {
	eng := &Engine{}
	res, err := eng.Assemble([]Unit{Enabled(true, jobUnit())}, &recordingRegistrar{})
	if err != nil {
		t.Fatalf("Enabled(true) Assemble: %v", err)
	}
	if _, ok := res.RouteMap["job.list"]; !ok {
		t.Errorf("enabled unit missing job.list: %v", res.RouteMap)
	}
}

// TestUnitIf_StubCannotSmuggleDanglingNav: if a stub (or any unit) contributes
// a nav item without a backing route, the fail-closed phase-3 check still
// catches it — the gate cannot bypass fail-closed resolution.
func TestUnitIf_StubCannotSmuggleDanglingNav(t *testing.T) {
	badStub := Unit{
		Key:       "conversation.portal",
		EntityKey: "conversation.portal",
		// No Routes, but a nav item — dangling by construction.
		Nav: NavContrib{Items: []NavItem{{Key: "ghost", Route: "portal.list", Label: "Inbox"}}},
	}
	unit := UnitIf(false, portalRealUnit(), badStub)

	eng := &Engine{}
	_, err := eng.Assemble([]Unit{unit}, &recordingRegistrar{})
	if err == nil || !strings.Contains(err.Error(), "unresolvable nav reference") {
		t.Fatalf("dangling stub nav: want fail-closed error, got %v", err)
	}
}
