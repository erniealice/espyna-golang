// Package compute_taxes_for_revenue implements the ComputeTaxesForRevenue use case.
//
// Load-bearing design decisions (do not change without rereading
// docs/plan/20260509-tax-integration/plan.md and flow.md):
//
//  1. Compute gates on compute_path_snapshot ∈ {SURCHARGE=1, WITHHOLDING=2}.
//     PERIODIC_ONLY and NONE registrations produce zero RevenueTaxLine rows.
//  2. Fail-closed for STANDARD/REDUCED lines when no SURCHARGE registration exists —
//     this is the key fix vs 20260505's silent mis-behavior.
//  3. asOf is always revenue.revenue_date (passed by caller), so superseded
//     tax_rate rows are correctly pinned to the recognition date.
//  4. Idempotency: DELETE + INSERT revenue_tax_line in one transaction.
//  5. Multi-currency guard is defensive only — compute receives the functional total_amount.
package compute_taxes_for_revenue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

const (
	entityRevenueTaxLine = "revenue_tax_line"

	// compute_path integer enum values (stored as int in DB by the adapter query).
	computePathSurcharge   = 1
	computePathWithholding = 2

	// settlement_status string values.
	settlementStatusOpen                   = "OPEN"
	settlementStatusCashReceivedWHTpending = "CASH_RECEIVED_WHT_PENDING"
	settlementStatusFullySettled           = "FULLY_SETTLED"

	// Tax treatment codes — match seed CSV values.
	treatmentStandard   = "STANDARD"
	treatmentReduced    = "REDUCED"
	treatmentZeroRated  = "ZERO_RATED"
	treatmentExempt     = "EXEMPT"
	treatmentOutOfScope = "OUT_OF_SCOPE"

	// direction strings for the FindApplicable adapter query.
	directionSurcharge   = "SURCHARGE"
	directionWithholding = "WITHHOLDING"
)

// regContext bundles the resolved kind + authority code alongside a registration.
// The proto TaxRegistration only carries FK IDs; this struct holds the resolved values.
type regContext struct {
	reg           *taxregistrationpb.TaxRegistration
	defaultKind   string // TaxRegistrationKind.default_rate_kind
	authorityCode string // TaxAuthority.code (short code like "BIR")
	jurisdiction  string // TaxRegistrationKind.jurisdiction
}

// aggKey identifies an aggregation bucket: one RevenueTaxLine per (tax_rate_id, direction).
type aggKey struct {
	taxRateID string
	direction revenuetaxlinepb.RevenueTaxLineDirection
}

// aggRow accumulates values across line items sharing the same tax rate + direction.
type aggRow struct {
	taxRate     *taxratepb.TaxRate
	regCtx      *regContext
	direction   revenuetaxlinepb.RevenueTaxLineDirection
	taxableBase int64
	taxAmount   int64
	lineIDs     []string
}

// ComputeTaxesRequest is the input to ComputeTaxesForRevenue.
type ComputeTaxesRequest struct {
	// RevenueID is the revenue to compute taxes for.
	RevenueID string
	// WorkspaceID is the owning workspace. Required — Revenue proto has no workspace_id
	// field; the caller (who has the workspace context) must supply it.
	WorkspaceID string
	// AsOf is the effective date for reading registrations and rates.
	// Zero value falls back to revenue.revenue_date.
	AsOf time.Time
	// DryRun, when true, skips all writes and returns preview lines only.
	DryRun bool
	// IsRecompute, when true, applies the recompute blocking rules.
	IsRecompute bool
}

// ComputeTaxesResponse is the result of ComputeTaxesForRevenue.
type ComputeTaxesResponse struct {
	Lines []*revenuetaxlinepb.RevenueTaxLine
}

// TaxRegistrationQueries is the narrow query interface from the tax_registration adapter.
type TaxRegistrationQueries interface {
	FindActiveByComputePath(ctx context.Context, partyType, partyID, computePath, jurisdiction string, asOf time.Time) (*taxregistrationpb.TaxRegistration, error)
}

// FindApplicableQueries is the narrow query interface from the tax_rate adapter.
type FindApplicableQueries interface {
	FindApplicable(ctx context.Context, workspaceID, jurisdiction, authorityCode, kind, treatment, direction string, asOf time.Time) (*taxratepb.TaxRate, error)
}

// FindByCodeQueries is the narrow query interface from the tax_class adapter.
type FindByCodeQueries interface {
	FindByCode(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error)
}

// RevenueTaxLineQueries is the narrow write interface from the revenue_tax_line adapter.
type RevenueTaxLineQueries interface {
	InsertForRevenue(ctx context.Context, lines []*revenuetaxlinepb.RevenueTaxLine) error
	DeleteByRevenueID(ctx context.Context, revenueID string) error
}

