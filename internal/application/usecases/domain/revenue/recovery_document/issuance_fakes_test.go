package recovery_document

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// --- generic in-memory store: partial-update = proto.Merge, string filters (equals / starts_with) ---

type store struct {
	mu   sync.Mutex
	rows map[string]proto.Message
}

func newStore() *store { return &store{rows: map[string]proto.Message{}} }

func idOf(m proto.Message) string {
	return m.ProtoReflect().Get(m.ProtoReflect().Descriptor().Fields().ByName("id")).String()
}

func (s *store) create(m proto.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[idOf(m)] = proto.Clone(m)
}
func (s *store) read(id string) (proto.Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.rows[id]
	if !ok {
		return nil, false
	}
	return proto.Clone(m), true
}
func (s *store) update(patch proto.Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.rows[idOf(patch)]
	if !ok {
		return false
	}
	proto.Merge(cur, patch)
	return true
}
func fieldString(m proto.Message, name string) string {
	fd := m.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
	if fd == nil {
		return ""
	}
	v := m.ProtoReflect().Get(fd)
	if fd.Kind() == protoreflect.EnumKind {
		return string(fd.Enum().Values().ByNumber(v.Enum()).Name())
	}
	return v.String()
}
func (s *store) list(f *commonpb.FilterRequest) []proto.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []string
	for id := range s.rows {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []proto.Message
	for _, id := range ids {
		m := s.rows[id]
		ok := true
		for _, tf := range f.GetFilters() {
			sf := tf.GetStringFilter()
			got := fieldString(m, tf.GetField())
			switch sf.GetOperator() {
			case commonpb.StringOperator_STRING_STARTS_WITH:
				ok = ok && strings.HasPrefix(got, sf.GetValue())
			default:
				ok = ok && got == sf.GetValue()
			}
		}
		if ok {
			out = append(out, proto.Clone(m))
		}
	}
	return out
}

var errFakeNotFound = errors.New("record not found")

// --- typed fakes ---

type fakeDocs struct {
	recoverydocumentpb.UnimplementedRecoveryDocumentDomainServiceServer
	s     *store
	locks sync.Map // id -> *sync.Mutex
}

// LockRecoveryDocumentForUpdate holds the row lock until the owning fake transaction ends.
func (f *fakeDocs) LockRecoveryDocumentForUpdate(ctx context.Context, id string) (*recoverydocumentpb.RecoveryDocument, error) {
	st := txFrom(ctx)
	if st == nil {
		return nil, errors.New("lock outside transaction")
	}
	mu, _ := f.locks.LoadOrStore(id, &sync.Mutex{})
	st.acquire(mu.(*sync.Mutex))
	m, ok := f.s.read(id)
	if !ok {
		return nil, errFakeNotFound
	}
	return m.(*recoverydocumentpb.RecoveryDocument), nil
}

func (f *fakeDocs) CreateRecoveryDocument(_ context.Context, r *recoverydocumentpb.CreateRecoveryDocumentRequest) (*recoverydocumentpb.CreateRecoveryDocumentResponse, error) {
	// mirror the unique (series, sequence_number) and (workspace, issuance_key) indexes
	for _, m := range f.s.list(nil) {
		d := m.(*recoverydocumentpb.RecoveryDocument)
		if d.GetIssuanceKey() == r.Data.GetIssuanceKey() ||
			(d.GetDocumentSeriesId() == r.Data.GetDocumentSeriesId() && d.GetSequenceNumber() == r.Data.GetSequenceNumber()) {
			return nil, errors.New("unique violation")
		}
	}
	f.s.create(r.Data)
	return &recoverydocumentpb.CreateRecoveryDocumentResponse{Data: []*recoverydocumentpb.RecoveryDocument{r.Data}, Success: true}, nil
}
func (f *fakeDocs) ReadRecoveryDocument(_ context.Context, r *recoverydocumentpb.ReadRecoveryDocumentRequest) (*recoverydocumentpb.ReadRecoveryDocumentResponse, error) {
	m, ok := f.s.read(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &recoverydocumentpb.ReadRecoveryDocumentResponse{Data: []*recoverydocumentpb.RecoveryDocument{m.(*recoverydocumentpb.RecoveryDocument)}}, nil
}
func (f *fakeDocs) UpdateRecoveryDocument(_ context.Context, r *recoverydocumentpb.UpdateRecoveryDocumentRequest) (*recoverydocumentpb.UpdateRecoveryDocumentResponse, error) {
	if !f.s.update(r.Data) {
		return nil, errFakeNotFound
	}
	m, _ := f.s.read(r.Data.Id)
	return &recoverydocumentpb.UpdateRecoveryDocumentResponse{Data: []*recoverydocumentpb.RecoveryDocument{m.(*recoverydocumentpb.RecoveryDocument)}, Success: true}, nil
}
func (f *fakeDocs) ListRecoveryDocuments(_ context.Context, r *recoverydocumentpb.ListRecoveryDocumentsRequest) (*recoverydocumentpb.ListRecoveryDocumentsResponse, error) {
	out := &recoverydocumentpb.ListRecoveryDocumentsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*recoverydocumentpb.RecoveryDocument))
	}
	return out, nil
}

