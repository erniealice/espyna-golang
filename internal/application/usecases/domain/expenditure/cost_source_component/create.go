package cost_source_component

import (
	"context"
	"fmt"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// CreateCostSourceComponentUseCase creates an unclaimed component (source_version 1). Callers
// cannot set claim fields, workspace or id. The parent expenditure, line item and tax treatment
// are read before use (C5).
type CreateCostSourceComponentUseCase struct{ c *core }

func (uc *CreateCostSourceComponentUseCase) Execute(ctx context.Context, req *pb.CreateCostSourceComponentRequest) (*pb.CreateCostSourceComponentResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if _, err := contextutil.RequireUserIDFromContext(ctx); err != nil {
		return nil, uc.c.refuse(ctx, codeValidation, "authenticated user required")
	}
	if req == nil || req.Data == nil {
		return nil, uc.c.refuse(ctx, codeValidation, "data is required")
	}
	d := req.Data
	if blank(d.GetExpenditureId()) {
		return nil, uc.c.refuse(ctx, codeValidation, "expenditure_id is required")
	}
	if err := uc.c.validateFields(ctx, d); err != nil {
		return nil, err
	}
	if err := uc.c.expenditureExists(ctx, d.GetExpenditureId()); err != nil {
		return nil, err
	}
	if err := uc.c.validateReferences(ctx, d.GetExpenditureId(), d.ExpenditureLineItemId, d.TaxTreatmentId); err != nil {
		return nil, err
	}
	if uc.c.svc.IDGenerator == nil || uc.c.repos.CostSourceComponent == nil {
		return nil, fmt.Errorf("cost_source_component: dependencies unavailable")
	}
	ms, ts := stamp()
	row := &pb.CostSourceComponent{
		Id:                    uc.c.svc.IDGenerator.GenerateID(),
		ExpenditureId:         d.GetExpenditureId(),
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
		SourceVersion:         1,
		Active:                true,
		DateCreated:           i64p(ms),
		DateCreatedString:     strp(ts),
		DateModified:          i64p(ms),
		DateModifiedString:    strp(ts),
	}
	return uc.c.repos.CostSourceComponent.CreateCostSourceComponent(ctx, &pb.CreateCostSourceComponentRequest{Data: row})
}
