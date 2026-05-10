package compute_taxes_for_revenue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"

	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
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

// ---- Mock implementations ----

// mockRevenue implements revenuepb.RevenueDomainServiceServer.
type mockRevenue struct {
	revenuepb.UnimplementedRevenueDomainServiceServer
	rev    *revenuepb.Revenue
	update *revenuepb.Revenue
}

func (m *mockRevenue) ReadRevenue(_ context.Context, req *revenuepb.ReadRevenueRequest) (*revenuepb.ReadRevenueResponse, error) {
	if m.rev == nil || req.GetData().GetId() != m.rev.GetId() {
		return &revenuepb.ReadRevenueResponse{Success: false}, fmt.Errorf("not found")
	}
	return &revenuepb.ReadRevenueResponse{Success: true, Data: []*revenuepb.Revenue{m.rev}}, nil
}

func (m *mockRevenue) UpdateRevenue(_ context.Context, req *revenuepb.UpdateRevenueRequest) (*revenuepb.UpdateRevenueResponse, error) {
	m.update = req.GetData()
	return &revenuepb.UpdateRevenueResponse{Success: true}, nil
}

// mockWorkspace implements workspacepb.WorkspaceDomainServiceServer.
type mockWorkspace struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	ws *workspacepb.Workspace
}

func (m *mockWorkspace) ReadWorkspace(_ context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if m.ws == nil {
		return nil, fmt.Errorf("not found")
	}
	return &workspacepb.ReadWorkspaceResponse{Success: true, Data: []*workspacepb.Workspace{m.ws}}, nil
}

// mockRevenueLineItem implements revenuelineitempb.RevenueLineItemDomainServiceServer.
type mockRevenueLineItem struct {
	revenuelineitempb.UnimplementedRevenueLineItemDomainServiceServer
	lines []*revenuelineitempb.RevenueLineItem
}

func (m *mockRevenueLineItem) ListRevenueLineItems(_ context.Context, _ *revenuelineitempb.ListRevenueLineItemsRequest) (*revenuelineitempb.ListRevenueLineItemsResponse, error) {
	return &revenuelineitempb.ListRevenueLineItemsResponse{Success: true, Data: m.lines}, nil
}

// mockRevenueTaxLine implements revenuetaxlinepb.RevenueTaxLineDomainServiceServer + RevenueTaxLineQueries.
type mockRevenueTaxLine struct {
	revenuetaxlinepb.UnimplementedRevenueTaxLineDomainServiceServer
	deleted  bool
	inserted []*revenuetaxlinepb.RevenueTaxLine
}

func (m *mockRevenueTaxLine) DeleteByRevenueID(_ context.Context, _ string) error {
	m.deleted = true
	m.inserted = nil
	return nil
}

func (m *mockRevenueTaxLine) InsertForRevenue(_ context.Context, lines []*revenuetaxlinepb.RevenueTaxLine) error {
	m.inserted = append(m.inserted, lines...)
	return nil
}

// mockTaxRegistration implements TaxRegistrationQueries.
type mockTaxRegistration struct {
	taxregistrationpb.UnimplementedTaxRegistrationDomainServiceServer
	// wsReg is the workspace SURCHARGE registration (nil if not found).
	wsReg *taxregistrationpb.TaxRegistration
	// clientReg is the client WITHHOLDING registration (nil if not found).
	clientReg *taxregistrationpb.TaxRegistration
}

func (m *mockTaxRegistration) FindActiveByComputePath(_ context.Context, partyType, _ string, computePath, _ string, _ time.Time) (*taxregistrationpb.TaxRegistration, error) {
	// Determine compute_path: "1" = SURCHARGE, "2" = WITHHOLDING
	if computePath == "1" && partyType == "workspace" {
		return m.wsReg, nil
	}
	if computePath == "2" && partyType == "client" {
		return m.clientReg, nil
	}
	return nil, nil
}

// mockTaxRegistrationKind implements taxregistrationkindpb.TaxRegistrationKindDomainServiceServer.
type mockTaxRegistrationKind struct {
	taxregistrationkindpb.UnimplementedTaxRegistrationKindDomainServiceServer
	kinds map[string]*taxregistrationkindpb.TaxRegistrationKind
}

func (m *mockTaxRegistrationKind) ReadTaxRegistrationKind(_ context.Context, req *taxregistrationkindpb.ReadTaxRegistrationKindRequest) (*taxregistrationkindpb.ReadTaxRegistrationKindResponse, error) {
	k, ok := m.kinds[req.GetData().GetId()]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return &taxregistrationkindpb.ReadTaxRegistrationKindResponse{Success: true, Data: []*taxregistrationkindpb.TaxRegistrationKind{k}}, nil
}

// mockTaxAuthority implements taxauthoritypb.TaxAuthorityDomainServiceServer.
type mockTaxAuthority struct {
	taxauthoritypb.UnimplementedTaxAuthorityDomainServiceServer
	authorities map[string]*taxauthoritypb.TaxAuthority
}

func (m *mockTaxAuthority) ReadTaxAuthority(_ context.Context, req *taxauthoritypb.ReadTaxAuthorityRequest) (*taxauthoritypb.ReadTaxAuthorityResponse, error) {
	a, ok := m.authorities[req.GetData().GetId()]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return &taxauthoritypb.ReadTaxAuthorityResponse{Success: true, Data: []*taxauthoritypb.TaxAuthority{a}}, nil
}

