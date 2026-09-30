package allocation_batch

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenserecognition "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expense_recognition"
	billablecharge "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/billable_charge"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	chargepolicycomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	chargepolicyversionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	"google.golang.org/protobuf/proto"
)

// ---- fakes -----------------------------------------------------------------

type ctxTxKey struct{}

type fakeTx struct {
	held    map[string]bool
	unlocks []func()
}

// lockTable emulates SELECT ... FOR UPDATE: a row lock is held until the owning transaction ends.
type lockTable struct {
	mu sync.Mutex
	m  map[string]*sync.Mutex
}

func (l *lockTable) lock(ctx context.Context, key string) error {
	tx, _ := ctx.Value(ctxTxKey{}).(*fakeTx)
	if tx == nil {
		return errors.New("lock outside a transaction")
	}
	if tx.held[key] {
		return nil // re-entrant within one transaction, like postgres
	}
	l.mu.Lock()
	if l.m == nil {
		l.m = map[string]*sync.Mutex{}
	}
	mu := l.m[key]
	if mu == nil {
		mu = &sync.Mutex{}
		l.m[key] = mu
	}
	l.mu.Unlock()
	mu.Lock()
	tx.held[key] = true
	tx.unlocks = append(tx.unlocks, mu.Unlock)
	return nil
}

type fakeTransactor struct{}

func (fakeTransactor) SupportsTransactions() bool               { return true }
func (fakeTransactor) IsTransactionActive(context.Context) bool { return false }
func (fakeTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx := &fakeTx{held: map[string]bool{}}
	defer func() {
		for _, u := range tx.unlocks {
			u()
		}
	}()
	return fn(context.WithValue(ctx, ctxTxKey{}, tx))
}

type seqIDs struct{ n atomic.Int64 }

func (s *seqIDs) GenerateID() string                        { return fmt.Sprintf("id-%04d", s.n.Add(1)) }
func (s *seqIDs) IsEnabled() bool                           { return true }
func (s *seqIDs) GetProviderInfo() string                   { return "seq" }
func (s *seqIDs) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

type allowAuthz struct{ deny bool }

func (a allowAuthz) IsEnabled() bool { return true }
func (a allowAuthz) HasPermission(context.Context, string, string) (bool, error) {
	return !a.deny, nil
}

type fakeComps struct {
	costsourcecomponentpb.UnimplementedCostSourceComponentDomainServiceServer
	mu    sync.Mutex
	rows  map[string]*costsourcecomponentpb.CostSourceComponent
	locks *lockTable
}

func (f *fakeComps) get(id string) *costsourcecomponentpb.CostSourceComponent {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.rows[id]; ok {
		return proto.Clone(r).(*costsourcecomponentpb.CostSourceComponent)
	}
	return nil
}

// eqFilter returns the field and value of the request's (single) string-equality filter.
func eqFilter(f *commonpb.FilterRequest) (string, string) {
	if f == nil || len(f.GetFilters()) == 0 {
		return "", ""
	}
	tf := f.GetFilters()[0]
	return tf.GetField(), tf.GetStringFilter().GetValue()
}