type fakeLines struct {
	recoverydocumentlinepb.UnimplementedRecoveryDocumentLineDomainServiceServer
	s *store
}

func (f *fakeLines) CreateRecoveryDocumentLine(_ context.Context, r *recoverydocumentlinepb.CreateRecoveryDocumentLineRequest) (*recoverydocumentlinepb.CreateRecoveryDocumentLineResponse, error) {
	for _, m := range f.s.list(nil) { // unique (charge_component_id)
		if m.(*recoverydocumentlinepb.RecoveryDocumentLine).GetChargeComponentId() == r.Data.GetChargeComponentId() {
			return nil, errors.New("unique violation: component already on a document")
		}
	}
	f.s.create(r.Data)
	return &recoverydocumentlinepb.CreateRecoveryDocumentLineResponse{Success: true}, nil
}
func (f *fakeLines) ListRecoveryDocumentLines(_ context.Context, r *recoverydocumentlinepb.ListRecoveryDocumentLinesRequest) (*recoverydocumentlinepb.ListRecoveryDocumentLinesResponse, error) {
	out := &recoverydocumentlinepb.ListRecoveryDocumentLinesResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*recoverydocumentlinepb.RecoveryDocumentLine))
	}
	return out, nil
}

// fakeSeries also implements DocumentSeriesLocker: the row lock is held until the owning
// fake transaction ends (exactly like SELECT ... FOR UPDATE).
type fakeSeries struct {
	documentseriespb.UnimplementedDocumentSeriesDomainServiceServer
	s     *store
	locks sync.Map // id -> *sync.Mutex
}

