package domain

import (
	"fmt"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
	ratingdescriptionsetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	ratingdescriptionsetentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	ratingdescriptionsetproductplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

// fakeRDSProvider is a contracts.RepositoryProvider that registers only the
// entities in `has` — modelling mock_db / firestore, which register no
// rating_description_set* factories (codex-review-impl2 #8).
type fakeRDSProvider struct{ has map[string]bool }

func (p fakeRDSProvider) GetConnection() any { return nil }

func (p fakeRDSProvider) CreateRepository(entity string, _ any, _ string) (any, error) {
	if !p.has[entity] {
		return nil, fmt.Errorf("no repository factory registered for fake:%s", entity)
	}
	switch entity {
	case entityid.RatingDescriptionSet:
		return &ratingdescriptionsetpb.UnimplementedRatingDescriptionSetDomainServiceServer{}, nil
	case entityid.RatingDescriptionSetEntry:
		return &ratingdescriptionsetentrypb.UnimplementedRatingDescriptionSetEntryDomainServiceServer{}, nil
	case entityid.RatingDescriptionSetProductPlan:
		return &ratingdescriptionsetproductplanpb.UnimplementedRatingDescriptionSetProductPlanDomainServiceServer{}, nil
	}
	return nil, fmt.Errorf("unexpected entity %s", entity)
}

func TestNewRatingDescriptionRepositories_OptionalAllOrNothing(t *testing.T) {
	all := map[string]bool{entityid.RatingDescriptionSet: true, entityid.RatingDescriptionSetEntry: true, entityid.RatingDescriptionSetProductPlan: true}
	cases := []struct {
		name    string
		has     map[string]bool
		wantNil bool
	}{
		{"provider without the capability (mock_db/firestore)", map[string]bool{}, true},
		{"partial registration is treated as unsupported", map[string]bool{entityid.RatingDescriptionSet: true}, true},
		{"postgres-like provider registers all three", all, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set, entry, link := newRatingDescriptionRepositories(fakeRDSProvider{has: tc.has}, nil, nil)
			gotNil := set == nil && entry == nil && link == nil
			gotAll := set != nil && entry != nil && link != nil
			if tc.wantNil && !gotNil {
				t.Fatalf("want all nil (capability disabled, no error), got set=%v entry=%v link=%v", set, entry, link)
			}
			if !tc.wantNil && !gotAll {
				t.Fatalf("want all three repositories, got set=%v entry=%v link=%v", set, entry, link)
			}
		})
	}
}