func (f *fakeComps) ReadCostSourceComponent(_ context.Context, r *costsourcecomponentpb.ReadCostSourceComponentRequest) (*costsourcecomponentpb.ReadCostSourceComponentResponse, error) {
	if c := f.get(r.Data.Id); c != nil {
		return &costsourcecomponentpb.ReadCostSourceComponentResponse{Data: []*costsourcecomponentpb.CostSourceComponent{c}}, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeComps) ListCostSourceComponents(_ context.Context, r *costsourcecomponentpb.ListCostSourceComponentsRequest) (*costsourcecomponentpb.ListCostSourceComponentsResponse, error) {
	_, v := eqFilter(r.Filters)
	resp := &costsourcecomponentpb.ListCostSourceComponentsResponse{Success: true}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.rows {
		if c.ExpenditureId == v {
			resp.Data = append(resp.Data, proto.Clone(c).(*costsourcecomponentpb.CostSourceComponent))
		}
	}
	return resp, nil
}
func (f *fakeComps) UpdateCostSourceComponent(_ context.Context, r *costsourcecomponentpb.UpdateCostSourceComponentRequest) (*costsourcecomponentpb.UpdateCostSourceComponentResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	proto.Merge(f.rows[r.Data.Id], r.Data)
	return &costsourcecomponentpb.UpdateCostSourceComponentResponse{Success: true}, nil
}
func (f *fakeComps) LockCostSourceComponentForUpdate(ctx context.Context, id string) (*costsourcecomponentpb.CostSourceComponent, error) {
	if err := f.locks.lock(ctx, "csc:"+id); err != nil {
		return nil, err
	}
	if c := f.get(id); c != nil {
		return c, nil
	}
	return nil, fmt.Errorf("cost_source_component lock: %w", domainports.ErrLockedRowNotFound)
}
func (f *fakeComps) LockCostSourceComponentsByExpenditure(ctx context.Context, exp string) ([]*costsourcecomponentpb.CostSourceComponent, error) {
	f.mu.Lock()
	var ids []string
	for id, c := range f.rows {
		if c.ExpenditureId == exp {
			ids = append(ids, id)
		}
	}
	f.mu.Unlock()
	sort.Strings(ids)
	var out []*costsourcecomponentpb.CostSourceComponent
	for _, id := range ids {
		c, err := f.LockCostSourceComponentForUpdate(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

type fakeBatches struct {
	allocationbatchpb.UnimplementedAllocationBatchDomainServiceServer
	mu       sync.Mutex
	rows     map[string]*allocationbatchpb.AllocationBatch
	locks    *lockTable
	listErr  error // injected repository failure on List
	lockFail error // injected infrastructure failure on the lockers
}

func (f *fakeBatches) get(id string) *allocationbatchpb.AllocationBatch {
	f.mu.Lock()
	defer f.mu.Unlock()
	if b, ok := f.rows[id]; ok {
		return proto.Clone(b).(*allocationbatchpb.AllocationBatch)
	}
	return nil
}

func (f *fakeBatches) LockAllocationBatchForUpdate(ctx context.Context, id string) (*allocationbatchpb.AllocationBatch, error) {
	if f.lockFail != nil {
		return nil, f.lockFail
	}
	if err := f.locks.lock(ctx, "batch:"+id); err != nil {
		return nil, err
	}
	if b := f.get(id); b != nil {
		return b, nil
	}
	return nil, fmt.Errorf("allocation_batch lock: %w", domainports.ErrLockedRowNotFound)
}

func (f *fakeBatches) LockAllocationBatchesByComponent(ctx context.Context, comp string) ([]*allocationbatchpb.AllocationBatch, error) {
	if f.lockFail != nil {
		return nil, f.lockFail
	}
	f.mu.Lock()
	var ids []string
	for id, b := range f.rows {
		if b.CostSourceComponentId == comp {
			ids = append(ids, id)
		}
	}
	f.mu.Unlock()
	sort.Slice(ids, func(i, j int) bool { return f.get(ids[i]).Revision < f.get(ids[j]).Revision })
	var out []*allocationbatchpb.AllocationBatch
	for _, id := range ids {
		b, err := f.LockAllocationBatchForUpdate(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (f *fakeBatches) CreateAllocationBatch(_ context.Context, r *allocationbatchpb.CreateAllocationBatchRequest) (*allocationbatchpb.CreateAllocationBatchResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[r.Data.Id] = proto.Clone(r.Data).(*allocationbatchpb.AllocationBatch)
	return &allocationbatchpb.CreateAllocationBatchResponse{Data: []*allocationbatchpb.AllocationBatch{r.Data}, Success: true}, nil
}
func (f *fakeBatches) ReadAllocationBatch(_ context.Context, r *allocationbatchpb.ReadAllocationBatchRequest) (*allocationbatchpb.ReadAllocationBatchResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if b, ok := f.rows[r.Data.Id]; ok {
		return &allocationbatchpb.ReadAllocationBatchResponse{Data: []*allocationbatchpb.AllocationBatch{proto.Clone(b).(*allocationbatchpb.AllocationBatch)}}, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeBatches) ListAllocationBatches(_ context.Context, r *allocationbatchpb.ListAllocationBatchesRequest) (*allocationbatchpb.ListAllocationBatchesResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	field, v := eqFilter(r.Filters)
	f.mu.Lock()
	defer f.mu.Unlock()
	resp := &allocationbatchpb.ListAllocationBatchesResponse{Success: true}
	for _, b := range f.rows {
		if (field == "id" && b.Id == v) || (field == "cost_source_component_id" && b.CostSourceComponentId == v) {
			resp.Data = append(resp.Data, proto.Clone(b).(*allocationbatchpb.AllocationBatch))
		}
	}
	return resp, nil
}
func (f *fakeBatches) UpdateAllocationBatch(_ context.Context, r *allocationbatchpb.UpdateAllocationBatchRequest) (*allocationbatchpb.UpdateAllocationBatchResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	proto.Merge(f.rows[r.Data.Id], r.Data)
	return &allocationbatchpb.UpdateAllocationBatchResponse{Success: true}, nil
}

type fakeShares struct {
	allocationsharepb.UnimplementedAllocationShareDomainServiceServer
	mu   sync.Mutex
	rows map[string]*allocationsharepb.AllocationShare
}

func (f *fakeShares) CreateAllocationShare(_ context.Context, r *allocationsharepb.CreateAllocationShareRequest) (*allocationsharepb.CreateAllocationShareResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[r.Data.Id] = proto.Clone(r.Data).(*allocationsharepb.AllocationShare)
	return &allocationsharepb.CreateAllocationShareResponse{Data: []*allocationsharepb.AllocationShare{r.Data}, Success: true}, nil
}
func (f *fakeShares) ListAllocationShares(_ context.Context, r *allocationsharepb.ListAllocationSharesRequest) (*allocationsharepb.ListAllocationSharesResponse, error) {
	_, v := eqFilter(r.Filters)
	f.mu.Lock()
	defer f.mu.Unlock()
	resp := &allocationsharepb.ListAllocationSharesResponse{Success: true}
	for _, s := range f.rows {
		if s.AllocationBatchId == v {
			resp.Data = append(resp.Data, proto.Clone(s).(*allocationsharepb.AllocationShare))
		}
	}
	return resp, nil
}
func (f *fakeShares) UpdateAllocationShare(_ context.Context, r *allocationsharepb.UpdateAllocationShareRequest) (*allocationsharepb.UpdateAllocationShareResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	proto.Merge(f.rows[r.Data.Id], r.Data)
	return &allocationsharepb.UpdateAllocationShareResponse{Success: true}, nil
}
func (f *fakeShares) DeleteAllocationShare(_ context.Context, r *allocationsharepb.DeleteAllocationShareRequest) (*allocationsharepb.DeleteAllocationShareResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rows, r.Data.Id)
	return &allocationsharepb.DeleteAllocationShareResponse{Success: true}, nil
}

type fakeTerms struct {
	agreementlinetermpb.UnimplementedAgreementLineTermDomainServiceServer
	rows []*agreementlinetermpb.AgreementLineTerm
}

func (f *fakeTerms) ListAgreementLineTerms(_ context.Context, r *agreementlinetermpb.ListAgreementLineTermsRequest) (*agreementlinetermpb.ListAgreementLineTermsResponse, error) {
	_, v := eqFilter(r.Filters)
	resp := &agreementlinetermpb.ListAgreementLineTermsResponse{Success: true}
	for _, t := range f.rows {
		if t.SubscriptionId == v {
			resp.Data = append(resp.Data, t)
		}
	}
	return resp, nil
}

type fakeVersions struct {
	chargepolicyversionpb.UnimplementedChargePolicyVersionDomainServiceServer
	row *chargepolicyversionpb.ChargePolicyVersion
}

func (f *fakeVersions) ReadChargePolicyVersion(_ context.Context, r *chargepolicyversionpb.ReadChargePolicyVersionRequest) (*chargepolicyversionpb.ReadChargePolicyVersionResponse, error) {
	if f.row == nil || f.row.Id != r.Data.Id {
		return nil, errors.New("not found")
	}
	return &chargepolicyversionpb.ReadChargePolicyVersionResponse{Data: []*chargepolicyversionpb.ChargePolicyVersion{f.row}}, nil
}

type fakePolicyComps struct {
	chargepolicycomponentpb.UnimplementedChargePolicyComponentDomainServiceServer
	rows []*chargepolicycomponentpb.ChargePolicyComponent
}

func (f *fakePolicyComps) ListChargePolicyComponents(_ context.Context, r *chargepolicycomponentpb.ListChargePolicyComponentsRequest) (*chargepolicycomponentpb.ListChargePolicyComponentsResponse, error) {
	_, v := eqFilter(r.Filters)
	resp := &chargepolicycomponentpb.ListChargePolicyComponentsResponse{Success: true}
	for _, c := range f.rows {
		if c.ChargePolicyVersionId == v {
			resp.Data = append(resp.Data, c)
		}
	}
	return resp, nil
}

type fakeCharges struct {
	billablechargepb.UnimplementedBillableChargeDomainServiceServer
	mu   sync.Mutex
	rows map[string]*billablechargepb.BillableCharge
}

func (f *fakeCharges) CreateBillableCharge(_ context.Context, r *billablechargepb.CreateBillableChargeRequest) (*billablechargepb.CreateBillableChargeResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.rows {
		if x.ObligationKey == r.Data.ObligationKey {
			return nil, errors.New("unique violation (workspace, obligation_key)")
		}
	}
	f.rows[r.Data.Id] = proto.Clone(r.Data).(*billablechargepb.BillableCharge)
	return &billablechargepb.CreateBillableChargeResponse{Data: []*billablechargepb.BillableCharge{r.Data}, Success: true}, nil
}
func (f *fakeCharges) ListBillableCharges(_ context.Context, r *billablechargepb.ListBillableChargesRequest) (*billablechargepb.ListBillableChargesResponse, error) {
	_, v := eqFilter(r.Filters)
	f.mu.Lock()
	defer f.mu.Unlock()
	resp := &billablechargepb.ListBillableChargesResponse{Success: true}
	for _, c := range f.rows {
		if c.ObligationKey == v {
			resp.Data = append(resp.Data, proto.Clone(c).(*billablechargepb.BillableCharge))
		}
	}
	return resp, nil
}

type fakeChargeComps struct {
	chargecomponentpb.UnimplementedChargeComponentDomainServiceServer
	mu   sync.Mutex
	rows []*chargecomponentpb.ChargeComponent
}

func (f *fakeChargeComps) CreateChargeComponent(_ context.Context, r *chargecomponentpb.CreateChargeComponentRequest) (*chargecomponentpb.CreateChargeComponentResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, proto.Clone(r.Data).(*chargecomponentpb.ChargeComponent))
	return &chargecomponentpb.CreateChargeComponentResponse{Success: true}, nil
}

// ---- world -----------------------------------------------------------------

type world struct {
	uc      *UseCases
	comps   *fakeComps
	batches *fakeBatches
	shares  *fakeShares
	terms   *fakeTerms
	charges *fakeCharges
	cComps  *fakeChargeComps
	locks   *lockTable
	ids     *seqIDs
	authz   allowAuthz
}

func newWorld() *world {
	w := &world{locks: &lockTable{}, ids: &seqIDs{}}
	w.comps = &fakeComps{rows: map[string]*costsourcecomponentpb.CostSourceComponent{}, locks: w.locks}
	w.batches = &fakeBatches{rows: map[string]*allocationbatchpb.AllocationBatch{}, locks: w.locks}
	w.shares = &fakeShares{rows: map[string]*allocationsharepb.AllocationShare{}}
	w.charges = &fakeCharges{rows: map[string]*billablechargepb.BillableCharge{}}
	w.cComps = &fakeChargeComps{}
	from := "2026-01-01"
	w.terms = &fakeTerms{rows: []*agreementlinetermpb.AgreementLineTerm{
		{Id: "term-1", SubscriptionId: "sub-1", ClientId: "cli-1", ChargePolicyVersionId: "ver-1", EffectiveFrom: from, Active: true},
		{Id: "term-2", SubscriptionId: "sub-2", ClientId: "cli-2", ChargePolicyVersionId: "ver-1", EffectiveFrom: from, Active: true},
	}}
	w.comps.rows["csc-1"] = &costsourcecomponentpb.CostSourceComponent{
		Id: "csc-1", ExpenditureId: "exp-1", Amount: 10000, Currency: "PHP",
		ComponentKind: costsourcecomponentpb.CostSourceComponentKind_COST_SOURCE_COMPONENT_KIND_ENERGY,
		ServiceFrom:   strp("2026-01-01"), ServiceTo: strp("2026-02-01"), SourceVersion: 1, Active: true,
	}
	ver := &fakeVersions{row: &chargepolicyversionpb.ChargePolicyVersion{Id: "ver-1", TaxPosition: enumspb.TaxPosition_TAX_POSITION_EXCLUDED_REIMBURSEMENT.Enum()}}
	pc := &fakePolicyComps{rows: []*chargepolicycomponentpb.ChargePolicyComponent{{
		Id: "pc-1", ChargePolicyVersionId: "ver-1", Active: true,
		ComponentRole:    enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST,
		DocumentKind:     enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT,
		BookPresentation: enumspb.BookPresentation_BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT,
	}}}
	w.rebuild(ver, pc)
	return w
}

func (w *world) rebuild(ver *fakeVersions, pc *fakePolicyComps) {
	w.uc = NewUseCases(Repositories{
		AllocationBatch: w.batches, AllocationShare: w.shares, CostSourceComponent: w.comps, AgreementLineTerm: w.terms,
		ChargePolicyVersion: ver, ChargePolicyComponent: pc, BillableCharge: w.charges, ChargeComponent: w.cComps,
	}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), Transactor: fakeTransactor{}, IDGenerator: w.ids,
		ActionGatekeeper: actiongate.NewActionGatekeeper(&w.authz, ports.NewNoOpTranslator()),
	})
}

func uctx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

// standardShares: lessor own use 25%, sub-1 50%, sub-2 25% of weight 4.
func standardShares() []*allocationsharepb.AllocationShare {
	return []*allocationsharepb.AllocationShare{
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, BasisNumerator: 1, BasisDenominator: 4},
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: strp("sub-1"), ClientId: strp("cli-1"), BasisNumerator: 2, BasisDenominator: 4},
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: strp("sub-2"), ClientId: strp("cli-2"), BasisNumerator: 1, BasisDenominator: 4},
	}
}

func createReq(shares []*allocationsharepb.AllocationShare) *allocationbatchpb.CreateAllocationBatchRequest {
	return &allocationbatchpb.CreateAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{CostSourceComponentId: "csc-1"}, Shares: shares}
}
func publishReq(id string) *allocationbatchpb.PublishAllocationBatchRequest {
	return &allocationbatchpb.PublishAllocationBatchRequest{AllocationBatchId: id}
}
func updateReq(id string, shares []*allocationsharepb.AllocationShare) *allocationbatchpb.UpdateAllocationBatchSharesRequest {
	return &allocationbatchpb.UpdateAllocationBatchSharesRequest{AllocationBatchId: id, Shares: shares}
}