// ComputeTaxesForRevenueUseCase implements the tax compute algorithm from flow.md Phase E.
type ComputeTaxesForRevenueUseCase struct {
	repositories ComputeTaxesRepositories
	services     ComputeTaxesServices
}

// NewComputeTaxesForRevenueUseCase creates the use case.
func NewComputeTaxesForRevenueUseCase(
	repos ComputeTaxesRepositories,
	services ComputeTaxesServices,
) *ComputeTaxesForRevenueUseCase {
	return &ComputeTaxesForRevenueUseCase{repositories: repos, services: services}
}

// Execute runs the full tax compute algorithm.
func (uc *ComputeTaxesForRevenueUseCase) Execute(
	ctx context.Context,
	req *ComputeTaxesRequest,
) (*ComputeTaxesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueTaxLine, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil || req.RevenueID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.revenue_id_required",
			"Revenue ID is required for tax computation [DEFAULT]"))
	}
	if req.WorkspaceID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.workspace_id_required",
			"Workspace ID is required for tax computation [DEFAULT]"))
	}

	if !req.DryRun && uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *ComputeTaxesResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("compute taxes failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *ComputeTaxesForRevenueUseCase) executeCore(ctx context.Context, req *ComputeTaxesRequest) (*ComputeTaxesResponse, error) {
	// 1. Load revenue.
	rev, err := uc.readRevenue(ctx, req.RevenueID)
	if err != nil {
		return nil, err
	}

	// 2. Load workspace for functional currency and tax settings.
	ws, err := uc.readWorkspace(ctx, req.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// 3. Guard: tax_computation_enabled=false → no-op (do NOT delete existing rows).
	if !ws.GetTaxComputationEnabled() {
		return &ComputeTaxesResponse{Lines: nil}, nil
	}

	// 4. Multi-currency guard (defensive only).
	if err := uc.checkMultiCurrencyGuard(ctx, rev, ws); err != nil {
		return nil, err
	}

	// 5. Recompute blocking rules (only on IsRecompute=true paths).
	if req.IsRecompute {
		if err := uc.checkRecomputeBlockers(ctx, rev); err != nil {
			return nil, err
		}
	}

	// 6. Resolve asOf.
	asOf := req.AsOf
	if asOf.IsZero() {
		asOf, err = parseDateString(rev.GetRevenueDate())
		if err != nil {
			return nil, fmt.Errorf("invalid revenue_date %q: %w", rev.GetRevenueDate(), err)
		}
	}

	// 7. Resolve jurisdiction.
	jurisdiction := ws.GetHomeJurisdiction()
	if jurisdiction == "" {
		jurisdiction = ws.GetComplianceRegion()
	}

	workspaceID := req.WorkspaceID
	clientID := rev.GetClientId()

	// 8. Load line items.
	lineItems, err := uc.listLineItems(ctx, rev.GetId())
	if err != nil {
		return nil, err
	}
	if len(lineItems) == 0 {
		return &ComputeTaxesResponse{Lines: nil}, nil
	}

	// 9. Read workspace SURCHARGE registration active@asOf, resolved with kind+authority context.
	wsRegCtx, err := uc.findAndResolveRegistration(ctx, "workspace", workspaceID, computePathSurcharge, jurisdiction, asOf)
	if err != nil {
		return nil, fmt.Errorf("workspace SURCHARGE registration lookup: %w", err)
	}

	// 10. Read client WITHHOLDING registration active@asOf.
	var clientRegCtx *regContext
	if clientID != "" {
		clientRegCtx, err = uc.findAndResolveRegistration(ctx, "client", clientID, computePathWithholding, jurisdiction, asOf)
		if err != nil {
			return nil, fmt.Errorf("client WITHHOLDING registration lookup: %w", err)
		}
	}

	taxInclusive := rev.GetTaxInclusivePricingSnapshot()

	// 11. Per-line SURCHARGE + WITHHOLDING passes, aggregated by (tax_rate_id, direction).
	surchargeAgg := make(map[aggKey]*aggRow)
	withholdingAgg := make(map[aggKey]*aggRow)

	for _, line := range lineItems {
		treatmentCode, err := uc.resolveTreatmentCode(ctx, line)
		if err != nil {
			return nil, err
		}

		if err := uc.surchargePassForLine(ctx, line, treatmentCode, wsRegCtx, workspaceID, jurisdiction, taxInclusive, asOf, surchargeAgg); err != nil {
			return nil, err
		}

		if clientRegCtx != nil {
			// Phase 4 C4: WHT must use the net (post-extraction) taxable base.
			// For exclusive pricing: net = raw line_amount.
			// For inclusive pricing: net = round(line_amount / (1 + surchargeRate/10000)).
			// We compute net here in the outer loop so withholdingPassForLine is clean.
			whtNetBase := uc.computeNetTaxableBase(ctx, line, wsRegCtx, workspaceID, taxInclusive, asOf)
			if err := uc.withholdingPassForLine(ctx, line, whtNetBase, clientRegCtx, workspaceID, jurisdiction, asOf, withholdingAgg); err != nil {
				return nil, err
			}
		}
	}

	// 12. Flatten aggregated rows into RevenueTaxLine records.
	now := time.Now()
	var taxLines []*revenuetaxlinepb.RevenueTaxLine
	for _, row := range surchargeAgg {
		taxLines = append(taxLines, uc.buildRevenueTaxLine(workspaceID, rev.GetId(), row, now))
	}
	for _, row := range withholdingAgg {
		taxLines = append(taxLines, uc.buildRevenueTaxLine(workspaceID, rev.GetId(), row, now))
	}

	// 13. Dry-run: return preview without writing.
	if req.DryRun {
		return &ComputeTaxesResponse{Lines: taxLines}, nil
	}

	// 14. Idempotency: DELETE then INSERT in the same transaction.
	if err := uc.deleteAndInsertTaxLines(ctx, rev.GetId(), taxLines); err != nil {
		return nil, err
	}

	// 15. Update revenue denorm fields.
	if err := uc.updateRevenueDenorm(ctx, rev, taxLines); err != nil {
		return nil, fmt.Errorf("update revenue denorm: %w", err)
	}

	return &ComputeTaxesResponse{Lines: taxLines}, nil
}

// findAndResolveRegistration finds the registration for (partyType, partyID, computePath, jurisdiction)
// and resolves the associated kind's default_rate_kind and authority code.
func (uc *ComputeTaxesForRevenueUseCase) findAndResolveRegistration(
	ctx context.Context,
	partyType, partyID string,
	computePathInt int,
	jurisdiction string,
	asOf time.Time,
) (*regContext, error) {
	if uc.repositories.TaxRegistration == nil {
		return nil, nil
	}
	q, ok := uc.repositories.TaxRegistration.(TaxRegistrationQueries)
	if !ok {
		return nil, nil
	}

	computePathStr := fmt.Sprintf("%d", computePathInt)
	reg, err := q.FindActiveByComputePath(ctx, partyType, partyID, computePathStr, jurisdiction, asOf)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, nil
	}

	// Resolve TaxRegistrationKind to get default_rate_kind and jurisdiction.
	var defaultKind, kindJurisdiction string
	if reg.GetTaxRegistrationKindId() != "" && uc.repositories.TaxRegistrationKind != nil {
		kindResp, err := uc.repositories.TaxRegistrationKind.ReadTaxRegistrationKind(ctx, &taxregistrationkindpb.ReadTaxRegistrationKindRequest{
			Data: &taxregistrationkindpb.TaxRegistrationKind{Id: reg.GetTaxRegistrationKindId()},
		})
		if err == nil && kindResp != nil && len(kindResp.GetData()) > 0 {
			kind := kindResp.GetData()[0]
			defaultKind = kind.GetDefaultRateKind()
			kindJurisdiction = kind.GetJurisdiction()
		}
	}
	if kindJurisdiction == "" {
		kindJurisdiction = jurisdiction
	}

	// Resolve TaxAuthority to get the short code (e.g. "BIR").
	var authorityCode string
	if reg.GetTaxAuthorityId() != "" && uc.repositories.TaxAuthority != nil {
		authResp, err := uc.repositories.TaxAuthority.ReadTaxAuthority(ctx, &taxauthoritypb.ReadTaxAuthorityRequest{
			Data: &taxauthoritypb.TaxAuthority{Id: reg.GetTaxAuthorityId()},
		})
		if err == nil && authResp != nil && len(authResp.GetData()) > 0 {
			authorityCode = authResp.GetData()[0].GetCode()
		}
	}

	return &regContext{
		reg:           reg,
		defaultKind:   defaultKind,
		authorityCode: authorityCode,
		jurisdiction:  kindJurisdiction,
	}, nil
}

