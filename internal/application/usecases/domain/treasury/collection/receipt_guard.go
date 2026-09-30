package collection

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// errReceiptHasApplications refuses an edit or delete of a collection that anchors APPLIED
// collection_application rows (Lyngua collection.errors.receipt_has_applications). Correction goes
// through reversing the applications first (build-spec §7c C25; C6 immutable financial rows).
var errReceiptHasApplications = usecaseerr.New("", "receipt_has_applications",
	"This payment has been applied. Reverse its applications first.")

// refuseAppliedReceipt runs inside the caller's transaction: it locks the collection row, then
// refuses when any active APPLIED collection_application references it. The lock serialises the
// edit against a concurrent application of the same receipt.
//
// A provider without collection_application (applications nil) cannot hold an application row, so
// there is nothing to protect. A missing row passes through so the legacy update/delete report it
// exactly as before; any other lock failure fails closed.
func refuseAppliedReceipt(ctx context.Context, tr ports.Translator,
	collections collectionpb.CollectionDomainServiceServer,
	applications collectionapplicationpb.CollectionApplicationDomainServiceServer, id string,
) error {
	if applications == nil {
		return nil
	}
	locker, ok := collections.(domainports.CollectionLocker)
	if !ok {
		return fmt.Errorf("collection: repository cannot lock rows")
	}
	if _, err := locker.LockCollectionForUpdate(ctx, id); err != nil {
		if usecaseerr.IsNotFound(err) {
			return nil
		}
		return usecaseerr.RepoErr("collection", "lock treasury_collection", err, id)
	}
	apps, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*collectionapplicationpb.CollectionApplication, error) {
		resp, err := applications.ListCollectionApplications(ctx, &collectionapplicationpb.ListCollectionApplicationsRequest{
			Filters: listdata.EqFilter("treasury_collection_id", id), Sort: s, Pagination: p,
		})
		return resp.GetData(), err
	})
	if err != nil {
		return usecaseerr.RepoErr("collection", "list collection_applications", err, id)
	}
	for _, a := range apps {
		if a.GetActive() && a.GetStatus() == collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED {
			return usecaseerr.Localize(ctx, tr, "collection", errReceiptHasApplications)
		}
	}
	return nil
}