// draftResult is the created batch with its shares.
type draftResult struct {
	Batch  *allocationbatchpb.AllocationBatch
	Shares []*allocationsharepb.AllocationShare
}

func (w *world) draft(t *testing.T, shares []*allocationsharepb.AllocationShare) *draftResult {
	t.Helper()
	res, err := w.uc.CreateAllocationBatch.Execute(uctx(), createReq(shares))
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if !res.Success || len(res.Data) != 1 {
		t.Fatalf("create response shape: %+v", res)
	}
	return &draftResult{Batch: res.Data[0], Shares: res.Shares}
}

func code(err error) string {
	var ce interface{ ErrorCode() string }
	if errors.As(err, &ce) {
		return ce.ErrorCode()
	}
	return ""
}

// ---- N5 --------------------------------------------------------------------

func TestLargestRemainderStableOrder(t *testing.T) {
	got, ok := LargestRemainderSplit(100, 3, []SplitInput{{1, 1}, {2, 1}, {3, 1}})
	if !ok || got[0] != 34 || got[1] != 33 || got[2] != 33 {
		t.Fatalf("equal thirds: %v ok=%v", got, ok)
	}
	// remainders 6/7, 4/7, 4/7: the 2 leftover centavos go to the largest remainder, then the
	// earlier sequence_order of the tie (index 2 has the LOWER sequence_order here).
	got, ok = LargestRemainderSplit(100, 7, []SplitInput{{1, 3}, {9, 2}, {5, 2}})
	if !ok || got[0] != 43 || got[1] != 28 || got[2] != 29 {
		t.Fatalf("tie by sequence_order: %v ok=%v", got, ok)
	}
	// identical input, identical output (replay-safe)
	again, _ := LargestRemainderSplit(100, 7, []SplitInput{{1, 3}, {9, 2}, {5, 2}})
	for i := range got {
		if got[i] != again[i] {
			t.Fatal("not deterministic")
		}
	}
	// refusals
	if _, ok := LargestRemainderSplit(100, 4, []SplitInput{{1, 1}, {2, 1}}); ok {
		t.Fatal("weights not summing to the denominator must be refused")
	}
	if _, ok := LargestRemainderSplit(100, 0, []SplitInput{{1, 0}}); ok {
		t.Fatal("zero denominator must be refused")
	}
	if _, ok := LargestRemainderSplit(100, 2, []SplitInput{{1, -1}, {2, 3}}); ok {
		t.Fatal("negative weight must be refused")
	}
	// property: exact sum, each within 1 centavo of the ideal share, deterministic
	rng := rand.New(rand.NewSource(7))
	for n := 0; n < 500; n++ {
		k := 1 + rng.Intn(8)
		den := int64(0)
		in := make([]SplitInput, k)
		for i := range in {
			in[i] = SplitInput{SequenceOrder: int32(i + 1), Numerator: int64(rng.Intn(50))}
			den += in[i].Numerator
		}
		if den == 0 {
			continue
		}
		total := int64(rng.Intn(1_000_000_000))
		out, ok := LargestRemainderSplit(total, den, in)
		if !ok {
			t.Fatalf("unexpected refusal %v", in)
		}
		var sum int64
		for i, a := range out {
			sum += a
			ideal := float64(total) * float64(in[i].Numerator) / float64(den)
			if d := float64(a) - ideal; d < -1 || d > 1 {
				t.Fatalf("share %d off by %v", i, d)
			}
		}
		if sum != total {
			t.Fatalf("sum %d != %d", sum, total)
		}
	}
}