func (f *fakeSeries) ReadDocumentSeries(_ context.Context, r *documentseriespb.ReadDocumentSeriesRequest) (*documentseriespb.ReadDocumentSeriesResponse, error) {
	m, ok := f.s.read(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &documentseriespb.ReadDocumentSeriesResponse{Data: []*documentseriespb.DocumentSeries{m.(*documentseriespb.DocumentSeries)}}, nil
}
func (f *fakeSeries) UpdateDocumentSeries(_ context.Context, r *documentseriespb.UpdateDocumentSeriesRequest) (*documentseriespb.UpdateDocumentSeriesResponse, error) {
	if !f.s.update(r.Data) {
		return nil, errFakeNotFound
	}
	return &documentseriespb.UpdateDocumentSeriesResponse{Success: true}, nil
}
func (f *fakeSeries) LockDocumentSeriesForUpdate(ctx context.Context, id string) (*documentseriespb.DocumentSeries, error) {
	st := txFrom(ctx)
	if st == nil {
		return nil, errors.New("lock outside transaction")
	}
	mu, _ := f.locks.LoadOrStore(id, &sync.Mutex{})
	st.acquire(mu.(*sync.Mutex))
	m, ok := f.s.read(id) // read AFTER the lock is held
	if !ok {
		return nil, errFakeNotFound
	}
	return m.(*documentseriespb.DocumentSeries), nil
}

type fakeCharges struct {
	billablechargepb.UnimplementedBillableChargeDomainServiceServer
	s     *store
	locks sync.Map // id -> *sync.Mutex
}

// LockBillableChargeForUpdate holds the row lock until the owning fake transaction ends.
func (f *fakeCharges) LockBillableChargeForUpdate(ctx context.Context, id string) (*billablechargepb.BillableCharge, error) {
	st := txFrom(ctx)
	if st == nil {
		return nil, errors.New("lock outside transaction")
	}
	mu, _ := f.locks.LoadOrStore(id, &sync.Mutex{})
	st.acquire(mu.(*sync.Mutex))
	m, ok := f.s.read(id)
	if !ok {
		return nil, errFakeNotFound
	}
	return m.(*billablechargepb.BillableCharge), nil
}

func (f *fakeCharges) CreateBillableCharge(_ context.Context, r *billablechargepb.CreateBillableChargeRequest) (*billablechargepb.CreateBillableChargeResponse, error) {
	for _, m := range f.s.list(nil) { // unique obligation_key
		if m.(*billablechargepb.BillableCharge).GetObligationKey() == r.Data.GetObligationKey() {
			return nil, errors.New("unique violation: obligation_key")
		}
	}
	f.s.create(r.Data)
	return &billablechargepb.CreateBillableChargeResponse{Success: true}, nil
}
func (f *fakeCharges) ReadBillableCharge(_ context.Context, r *billablechargepb.ReadBillableChargeRequest) (*billablechargepb.ReadBillableChargeResponse, error) {
	m, ok := f.s.read(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &billablechargepb.ReadBillableChargeResponse{Data: []*billablechargepb.BillableCharge{m.(*billablechargepb.BillableCharge)}}, nil
}
func (f *fakeCharges) UpdateBillableCharge(_ context.Context, r *billablechargepb.UpdateBillableChargeRequest) (*billablechargepb.UpdateBillableChargeResponse, error) {
	if !f.s.update(r.Data) {
		return nil, errFakeNotFound
	}
	return &billablechargepb.UpdateBillableChargeResponse{Success: true}, nil
}
func (f *fakeCharges) ListBillableCharges(_ context.Context, r *billablechargepb.ListBillableChargesRequest) (*billablechargepb.ListBillableChargesResponse, error) {
	out := &billablechargepb.ListBillableChargesResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*billablechargepb.BillableCharge))
	}
	return out, nil
}

type fakeComps struct {
	chargecomponentpb.UnimplementedChargeComponentDomainServiceServer
	s *store
}

func (f *fakeComps) CreateChargeComponent(_ context.Context, r *chargecomponentpb.CreateChargeComponentRequest) (*chargecomponentpb.CreateChargeComponentResponse, error) {
	f.s.create(r.Data)
	return &chargecomponentpb.CreateChargeComponentResponse{Success: true}, nil
}
func (f *fakeComps) ReadChargeComponent(_ context.Context, r *chargecomponentpb.ReadChargeComponentRequest) (*chargecomponentpb.ReadChargeComponentResponse, error) {
	m, ok := f.s.read(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &chargecomponentpb.ReadChargeComponentResponse{Data: []*chargecomponentpb.ChargeComponent{m.(*chargecomponentpb.ChargeComponent)}}, nil
}
func (f *fakeComps) ListChargeComponents(_ context.Context, r *chargecomponentpb.ListChargeComponentsRequest) (*chargecomponentpb.ListChargeComponentsResponse, error) {
	out := &chargecomponentpb.ListChargeComponentsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*chargecomponentpb.ChargeComponent))
	}
	return out, nil
}

type fakePostings struct {
	postingpb.UnimplementedChargePolicyPostingDomainServiceServer
	s *store
}

func (f *fakePostings) ListChargePolicyPostings(_ context.Context, r *postingpb.ListChargePolicyPostingsRequest) (*postingpb.ListChargePolicyPostingsResponse, error) {
	out := &postingpb.ListChargePolicyPostingsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*postingpb.ChargePolicyPosting))
	}
	return out, nil
}

type fakeEffects struct {
	chargeeffectpb.UnimplementedChargeEffectDomainServiceServer
	s *store
}