// surchargePassForLine performs the SURCHARGE computation for one line.
func (uc *ComputeTaxesForRevenueUseCase) surchargePassForLine(
	ctx context.Context,
	line *revenuelineitempb.RevenueLineItem,
	treatmentCode string,
	wsRegCtx *regContext,
	workspaceID, jurisdiction string,
	taxInclusive bool,
	asOf time.Time,
	agg map[aggKey]*aggRow,
) error {
	// EXEMPT and OUT_OF_SCOPE: no SURCHARGE row.
	if treatmentCode == treatmentExempt || treatmentCode == treatmentOutOfScope {
		return nil
	}

	// ZERO_RATED: write a row with tax_amount=0 (audit trail).
	// Phase 4 M1: tax_rate_id is NULL when no applicable rate row exists for ZERO_RATED.
	// The aggKey uses a synthetic string for internal map deduplication only; the resulting
	// RevenueTaxLine will have TaxRateId=nil (nil pointer in proto optional field).
	if treatmentCode == treatmentZeroRated {
		if wsRegCtx == nil {
			return nil // No registration → no audit row for ZERO_RATED.
		}
		rate, _ := uc.findApplicableRate(ctx, workspaceID, wsRegCtx, treatmentCode, directionSurcharge, asOf)
		// Use the rate ID if found; otherwise use a stable synthetic key for map deduplication only.
		// The aggRow.taxRate field is nil when there is no rate, so TaxRateId will be nil in the output.
		aggKeyID := "zero-rated-null:" + wsRegCtx.reg.GetId()
		if rate != nil {
			aggKeyID = rate.GetId()
		}
		key := aggKey{taxRateID: aggKeyID, direction: revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE}
		if existing := agg[key]; existing == nil {
			agg[key] = &aggRow{
				taxRate:     rate, // nil is valid — buildRevenueTaxLine omits TaxRateId when nil.
				regCtx:      wsRegCtx,
				direction:   revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE,
				taxableBase: line.GetLineAmount(),
				taxAmount:   0,
				lineIDs:     []string{line.GetId()},
			}
		} else {
			existing.taxableBase += line.GetLineAmount()
			existing.lineIDs = append(existing.lineIDs, line.GetId())
		}
		return nil
	}

	// STANDARD / REDUCED: fail-closed if no workspace SURCHARGE registration.
	if wsRegCtx == nil {
		return fmt.Errorf(
			"workspace has no SURCHARGE-path registration in jurisdiction %q but product/line treatment is %s. "+
				"Either add a registration that can collect surcharge tax, or change the treatment to EXEMPT/OUT_OF_SCOPE",
			jurisdiction, treatmentCode)
	}

	// Fail-closed if no default_rate_kind on the registration kind.
	if wsRegCtx.defaultKind == "" {
		return fmt.Errorf(
			"workspace SURCHARGE registration kind has no default_rate_kind configured in jurisdiction %q", jurisdiction)
	}

	// Lookup the applicable rate.
	rate, err := uc.findApplicableRate(ctx, workspaceID, wsRegCtx, treatmentCode, directionSurcharge, asOf)
	if err != nil {
		return err
	}
	if rate == nil {
		return fmt.Errorf(
			"no active rate found for kind %q, treatment %q, on %s in %q",
			wsRegCtx.defaultKind, treatmentCode, asOf.Format("2006-01-02"), wsRegCtx.jurisdiction)
	}

	// Inclusive vs exclusive arithmetic.
	lineAmount := line.GetLineAmount()
	var taxableBase int64
	if taxInclusive {
		grossFactor := 1.0 + float64(rate.GetRateBasisPoints())/10000.0
		taxableBase = int64(math.Round(float64(lineAmount) / grossFactor))
	} else {
		taxableBase = lineAmount
	}
	taxAmount := int64(math.Round(float64(taxableBase) * float64(rate.GetRateBasisPoints()) / 10000.0))

	key := aggKey{taxRateID: rate.GetId(), direction: revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE}
	if existing := agg[key]; existing == nil {
		agg[key] = &aggRow{
			taxRate:     rate,
			regCtx:      wsRegCtx,
			direction:   revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE,
			taxableBase: taxableBase,
			taxAmount:   taxAmount,
			lineIDs:     []string{line.GetId()},
		}
	} else {
		existing.taxableBase += taxableBase
		existing.taxAmount += taxAmount
		existing.lineIDs = append(existing.lineIDs, line.GetId())
	}
	return nil
}

