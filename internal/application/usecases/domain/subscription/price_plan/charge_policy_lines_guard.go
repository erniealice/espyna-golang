package price_plan

import (
	"context"
	"errors"
	"fmt"
	"log"

	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	productpriceplanuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/product_price_plan"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// guardOptedInLines is the C12 parent-side half of the charge policy opt-in guard
// (20260927-usage-and-pass-through-charges, R3): the package-line guard only runs when a LINE is
// written, so changing the parent price plan's billing_kind (e.g. RECURRING -> ONE_TIME) or
// amount_basis (-> TOTAL_PACKAGE) after a line opted in would leave a policy bound to a plan that
// no longer allows one. The EFFECTIVE plan (stored row overlaid with the request's non-zero
// billing_kind / amount_basis) is checked against the same rule the line guard uses
// (productpriceplanuc.PricePlanAllowsChargePolicy); a violation is refused while any line still
// holds a charge_policy_id. Read failures fail closed.
func (uc *UpdatePricePlanUseCase) guardOptedInLines(ctx context.Context, patch *priceplanpb.PricePlan) error {
	if !touchesGuardedFields(patch) {
		return nil // neither guarded field is in the request
	}
	if uc.repositories.ProductPricePlan == nil {
		// Fail closed (C12): without the line repository the opted-in lines cannot be checked.
		return productpriceplanuc.GuardRefusal(ctx, uc.services.Translator, "price_plan", productpriceplanuc.GuardCodeUnverifiable, "")
	}
	stored, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{Data: &priceplanpb.PricePlan{Id: patch.GetId()}})
	if err != nil {
		if productpriceplanuc.IsRepoNotFound(err) {
			return nil // the update itself reports the missing row
		}
		log.Printf("price_plan: read price_plan for charge policy guard failed (ids=[%s]): %v", patch.GetId(), err)
		return fmt.Errorf("price_plan: read price_plan: %w", err)
	}
	if stored == nil || len(stored.Data) == 0 {
		return nil
	}
	eff := &priceplanpb.PricePlan{BillingKind: stored.Data[0].GetBillingKind(), AmountBasis: stored.Data[0].GetAmountBasis()}
	if patch.GetBillingKind() != priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED {
		eff.BillingKind = patch.GetBillingKind()
	}
	if patch.GetAmountBasis() != priceplanpb.AmountBasis_AMOUNT_BASIS_UNSPECIFIED {
		eff.AmountBasis = patch.GetAmountBasis()
	}
	if productpriceplanuc.PricePlanAllowsChargePolicy(eff) {
		return nil
	}
	opted, err := uc.optedInLineIDs(ctx, patch.GetId())
	if err != nil {
		return err
	}
	if len(opted) > 0 {
		return productpriceplanuc.GuardRefusal(ctx, uc.services.Translator, "price_plan", productpriceplanuc.GuardCodeBillingKind,
			fmt.Sprintf("%d package line(s) use a charge policy", len(opted)))
	}
	return nil
}

// touchesGuardedFields reports whether the request changes billing_kind or amount_basis, the only
// parent fields the charge policy opt-in rule depends on.
func touchesGuardedFields(patch *priceplanpb.PricePlan) bool {
	return patch != nil && (patch.GetBillingKind() != priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED ||
		patch.GetAmountBasis() != priceplanpb.AmountBasis_AMOUNT_BASIS_UNSPECIFIED)
}

// guardedUpdate runs the opt-in re-check and the write under the price plan's lock in ONE
// transaction, so a concurrent line opt-in (whose guard takes the same lock) cannot slip between
// the check and the write (C12, R4 m3). A request that touches neither guarded field needs no
// lock. A repository that cannot lock or a runtime without transactions refuses (fail closed, C4).
func (uc *UpdatePricePlanUseCase) guardedUpdate(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	write := func(c context.Context) (*priceplanpb.UpdatePricePlanResponse, error) {
		return uc.repositories.PricePlan.UpdatePricePlan(c, req)
	}
	if !touchesGuardedFields(req.Data) {
		return write(ctx)
	}
	locker, canLock := uc.repositories.PricePlan.(domainports.PricePlanLocker)
	tx := uc.services.Transactor
	if !canLock || tx == nil || !tx.SupportsTransactions() {
		return nil, productpriceplanuc.GuardRefusal(ctx, uc.services.Translator, "price_plan", productpriceplanuc.GuardCodeUnverifiable, "")
	}
	var out *priceplanpb.UpdatePricePlanResponse
	err := tx.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		if lErr := locker.LockPricePlanForUpdate(txCtx, req.Data.GetId()); lErr != nil && !productpriceplanuc.IsRepoNotFound(lErr) {
			log.Printf("price_plan: lock for charge policy guard failed (ids=[%s]): %v", req.Data.GetId(), lErr)
			return fmt.Errorf("price_plan: lock price_plan: %w", lErr)
		}
		if gErr := uc.guardOptedInLines(txCtx, req.Data); gErr != nil {
			return gErr
		}
		res, wErr := write(txCtx)
		out = res
		return wErr
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// optedInLineIDs pages through every line of the price plan and returns those bound to a policy.
func (uc *UpdatePricePlanUseCase) optedInLineIDs(ctx context.Context, pricePlanID string) ([]string, error) {
	var out []string
	for page := int32(1); page <= 1000; page++ {
		resp, err := uc.repositories.ProductPricePlan.ListProductPricePlans(ctx, &productpriceplanpb.ListProductPricePlansRequest{
			Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
				Field: "price_plan_id",
				FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
					Value: pricePlanID, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
				}},
			}}},
			Pagination: &commonpb.PaginationRequest{Limit: 100, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}}},
		})
		if err != nil {
			log.Printf("price_plan: list product_price_plan for charge policy guard failed (ids=[%s]): %v", pricePlanID, err)
			return nil, errors.Join(errors.New("price_plan: list product_price_plan"), err)
		}
		for _, l := range resp.GetData() {
			if l.GetPricePlanId() == pricePlanID && l.GetChargePolicyId() != "" {
				out = append(out, l.GetId())
			}
		}
		if len(resp.GetData()) < 100 { // the response carries no pagination metadata: a short page is the last
			return out, nil
		}
	}
	return out, nil
}
