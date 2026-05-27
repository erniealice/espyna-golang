package collectionmethodgrant

import (
	"testing"

	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// TestQ6_CollectionMethodGrant_HasNoEventShapedFields enforces the Q6 CONFIG-only
// lock (treasury-domain-rebuild phases.md Stage 3 acceptance + entities.md §E-4):
// the collection_method_grant message is CONFIG, never an EVENT. It must carry NO
// usage_count / last_used_at / redemption_count — nor any event-shaped counter of
// any kind. Usage events live on treasury_collection / revenue, never on the grant.
//
// This walks the generated proto message descriptor (reflection over the proto
// message) and fails if any forbidden field name is present. Adding such a field
// to the proto is a Q6 violation that this test catches before review.
func TestQ6_CollectionMethodGrant_HasNoEventShapedFields(t *testing.T) {
	// Explicit denylist of the named event-shaped fields the spec calls out.
	forbidden := map[string]bool{
		"usage_count":      true,
		"usagecount":       true,
		"last_used_at":     true,
		"lastusedat":       true,
		"redemption_count": true,
		"redemptioncount":  true,
	}

	// Broader heuristic: any field whose name embeds an event-shaped substring is
	// suspicious for a CONFIG entity. These catch near-misses (e.g. "use_count",
	// "times_redeemed", "last_redeemed_at") that would also violate Q6.
	suspiciousSubstrings := []string{
		"usage", "used", "redemption", "redeemed", "_count", "last_",
	}

	msg := &grantpb.CollectionMethodGrant{}
	fields := msg.ProtoReflect().Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		name := string(fd.Name()) // proto field name, e.g. "collection_method_id"
		jsonName := fd.JSONName() // camelCase, e.g. "collectionMethodId"

		if forbidden[name] || forbidden[jsonName] {
			t.Errorf("Q6 VIOLATION: collection_method_grant carries forbidden event-shaped field %q — grants are CONFIG, not EVENT; usage lives on treasury_collection/revenue", name)
			continue
		}
		for _, sub := range suspiciousSubstrings {
			if contains(name, sub) {
				t.Errorf("Q6 VIOLATION (heuristic): collection_method_grant field %q embeds event-shaped token %q — grants must not carry usage/redemption counters", name, sub)
				break
			}
		}
	}

	// Sanity: the message must still carry its expected CONFIG fields, proving the
	// reflection walk actually inspected a populated descriptor.
	if fields.ByName("collection_method_id") == nil {
		t.Fatalf("expected collection_method_id field on CollectionMethodGrant; descriptor walk found nothing — test is not exercising the real message")
	}
	if fields.ByName("status") == nil {
		t.Fatalf("expected status field on CollectionMethodGrant")
	}
}

// contains reports whether s contains substr (avoids importing strings just for
// one call in a focused test, keeping the intent explicit).
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