// withholdingPassForLine performs the WITHHOLDING computation for one line.
// netTaxableBase is the post-SURCHARGE-extraction base (pre-computed by the caller):
// exclusive mode: equals line_amount; inclusive mode: equals round(line_amount/(1+rate)).
func (uc *ComputeTaxesForRevenueUseCase) withholdingPassForLine(
	ctx context.Context,
	line *revenuelineitempb.RevenueLineItem,
	netTaxableBase int64,
	clientRegCtx *regContext,
	workspaceID, jurisdiction string,
	asOf time.Time,
	agg map[aggKey]*aggRow,
) error {
	// Resolve withholding class code: line snapshot → product → "" (skip line).
	whtClassCode := uc.resolveWithholdingClassCode(ctx, line)
	if whtClassCode == "" {
		return nil
	}

	// Lookup TaxClass by code + WITHHOLDING direction.
	taxClass, err := uc.findTaxClassByCode(ctx, whtClassCode, directionWithholding)
	if err != nil {
		return err
	}
	if taxClass == nil {
		return nil // TaxClass not found — skip gracefully.
	}

	// If requires_counterparty_role is set, verify the client registration's party_role matches.
	// Note: TaxClassCounterpartyRole and TaxRegistrationPartyRoleSnapshot are separate enums
	// with different integer values for the same semantic (BUYER=1 vs BUYER=2). Compare by name.
	if taxClass.RequiresCounterpartyRole != nil {
		if clientRegCtx == nil || clientRegCtx.reg == nil {
			return nil
		}
		requiredRoleName := taxClass.RequiresCounterpartyRole.String()     // e.g. "TAX_CLASS_COUNTERPARTY_ROLE_BUYER"
		actualRoleName := clientRegCtx.reg.GetPartyRoleSnapshot().String() // e.g. "TAX_REGISTRATION_PARTY_ROLE_SNAPSHOT_BUYER"
		// Both end with _BUYER or _SELLER — compare the suffix.
		requiredSuffix := ""
		if strings.HasSuffix(requiredRoleName, "_BUYER") {
			requiredSuffix = "BUYER"
		} else if strings.HasSuffix(requiredRoleName, "_SELLER") {
			requiredSuffix = "SELLER"
		}
		actualSuffix := ""
		if strings.HasSuffix(actualRoleName, "_BUYER") {
			actualSuffix = "BUYER"
		} else if strings.HasSuffix(actualRoleName, "_SELLER") {
			actualSuffix = "SELLER"
		}
		if requiredSuffix == "" || requiredSuffix != actualSuffix {
			return nil // Role mismatch — skip line.
		}
	}

	// Lookup rate by (kind=tax_class.default_rate_kind, direction=WITHHOLDING, asOf).
	rateKind := taxClass.GetDefaultRateKind()
	if rateKind == "" {
		return nil // No rate kind configured — skip.
	}

	rate, err := uc.findApplicableRateByKind(ctx, workspaceID, clientRegCtx.jurisdiction, clientRegCtx.authorityCode, rateKind, directionWithholding, asOf)
	if err != nil {
		return err
	}
	if rate == nil {
		return fmt.Errorf(
			"no active WITHHOLDING rate found for kind %q on %s in %q",
			rateKind, asOf.Format("2006-01-02"), jurisdiction)
	}

	// Taxable base is the pre-computed net base (exclusive: = line_amount; inclusive: = extracted net).
	taxableBase := netTaxableBase
	taxAmount := int64(math.Round(float64(taxableBase) * float64(rate.GetRateBasisPoints()) / 10000.0))

	key := aggKey{taxRateID: rate.GetId(), direction: revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_WITHHOLDING}
	if existing := agg[key]; existing == nil {
		agg[key] = &aggRow{
			taxRate:     rate,
			regCtx:      clientRegCtx,
			direction:   revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_WITHHOLDING,
			taxableBase: taxableBase,
			taxAmount:   taxAmount,
			lineIDs:     []string{line.GetId()},
		}
	} else {
		existing.taxableBase += taxableBase
		existing.taxAmount += taxAmount
		existing.lineIDs = append(existing.lineIDs, line.GetId())
	}
	return nil
}

