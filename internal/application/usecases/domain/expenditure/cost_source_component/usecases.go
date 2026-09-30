// Package cost_source_component holds the use cases of cost_source_component (domain
// expenditure; 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.3): the
// measured slices of a supplier cost that can be allocated to customers or recognised as own
// expense. Authorization: expenditure:read for list/read, expenditure:update for writes (spec
// §6.4: cost-source components use the expenditure codes). Allocation/recognition claims are
// written by other use cases under the row lock; update and delete here are refused once a
// component is claimed. One operation per file (C10): create.go, update.go, delete.go, read.go, list.go.
package cost_source_component

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

// Repositories groups the repository dependencies.
type Repositories struct {
	CostSourceComponent pb.CostSourceComponentDomainServiceServer
	// Expenditure validates that the parent expenditure exists in the caller's workspace.
	Expenditure expenditurepb.ExpenditureDomainServiceServer
	// ExpenditureLineItem validates a request-supplied expenditure_line_item_id belongs to the
	// same expenditure (C5: parent references are read before use; nil = such a reference is refused).
	ExpenditureLineItem expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	// TaxTreatment validates a request-supplied tax_treatment_id exists and is active (a global
	// catalog table: it has no workspace column, so existence is the checkable property).
	TaxTreatment taxtreatmentpb.TaxTreatmentDomainServiceServer
	// Reconciliation collaborators (ReconcileCostSource, read only): the component's allocation
	// batches and shares and the billable charges raised from its recoverable shares.
	AllocationBatch allocationbatchpb.AllocationBatchDomainServiceServer
	AllocationShare allocationsharepb.AllocationShareDomainServiceServer
	BillableCharge  billablechargepb.BillableChargeDomainServiceServer
}

// Services groups the shared service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type core struct {
	repos Repositories
	svc   Services
}

func (c *core) gate(ctx context.Context, action string) error {
	return c.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Expenditure, Action: action})
}

func (c *core) refuse(ctx context.Context, code string, detail ...string) error {
	return refuse(ctx, c.svc.Translator, code, detail...)
}

func (c *core) inTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if c.svc.Transactor == nil || !c.svc.Transactor.SupportsTransactions() {
		return c.refuse(ctx, codeTransactionRequired)
	}
	return c.svc.Transactor.ExecuteInTransaction(ctx, fn)
}

func blank(s string) bool   { return strings.TrimSpace(s) == "" }
func strp(s string) *string { return &s }
func i64p(i int64) *int64   { return &i }

func stamp() (int64, string) {
	now := time.Now()
	return now.UnixMilli(), now.Format(time.RFC3339)
}

// lock takes the component row FOR UPDATE through the adapter's locker. A repository that cannot
// lock (C4) fails closed: there is no unlocked-read fallback. A missing/foreign row is not_found
// only when the adapter says so (ErrLockedRowNotFound); any other repository error is logged and
// returned as an internal error, never reported as not_found (C9).
func (c *core) lock(ctx context.Context, id string) (*pb.CostSourceComponent, error) {
	l, ok := c.repos.CostSourceComponent.(domainports.CostSourceComponentLocker)
	if !ok {
		return nil, c.refuse(ctx, codeLockUnavailable)
	}
	v, err := l.LockCostSourceComponentForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, domainports.ErrLockedRowNotFound) {
			return nil, c.refuse(ctx, codeNotFound)
		}
		log.Printf("cost_source_component: lock component=%s: %v", id, err)
		return nil, fmt.Errorf("cost_source_component: lock: %w", err)
	}
	if v == nil {
		return nil, c.refuse(ctx, codeNotFound)
	}
	return v, nil
}

// expenditureExists proves the parent expenditure is readable in the actor's workspace. The
// generic Read cannot tell "missing" from "failed", so any read failure is logged (operation +
// ids) and refused as reference_invalid, never as not_found.
func (c *core) expenditureExists(ctx context.Context, id string) error {
	if c.repos.Expenditure == nil {
		return c.refuse(ctx, codeValidation, "expenditure repository unavailable (fail closed)")
	}
	resp, err := c.repos.Expenditure.ReadExpenditure(ctx, &expenditurepb.ReadExpenditureRequest{Data: &expenditurepb.Expenditure{Id: id}})
	if err != nil {
		log.Printf("cost_source_component: read expenditure=%s: %v", id, err)
		return c.refuse(ctx, codeReferenceInvalid)
	}
	if resp == nil || len(resp.Data) == 0 {
		return c.refuse(ctx, codeReferenceInvalid)
	}
	return nil
}

