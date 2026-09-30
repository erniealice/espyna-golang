// Package billable_charge holds the use cases of the billable_charge entity (domain subscription;
// 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.4): the read use cases
// (list, list page data, read) and AdjustBillableCharge. Charges are created by the allocation
// publish (create_from_allocation.go) and are immutable financial rows: the adapter refuses hard
// delete (C6); a correction is a new signed CORRECTION charge. One operation per file (C10); every
// Execute takes and returns esqyma messages (C1); named refusals carry ErrorCode() (C2, errors.go).
// Authorization: billable_charge:{list,read} (Check); adjust uses CheckStrict (C3).
package billable_charge

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// Repositories groups the repository dependencies.
type Repositories struct {
	BillableCharge billablechargepb.BillableChargeDomainServiceServer
	// ChargeComponent is required by AdjustBillableCharge (W3-B).
	ChargeComponent chargecomponentpb.ChargeComponentDomainServiceServer
}

// Services groups the shared service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// Transactor and IDGenerator are required by AdjustBillableCharge (W3-B).
	Transactor  ports.Transactor
	IDGenerator ports.IDGenerator
}

func (s Services) gate(ctx context.Context, action string) error {
	return s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.BillableCharge, Action: action})
}

// gateStrict is for security-sensitive status verbs (adjust): a would-be deny stays a deny even in
// shadow mode (C3).
func (s Services) gateStrict(ctx context.Context, action string) error {
	return s.ActionGatekeeper.CheckStrict(ctx, &actiongate.CheckActionRequest{Entity: entityid.BillableCharge, Action: action})
}

func (s Services) refuse(ctx context.Context, code string, detail ...string) error {
	return refuse(ctx, s.Translator, code, detail...)
}

// UseCases aggregates the billable_charge use cases.
type UseCases struct {
	ListBillableCharges           *ListBillableChargesUseCase
	GetBillableChargeListPageData *GetBillableChargeListPageDataUseCase
	ReadBillableCharge            *ReadBillableChargeUseCase
	AdjustBillableCharge          *AdjustBillableChargeUseCase
}

// NewUseCases wires the billable_charge use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		ListBillableCharges:           &ListBillableChargesUseCase{r, s},
		GetBillableChargeListPageData: &GetBillableChargeListPageDataUseCase{r, s},
		ReadBillableCharge:            &ReadBillableChargeUseCase{r, s},
		AdjustBillableCharge:          &AdjustBillableChargeUseCase{r, s},
	}
}