// ---- AC-UC-09 ---------------------------------------------------------------

func TestAllocateSupplierCostSumsAndLessorShare(t *testing.T) {
	w := newWorld()
	d := w.draft(t, standardShares())
	if d.Batch.Revision != 1 || d.Batch.Status != allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_DRAFT {
		t.Fatalf("draft wrong: %+v", d.Batch)
	}
	// preview amounts already exact: lessor 2500, tenants 5000 / 2500
	var sum int64
	for _, s := range d.Shares {
		sum += s.Amount
	}
	if sum != 10000 {
		t.Fatalf("draft shares sum %d", sum)
	}
	res, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id))
	if err != nil {
		t.Fatal(err)
	}
	if res.Data.Status != allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED || res.Data.GetPublishedBy() != "u1" {
		t.Fatalf("batch not published: %+v", res.Data)
	}
	got := map[string]int64{}
	sum = 0
	for _, s := range res.Shares {
		sum += s.Amount
		got[s.GetShareKind().String()+s.GetSubscriptionId()] += s.Amount
	}
	if sum != 10000 || got["ALLOCATION_SHARE_KIND_OWN_USE"] != 2500 || got["ALLOCATION_SHARE_KIND_RECOVERABLEsub-1"] != 5000 || got["ALLOCATION_SHARE_KIND_RECOVERABLEsub-2"] != 2500 {
		t.Fatalf("split wrong: sum=%d %v", sum, got)
	}
	// exactly the two RECOVERABLE shares produce charges; own use produces none
	if len(res.Charges) != 2 || len(w.charges.rows) != 2 || len(w.cComps.rows) != 2 {
		t.Fatalf("charges=%d rows=%d comps=%d", len(res.Charges), len(w.charges.rows), len(w.cComps.rows))
	}
	var chargeSum int64
	for _, c := range w.charges.rows {
		chargeSum += c.Amount
		want := billablecharge.AllocationObligationKey("csc-1", c.GetSubscriptionId(), "2026-01-01", "2026-02-01")
		if c.ObligationKey != want || c.Status != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN ||
			c.ChargeKind != billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_ORIGINAL || c.GetAgreementLineTermId() == "" ||
			c.GetChargePolicyVersionId() != "ver-1" || c.ContentHash == "" || c.Currency != "PHP" {
			t.Fatalf("charge wrong: %+v", c)
		}
	}
	if chargeSum != 7500 {
		t.Fatalf("recoverable charges must sum to the recoverable shares: %d", chargeSum)
	}
	for _, cc := range w.cComps.rows {
		if cc.ComponentRole != enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST ||
			cc.DocumentKind != enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT ||
			cc.TaxPosition != enumspb.TaxPosition_TAX_POSITION_EXCLUDED_REIMBURSEMENT || cc.GetCostSourceComponentId() != "csc-1" {
			t.Fatalf("component wrong: %+v", cc)
		}
	}
	// claim written
	c := w.comps.get("csc-1")
	if c.GetClaimKind() != costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_ALLOCATION || c.GetClaimRefId() != d.Batch.Id || c.GetClaimedAt() == 0 {
		t.Fatalf("claim not written: %+v", c)
	}
	// publishing again / editing after publish are refused
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); code(err) != "already_published" {
		t.Fatalf("want already_published, got %v", err)
	}
	if _, err := w.uc.UpdateAllocationBatchShares.Execute(uctx(), updateReq(d.Batch.Id, standardShares())); code(err) != "already_published" {
		t.Fatalf("want already_published on edit, got %v", err)
	}
}