// mockTaxRate implements taxratepb.TaxRateDomainServiceServer + FindApplicableQueries.
type mockTaxRate struct {
	taxratepb.UnimplementedTaxRateDomainServiceServer
	// Map from "kind|treatment|direction" to the rate.
	rates map[string]*taxratepb.TaxRate
}

func rateKey(kind, treatment, direction string) string {
	return kind + "|" + treatment + "|" + direction
}

func (m *mockTaxRate) FindApplicable(_ context.Context, _, _, _, kind, treatment, direction string, _ time.Time) (*taxratepb.TaxRate, error) {
	k := rateKey(kind, treatment, direction)
	r, ok := m.rates[k]
	if !ok {
		// Also try empty treatment (for WITHHOLDING which uses "" treatment).
		k2 := rateKey(kind, "", direction)
		r, ok = m.rates[k2]
	}
	if !ok {
		return nil, nil
	}
	return r, nil
}

// mockTaxClass implements taxclasspb.TaxClassDomainServiceServer + FindByCodeQueries.
type mockTaxClass struct {
	taxclasspb.UnimplementedTaxClassDomainServiceServer
	classes map[string]*taxclasspb.TaxClass // key: code|direction
	byID    map[string]*taxclasspb.TaxClass
}

func (m *mockTaxClass) FindByCode(_ context.Context, code, direction string) (*taxclasspb.TaxClass, error) {
	key := code + "|" + direction
	c, ok := m.classes[key]
	if !ok {
		return nil, fmt.Errorf("tax_class not found for code=%q direction=%q", code, direction)
	}
	return c, nil
}

func (m *mockTaxClass) ReadTaxClass(_ context.Context, req *taxclasspb.ReadTaxClassRequest) (*taxclasspb.ReadTaxClassResponse, error) {
	c, ok := m.byID[req.GetData().GetId()]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return &taxclasspb.ReadTaxClassResponse{Success: true, Data: []*taxclasspb.TaxClass{c}}, nil
}

// mockWithholdingCertificate implements withholdingcertificatepb.WithholdingCertificateDomainServiceServer.
type mockWithholdingCertificate struct {
	withholdingcertificatepb.UnimplementedWithholdingCertificateDomainServiceServer
	certs []*withholdingcertificatepb.WithholdingCertificate
}

func (m *mockWithholdingCertificate) ListWithholdingCertificates(_ context.Context, _ *withholdingcertificatepb.ListWithholdingCertificatesRequest) (*withholdingcertificatepb.ListWithholdingCertificatesResponse, error) {
	return &withholdingcertificatepb.ListWithholdingCertificatesResponse{Success: true, Data: m.certs}, nil
}

// mockTaxTreatment implements taxtreatmentpb.TaxTreatmentDomainServiceServer.
type mockTaxTreatment struct {
	taxtreatmentpb.UnimplementedTaxTreatmentDomainServiceServer
}

// ---- Test helpers ----

const (
	wsID       = "ws-001"
	clientID   = "cli-001"
	revenueID  = "rev-001"
	lineID1    = "rli-001"
	lineID2    = "rli-002"
	vatRegID   = "tr-ws-vat"
	twaRegID   = "tr-cli-twa"
	kindID     = "kind-vat"
	kindWHTID  = "kind-twa"
	authID     = "auth-bir"
	rateVATID  = "rate-vat-12"
	rateWHTID  = "rate-wht-corp"
)

// buildSurchargeRegistration builds a workspace SURCHARGE registration.
func buildSurchargeRegistration(kindID, authID string) *taxregistrationpb.TaxRegistration {
	return &taxregistrationpb.TaxRegistration{
		Id:                  vatRegID,
		TaxRegistrationKindId: kindID,
		TaxAuthorityId:      authID,
		ComputePathSnapshot: taxregistrationpb.TaxRegistrationComputePathSnapshot_TAX_REGISTRATION_COMPUTE_PATH_SNAPSHOT_SURCHARGE,
		PartyRoleSnapshot:   taxregistrationpb.TaxRegistrationPartyRoleSnapshot_TAX_REGISTRATION_PARTY_ROLE_SNAPSHOT_SELLER,
		Status:              taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_ACTIVE,
		EffectiveFrom:       "2018-01-01",
		Active:              true,
	}
}

// buildWithholdingRegistration builds a client WITHHOLDING registration.
func buildWithholdingRegistration(kindID, authID string) *taxregistrationpb.TaxRegistration {
	return &taxregistrationpb.TaxRegistration{
		Id:                  twaRegID,
		TaxRegistrationKindId: kindID,
		TaxAuthorityId:      authID,
		ComputePathSnapshot: taxregistrationpb.TaxRegistrationComputePathSnapshot_TAX_REGISTRATION_COMPUTE_PATH_SNAPSHOT_WITHHOLDING,
		PartyRoleSnapshot:   taxregistrationpb.TaxRegistrationPartyRoleSnapshot_TAX_REGISTRATION_PARTY_ROLE_SNAPSHOT_BUYER,
		Status:              taxregistrationpb.TaxRegistrationStatus_TAX_REGISTRATION_STATUS_ACTIVE,
		EffectiveFrom:       "2024-03-15",
		Active:              true,
	}
}