// resolveTreatmentCode resolves the treatment code for a line item.
// Order: line snapshot → product.tax_treatment_id → "STANDARD" fallback.
func (uc *ComputeTaxesForRevenueUseCase) resolveTreatmentCode(ctx context.Context, line *revenuelineitempb.RevenueLineItem) (string, error) {
	if snap := line.GetTaxTreatmentSnapshot(); snap != "" {
		return snap, nil
	}
	if prdID := line.GetProductId(); prdID != "" && uc.repositories.Product != nil {
		prod, _ := uc.readProduct(ctx, prdID)
		if prod != nil && prod.GetTaxTreatmentId() != "" && uc.repositories.TaxTreatment != nil {
			tr, _ := uc.readTaxTreatment(ctx, prod.GetTaxTreatmentId())
			if tr != nil && tr.GetCode() != "" {
				return tr.GetCode(), nil
			}
		}
	}
	return treatmentStandard, nil
}

// resolveWithholdingClassCode resolves the withholding class code for a line item.
// Order: line snapshot → product.withholding_class_id → "" (skip line).
func (uc *ComputeTaxesForRevenueUseCase) resolveWithholdingClassCode(ctx context.Context, line *revenuelineitempb.RevenueLineItem) string {
	if snap := line.GetWithholdingClassSnapshot(); snap != "" {
		return snap
	}
	if prdID := line.GetProductId(); prdID != "" && uc.repositories.Product != nil {
		prod, _ := uc.readProduct(ctx, prdID)
		if prod != nil && prod.GetWithholdingClassId() != "" && uc.repositories.TaxClass != nil {
			resp, err := uc.repositories.TaxClass.ReadTaxClass(ctx, &taxclasspb.ReadTaxClassRequest{
				Data: &taxclasspb.TaxClass{Id: prod.GetWithholdingClassId()},
			})
			if err == nil && resp != nil && len(resp.GetData()) > 0 {
				return resp.GetData()[0].GetCode()
			}
		}
	}
	return ""
}

