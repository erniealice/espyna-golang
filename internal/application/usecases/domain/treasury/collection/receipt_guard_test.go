package collection

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

type fakeCollections struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	locked, updated, deleted []string
	lockErr                  error
}

func (f *fakeCollections) LockCollectionForUpdate(_ context.Context, id string) (*collectionpb.Collection, error) {
	if f.lockErr != nil {
		return nil, f.lockErr
	}
	f.locked = append(f.locked, id)
	return &collectionpb.Collection{Id: id}, nil
}
func (f *fakeCollections) UpdateCollection(_ context.Context, r *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	f.updated = append(f.updated, r.GetData().GetId())
	return &collectionpb.UpdateCollectionResponse{Success: true, Data: []*collectionpb.Collection{r.GetData()}}, nil
}
func (f *fakeCollections) DeleteCollection(_ context.Context, r *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	f.deleted = append(f.deleted, r.GetData().GetId())
	return &collectionpb.DeleteCollectionResponse{Success: true}, nil
}

// unlockable has no LockCollectionForUpdate: the guard must fail closed.
type unlockable struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
}

type fakeApplications struct {
	collectionapplicationpb.UnimplementedCollectionApplicationDomainServiceServer
	rows []*collectionapplicationpb.CollectionApplication
}

func (f *fakeApplications) ListCollectionApplications(_ context.Context, r *collectionapplicationpb.ListCollectionApplicationsRequest) (*collectionapplicationpb.ListCollectionApplicationsResponse, error) {
	want := r.GetFilters().GetFilters()[0].GetStringFilter().GetValue()
	var out []*collectionapplicationpb.CollectionApplication
	for _, a := range f.rows {
		if a.GetTreasuryCollectionId() == want {
			out = append(out, a)
		}
	}
	return &collectionapplicationpb.ListCollectionApplicationsResponse{Data: out, Success: true}, nil
}

type tx struct{}

func (tx) SupportsTransactions() bool               { return true }
func (tx) IsTransactionActive(context.Context) bool { return false }
func (tx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func application(collectionID string, status collectionapplicationpb.ApplicationStatus, active bool) *collectionapplicationpb.CollectionApplication {
	return &collectionapplicationpb.CollectionApplication{Id: "ca-" + collectionID, TreasuryCollectionId: collectionID, Status: status, Active: active}
}

func guarded(cols collectionpb.CollectionDomainServiceServer, apps collectionapplicationpb.CollectionApplicationDomainServiceServer) *UseCases {
	return NewUseCases(CollectionRepositories{Collection: cols, CollectionApplication: apps}, CollectionServices{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx{}, Translator: ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	})
}

func userCtx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

func TestUpdateAndDeleteRefuseAReceiptWithAppliedApplications(t *testing.T) {
	cols := &fakeCollections{}
	uc := guarded(cols, &fakeApplications{rows: []*collectionapplicationpb.CollectionApplication{
		application("rcpt-1", collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED, true),
	}})
	_, err := uc.UpdateCollection.Execute(userCtx(), &collectionpb.UpdateCollectionRequest{Data: &collectionpb.Collection{Id: "rcpt-1", Amount: 1}})
	if !usecaseerr.IsCode(err, "receipt_has_applications") {
		t.Fatalf("update: want receipt_has_applications, got %v", err)
	}
	_, err = uc.DeleteCollection.Execute(userCtx(), &collectionpb.DeleteCollectionRequest{Data: &collectionpb.Collection{Id: "rcpt-1"}})
	if !usecaseerr.IsCode(err, "receipt_has_applications") {
		t.Fatalf("delete: want receipt_has_applications, got %v", err)
	}
	if len(cols.updated) != 0 || len(cols.deleted) != 0 {
		t.Fatalf("the refused receipt was written: updated=%v deleted=%v", cols.updated, cols.deleted)
	}
	if len(cols.locked) != 2 {
		t.Fatalf("each check must run under the collection row lock, locked=%v", cols.locked)
	}
}

func TestReversedOrInactiveApplicationsDoNotFreezeTheCollection(t *testing.T) {
	cols := &fakeCollections{}
	uc := guarded(cols, &fakeApplications{rows: []*collectionapplicationpb.CollectionApplication{
		application("rcpt-2", collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_REVERSED, true),
		application("rcpt-2", collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED, false),
		application("other", collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED, true),
	}})
	if _, err := uc.UpdateCollection.Execute(userCtx(), &collectionpb.UpdateCollectionRequest{Data: &collectionpb.Collection{Id: "rcpt-2"}}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := uc.DeleteCollection.Execute(userCtx(), &collectionpb.DeleteCollectionRequest{Data: &collectionpb.Collection{Id: "rcpt-2"}}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(cols.updated) != 1 || len(cols.deleted) != 1 {
		t.Fatalf("updated=%v deleted=%v", cols.updated, cols.deleted)
	}
}

func TestReceiptGuardFailsClosedWithoutALockOrOnALockError(t *testing.T) {
	apps := &fakeApplications{}
	if _, err := guarded(unlockable{}, apps).UpdateCollection.Execute(userCtx(), &collectionpb.UpdateCollectionRequest{Data: &collectionpb.Collection{Id: "x"}}); err == nil {
		t.Fatal("a repository that cannot lock must refuse the update")
	}
	boom := errors.New("connection reset")
	cols := &fakeCollections{lockErr: boom}
	if _, err := guarded(cols, apps).DeleteCollection.Execute(userCtx(), &collectionpb.DeleteCollectionRequest{Data: &collectionpb.Collection{Id: "x"}}); !errors.Is(err, boom) {
		t.Fatalf("an infrastructure lock error must fail the delete, got %v", err)
	}
	// A missing row passes to the legacy path, which reports it as before.
	cols = &fakeCollections{lockErr: domainports.ErrLockedRowNotFound}
	if _, err := guarded(cols, apps).DeleteCollection.Execute(userCtx(), &collectionpb.DeleteCollectionRequest{Data: &collectionpb.Collection{Id: "gone"}}); err != nil || len(cols.deleted) != 1 {
		t.Fatalf("missing row must reach the legacy delete: %v %v", err, cols.deleted)
	}
}

func TestCollectionWithoutApplicationRepositoryKeepsLegacyBehaviour(t *testing.T) {
	if _, err := guarded(unlockable{}, nil).DeleteCollection.Execute(userCtx(), &collectionpb.DeleteCollectionRequest{Data: &collectionpb.Collection{Id: "x"}}); err == nil {
		// unlockable's Unimplemented DeleteCollection returns an error: reaching it proves the guard was skipped.
		t.Fatal("expected the legacy delete's own error")
	} else if usecaseerr.IsCode(err, "receipt_has_applications") {
		t.Fatalf("no application repository: nothing to protect, got %v", err)
	}
}