func TestDraftValidationAndAuthorization(t *testing.T) {
	w := newWorld()
	mk := func(mut func([]*allocationsharepb.AllocationShare) []*allocationsharepb.AllocationShare) error {
		_, err := w.uc.CreateAllocationBatch.Execute(uctx(), createReq(mut(standardShares())))
		return err
	}
	if c := code(mk(func(s []*allocationsharepb.AllocationShare) []*allocationsharepb.AllocationShare {
		s[1].BasisNumerator = 3
		return s
	})); c != "shares_total_mismatch" {
		t.Fatalf("weights over total: %q", c)
	}
	if c := code(mk(func(s []*allocationsharepb.AllocationShare) []*allocationsharepb.AllocationShare {
		for i := range s {
			s[i].BasisDenominator = 0
		}
		return s
	})); c != "denominator_invalid" {
		t.Fatalf("zero total weight: %q", c)
	}
	if c := code(mk(func(s []*allocationsharepb.AllocationShare) []*allocationsharepb.AllocationShare {
		s[2].SubscriptionId = strp("sub-9")
		return s
	})); c != "agreement_term_missing" {
		t.Fatalf("unknown subscription: %q", c)
	}
	if c := code(mk(func(s []*allocationsharepb.AllocationShare) []*allocationsharepb.AllocationShare {
		s[2].SubscriptionId = strp("sub-1")
		return s
	})); c != "validation" {
		t.Fatalf("duplicate subscription: %q", c)
	}
	if c := code(mk(func(s []*allocationsharepb.AllocationShare) []*allocationsharepb.AllocationShare {
		s[0].SubscriptionId = strp("sub-1")
		return s
	})); c != "validation" {
		t.Fatalf("non-recoverable share must not carry a subscription: %q", c)
	}
	// only own use: draft is fine, publish refuses no_recoverable_share
	only := []*allocationsharepb.AllocationShare{{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, BasisNumerator: 1, BasisDenominator: 1}}
	d := w.draft(t, only)
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); code(err) != "no_recoverable_share" {
		t.Fatalf("want no_recoverable_share, got %v", err)
	}
	// update replaces shares and revision numbering advances on the next draft
	if _, err := w.uc.UpdateAllocationBatchShares.Execute(uctx(), updateReq(d.Batch.Id, standardShares())); err != nil {
		t.Fatal(err)
	}
	if n := len(w.shares.rows); n != 3 {
		t.Fatalf("shares not replaced: %d", n)
	}
	d2 := w.draft(t, standardShares())
	if d2.Batch.Revision != 2 {
		t.Fatalf("revision: %d", d2.Batch.Revision)
	}
	// denied caller fails closed on every write
	w.authz.deny = true
	if _, err := w.uc.CreateAllocationBatch.Execute(uctx(), createReq(standardShares())); err == nil {
		t.Fatal("create must be denied")
	}
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); err == nil {
		t.Fatal("publish must be denied")
	}
	if _, err := w.uc.UpdateAllocationBatchShares.Execute(uctx(), updateReq(d.Batch.Id, standardShares())); err == nil {
		t.Fatal("update must be denied")
	}
}

