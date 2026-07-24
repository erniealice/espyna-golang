//go:build postgresql

package subscription

import (
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	sgpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
)

// TestProtoGradingToMap_ActivePreserved proves the exact D-SG fix wiring used by
// UpdateSubscriptionGroup: the real storage-bridge marshal (protoGradingToMap)
// omits a false active, and ForceBoolField restores it so a Deactivate persists
// instead of silently no-opping.
func TestProtoGradingToMap_ActivePreserved(t *testing.T) {
	off := &sgpb.SubscriptionGroup{Id: "sg-1", Name: "Section A", Active: false}
	data, err := protoGradingToMap(off)
	if err != nil {
		t.Fatalf("protoGradingToMap: %v", err)
	}
	if _, ok := data["active"]; ok {
		t.Fatalf("precondition: protojson should omit active=false, got %v", data["active"])
	}
	postgresCore.ForceBoolField(off, data, "active")
	if v, ok := data["active"].(bool); !ok || v != false {
		t.Fatalf("active = %v (ok=%v), want false persisted", data["active"], ok)
	}

	on := &sgpb.SubscriptionGroup{Id: "sg-1", Name: "Section A", Active: true}
	dataOn, err := protoGradingToMap(on)
	if err != nil {
		t.Fatalf("protoGradingToMap: %v", err)
	}
	postgresCore.ForceBoolField(on, dataOn, "active")
	if v, ok := dataOn["active"].(bool); !ok || v != true {
		t.Fatalf("active = %v (ok=%v), want true", dataOn["active"], ok)
	}
}