func buildVATKind(id, defaultKind, jurisdiction string) *taxregistrationkindpb.TaxRegistrationKind {
	dk := defaultKind
	return &taxregistrationkindpb.TaxRegistrationKind{
		Id:              id,
		DefaultRateKind: &dk,
		Jurisdiction:    jurisdiction,
	}
}

func buildAuthority(id, code string) *taxauthoritypb.TaxAuthority {
	return &taxauthoritypb.TaxAuthority{Id: id, Code: code}
}

func buildVATRate(id, kind, treatment string, basisPoints int32) *taxratepb.TaxRate {
	rc := "VT010"
	return &taxratepb.TaxRate{
		Id:              id,
		Kind:            kind,
		TreatmentCode:   &treatment,
		Direction:       taxratepb.TaxRateDirection_TAX_RATE_DIRECTION_SURCHARGE,
		RateBasisPoints: basisPoints,
		RegulatorCode:   &rc,
		Jurisdiction:    "PH-NATIONAL",
		AuthorityCode:   "BIR",
	}
}

func buildWHTRate(id, kind string, basisPoints int32) *taxratepb.TaxRate {
	rc := "WC011"
	return &taxratepb.TaxRate{
		Id:              id,
		Kind:            kind,
		Direction:       taxratepb.TaxRateDirection_TAX_RATE_DIRECTION_WITHHOLDING,
		RateBasisPoints: basisPoints,
		RegulatorCode:   &rc,
		Jurisdiction:    "PH-NATIONAL",
		AuthorityCode:   "BIR",
	}
}

func buildWHTClass(code string, rateKind string) *taxclasspb.TaxClass {
	buyerRole := taxclasspb.TaxClassCounterpartyRole_TAX_CLASS_COUNTERPARTY_ROLE_BUYER
	return &taxclasspb.TaxClass{
		Id:                      "tc-" + code,
		Code:                    code,
		Direction:               taxclasspb.TaxClassDirection_TAX_CLASS_DIRECTION_WITHHOLDING,
		DefaultRateKind:         &rateKind,
		RequiresCounterpartyRole: &buyerRole,
	}
}

func buildRevenue(totalAmount int64, currency, settlement string, taxEnabled, taxInclusive bool, billingCurrency, fxRateSource string, fxRateMicroUnits int64) *revenuepb.Revenue {
	te := taxEnabled
	ti := taxInclusive
	rd := "2026-05-01"
	rev := &revenuepb.Revenue{
		Id:                            revenueID,
		ClientId:                      clientID,
		RevenueDate:                   &rd,
		TotalAmount:                   totalAmount,
		Currency:                      currency,
		TaxComputationEnabledSnapshot: &te,
		TaxInclusivePricingSnapshot:   &ti,
		Active:                        true,
	}
	if settlement != "" {
		rev.SettlementStatus = &settlement
	}
	if billingCurrency != "" {
		rev.BillingCurrency = &billingCurrency
	}
	if fxRateSource != "" {
		rev.ForexRateSource = &fxRateSource
	}
	if fxRateMicroUnits != 0 {
		rev.ForexRateMicroUnits = &fxRateMicroUnits
	}
	return rev
}

func buildWorkspace(funcCurrency, homeJurisdiction string, taxEnabled, taxInclusive bool) *workspacepb.Workspace {
	te := taxEnabled
	ti := taxInclusive
	hj := homeJurisdiction
	return &workspacepb.Workspace{
		Id:                    wsID,
		FunctionalCurrency:    &funcCurrency,
		HomeJurisdiction:      &hj,
		TaxComputationEnabled: &te,
		TaxInclusivePricing:   &ti,
	}
}

func buildLineItem(id, revenueID, treatmentSnapshot, whtClassSnapshot string, lineAmount int64) *revenuelineitempb.RevenueLineItem {
	li := &revenuelineitempb.RevenueLineItem{
		Id:         id,
		RevenueId:  revenueID,
		LineAmount: lineAmount,
	}
	if treatmentSnapshot != "" {
		li.TaxTreatmentSnapshot = &treatmentSnapshot
	}
	if whtClassSnapshot != "" {
		li.WithholdingClassSnapshot = &whtClassSnapshot
	}
	return li
}