// validateReferences proves the request-supplied parent references (C5): the line item must exist
// and belong to the SAME expenditure (the expenditure itself was read in the actor's workspace, so
// this is transitively workspace-safe even when the line-item adapter's scope is parent-join
// shadow mode); the tax treatment must exist and be active.
func (c *core) validateReferences(ctx context.Context, expenditureID string, lineItemID, taxTreatmentID *string) error {
	if lineItemID != nil && !blank(*lineItemID) {
		if c.repos.ExpenditureLineItem == nil {
			return c.refuse(ctx, codeValidation, "line item repository unavailable (fail closed)")
		}
		resp, err := c.repos.ExpenditureLineItem.ReadExpenditureLineItem(ctx, &expenditurelineitempb.ReadExpenditureLineItemRequest{Data: &expenditurelineitempb.ExpenditureLineItem{Id: *lineItemID}})
		if err != nil {
			log.Printf("cost_source_component: read expenditure_line_item=%s: %v", *lineItemID, err)
			return c.refuse(ctx, codeReferenceInvalid)
		}
		if resp == nil || len(resp.Data) == 0 || resp.Data[0].GetExpenditureId() != expenditureID {
			return c.refuse(ctx, codeReferenceInvalid)
		}
	}
	if taxTreatmentID != nil && !blank(*taxTreatmentID) {
		if c.repos.TaxTreatment == nil {
			return c.refuse(ctx, codeValidation, "tax treatment repository unavailable (fail closed)")
		}
		resp, err := c.repos.TaxTreatment.ReadTaxTreatment(ctx, &taxtreatmentpb.ReadTaxTreatmentRequest{Data: &taxtreatmentpb.TaxTreatment{Id: *taxTreatmentID}})
		if err != nil {
			log.Printf("cost_source_component: read tax_treatment=%s: %v", *taxTreatmentID, err)
			return c.refuse(ctx, codeReferenceInvalid)
		}
		if resp == nil || len(resp.Data) == 0 || !resp.Data[0].GetActive() {
			return c.refuse(ctx, codeReferenceInvalid)
		}
	}
	return nil
}

func (c *core) validateFields(ctx context.Context, d *pb.CostSourceComponent) error {
	if d.GetComponentKind() == pb.CostSourceComponentKind_COST_SOURCE_COMPONENT_KIND_UNSPECIFIED {
		return c.refuse(ctx, codeValidation, "component_kind is required")
	}
	if d.GetAmount() <= 0 {
		return c.refuse(ctx, codeValidation, "amount must be greater than zero (centavos)")
	}
	if blank(d.GetCurrency()) {
		return c.refuse(ctx, codeValidation, "currency is required")
	}
	if d.ServiceFrom != nil && d.ServiceTo != nil && d.GetServiceFrom() >= d.GetServiceTo() {
		return c.refuse(ctx, codeValidation, "service_from must be before service_to")
	}
	if d.BasisScale != nil && d.GetBasisScale() < 0 {
		return c.refuse(ctx, codeValidation, "basis_scale must not be negative")
	}
	return nil
}

// UseCases aggregates the cost_source_component use cases.
type UseCases struct {
	CreateCostSourceComponent *CreateCostSourceComponentUseCase
	UpdateCostSourceComponent *UpdateCostSourceComponentUseCase
	DeleteCostSourceComponent *DeleteCostSourceComponentUseCase
	ReadCostSourceComponent   *ReadCostSourceComponentUseCase
	ListCostSourceComponents  *ListCostSourceComponentsUseCase
	ReconcileCostSource       *ReconcileCostSourceUseCase
}

// NewUseCases wires the cost_source_component use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	c := &core{repos: r, svc: s}
	return &UseCases{
		CreateCostSourceComponent: &CreateCostSourceComponentUseCase{c},
		UpdateCostSourceComponent: &UpdateCostSourceComponentUseCase{c},
		DeleteCostSourceComponent: &DeleteCostSourceComponentUseCase{c},
		ReadCostSourceComponent:   &ReadCostSourceComponentUseCase{c},
		ListCostSourceComponents:  &ListCostSourceComponentsUseCase{c},
		ReconcileCostSource:       &ReconcileCostSourceUseCase{c},
	}
}
