package cost_source_component

import (
	"context"

	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// UpdateCostSourceComponentUseCase updates an UNCLAIMED component under its row lock and bumps
// source_version. The parent expenditure and the claim fields are immutable here; a supplied
// line item / tax treatment is read against the locked row's expenditure (C5).
type UpdateCostSourceComponentUseCase struct{ c *core }

func (uc *UpdateCostSourceComponentUseCase) Execute(ctx context.Context, req *pb.UpdateCostSourceComponentRequest) (*pb.UpdateCostSourceComponentResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, uc.c.refuse(ctx, codeValidation, "id is required")
	}
	var out *pb.UpdateCostSourceComponentResponse
	err := uc.c.inTx(ctx, func(txCtx context.Context) error {
		cur, err := uc.c.lock(txCtx, req.Data.GetId())
		if err != nil {
			return err
		}
		if cur.ClaimKind != nil {
			return uc.c.refuse(txCtx, codeClaimed)
		}
		d := req.Data
		merged := &pb.CostSourceComponent{
			ComponentKind: cur.GetComponentKind(), Amount: cur.GetAmount(), Currency: cur.GetCurrency(),
			ServiceFrom: cur.ServiceFrom, ServiceTo: cur.ServiceTo, BasisScale: cur.BasisScale,
		}
		if d.GetComponentKind() != pb.CostSourceComponentKind_COST_SOURCE_COMPONENT_KIND_UNSPECIFIED {
			merged.ComponentKind = d.GetComponentKind()
		}
		if d.GetAmount() != 0 {
			merged.Amount = d.GetAmount()
		}
		if d.GetCurrency() != "" {
			merged.Currency = d.GetCurrency()
		}
		if d.ServiceFrom != nil {
			merged.ServiceFrom = d.ServiceFrom
		}
		if d.ServiceTo != nil {
			merged.ServiceTo = d.ServiceTo
		}
		if d.BasisScale != nil {
			merged.BasisScale = d.BasisScale
		}
		if err := uc.c.validateFields(txCtx, merged); err != nil {
			return err
		}
		if err := uc.c.validateReferences(txCtx, cur.GetExpenditureId(), d.ExpenditureLineItemId, d.TaxTreatmentId); err != nil {
			return err
		}
		ms, ts := stamp()
		patch := &pb.CostSourceComponent{
			Id:                    cur.GetId(),
			ExpenditureLineItemId: d.ExpenditureLineItemId,
			ComponentKind:         d.GetComponentKind(),
			Description:           d.Description,
			BasisUnit:             d.BasisUnit,
			BasisQuantityScaled:   d.BasisQuantityScaled,
			BasisScale:            d.BasisScale,
			Amount:                d.GetAmount(),
			Currency:              d.GetCurrency(),
			TaxTreatmentId:        d.TaxTreatmentId,
			TaxFact:               d.TaxFact,
			ServiceFrom:           d.ServiceFrom,
			ServiceTo:             d.ServiceTo,
			SourceVersion:         cur.GetSourceVersion() + 1,
			DateModified:          i64p(ms),
			DateModifiedString:    strp(ts),
		}
		out, err = uc.c.repos.CostSourceComponent.UpdateCostSourceComponent(txCtx, &pb.UpdateCostSourceComponentRequest{Data: patch})
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