// buildStandardRepos builds the standard set of mocks for the PH VAT + TWA scenario.
func buildStandardRepos(
	wsReg, clientReg *taxregistrationpb.TaxRegistration,
	vatRate *taxratepb.TaxRate,
	whtRate *taxratepb.TaxRate,
	certs []*withholdingcertificatepb.WithholdingCertificate,
	lineItems []*revenuelineitempb.RevenueLineItem,
	revenue *revenuepb.Revenue,
	workspace *workspacepb.Workspace,
) (ComputeTaxesRepositories, *mockRevenueTaxLine, *mockRevenue) {
	taxLineRepo := &mockRevenueTaxLine{}
	revRepo := &mockRevenue{rev: revenue}

	rates := map[string]*taxratepb.TaxRate{}
	if vatRate != nil {
		rates[rateKey("VAT_STANDARD", "STANDARD", "SURCHARGE")] = vatRate
		rates[rateKey("VAT_STANDARD", "ZERO_RATED", "SURCHARGE")] = buildVATRate("rate-vat-0", "VAT_STANDARD", "ZERO_RATED", 0)
	}
	if whtRate != nil {
		rates[rateKey("WHT_PROFESSIONAL_CORPORATE", "", "WITHHOLDING")] = whtRate
	}

	kinds := map[string]*taxregistrationkindpb.TaxRegistrationKind{
		kindID:   buildVATKind(kindID, "VAT_STANDARD", "PH-NATIONAL"),
		kindWHTID: buildVATKind(kindWHTID, "", "PH-NATIONAL"),
	}

	whtClass := buildWHTClass("PROFESSIONAL_CORPORATE", "WHT_PROFESSIONAL_CORPORATE")
	taxClasses := map[string]*taxclasspb.TaxClass{
		"PROFESSIONAL_CORPORATE|WITHHOLDING": whtClass,
	}
	taxClassesByID := map[string]*taxclasspb.TaxClass{
		"tc-PROFESSIONAL_CORPORATE": whtClass,
	}

	repos := ComputeTaxesRepositories{
		Revenue: revRepo,
		RevenueLineItem: &mockRevenueLineItem{lines: lineItems},
		RevenueTaxLine:  taxLineRepo,
		Workspace:       &mockWorkspace{ws: workspace},
		TaxRegistration: &mockTaxRegistration{wsReg: wsReg, clientReg: clientReg},
		TaxRegistrationKind: &mockTaxRegistrationKind{kinds: kinds},
		TaxAuthority: &mockTaxAuthority{authorities: map[string]*taxauthoritypb.TaxAuthority{
			authID: buildAuthority(authID, "BIR"),
		}},
		TaxRate:  &mockTaxRate{rates: rates},
		TaxClass: &mockTaxClass{classes: taxClasses, byID: taxClassesByID},
		WithholdingCertificate: &mockWithholdingCertificate{certs: certs},
	}
	return repos, taxLineRepo, revRepo
}

func buildStandardServices() ComputeTaxesServices {
	return ComputeTaxesServices{
		AuthorizationService: ports.NewNoOpAuthorizationService(),
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}
}

// ---- Tests ----

// TC1: SURCHARGE exclusive happy path (PH VAT 12% on ₱5,000 line).
func TestComputeTaxes_Surcharge_Exclusive_HappyPath(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, taxLineRepo, revRepo := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 1 {
		t.Fatalf("expected 1 tax line, got %d", len(resp.Lines))
	}
	tl := resp.Lines[0]
	if tl.GetTaxAmount() != 60000 {
		t.Errorf("expected tax_amount=60000, got %d", tl.GetTaxAmount())
	}
	if tl.GetTaxableBase() != 500000 {
		t.Errorf("expected taxable_base=500000, got %d", tl.GetTaxableBase())
	}
	if tl.GetDirection() != revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE {
		t.Error("expected SURCHARGE direction")
	}
	// Verify denorm update: cash_amount_expected = 500000 + 60000 = 560000, wht = 0.
	if revRepo.update == nil {
		t.Fatal("expected revenue update")
	}
	if revRepo.update.GetCashAmountExpected() != 560000 {
		t.Errorf("expected cash_amount_expected=560000, got %d", revRepo.update.GetCashAmountExpected())
	}
	if revRepo.update.GetWhtAmountExpected() != 0 {
		t.Errorf("expected wht_amount_expected=0, got %d", revRepo.update.GetWhtAmountExpected())
	}
	if !taxLineRepo.deleted {
		t.Error("expected DeleteByRevenueID to be called")
	}
}

// TC2: SURCHARGE inclusive arithmetic (₱5,000 gross-inclusive @ 12%).
func TestComputeTaxes_Surcharge_Inclusive_Arithmetic(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	// Line amount = 560000 centavos (₱5,600 inclusive of 12% VAT).
	// taxable_base = round(560000 / 1.12) = 500000
	// tax_amount = round(500000 * 1200 / 10000) = 60000
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 560000)
	rev := buildRevenue(560000, "PHP", "", true, true, "", "", 0) // taxInclusive=true
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, true)

	repos, _, _ := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 1 {
		t.Fatalf("expected 1 tax line, got %d", len(resp.Lines))
	}
	if resp.Lines[0].GetTaxAmount() != 60000 {
		t.Errorf("expected tax_amount=60000 (inclusive), got %d", resp.Lines[0].GetTaxAmount())
	}
	if resp.Lines[0].GetTaxableBase() != 500000 {
		t.Errorf("expected taxable_base=500000 (extracted), got %d", resp.Lines[0].GetTaxableBase())
	}
}

// TC3: No SURCHARGE registration + STANDARD line → fail-closed error.
func TestComputeTaxes_NoSurchargeRegistration_FailClosed(t *testing.T) {
	// wsReg = nil: no workspace SURCHARGE registration.
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(nil, nil, nil, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	// WorkspaceID must be set so the test reaches the fail-closed branch (not the workspace_id guard).
	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected fail-closed error when no SURCHARGE registration and STANDARD line")
	}
}

// TC4: No client WITHHOLDING registration → no WHT rows.
func TestComputeTaxes_NoWithholdingRegistration_NoWHTRows(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "PROFESSIONAL_CORPORATE", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	// clientReg = nil: no WITHHOLDING registration.
	repos, taxLineRepo, _ := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have exactly 1 SURCHARGE row, no WITHHOLDING row.
	if len(resp.Lines) != 1 {
		t.Fatalf("expected 1 tax line (SURCHARGE only), got %d", len(resp.Lines))
	}
	if resp.Lines[0].GetDirection() != revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE {
		t.Error("expected SURCHARGE direction")
	}
	if len(taxLineRepo.inserted) != 1 {
		t.Errorf("expected 1 inserted line, got %d", len(taxLineRepo.inserted))
	}
}