// ---- term boundary ----------------------------------------------------------

func TestPublishRefusesTermBoundaryCrossed(t *testing.T) {
	w := newWorld()
	d := w.draft(t, standardShares())
	// the tenant's term ends mid-interval: the service period crosses the boundary
	w.terms.rows[0].EffectiveTo = strp("2026-01-15")
	w.terms.rows = append(w.terms.rows, &agreementlinetermpb.AgreementLineTerm{
		Id: "term-1b", SubscriptionId: "sub-1", ClientId: "cli-1", ChargePolicyVersionId: "ver-1", EffectiveFrom: "2026-01-15", Active: true})
	_, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id))
	if code(err) != "term_boundary_crossed" {
		t.Fatalf("want term_boundary_crossed, got %v", err)
	}
	if len(w.charges.rows) != 0 || w.comps.get("csc-1").ClaimKind != nil {
		t.Fatal("a refused publish must leave no charge and no claim")
	}
	if w.batches.rows[d.Batch.Id].Status != allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_DRAFT {
		t.Fatal("batch must stay DRAFT")
	}
	// a term of a different client never covers the interval
	w2 := newWorld()
	d2 := w2.draft(t, standardShares())
	w2.terms.rows[0].ClientId = "cli-other"
	if _, err := w2.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d2.Batch.Id)); code(err) != "agreement_term_missing" {
		t.Fatalf("want agreement_term_missing, got %v", err)
	}
	// a boundary that coincides with the interval end is fine (half-open)
	w3 := newWorld()
	d3 := w3.draft(t, standardShares())
	w3.terms.rows[0].EffectiveTo = strp("2026-02-01")
	if _, err := w3.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d3.Batch.Id)); err != nil {
		t.Fatalf("term ending exactly at service_to must cover: %v", err)
	}
}

