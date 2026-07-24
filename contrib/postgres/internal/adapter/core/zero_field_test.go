//go:build postgresql

package core

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	jtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	sgpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
)

// marshalToMap mirrors the storage-bridge marshal hop (default protojson
// options → map) that the adapters run before dbOps.Update. It is byte-identical
// to subscription/protoGradingToMap and the inline marshal in job_template.go.
func marshalToMap(t *testing.T, msg proto.Message) map[string]any {
	t.Helper()
	b, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func TestForceBoolField_SubscriptionGroupActive(t *testing.T) {
	// Deactivate: protojson omits a false active — ForceBoolField must restore it
	// so the column is written on the partial UPDATE (this is D-SG).
	off := &sgpb.SubscriptionGroup{Id: "sg-1", Name: "Section A", Active: false}
	data := marshalToMap(t, off)
	if _, ok := data["active"]; ok {
		t.Fatalf("precondition failed: protojson should omit active=false, got %v", data["active"])
	}
	ForceBoolField(off, data, "active")
	if v, ok := data["active"].(bool); !ok || v != false {
		t.Fatalf("active = %v (ok=%v), want false persisted", data["active"], ok)
	}

	// Reactivate: a true active survives unchanged.
	on := &sgpb.SubscriptionGroup{Id: "sg-1", Name: "Section A", Active: true}
	dataOn := marshalToMap(t, on)
	ForceBoolField(on, dataOn, "active")
	if v, ok := dataOn["active"].(bool); !ok || v != true {
		t.Fatalf("active = %v (ok=%v), want true", dataOn["active"], ok)
	}
}

func TestForceBoolField_JobTemplateActive(t *testing.T) {
	// D-JT: unchecking Active submits active=false, which protojson drops.
	off := &jtpb.JobTemplate{Id: "jt-1", Name: "Curriculum", Active: false}
	data := marshalToMap(t, off)
	if _, ok := data["active"]; ok {
		t.Fatalf("precondition failed: protojson should omit active=false, got %v", data["active"])
	}
	ForceBoolField(off, data, "active")
	if v, ok := data["active"].(bool); !ok || v != false {
		t.Fatalf("active = %v (ok=%v), want false persisted", data["active"], ok)
	}
}

func TestForceBoolField_TouchesOnlyNamedField(t *testing.T) {
	// name + kind + capacity_mode are all at their zero values, so protojson
	// omits them. ForceBoolField("active") must add ONLY active and leave every
	// other zero-valued field absent (no clobber).
	msg := &sgpb.SubscriptionGroup{Id: "sg-1", Active: false}
	data := marshalToMap(t, msg)
	ForceBoolField(msg, data, "active")
	for _, k := range []string{"name", "kind", "capacityMode"} {
		if _, ok := data[k]; ok {
			t.Fatalf("zero-valued %q must stay absent, got %v", k, data[k])
		}
	}
	if v, _ := data["active"].(bool); v != false {
		t.Fatalf("active = %v, want false", data["active"])
	}
}

func TestForceBoolField_NoOpGuards(t *testing.T) {
	msg := &sgpb.SubscriptionGroup{Id: "sg-1", Active: false}
	data := marshalToMap(t, msg)

	// Unknown field name → no-op.
	ForceBoolField(msg, data, "does_not_exist")
	if _, ok := data["does_not_exist"]; ok {
		t.Fatal("unknown field must be a no-op")
	}
	// Non-bool field (name is a string) → no-op.
	ForceBoolField(msg, data, "name")
	if _, ok := data["name"]; ok {
		t.Fatal("non-bool field must be a no-op")
	}
	// nil guards must not panic.
	ForceBoolField(nil, data, "active")
	ForceBoolField(msg, nil, "active")
}
