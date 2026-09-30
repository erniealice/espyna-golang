// Package allocation_share holds the read use cases of the allocation_share entity (domain expenditure;
// 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.4): list, list page data,
// read, one operation per file (C10). Shares are written with their batch by the allocation_batch
// create / update-shares use cases; the adapter refuses deleting a share of a published batch (C6).
// Authorization: allocation_batch:{list,read} (fail closed).
package allocation_share

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
)

// Repositories groups the repository dependencies.
type Repositories struct {
	AllocationShare allocationsharepb.AllocationShareDomainServiceServer
}

// Services groups the shared service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func (s Services) gate(ctx context.Context, action string) error {
	return s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.AllocationBatch, Action: action})
}

// UseCases aggregates the allocation_share read use cases.
type UseCases struct {
	ListAllocationShares           *ListAllocationSharesUseCase
	GetAllocationShareListPageData *GetAllocationShareListPageDataUseCase
	ReadAllocationShare            *ReadAllocationShareUseCase
}

// NewUseCases wires the allocation_share read use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		ListAllocationShares:           &ListAllocationSharesUseCase{r, s},
		GetAllocationShareListPageData: &GetAllocationShareListPageDataUseCase{r, s},
		ReadAllocationShare:            &ReadAllocationShareUseCase{r, s},
	}
}