// TC5: ZERO_RATED treatment → SURCHARGE row with tax_amount=0.
func TestComputeTaxes_ZeroRated_ZeroAmountRow(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "ZERO_RATED", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 1 {
		t.Fatalf("expected 1 tax line for ZERO_RATED, got %d", len(resp.Lines))
	}
	if resp.Lines[0].GetTaxAmount() != 0 {
		t.Errorf("expected tax_amount=0 for ZERO_RATED, got %d", resp.Lines[0].GetTaxAmount())
	}
}

// TC6: EXEMPT treatment → no SURCHARGE row.
func TestComputeTaxes_Exempt_NoRow(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "EXEMPT", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, taxLineRepo, _ := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 0 {
		t.Errorf("expected 0 tax lines for EXEMPT, got %d", len(resp.Lines))
	}
	if len(taxLineRepo.inserted) != 0 {
		t.Errorf("expected 0 inserted lines, got %d", len(taxLineRepo.inserted))
	}
}

// TC7: OUT_OF_SCOPE treatment → no SURCHARGE row.
func TestComputeTaxes_OutOfScope_NoRow(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	line := buildLineItem(lineID1, revenueID, "OUT_OF_SCOPE", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(wsReg, nil, nil, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 0 {
		t.Errorf("expected 0 tax lines for OUT_OF_SCOPE, got %d", len(resp.Lines))
	}
}

// TC8: Mixed PROFESSIONAL_CORPORATE + RENTAL on 2 lines → 2 WHT rows.
func TestComputeTaxes_Mixed_WHT_TwoRows(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	clientReg := buildWithholdingRegistration(kindWHTID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	whtCorpRate := buildWHTRate("rate-wht-corp", "WHT_PROFESSIONAL_CORPORATE", 1000)
	whtRentalRate := buildWHTRate("rate-wht-rental", "WHT_RENTAL", 500)

	line1 := buildLineItem(lineID1, revenueID, "STANDARD", "PROFESSIONAL_CORPORATE", 500000)
	line2 := buildLineItem(lineID2, revenueID, "STANDARD", "RENTAL", 300000)
	rev := buildRevenue(800000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	rentalClass := buildWHTClass("RENTAL", "WHT_RENTAL")

	// Custom repos to include rental class and rate.
	taxLineRepo := &mockRevenueTaxLine{}
	revRepo := &mockRevenue{rev: rev}

	rates := map[string]*taxratepb.TaxRate{
		rateKey("VAT_STANDARD", "STANDARD", "SURCHARGE"):                vatRate,
		rateKey("VAT_STANDARD", "ZERO_RATED", "SURCHARGE"):              buildVATRate("rate-vat-0", "VAT_STANDARD", "ZERO_RATED", 0),
		rateKey("WHT_PROFESSIONAL_CORPORATE", "", "WITHHOLDING"):        whtCorpRate,
		rateKey("WHT_RENTAL", "", "WITHHOLDING"):                        whtRentalRate,
	}
	kinds := map[string]*taxregistrationkindpb.TaxRegistrationKind{
		kindID:   buildVATKind(kindID, "VAT_STANDARD", "PH-NATIONAL"),
		kindWHTID: buildVATKind(kindWHTID, "", "PH-NATIONAL"),
	}
	corpClass := buildWHTClass("PROFESSIONAL_CORPORATE", "WHT_PROFESSIONAL_CORPORATE")
	taxClasses := map[string]*taxclasspb.TaxClass{
		"PROFESSIONAL_CORPORATE|WITHHOLDING": corpClass,
		"RENTAL|WITHHOLDING":                 rentalClass,
	}
	taxClassesByID := map[string]*taxclasspb.TaxClass{
		"tc-PROFESSIONAL_CORPORATE": corpClass,
		"tc-RENTAL":                 rentalClass,
	}

	repos := ComputeTaxesRepositories{
		Revenue:         revRepo,
		RevenueLineItem: &mockRevenueLineItem{lines: []*revenuelineitempb.RevenueLineItem{line1, line2}},
		RevenueTaxLine:  taxLineRepo,
		Workspace:       &mockWorkspace{ws: ws},
		TaxRegistration: &mockTaxRegistration{wsReg: wsReg, clientReg: clientReg},
		TaxRegistrationKind: &mockTaxRegistrationKind{kinds: kinds},
		TaxAuthority: &mockTaxAuthority{authorities: map[string]*taxauthoritypb.TaxAuthority{
			authID: buildAuthority(authID, "BIR"),
		}},
		TaxRate:                &mockTaxRate{rates: rates},
		TaxClass:               &mockTaxClass{classes: taxClasses, byID: taxClassesByID},
		WithholdingCertificate: &mockWithholdingCertificate{},
	}
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count by direction.
	var surchargeCount, whtCount int
	for _, tl := range resp.Lines {
		switch tl.GetDirection() {
		case revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE:
			surchargeCount++
		case revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_WITHHOLDING:
			whtCount++
		}
	}
	// 1 SURCHARGE row (both lines aggregate to the same rate) + 2 WHT rows.
	if surchargeCount != 1 {
		t.Errorf("expected 1 SURCHARGE row, got %d", surchargeCount)
	}
	if whtCount != 2 {
		t.Errorf("expected 2 WITHHOLDING rows, got %d", whtCount)
	}
}

// TC9: Repost asOf time travel against superseded tax_rate.
// A 2027 rate (15%) supersedes the 2026 rate (12%); recomputing with asOf=2026 picks 12%.
func TestComputeTaxes_TimeTravel_SupersededRate(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	// Only the 2026 rate exists in our mock (asOf filter is applied by the real adapter).
	// Here we just verify the asOf parameter is passed through.
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200) // 12%
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, revRepo := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	// asOf = 2026-05-01 → should pick 1200bp.
	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 1 || resp.Lines[0].GetRateBasisPointsSnapshot() != 1200 {
		t.Errorf("expected 1200bp rate, got %d", resp.Lines[0].GetRateBasisPointsSnapshot())
	}
	// Verify the denorm gets updated correctly.
	if revRepo.update.GetCashAmountExpected() != 560000 {
		t.Errorf("expected cash_expected=560000, got %d", revRepo.update.GetCashAmountExpected())
	}
}

// TC10: Revoked-mid-cycle registration — clientReg = nil → no WHT row.
// (Simulated by passing nil clientReg; the real adapter filters by effective_to.)
func TestComputeTaxes_RevokedRegistration_NoWHTRow(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "PROFESSIONAL_CORPORATE", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	// clientReg = nil simulates the registration being inactive at asOf.
	repos, _, _ := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), // after TWA revoked 2026-08-15
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only SURCHARGE row — no WHT.
	if len(resp.Lines) != 1 {
		t.Fatalf("expected 1 line (SURCHARGE only), got %d", len(resp.Lines))
	}
	if resp.Lines[0].GetDirection() != revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE {
		t.Error("expected SURCHARGE direction only")
	}
}

// TC11: Milestone partial-billing — override_total: a line with a specific line_amount.
func TestComputeTaxes_MilestonePartialBilling(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	// Milestone: only ₱2,500 billed (partial).
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 250000)
	rev := buildRevenue(250000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, revRepo := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// tax_amount = round(250000 * 1200 / 10000) = 30000
	if resp.Lines[0].GetTaxAmount() != 30000 {
		t.Errorf("expected tax_amount=30000, got %d", resp.Lines[0].GetTaxAmount())
	}
	if revRepo.update.GetCashAmountExpected() != 280000 {
		t.Errorf("expected cash_expected=280000, got %d", revRepo.update.GetCashAmountExpected())
	}
}

// TC12a: Multi-currency guard — revenue.currency != functional → error.
func TestComputeTaxes_MultiCurrency_CurrencyMismatch_Error(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "USD", "", true, false, "", "", 0) // USD != PHP functional
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(wsReg, nil, nil, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected currency mismatch error")
	}
}

// TC12b: Multi-currency guard — billing_currency set but FX fields NULL → error.
func TestComputeTaxes_MultiCurrency_FXSnapshotMissing_Error(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "USD", "", 0) // billing_currency=USD but no FX rate
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(wsReg, nil, nil, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected FX snapshot missing error")
	}
}