// ---- AC-UC-30 ---------------------------------------------------------------

func TestObligationKeyReplayAndConflict(t *testing.T) {
	w := newWorld()
	d := w.draft(t, standardShares())
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); err != nil {
		t.Fatal(err)
	}
	before := len(w.charges.rows)
	// the first allocation is voided out-of-band (revision flow): release the claim, supersede batch 1
	w.comps.rows["csc-1"].ClaimKind, w.comps.rows["csc-1"].ClaimRefId, w.comps.rows["csc-1"].ClaimedAt = nil, nil, nil
	w.batches.rows[d.Batch.Id].Status = allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_SUPERSEDED

	// same content -> replay: no new charge, existing ones returned
	d2 := w.draft(t, standardShares())
	res, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d2.Batch.Id))
	if err != nil {
		t.Fatalf("identical replay must succeed: %v", err)
	}
	if res.ReplayedCount != 2 || len(w.charges.rows) != before || len(w.cComps.rows) != 2 {
		t.Fatalf("replay wrote rows: replayed=%d charges=%d comps=%d", res.ReplayedCount, len(w.charges.rows), len(w.cComps.rows))
	}

	// different content (component amount changed) under the same key -> obligation_conflict
	w.comps.rows["csc-1"].ClaimKind, w.comps.rows["csc-1"].ClaimRefId, w.comps.rows["csc-1"].ClaimedAt = nil, nil, nil
	w.batches.rows[d2.Batch.Id].Status = allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_SUPERSEDED
	w.comps.rows["csc-1"].Amount = 12000
	d3 := w.draft(t, standardShares())
	_, err = w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d3.Batch.Id))
	if code(err) != "obligation_conflict" {
		t.Fatalf("want obligation_conflict, got %v", err)
	}
	if len(w.charges.rows) != before {
		t.Fatal("a conflict must not add charges")
	}
	// the charge constructor itself: same key + same hash replays, hash is content-sensitive
	in := &billablecharge.AllocationChargeInput{SubscriptionID: "s", ClientID: "c", Amount: 5, Currency: "PHP", ServiceFrom: "a", ServiceTo: "b"}
	h1 := billablecharge.AllocationContentHash(in)
	in.Amount = 6
	if h1 == billablecharge.AllocationContentHash(in) {
		t.Fatal("content hash must depend on the amount")
	}
}

// ---- AC-UC-29 ---------------------------------------------------------------

type fakeRecogs struct {
	expenserecognitionpb.UnimplementedExpenseRecognitionDomainServiceServer
	mu    sync.Mutex
	count int
}