// computeNetTaxableBase returns the net (post-SURCHARGE-extraction) taxable base for a line.
// For exclusive pricing: net = line_amount (no extraction needed).
// For inclusive pricing: the price already includes the surcharge, so we back-extract to get net:
//
//	net = round(line_amount / (1 + surcharge_rate_bp / 10000))
//
// If the surcharge rate cannot be resolved (no wsRegCtx, adapter unavailable), falls back to
// line_amount — a conservative assumption that avoids a fatal error on the WHT pass.
func (uc *ComputeTaxesForRevenueUseCase) computeNetTaxableBase(
	ctx context.Context,
	line *revenuelineitempb.RevenueLineItem,
	wsRegCtx *regContext,
	workspaceID string,
	taxInclusive bool,
	asOf time.Time,
) int64 {
	lineAmount := line.GetLineAmount()
	if !taxInclusive || wsRegCtx == nil {
		return lineAmount
	}

	// Resolve treatment code to find the applicable SURCHARGE rate for extraction.
	treatmentCode, err := uc.resolveTreatmentCode(ctx, line)
	if err != nil || treatmentCode == treatmentExempt || treatmentCode == treatmentOutOfScope {
		return lineAmount
	}

	rate, err := uc.findApplicableRate(ctx, workspaceID, wsRegCtx, treatmentCode, directionSurcharge, asOf)
	if err != nil || rate == nil || rate.GetRateBasisPoints() == 0 {
		// ZERO_RATED or no rate → no extraction needed.
		return lineAmount
	}

	grossFactor := 1.0 + float64(rate.GetRateBasisPoints())/10000.0
	return int64(math.Round(float64(lineAmount) / grossFactor))
}

// findApplicableRate wraps FindApplicable using the regContext's resolved kind + authority.
func (uc *ComputeTaxesForRevenueUseCase) findApplicableRate(
	ctx context.Context,
	workspaceID string,
	regCtx *regContext,
	treatment, direction string,
	asOf time.Time,
) (*taxratepb.TaxRate, error) {
	if uc.repositories.TaxRate == nil {
		return nil, nil
	}
	q, ok := uc.repositories.TaxRate.(FindApplicableQueries)
	if !ok {
		return nil, nil
	}
	return q.FindApplicable(ctx, workspaceID, regCtx.jurisdiction, regCtx.authorityCode, regCtx.defaultKind, treatment, direction, asOf)
}

// findApplicableRateByKind wraps FindApplicable with an explicit kind (for WITHHOLDING).
func (uc *ComputeTaxesForRevenueUseCase) findApplicableRateByKind(
	ctx context.Context,
	workspaceID, jurisdiction, authorityCode, kind, direction string,
	asOf time.Time,
) (*taxratepb.TaxRate, error) {
	if uc.repositories.TaxRate == nil {
		return nil, nil
	}
	q, ok := uc.repositories.TaxRate.(FindApplicableQueries)
	if !ok {
		return nil, nil
	}
	return q.FindApplicable(ctx, workspaceID, jurisdiction, authorityCode, kind, "", direction, asOf)
}

// findTaxClassByCode wraps FindByCode from the tax_class adapter.
func (uc *ComputeTaxesForRevenueUseCase) findTaxClassByCode(ctx context.Context, code, direction string) (*taxclasspb.TaxClass, error) {
	if uc.repositories.TaxClass == nil {
		return nil, nil
	}
	q, ok := uc.repositories.TaxClass.(FindByCodeQueries)
	if !ok {
		return nil, nil
	}
	tc, err := q.FindByCode(ctx, code, direction)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return nil, nil
	}
	return tc, err
}

