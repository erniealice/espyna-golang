package treasury

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// C32: the S1 collection_application aggregate is nil unless its repository is wired, so a
// consumer's Mount fails closed on a provider without the table.
func TestCollectionApplicationAggregateIsNilWithoutItsRepository(t *testing.T) {
	gate := actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
	if uc := NewUseCases(TreasuryRepositories{}, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate); uc.CollectionApplication != nil {
		t.Fatal("collection_application aggregate must be nil without its repository")
	}
	wired := TreasuryRepositories{CollectionApplication: collectionapplicationpb.UnimplementedCollectionApplicationDomainServiceServer{}}
	if uc := NewUseCases(wired, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate); uc.CollectionApplication == nil || uc.CollectionApplication.ReceiveAndApplyCollection == nil {
		t.Fatal("collection_application aggregate must be built when its repository is wired")
	}
}