func (f *fakeRecogs) CreateExpenseRecognition(_ context.Context, r *expenserecognitionpb.CreateExpenseRecognitionRequest) (*expenserecognitionpb.CreateExpenseRecognitionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	return &expenserecognitionpb.CreateExpenseRecognitionResponse{Data: []*expenserecognitionpb.ExpenseRecognition{r.Data}, Success: true}, nil
}

type fakeExpenditures struct {
	expenditurepb.UnimplementedExpenditureDomainServiceServer
}

func (fakeExpenditures) ReadExpenditure(_ context.Context, r *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	return &expenditurepb.ReadExpenditureResponse{Data: []*expenditurepb.Expenditure{{Id: r.Data.Id}}}, nil
}

func (w *world) recognizer(recs *fakeRecogs) *expenserecognition.RecognizeFromExpenditureUseCase {
	return expenserecognition.NewRecognizeFromExpenditureUseCase(
		expenserecognition.RecognizeFromExpenditureRepositories{
			ExpenseRecognition: recs, Expenditure: fakeExpenditures{}, CostSourceComponent: w.comps,
		},
		expenserecognition.RecognizeFromExpenditureServices{
			Authorizer: ports.NewNoOpAuthorizer(), Transactor: fakeTransactor{}, Translator: ports.NewNoOpTranslator(), IDGenerator: w.ids,
			ActionGatekeeper: actiongate.NewActionGatekeeper(&w.authz, ports.NewNoOpTranslator()),
		})
}

func TestSourceClaimBothOrders(t *testing.T) {
	// (1) allocation first, then recognition is refused
	w := newWorld()
	recs := &fakeRecogs{}
	d := w.draft(t, standardShares())
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); err != nil {
		t.Fatal(err)
	}
	_, err := w.recognizer(recs).Execute(uctx(), &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "exp-1"})
	if code(err) != "source_claimed_by_allocation" {
		t.Fatalf("recognition after allocation: want source_claimed_by_allocation, got %v", err)
	}
	if recs.count != 0 {
		t.Fatal("refused recognition must not write")
	}

	// (2) recognition first, then a draft and a publish are refused
	w = newWorld()
	recs = &fakeRecogs{}
	d = w.draft(t, standardShares()) // draft before the claim exists
	if _, err := w.recognizer(recs).Execute(uctx(), &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "exp-1"}); err != nil {
		t.Fatal(err)
	}
	if c := w.comps.get("csc-1"); c.GetClaimKind() != costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_RECOGNITION {
		t.Fatalf("recognition must claim the source: %+v", c)
	}
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); code(err) != "source_claimed_by_recognition" {
		t.Fatalf("publish after recognition: got %v", err)
	}
	if _, err := w.uc.CreateAllocationBatch.Execute(uctx(), createReq(standardShares())); code(err) != "source_claimed_by_recognition" {
		t.Fatalf("draft after recognition: got %v", err)
	}
	if len(w.charges.rows) != 0 {
		t.Fatal("no charge may exist")
	}
	// a second recognition of the same recognised source (multi-period amortisation) stays allowed
	if _, err := w.recognizer(recs).Execute(uctx(), &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "exp-1", RecognitionPeriod: strp("2026-03")}); err != nil {
		t.Fatalf("re-recognition of a recognised source must stay allowed: %v", err)
	}
}

// Concurrent publish vs recognize on the same component: exactly one wins, the other is refused
// with the matching claim error, and the surviving claim matches the winner.
func TestSourceClaimBothOrdersRace(t *testing.T) {
	for i := 0; i < 60; i++ {
		w := newWorld()
		recs := &fakeRecogs{}
		d := w.draft(t, standardShares())
		rec := w.recognizer(recs)
		var wg sync.WaitGroup
		var pubErr, recErr error
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, pubErr = w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id))
		}()
		go func() {
			defer wg.Done()
			<-start
			_, recErr = rec.Execute(uctx(), &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "exp-1"})
		}()
		close(start)
		wg.Wait()
		c := w.comps.get("csc-1")
		switch {
		case pubErr == nil && recErr != nil:
			if code(recErr) != "source_claimed_by_allocation" || c.GetClaimKind() != costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_ALLOCATION || recs.count != 0 || len(w.charges.rows) != 2 {
				t.Fatalf("iter %d: publish won but state wrong: recErr=%v claim=%v recs=%d charges=%d", i, recErr, c.GetClaimKind(), recs.count, len(w.charges.rows))
			}
		case recErr == nil && pubErr != nil:
			if code(pubErr) != "source_claimed_by_recognition" || c.GetClaimKind() != costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_RECOGNITION || recs.count != 1 || len(w.charges.rows) != 0 {
				t.Fatalf("iter %d: recognize won but state wrong: pubErr=%v claim=%v recs=%d charges=%d", i, pubErr, c.GetClaimKind(), recs.count, len(w.charges.rows))
			}
		default:
			t.Fatalf("iter %d: exactly one must win: pubErr=%v recErr=%v", i, pubErr, recErr)
		}
	}
}