// buildRevenueTaxLine constructs a RevenueTaxLine from an aggregated row.
func (uc *ComputeTaxesForRevenueUseCase) buildRevenueTaxLine(
	workspaceID, revenueID string, row *aggRow, now time.Time,
) *revenuetaxlinepb.RevenueTaxLine {
	line := &revenuetaxlinepb.RevenueTaxLine{
		WorkspaceId:          workspaceID,
		RevenueId:            revenueID,
		Direction:            row.direction,
		TaxableBase:          row.taxableBase,
		TaxAmount:            row.taxAmount,
		AppliedToLineItemIds: row.lineIDs,
		Active:               true,
	}

	if uc.services.IDService != nil {
		line.Id = uc.services.IDService.GenerateID()
	}

	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	computedAt := now.Format("2006-01-02")
	line.DateCreated = &ms
	line.DateCreatedString = &s
	line.DateModified = &ms
	line.DateModifiedString = &s
	line.ComputedAt = &computedAt

	if row.regCtx != nil && row.regCtx.reg != nil {
		line.SourceRegistrationIdSnapshot = row.regCtx.reg.GetId()
		line.AuthorityCodeSnapshot = row.regCtx.authorityCode
	}

	if row.taxRate != nil {
		rateID := row.taxRate.GetId()
		line.TaxRateId = &rateID
		line.TaxKindSnapshot = row.taxRate.GetKind()
		line.RateBasisPointsSnapshot = row.taxRate.GetRateBasisPoints()
		if rc := row.taxRate.GetRegulatorCode(); rc != "" {
			line.RegulatorCodeSnapshot = &rc
		}
	}

	return line
}

// deleteAndInsertTaxLines implements the idempotency DELETE + INSERT.
func (uc *ComputeTaxesForRevenueUseCase) deleteAndInsertTaxLines(
	ctx context.Context, revenueID string, lines []*revenuetaxlinepb.RevenueTaxLine,
) error {
	q, ok := uc.repositories.RevenueTaxLine.(RevenueTaxLineQueries)
	if !ok {
		return fmt.Errorf("revenue_tax_line repository does not support write operations")
	}
	if err := q.DeleteByRevenueID(ctx, revenueID); err != nil {
		return fmt.Errorf("delete existing revenue_tax_line: %w", err)
	}
	if len(lines) > 0 {
		if err := q.InsertForRevenue(ctx, lines); err != nil {
			return fmt.Errorf("insert revenue_tax_line: %w", err)
		}
	}
	return nil
}

// updateRevenueDenorm updates cash_amount_expected, wht_amount_expected, settlement_status.
func (uc *ComputeTaxesForRevenueUseCase) updateRevenueDenorm(
	ctx context.Context, rev *revenuepb.Revenue, taxLines []*revenuetaxlinepb.RevenueTaxLine,
) error {
	if uc.repositories.Revenue == nil {
		return nil
	}

	var sumSurcharge, sumWithholding int64
	for _, tl := range taxLines {
		switch tl.GetDirection() {
		case revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE:
			sumSurcharge += tl.GetTaxAmount()
		case revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_WITHHOLDING:
			sumWithholding += tl.GetTaxAmount()
		}
	}

	// Phase 4 C5: Inclusive pricing denorm correction.
	// Exclusive: total_amount = net revenue; cash = net + surcharge - withholding.
	// Inclusive: total_amount = gross-of-surcharge; adding surcharge again would double-count.
	//            cash = total_amount - withholding.
	var cashExpected int64
	if rev.GetTaxInclusivePricingSnapshot() {
		cashExpected = rev.GetTotalAmount() - sumWithholding
	} else {
		cashExpected = rev.GetTotalAmount() + sumSurcharge - sumWithholding
	}
	whtExpected := sumWithholding
	ss := settlementStatusOpen

	_, err := uc.repositories.Revenue.UpdateRevenue(ctx, &revenuepb.UpdateRevenueRequest{
		Data: &revenuepb.Revenue{
			Id:                 rev.GetId(),
			CashAmountExpected: &cashExpected,
			WhtAmountExpected:  &whtExpected,
			SettlementStatus:   &ss,
		},
	})
	return err
}

// checkMultiCurrencyGuard enforces the defensive FX guard.
func (uc *ComputeTaxesForRevenueUseCase) checkMultiCurrencyGuard(
	ctx context.Context, rev *revenuepb.Revenue, ws *workspacepb.Workspace,
) error {
	funcCurrency := ws.GetFunctionalCurrency()
	revCurrency := rev.GetCurrency()
	if funcCurrency != "" && revCurrency != "" && revCurrency != funcCurrency {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.currency_mismatch",
			fmt.Sprintf("revenue currency %q does not match workspace functional currency %q [DEFAULT]",
				revCurrency, funcCurrency)))
	}
	if rev.GetBillingCurrency() != "" {
		if rev.GetForexRateMicroUnits() == 0 || rev.GetForexRateSource() == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
				"revenue_tax_line.validation.fx_snapshot_required",
				"billing_currency is set but forex_rate_micro_units or forex_rate_source is missing [DEFAULT]"))
		}
	}
	return nil
}

