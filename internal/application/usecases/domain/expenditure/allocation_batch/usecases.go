// Package allocation_batch holds the use cases of the allocation_batch entity (domain expenditure;
// 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.3/§6.4): the read use
// cases (list, list page data, read) and the write use cases create / update-shares / publish.
// One operation per file (C10); every Execute takes and returns esqyma messages (C1); named
// refusals carry ErrorCode() (C2, errors.go).
// Authorization: allocation_batch:{list,read} for reads (Check); create/update/publish use CheckStrict (C3).
package allocation_batch

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	chargepolicycomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	chargepolicyversionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// Repositories groups the repository dependencies.
type Repositories struct {
	AllocationBatch allocationbatchpb.AllocationBatchDomainServiceServer
	// Write behaviour (W3-A): required by CreateAllocationBatch / UpdateAllocationBatchShares /
	// PublishAllocationBatch; nil = those use cases fail closed.
	AllocationShare       allocationsharepb.AllocationShareDomainServiceServer
	CostSourceComponent   costsourcecomponentpb.CostSourceComponentDomainServiceServer
	AgreementLineTerm     agreementlinetermpb.AgreementLineTermDomainServiceServer
	ChargePolicyVersion   chargepolicyversionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent chargepolicycomponentpb.ChargePolicyComponentDomainServiceServer
	BillableCharge        billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent       chargecomponentpb.ChargeComponentDomainServiceServer
}

// Services groups the shared service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor  // write use cases (W3-A)
	IDGenerator      ports.IDGenerator // write use cases (W3-A)
}

func (s Services) gate(ctx context.Context, action string) error {
	return s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.AllocationBatch, Action: action})
}

// UseCases aggregates the allocation_batch use cases.
type UseCases struct {
	ListAllocationBatches          *ListAllocationBatchesUseCase
	GetAllocationBatchListPageData *GetAllocationBatchListPageDataUseCase
	ReadAllocationBatch            *ReadAllocationBatchUseCase
	// Write behaviour.
	CreateAllocationBatch       *CreateAllocationBatchUseCase
	UpdateAllocationBatchShares *UpdateAllocationBatchSharesUseCase
	PublishAllocationBatch      *PublishAllocationBatchUseCase
}

// NewUseCases wires the allocation_batch use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	wc := &writeCore{r: r, s: s}
	return &UseCases{
		CreateAllocationBatch:          &CreateAllocationBatchUseCase{wc},
		UpdateAllocationBatchShares:    &UpdateAllocationBatchSharesUseCase{wc},
		PublishAllocationBatch:         &PublishAllocationBatchUseCase{wc},
		ListAllocationBatches:          &ListAllocationBatchesUseCase{r, s},
		GetAllocationBatchListPageData: &GetAllocationBatchListPageDataUseCase{r, s},
		ReadAllocationBatch:            &ReadAllocationBatchUseCase{r, s},
	}
}