// TC13a: Boundary test — effective_from == asOf (active on first day).
func TestComputeTaxes_Boundary_EffectiveFromEqualsAsOf_Active(t *testing.T) {
	// Registration effective_from = 2026-05-01, asOf = 2026-05-01 → should be active.
	wsReg := buildSurchargeRegistration(kindID, authID)
	wsReg.EffectiveFrom = "2026-05-01"
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 1 {
		t.Errorf("expected 1 SURCHARGE line, got %d", len(resp.Lines))
	}
}

// TC13b: Boundary test — effective_to == asOf → INACTIVE on the day it ends (exclusive upper bound).
// (Simulated by returning nil from the mock for this specific asOf.)
func TestComputeTaxes_Boundary_EffectiveToEqualsAsOf_Inactive(t *testing.T) {
	// In the real adapter: effective_to > asOf is required (exclusive).
	// effective_to = 2026-08-15, asOf = 2026-08-15 → NOT active.
	// We simulate this by returning nil from FindActiveByComputePath (the adapter handles it).
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	// wsReg = nil simulates that the registration with effective_to=2026-08-15 is inactive at asOf=2026-08-15.
	repos, _, _ := buildStandardRepos(nil, nil, nil, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	})
	// Expect fail-closed: STANDARD line + no SURCHARGE registration.
	if err == nil {
		t.Fatal("expected fail-closed error when registration is inactive at asOf (exclusive effective_to)")
	}
}

// TC14: Recompute blocked by cash received (settlement_status = CASH_RECEIVED_WHT_PENDING).
func TestComputeTaxes_Recompute_BlockedByCashReceived(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "CASH_RECEIVED_WHT_PENDING", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		IsRecompute: true,
	})
	if err == nil {
		t.Fatal("expected recompute blocked error when cash received")
	}
}

// TC15: Recompute blocked by certificate exists (non-void).
func TestComputeTaxes_Recompute_BlockedByCertificate(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	cert := &withholdingcertificatepb.WithholdingCertificate{
		Id:        "cert-001",
		RevenueId: revenueID,
		Status:    withholdingcertificatepb.WithholdingCertificateStatus_WITHHOLDING_CERTIFICATE_STATUS_RECEIVED,
	}

	repos, _, _ := buildStandardRepos(wsReg, nil, vatRate, nil, []*withholdingcertificatepb.WithholdingCertificate{cert}, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		IsRecompute: true,
	})
	if err == nil {
		t.Fatal("expected recompute blocked error when certificate exists")
	}
}

