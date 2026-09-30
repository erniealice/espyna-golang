// Package charge_component holds the read use cases of the charge_component entity (domain subscription;
// 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.4): list, list page data,
// read, one operation per file (C10). Components are created with their charge by the allocation
// publish / adjust use cases and are immutable financial rows: the adapter refuses hard delete (C6).
// Authorization: billable_charge:{list,read} (fail closed).
package charge_component

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// Repositories groups the repository dependencies.
type Repositories struct {
	ChargeComponent chargecomponentpb.ChargeComponentDomainServiceServer
}

// Services groups the shared service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func (s Services) gate(ctx context.Context, action string) error {
	return s.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.BillableCharge, Action: action})
}

// UseCases aggregates the charge_component read use cases.
type UseCases struct {
	ListChargeComponents           *ListChargeComponentsUseCase
	GetChargeComponentListPageData *GetChargeComponentListPageDataUseCase
	ReadChargeComponent            *ReadChargeComponentUseCase
}

// NewUseCases wires the charge_component read use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		ListChargeComponents:           &ListChargeComponentsUseCase{r, s},
		GetChargeComponentListPageData: &GetChargeComponentListPageDataUseCase{r, s},
		ReadChargeComponent:            &ReadChargeComponentUseCase{r, s},
	}
}
