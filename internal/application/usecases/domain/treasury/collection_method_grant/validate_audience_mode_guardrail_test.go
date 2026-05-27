package collectionmethodgrant

import (
	"context"
	"testing"

	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
	grantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_grant"
)

// fakeCollectionMethodRepo returns a single CollectionMethod with the configured
// audience_mode, so the guardrail can be exercised without a database.
type fakeCollectionMethodRepo struct {
	collectionmethodpb.UnimplementedCollectionMethodDomainServiceServer
	mode collectionmethodpb.CollectionMethodAudienceMode
}

func (f *fakeCollectionMethodRepo) ReadCollectionMethod(ctx context.Context, req *collectionmethodpb.ReadCollectionMethodRequest) (*collectionmethodpb.ReadCollectionMethodResponse, error) {
	return &collectionmethodpb.ReadCollectionMethodResponse{
		Success: true,
		Data: []*collectionmethodpb.CollectionMethod{
			{Id: req.GetData().GetId(), AudienceMode: f.mode},
		},
	}, nil
}

// fakeGrantRepo serves a fixed set of grants from ListCollectionMethodGrants so
// countActiveGrantsForMethod can be exercised without a database.
type fakeGrantRepo struct {
	grantpb.UnimplementedCollectionMethodGrantDomainServiceServer
	grants []*grantpb.CollectionMethodGrant
}

func (f *fakeGrantRepo) ListCollectionMethodGrants(ctx context.Context, req *grantpb.ListCollectionMethodGrantsRequest) (*grantpb.ListCollectionMethodGrantsResponse, error) {
	return &grantpb.ListCollectionMethodGrantsResponse{Success: true, Data: f.grants}, nil
}

func activeGrant(methodID string) *grantpb.CollectionMethodGrant {
	return &grantpb.CollectionMethodGrant{
		CollectionMethodId: methodID,
		Status:             grantpb.CollectionMethodGrantStatus_COLLECTION_METHOD_GRANT_STATUS_ACTIVE,
	}
}

func TestValidateAudienceModeGuardrail(t *testing.T) {
	const methodID = "cm-1"
	mode := func(m collectionmethodpb.CollectionMethodAudienceMode) *fakeCollectionMethodRepo {
		return &fakeCollectionMethodRepo{mode: m}
	}

	cases := []struct {
		name      string
		repo      *fakeCollectionMethodRepo
		count     int
		expectErr bool
	}{
		// OPEN → zero grants allowed; any grant is rejected.
		{"open_zero_ok", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_OPEN), 0, false},
		{"open_one_rejected", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_OPEN), 1, true},
		// RESTRICTED → at least one grant required.
		{"restricted_zero_rejected", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_RESTRICTED), 0, true},
		{"restricted_one_ok", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_RESTRICTED), 1, false},
		{"restricted_many_ok", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_RESTRICTED), 5, false},
		// SINGLE_CLIENT → exactly one grant required.
		{"single_zero_rejected", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_SINGLE_CLIENT), 0, true},
		{"single_one_ok", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_SINGLE_CLIENT), 1, false},
		{"single_two_rejected", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_SINGLE_CLIENT), 2, true},
		// SEGMENT_SCOPED → reserved v2, always rejected.
		{"segment_scoped_rejected", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_SEGMENT_SCOPED), 1, true},
		// UNSPECIFIED → permissive (legacy methods with no declared policy).
		{"unspecified_permissive", mode(collectionmethodpb.CollectionMethodAudienceMode_COLLECTION_METHOD_AUDIENCE_MODE_UNSPECIFIED), 3, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAudienceModeGuardrail(context.Background(), nil, tc.repo, methodID, tc.count)
			if tc.expectErr && err == nil {
				t.Fatalf("expected guardrail error for count=%d, got nil", tc.count)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("expected no guardrail error for count=%d, got %v", tc.count, err)
			}
		})
	}

	// nil method repo → no-op (mock/dev wiring).
	if err := validateAudienceModeGuardrail(context.Background(), nil, nil, methodID, 99); err != nil {
		t.Fatalf("nil method repo should be a no-op, got %v", err)
	}
}

func TestCountActiveGrantsForMethod(t *testing.T) {
	const methodID = "cm-1"
	repo := &fakeGrantRepo{grants: []*grantpb.CollectionMethodGrant{
		activeGrant(methodID),
		activeGrant(methodID),
		activeGrant("cm-other"), // different method — excluded
		{CollectionMethodId: methodID, Status: grantpb.CollectionMethodGrantStatus_COLLECTION_METHOD_GRANT_STATUS_REVOKED}, // revoked — excluded
	}}

	got, err := countActiveGrantsForMethod(context.Background(), repo, methodID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 2 {
		t.Fatalf("expected 2 ACTIVE grants for %s, got %d", methodID, got)
	}

	// nil grant repo → 0.
	if got, _ := countActiveGrantsForMethod(context.Background(), nil, methodID); got != 0 {
		t.Fatalf("nil grant repo should return 0, got %d", got)
	}
}