func (f *fakeEffects) CreateChargeEffect(_ context.Context, r *chargeeffectpb.CreateChargeEffectRequest) (*chargeeffectpb.CreateChargeEffectResponse, error) {
	f.s.create(r.Data)
	return &chargeeffectpb.CreateChargeEffectResponse{Success: true}, nil
}
func (f *fakeEffects) ListChargeEffects(_ context.Context, r *chargeeffectpb.ListChargeEffectsRequest) (*chargeeffectpb.ListChargeEffectsResponse, error) {
	out := &chargeeffectpb.ListChargeEffectsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*chargeeffectpb.ChargeEffect))
	}
	return out, nil
}

type fakeApps struct {
	collectionapplicationpb.UnimplementedCollectionApplicationDomainServiceServer
	s *store
}

func (f *fakeApps) ListCollectionApplications(_ context.Context, r *collectionapplicationpb.ListCollectionApplicationsRequest) (*collectionapplicationpb.ListCollectionApplicationsResponse, error) {
	out := &collectionapplicationpb.ListCollectionApplicationsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*collectionapplicationpb.CollectionApplication))
	}
	return out, nil
}

// --- fake transactor: rolls back nothing (fakes have no undo) but releases row locks at the end ---

type txState struct {
	held []*sync.Mutex
	own  map[*sync.Mutex]bool
}

func (t *txState) acquire(mu *sync.Mutex) {
	if t.own[mu] {
		return
	}
	mu.Lock()
	t.own[mu] = true
	t.held = append(t.held, mu)
}

type txKey struct{}

func txFrom(ctx context.Context) *txState { st, _ := ctx.Value(txKey{}).(*txState); return st }

type fakeTx struct{ rollbackOnErr func() }

func (fakeTx) SupportsTransactions() bool               { return true }
func (fakeTx) IsTransactionActive(context.Context) bool { return false }
func (fakeTx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	st := &txState{own: map[*sync.Mutex]bool{}}
	defer func() {
		for _, mu := range st.held {
			mu.Unlock()
		}
	}()
	return fn(context.WithValue(ctx, txKey{}, st))
}

type authz struct{ deny bool }

// strictAuthz simulates the shadow-mode RBAC authorizer: the ordinary check ALLOWS (shadow), the
// strict (shadow-immune) check DENIES. A use case that uses Check instead of CheckStrict passes it.
type strictAuthz struct{}

func (strictAuthz) IsEnabled() bool { return true }
func (strictAuthz) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}
func (strictAuthz) HasPermissionStrict(context.Context, string, string) (bool, error) {
	return false, nil
}

func (a *authz) IsEnabled() bool { return true }
func (a *authz) HasPermission(context.Context, string, string) (bool, error) {
	return !a.deny, nil
}

type seqIDs struct{ n atomic.Int64 }

func (s *seqIDs) GenerateID() string                        { return fmt.Sprintf("id-%04d", s.n.Add(1)) }
func (s *seqIDs) IsEnabled() bool                           { return true }
func (s *seqIDs) GetProviderInfo() string                   { return "seq" }
func (s *seqIDs) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

// --- harness ---

type harness struct {
	uc      *UseCases
	svc     Services
	docs    *fakeDocs
	lines   *fakeLines
	series  *fakeSeries
	charges *fakeCharges
	comps   *fakeComps
	effects *fakeEffects
	apps    *fakeApps
	posts   *fakePostings
	clients *fakeClients
}

// fakeClients is the row-scoped client reader of the balance/aging reports: an id it does not hold
// reads as "record not found" (outside the caller's row scope or workspace).
type fakeClients struct {
	clientpb.UnimplementedClientDomainServiceServer
	s *store
}

func (f *fakeClients) ReadClient(_ context.Context, r *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	m, ok := f.s.read(r.GetData().GetId())
	if !ok {
		return nil, errFakeNotFound
	}
	return &clientpb.ReadClientResponse{Data: []*clientpb.Client{m.(*clientpb.Client)}}, nil
}

func newHarness(deny bool) *harness { return newHarnessWith(&authz{deny: deny}) }

