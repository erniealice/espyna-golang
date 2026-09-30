package domain

import (
	"context"
	"errors"

	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// Known-cost recovery (S1) hand-written lock ports (20260927-usage-and-pass-through-charges,
// build-spec §6.2/§6.3). The Wave 3 behaviour use cases type-assert these on the generated
// repositories (same pattern as ChargePolicyLocker). Every method requires a trusted workspace
// on ctx AND an ambient transaction (it runs through the tx executor, never the pool), and fails
// closed (not found) for a foreign or missing row.

// ErrLockedRowNotFound is wrapped (%w) by a locker adapter when the row is missing or belongs to
// another workspace, so a use case can tell "not found" (a named refusal) from an infrastructure
// error (timeout, aborted transaction) and never report the latter as not_found (C9).
var ErrLockedRowNotFound = errors.New("row not found")

// ErrImmutableRow is wrapped (%w) by an adapter Delete of an immutable financial row (C6): a
// billable_charge, charge_component or PUBLISHED allocation is corrected by a new row, never removed.
var ErrImmutableRow = errors.New("row is immutable and cannot be deleted")

// CostSourceComponentLocker locks cost_source_component rows FOR UPDATE.
type CostSourceComponentLocker interface {
	LockCostSourceComponentForUpdate(ctx context.Context, id string) (*costsourcecomponentpb.CostSourceComponent, error)
	// LockCostSourceComponentsByExpenditure locks every component of the expenditure, id ascending.
	LockCostSourceComponentsByExpenditure(ctx context.Context, expenditureID string) ([]*costsourcecomponentpb.CostSourceComponent, error)
}

// AllocationBatchLocker locks allocation_batch rows FOR UPDATE.
type AllocationBatchLocker interface {
	LockAllocationBatchForUpdate(ctx context.Context, id string) (*allocationbatchpb.AllocationBatch, error)
	// LockAllocationBatchesByComponent locks every batch of the component, revision ascending.
	LockAllocationBatchesByComponent(ctx context.Context, componentID string) ([]*allocationbatchpb.AllocationBatch, error)
}

// DocumentSeriesLocker locks a document_series row FOR UPDATE (number allocation).
type DocumentSeriesLocker interface {
	LockDocumentSeriesForUpdate(ctx context.Context, id string) (*documentseriespb.DocumentSeries, error)
}

// BillableChargeLocker locks a billable_charge row FOR UPDATE.
type BillableChargeLocker interface {
	LockBillableChargeForUpdate(ctx context.Context, id string) (*billablechargepb.BillableCharge, error)
}

// RecoveryDocumentLocker locks a recovery_document row FOR UPDATE.
type RecoveryDocumentLocker interface {
	LockRecoveryDocumentForUpdate(ctx context.Context, id string) (*recoverydocumentpb.RecoveryDocument, error)
}

// RevenueLocker locks a revenue (invoice) row FOR UPDATE. Receive & apply takes it on every open
// invoice of the client so two concurrent receipts cannot both consume the same open balance.
type RevenueLocker interface {
	LockRevenueForUpdate(ctx context.Context, id string) (*revenuepb.Revenue, error)
}

// CollectionApplicationLocker locks a collection_application row FOR UPDATE. Reversal takes it so two
// concurrent reversals of the same application cannot both write a reversing row.
type CollectionApplicationLocker interface {
	LockCollectionApplicationForUpdate(ctx context.Context, id string) (*collectionapplicationpb.CollectionApplication, error)
}

// CollectionLocker locks a treasury_collection row FOR UPDATE. The legacy collection update/delete
// take it before refusing a receipt that anchors APPLIED applications (build-spec §7c C25).
type CollectionLocker interface {
	LockCollectionForUpdate(ctx context.Context, id string) (*collectionpb.Collection, error)
}