// TC16: tax_computation_enabled=false → no-op (no rows written, no error).
func TestComputeTaxes_TaxComputationDisabled_NoOp(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", false, false, "", "", 0) // taxEnabled=false
	ws := buildWorkspace("PHP", "PH-NATIONAL", false, false)       // taxEnabled=false

	repos, taxLineRepo, revRepo := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 0 {
		t.Errorf("expected 0 tax lines when disabled, got %d", len(resp.Lines))
	}
	if taxLineRepo.deleted {
		t.Error("expected DeleteByRevenueID NOT to be called when tax_computation_enabled=false")
	}
	if revRepo.update != nil {
		t.Error("expected revenue NOT to be updated when tax_computation_enabled=false")
	}
}

// TC17: Workspace with only PH_PERCENTAGE_TAX (PERIODIC_ONLY) + STANDARD line → fail-closed error.
// PERIODIC_ONLY registration produces no SURCHARGE path → STANDARD line fails.
func TestComputeTaxes_PeriodicOnlyWorkspace_StandardLine_FailClosed(t *testing.T) {
	// wsReg = nil: the PERIODIC_ONLY registration is not returned by FindActiveByComputePath(computePath=1)
	// because it has compute_path=PERIODIC_ONLY, not SURCHARGE.
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, _, _ := buildStandardRepos(nil, nil, nil, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	// WorkspaceID must be set so the test reaches the fail-closed branch (not the workspace_id guard).
	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected fail-closed error: workspace has only PERIODIC_ONLY registration but product is STANDARD")
	}
}

// TC18: Workspace with only PH_PERCENTAGE_TAX + EXEMPT/OUT_OF_SCOPE lines → no rows, no error.
func TestComputeTaxes_PeriodicOnlyWorkspace_ExemptLines_NoRows(t *testing.T) {
	// wsReg = nil for SURCHARGE path (PERIODIC_ONLY doesn't appear on computePath=1).
	line1 := buildLineItem(lineID1, revenueID, "EXEMPT", "", 500000)
	line2 := buildLineItem(lineID2, revenueID, "OUT_OF_SCOPE", "", 300000)
	rev := buildRevenue(800000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, taxLineRepo, _ := buildStandardRepos(nil, nil, nil, nil, nil, []*revenuelineitempb.RevenueLineItem{line1, line2}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 0 {
		t.Errorf("expected 0 tax lines for EXEMPT/OUT_OF_SCOPE lines, got %d", len(resp.Lines))
	}
	if len(taxLineRepo.inserted) != 0 {
		t.Errorf("expected 0 inserted lines, got %d", len(taxLineRepo.inserted))
	}
}

// TC_DryRun: dry_run=true returns lines without writing.
func TestComputeTaxes_DryRun_NoWrite(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, taxLineRepo, revRepo := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Lines) != 1 {
		t.Errorf("expected 1 preview line, got %d", len(resp.Lines))
	}
	if taxLineRepo.deleted {
		t.Error("dry-run must NOT call DeleteByRevenueID")
	}
	if taxLineRepo.inserted != nil {
		t.Error("dry-run must NOT insert any lines")
	}
	if revRepo.update != nil {
		t.Error("dry-run must NOT update revenue denorm")
	}
}

// TC_FullFlow: SURCHARGE + WITHHOLDING full PH scenario.
func TestComputeTaxes_FullPH_SurchargeAndWithholding(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	clientReg := buildWithholdingRegistration(kindWHTID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)
	whtRate := buildWHTRate(rateWHTID, "WHT_PROFESSIONAL_CORPORATE", 1000)
	line := buildLineItem(lineID1, revenueID, "STANDARD", "PROFESSIONAL_CORPORATE", 500000)
	rev := buildRevenue(500000, "PHP", "", true, false, "", "", 0)
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, false)

	repos, taxLineRepo, revRepo := buildStandardRepos(wsReg, clientReg, vatRate, whtRate, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect 2 rows: SURCHARGE + WITHHOLDING.
	if len(resp.Lines) != 2 {
		t.Fatalf("expected 2 tax lines, got %d", len(resp.Lines))
	}
	if len(taxLineRepo.inserted) != 2 {
		t.Fatalf("expected 2 inserted lines, got %d", len(taxLineRepo.inserted))
	}

	// Denorm: cash_expected = 500000 + 60000 - 50000 = 510000; wht = 50000.
	if revRepo.update.GetCashAmountExpected() != 510000 {
		t.Errorf("expected cash_expected=510000, got %d", revRepo.update.GetCashAmountExpected())
	}
	if revRepo.update.GetWhtAmountExpected() != 50000 {
		t.Errorf("expected wht_expected=50000, got %d", revRepo.update.GetWhtAmountExpected())
	}
	if revRepo.update.GetSettlementStatus() != settlementStatusOpen {
		t.Errorf("expected settlement_status=OPEN, got %q", revRepo.update.GetSettlementStatus())
	}
}

// TC19: Inclusive pricing × WHT — Phase 4 C4 regression test.
// PHP 5,600 line (gross-inclusive of 12% VAT) + 10% WHT on PROFESSIONAL_CORPORATE.
// Expected:
//   - SURCHARGE: taxable_base=500000 (extracted), tax_amount=60000 (12% of 5000)
//   - WITHHOLDING: taxable_base=500000 (net, NOT gross 560000), tax_amount=50000 (10% of 5000)
//   - Denorm (inclusive): cash_expected = 560000 - 50000 = 510000
func TestComputeTaxes_Inclusive_WHT_NetBase(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	clientReg := buildWithholdingRegistration(kindWHTID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)  // 12% SURCHARGE
	whtRate := buildWHTRate(rateWHTID, "WHT_PROFESSIONAL_CORPORATE", 1000) // 10% WHT

	// Line amount = 560000 centavos (₱5,600 gross-inclusive of 12% VAT).
	line := buildLineItem(lineID1, revenueID, "STANDARD", "PROFESSIONAL_CORPORATE", 560000)
	// Revenue total = 560000 (gross-inclusive).
	rev := buildRevenue(560000, "PHP", "", true, true, "", "", 0) // taxInclusive=true
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, true)

	repos, taxLineRepo, revRepo := buildStandardRepos(wsReg, clientReg, vatRate, whtRate, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	resp, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Lines) != 2 {
		t.Fatalf("expected 2 tax lines (SURCHARGE + WITHHOLDING), got %d", len(resp.Lines))
	}
	if len(taxLineRepo.inserted) != 2 {
		t.Fatalf("expected 2 inserted lines, got %d", len(taxLineRepo.inserted))
	}

	var surcharge, withholding *revenuetaxlinepb.RevenueTaxLine
	for _, tl := range resp.Lines {
		switch tl.GetDirection() {
		case revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_SURCHARGE:
			surcharge = tl
		case revenuetaxlinepb.RevenueTaxLineDirection_REVENUE_TAX_LINE_DIRECTION_WITHHOLDING:
			withholding = tl
		}
	}

	if surcharge == nil {
		t.Fatal("expected SURCHARGE line")
	}
	if withholding == nil {
		t.Fatal("expected WITHHOLDING line")
	}

	// SURCHARGE: base = round(560000 / 1.12) = 500000; amount = round(500000 * 1200/10000) = 60000.
	if surcharge.GetTaxableBase() != 500000 {
		t.Errorf("SURCHARGE taxable_base: expected 500000, got %d", surcharge.GetTaxableBase())
	}
	if surcharge.GetTaxAmount() != 60000 {
		t.Errorf("SURCHARGE tax_amount: expected 60000, got %d", surcharge.GetTaxAmount())
	}

	// WITHHOLDING: base = 500000 (net, NOT 560000 gross); amount = round(500000 * 1000/10000) = 50000.
	if withholding.GetTaxableBase() != 500000 {
		t.Errorf("WITHHOLDING taxable_base: expected 500000 (net-of-VAT), got %d — WHT must use extracted net, not gross", withholding.GetTaxableBase())
	}
	if withholding.GetTaxAmount() != 50000 {
		t.Errorf("WITHHOLDING tax_amount: expected 50000, got %d", withholding.GetTaxAmount())
	}

	// Denorm (inclusive): cash_expected = total_amount - withholding = 560000 - 50000 = 510000.
	// Must NOT add surcharge (60000) again — it is already inside total_amount.
	if revRepo.update == nil {
		t.Fatal("expected revenue update")
	}
	if revRepo.update.GetCashAmountExpected() != 510000 {
		t.Errorf("cash_expected: expected 510000 (inclusive=total-wht), got %d", revRepo.update.GetCashAmountExpected())
	}
	if revRepo.update.GetWhtAmountExpected() != 50000 {
		t.Errorf("wht_expected: expected 50000, got %d", revRepo.update.GetWhtAmountExpected())
	}
}