// checkRecomputeBlockers enforces the recompute hard-block rules.
func (uc *ComputeTaxesForRevenueUseCase) checkRecomputeBlockers(ctx context.Context, rev *revenuepb.Revenue) error {
	ss := rev.GetSettlementStatus()
	if ss == settlementStatusFullySettled || ss == settlementStatusCashReceivedWHTpending {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.recompute_blocked_settled",
			"recompute is blocked: revenue has been (partially) settled. Reverse the cash receipt before recomputing [DEFAULT]"))
	}
	if uc.repositories.WithholdingCertificate != nil {
		revenueID := rev.GetId()
		resp, err := uc.repositories.WithholdingCertificate.ListWithholdingCertificates(ctx,
			&withholdingcertificatepb.ListWithholdingCertificatesRequest{RevenueId: &revenueID})
		if err == nil && resp != nil {
			for _, cert := range resp.GetData() {
				if cert.GetStatus() != withholdingcertificatepb.WithholdingCertificateStatus_WITHHOLDING_CERTIFICATE_STATUS_VOIDED {
					return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
						"revenue_tax_line.validation.recompute_blocked_certificate",
						"recompute is blocked: a non-void WithholdingCertificate exists. Void all certificates before recomputing [DEFAULT]"))
				}
			}
		}
	}
	return nil
}

// --- Repository read helpers ---

func (uc *ComputeTaxesForRevenueUseCase) readRevenue(ctx context.Context, id string) (*revenuepb.Revenue, error) {
	if uc.repositories.Revenue == nil {
		return nil, fmt.Errorf("revenue repository is not configured")
	}
	resp, err := uc.repositories.Revenue.ReadRevenue(ctx, &revenuepb.ReadRevenueRequest{Data: &revenuepb.Revenue{Id: id}})
	if err != nil {
		return nil, fmt.Errorf("read revenue %q: %w", id, err)
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, fmt.Errorf("revenue %q not found", id)
	}
	return resp.GetData()[0], nil
}

func (uc *ComputeTaxesForRevenueUseCase) readWorkspace(ctx context.Context, id string) (*workspacepb.Workspace, error) {
	if uc.repositories.Workspace == nil {
		return nil, fmt.Errorf("workspace repository is not configured")
	}
	resp, err := uc.repositories.Workspace.ReadWorkspace(ctx, &workspacepb.ReadWorkspaceRequest{Data: &workspacepb.Workspace{Id: id}})
	if err != nil {
		return nil, fmt.Errorf("read workspace %q: %w", id, err)
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, fmt.Errorf("workspace %q not found", id)
	}
	return resp.GetData()[0], nil
}

func (uc *ComputeTaxesForRevenueUseCase) readProduct(ctx context.Context, id string) (*productpb.Product, error) {
	if uc.repositories.Product == nil {
		return nil, nil
	}
	resp, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{Data: &productpb.Product{Id: id}})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

func (uc *ComputeTaxesForRevenueUseCase) readTaxTreatment(ctx context.Context, id string) (*taxtreatmentpb.TaxTreatment, error) {
	if uc.repositories.TaxTreatment == nil {
		return nil, nil
	}
	resp, err := uc.repositories.TaxTreatment.ReadTaxTreatment(ctx, &taxtreatmentpb.ReadTaxTreatmentRequest{Data: &taxtreatmentpb.TaxTreatment{Id: id}})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, nil
	}
	return resp.GetData()[0], nil
}

func (uc *ComputeTaxesForRevenueUseCase) listLineItems(ctx context.Context, revenueID string) ([]*revenuelineitempb.RevenueLineItem, error) {
	if uc.repositories.RevenueLineItem == nil {
		return nil, nil
	}
	resp, err := uc.repositories.RevenueLineItem.ListRevenueLineItems(ctx, &revenuelineitempb.ListRevenueLineItemsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "revenue_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    revenueID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list line items for revenue %q: %w", revenueID, err)
	}
	if resp == nil {
		return nil, nil
	}
	return resp.GetData(), nil
}

// ExecuteForRevenue is a narrow convenience wrapper that satisfies the
// ComputeTaxesForRevenueInvoker interface used by the revenue package.
// It calls Execute with IsRecompute=false (first-time compute path) and
// discards the response — callers only care whether the compute succeeded.
func (uc *ComputeTaxesForRevenueUseCase) ExecuteForRevenue(ctx context.Context, revenueID, workspaceID string) error {
	_, err := uc.Execute(ctx, &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: workspaceID,
		IsRecompute: false,
	})
	return err
}

// parseDateString parses a YYYY-MM-DD string into a time.Time.
func parseDateString(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}
	return time.Parse("2006-01-02", s)
}