func newHarnessWith(a actiongate.Authorizer) *harness {
	h := &harness{
		docs: &fakeDocs{s: newStore()}, lines: &fakeLines{s: newStore()}, series: &fakeSeries{s: newStore()},
		charges: &fakeCharges{s: newStore()}, comps: &fakeComps{s: newStore()}, effects: &fakeEffects{s: newStore()},
		apps: &fakeApps{s: newStore()}, posts: &fakePostings{s: newStore()}, clients: &fakeClients{s: newStore()},
	}
	for _, c := range []string{"c1", "c2"} {
		h.clients.s.create(&clientpb.Client{Id: c})
	}
	gate := actiongate.NewActionGatekeeper(a, ports.NewNoOpTranslator())
	ids := &seqIDs{}
	h.svc = Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, Transactor: fakeTx{}, IDGenerator: ids}
	h.uc = NewUseCases(Repositories{
		RecoveryDocument: h.docs, RecoveryDocumentLine: h.lines, DocumentSeries: h.series, BillableCharge: h.charges,
		ChargeComponent: h.comps, ChargePolicyPosting: h.posts, ChargeEffect: h.effects, CollectionApplication: h.apps,
		Client: h.clients,
	}, h.svc)
	// policy version "v1": ISSUE and APPLICATION mappings
	for i, p := range []struct {
		ev   enumspb.ChargePostingEvent
		role enumspb.ChargePostingRole
		acct string
	}{
		{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE, "acc-recv"},
		{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CLEARING, "acc-clear"},
	} {
		h.posts.s.create(&postingpb.ChargePolicyPosting{Id: fmt.Sprintf("p%d", i), ChargePolicyVersionId: "v1", Event: p.ev, PostingRole: p.role, AccountId: p.acct, Active: true})
	}
	return h
}

func (h *harness) ctx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

func (h *harness) addSeries(id string, status documentseriespb.DocumentSeriesStatus) {
	h.series.s.create(&documentseriespb.DocumentSeries{
		Id: id, Code: strings.ToUpper(id), IssuerName: "Acme", DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT,
		Prefix: proto.String("SOA-"), NextNumber: 1, NumberPadding: 4, Status: status,
	})
}

// addCharge seeds an OPEN ORIGINAL charge with one component of the same amount.
func (h *harness) addCharge(id, client, sub string, amount int64, currency string) {
	h.charges.s.create(&billablechargepb.BillableCharge{
		Id: id, ObligationKey: "KEY:" + id, ContentHash: "h", ClientId: proto.String(client), SubscriptionId: proto.String(sub),
		ChargePolicyVersionId: proto.String("v1"), ChargeKind: billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_ORIGINAL,
		Amount: amount, Currency: currency, Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN,
		ServiceFrom: proto.String("2026-09-01"), ServiceTo: proto.String("2026-10-01"), Active: true,
	})
	h.comps.s.create(&chargecomponentpb.ChargeComponent{
		Id: "cc-" + id, BillableChargeId: id, ComponentRole: enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST,
		DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT, Amount: amount, Currency: currency, Active: true,
	})
}

// addCorrection seeds an OPEN CORRECTION charge (a downward adjust of an ISSUED charge, built the way
// billable_charge.AdjustBillableCharge builds it) with one recovery-document component.
func (h *harness) addCorrection(id, predecessor string, delta int64) {
	pred := h.charge(predecessor)
	h.charges.s.create(&billablechargepb.BillableCharge{
		Id: id, ObligationKey: "KEY:" + predecessor + "#r1", ContentHash: "h", ClientId: pred.ClientId, SubscriptionId: pred.SubscriptionId,
		ChargePolicyVersionId: pred.ChargePolicyVersionId, ChargeKind: billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_CORRECTION,
		PredecessorId: proto.String(predecessor), Amount: delta, Currency: pred.GetCurrency(),
		Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN, Active: true,
	})
	h.comps.s.create(&chargecomponentpb.ChargeComponent{
		Id: "cc-" + id, BillableChargeId: id, ComponentRole: enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST,
		DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT, Amount: delta, Currency: pred.GetCurrency(), Active: true,
	})
}

func (h *harness) charge(id string) *billablechargepb.BillableCharge {
	m, _ := h.charges.s.read(id)
	return m.(*billablechargepb.BillableCharge)
}