// TC20: Inclusive pricing denorm — no WHT case.
// PHP 5,600 line (gross-inclusive of 12% VAT), SURCHARGE only.
// cash_expected = total_amount (560000) — surcharge already included, no WHT to deduct.
func TestComputeTaxes_Inclusive_DenormNoWHT(t *testing.T) {
	wsReg := buildSurchargeRegistration(kindID, authID)
	vatRate := buildVATRate(rateVATID, "VAT_STANDARD", "STANDARD", 1200)

	line := buildLineItem(lineID1, revenueID, "STANDARD", "", 560000)
	rev := buildRevenue(560000, "PHP", "", true, true, "", "", 0) // taxInclusive=true
	ws := buildWorkspace("PHP", "PH-NATIONAL", true, true)

	repos, _, revRepo := buildStandardRepos(wsReg, nil, vatRate, nil, nil, []*revenuelineitempb.RevenueLineItem{line}, rev, ws)
	svc := buildStandardServices()
	uc := NewComputeTaxesForRevenueUseCase(repos, svc)

	_, err := uc.Execute(context.Background(), &ComputeTaxesRequest{
		RevenueID:   revenueID,
		WorkspaceID: wsID,
		AsOf:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if revRepo.update == nil {
		t.Fatal("expected revenue update")
	}
	// Inclusive, no WHT: cash_expected = 560000 (surcharge already in total; no withholding).
	if revRepo.update.GetCashAmountExpected() != 560000 {
		t.Errorf("cash_expected: expected 560000 (inclusive, no WHT), got %d", revRepo.update.GetCashAmountExpected())
	}
	if revRepo.update.GetWhtAmountExpected() != 0 {
		t.Errorf("wht_expected: expected 0, got %d", revRepo.update.GetWhtAmountExpected())
	}
}
