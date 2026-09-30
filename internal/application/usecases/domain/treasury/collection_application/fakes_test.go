package collection_application

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type store struct {
	mu   sync.Mutex
	rows map[string]proto.Message
}

func newStore() *store { return &store{rows: map[string]proto.Message{}} }

func idOf(m proto.Message) string {
	return m.ProtoReflect().Get(m.ProtoReflect().Descriptor().Fields().ByName("id")).String()
}
func (s *store) put(m proto.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[idOf(m)] = proto.Clone(m)
}
func (s *store) get(id string) (proto.Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.rows[id]
	if !ok {
		return nil, false
	}
	return proto.Clone(m), true
}
func (s *store) patch(p proto.Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.rows[idOf(p)]
	if !ok {
		return false
	}
	proto.Merge(cur, p)
	return true
}
func fieldString(m proto.Message, name string) string {
	fd := m.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
	if fd == nil {
		return ""
	}
	return m.ProtoReflect().Get(fd).String()
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
		ok := true
		for _, tf := range f.GetFilters() {
			ok = ok && fieldString(s.rows[id], tf.GetField()) == tf.GetStringFilter().GetValue()
		}
		if ok {
			out = append(out, proto.Clone(s.rows[id]))
		}
	}
	return out
}

// errFakeNotFound mirrors the real adapters: the generic read says "record not found"; lockers
// wrap domainports.ErrLockedRowNotFound (C9).
var errFakeNotFound = errors.New("record not found")

type fakeClients struct {
	clientpb.UnimplementedClientDomainServiceServer
	s *store
}

func (f *fakeClients) ReadClient(_ context.Context, r *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	m, ok := f.s.get(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &clientpb.ReadClientResponse{Data: []*clientpb.Client{m.(*clientpb.Client)}}, nil
}

type fakeMethods struct {
	collectionmethodpb.UnimplementedCollectionMethodDomainServiceServer
	s *store
}

func (f *fakeMethods) ReadCollectionMethod(_ context.Context, r *collectionmethodpb.ReadCollectionMethodRequest) (*collectionmethodpb.ReadCollectionMethodResponse, error) {
	m, ok := f.s.get(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &collectionmethodpb.ReadCollectionMethodResponse{Data: []*collectionmethodpb.CollectionMethod{m.(*collectionmethodpb.CollectionMethod)}}, nil
}

// noLockDocs is a recovery-document repository WITHOUT the row locker (C4: must fail closed).
type noLockDocs struct {
	recoverydocumentpb.RecoveryDocumentDomainServiceServer
}

type fakeCollections struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	s *store
}

func (f *fakeCollections) CreateCollection(_ context.Context, r *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	f.s.put(r.Data)
	return &collectionpb.CreateCollectionResponse{Success: true, Data: []*collectionpb.Collection{r.Data}}, nil
}
func (f *fakeCollections) ListCollections(_ context.Context, r *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	out := &collectionpb.ListCollectionsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*collectionpb.Collection))
	}
	return out, nil
}

type fakeApps struct {
	collectionapplicationpb.UnimplementedCollectionApplicationDomainServiceServer
	s     *store
	locks sync.Map
}

// LockCollectionApplicationForUpdate holds the row lock until the owning fake transaction ends.
func (f *fakeApps) LockCollectionApplicationForUpdate(ctx context.Context, id string) (*collectionapplicationpb.CollectionApplication, error) {
	st := txFrom(ctx)
	if st == nil {
		return nil, errors.New("lock outside transaction")
	}
	mu, _ := f.locks.LoadOrStore(id, &sync.Mutex{})
	st.acquire(mu.(*sync.Mutex))
	m, ok := f.s.get(id)
	if !ok {
		return nil, errFakeNotFound
	}
	return m.(*collectionapplicationpb.CollectionApplication), nil
}

func (f *fakeApps) CreateCollectionApplication(_ context.Context, r *collectionapplicationpb.CreateCollectionApplicationRequest) (*collectionapplicationpb.CreateCollectionApplicationResponse, error) {
	f.s.put(r.Data)
	return &collectionapplicationpb.CreateCollectionApplicationResponse{Success: true, Data: []*collectionapplicationpb.CollectionApplication{r.Data}}, nil
}
func (f *fakeApps) ReadCollectionApplication(_ context.Context, r *collectionapplicationpb.ReadCollectionApplicationRequest) (*collectionapplicationpb.ReadCollectionApplicationResponse, error) {
	m, ok := f.s.get(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &collectionapplicationpb.ReadCollectionApplicationResponse{Data: []*collectionapplicationpb.CollectionApplication{m.(*collectionapplicationpb.CollectionApplication)}}, nil
}
func (f *fakeApps) UpdateCollectionApplication(_ context.Context, r *collectionapplicationpb.UpdateCollectionApplicationRequest) (*collectionapplicationpb.UpdateCollectionApplicationResponse, error) {
	if !f.s.patch(r.Data) {
		return nil, errFakeNotFound
	}
	return &collectionapplicationpb.UpdateCollectionApplicationResponse{Success: true}, nil
}
func (f *fakeApps) ListCollectionApplications(_ context.Context, r *collectionapplicationpb.ListCollectionApplicationsRequest) (*collectionapplicationpb.ListCollectionApplicationsResponse, error) {
	out := &collectionapplicationpb.ListCollectionApplicationsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*collectionapplicationpb.CollectionApplication))
	}
	return out, nil
}

type fakeRevenues struct {
	revenuepb.UnimplementedRevenueDomainServiceServer
	s     *store
	locks sync.Map
}

// LockRevenueForUpdate holds the row lock until the owning fake transaction ends.
func (f *fakeRevenues) LockRevenueForUpdate(ctx context.Context, id string) (*revenuepb.Revenue, error) {
	st := txFrom(ctx)
	if st == nil {
		return nil, errors.New("lock outside transaction")
	}
	mu, _ := f.locks.LoadOrStore(id, &sync.Mutex{})
	st.acquire(mu.(*sync.Mutex))
	m, ok := f.s.get(id)
	if !ok {
		return nil, errFakeNotFound
	}
	return m.(*revenuepb.Revenue), nil
}

func (f *fakeRevenues) ListRevenues(_ context.Context, r *revenuepb.ListRevenuesRequest) (*revenuepb.ListRevenuesResponse, error) {
	out := &revenuepb.ListRevenuesResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*revenuepb.Revenue))
	}
	return out, nil
}

type fakePayments struct {
	revenuepaymentpb.UnimplementedRevenuePaymentDomainServiceServer
	s *store
}

func (f *fakePayments) ListRevenuePayments(_ context.Context, r *revenuepaymentpb.ListRevenuePaymentsRequest) (*revenuepaymentpb.ListRevenuePaymentsResponse, error) {
	out := &revenuepaymentpb.ListRevenuePaymentsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*revenuepaymentpb.RevenuePayment))
	}
	return out, nil
}

// fakeDocs also implements RecoveryDocumentLocker; the lock is held until the fake tx ends.
type fakeDocs struct {
	recoverydocumentpb.UnimplementedRecoveryDocumentDomainServiceServer
	s     *store
	locks sync.Map
}

func (f *fakeDocs) ListRecoveryDocuments(_ context.Context, r *recoverydocumentpb.ListRecoveryDocumentsRequest) (*recoverydocumentpb.ListRecoveryDocumentsResponse, error) {
	out := &recoverydocumentpb.ListRecoveryDocumentsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*recoverydocumentpb.RecoveryDocument))
	}
	return out, nil
}
func (f *fakeDocs) ReadRecoveryDocument(_ context.Context, r *recoverydocumentpb.ReadRecoveryDocumentRequest) (*recoverydocumentpb.ReadRecoveryDocumentResponse, error) {
	m, ok := f.s.get(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &recoverydocumentpb.ReadRecoveryDocumentResponse{Data: []*recoverydocumentpb.RecoveryDocument{m.(*recoverydocumentpb.RecoveryDocument)}}, nil
}
func (f *fakeDocs) LockRecoveryDocumentForUpdate(ctx context.Context, id string) (*recoverydocumentpb.RecoveryDocument, error) {
	st := txFrom(ctx)
	if st == nil {
		return nil, errors.New("lock outside transaction")
	}
	mu, _ := f.locks.LoadOrStore(id, &sync.Mutex{})
	st.acquire(mu.(*sync.Mutex))
	m, ok := f.s.get(id)
	if !ok {
		return nil, errFakeNotFound
	}
	return m.(*recoverydocumentpb.RecoveryDocument), nil
}

type fakeLines struct {
	recoverydocumentlinepb.UnimplementedRecoveryDocumentLineDomainServiceServer
	s *store
}

func (f *fakeLines) ListRecoveryDocumentLines(_ context.Context, r *recoverydocumentlinepb.ListRecoveryDocumentLinesRequest) (*recoverydocumentlinepb.ListRecoveryDocumentLinesResponse, error) {
	out := &recoverydocumentlinepb.ListRecoveryDocumentLinesResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*recoverydocumentlinepb.RecoveryDocumentLine))
	}
	return out, nil
}

type fakeCharges struct {
	billablechargepb.UnimplementedBillableChargeDomainServiceServer
	s *store
}

func (f *fakeCharges) ReadBillableCharge(_ context.Context, r *billablechargepb.ReadBillableChargeRequest) (*billablechargepb.ReadBillableChargeResponse, error) {
	m, ok := f.s.get(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &billablechargepb.ReadBillableChargeResponse{Data: []*billablechargepb.BillableCharge{m.(*billablechargepb.BillableCharge)}}, nil
}

type fakeComps struct {
	chargecomponentpb.UnimplementedChargeComponentDomainServiceServer
	s *store
}

func (f *fakeComps) ReadChargeComponent(_ context.Context, r *chargecomponentpb.ReadChargeComponentRequest) (*chargecomponentpb.ReadChargeComponentResponse, error) {
	m, ok := f.s.get(r.Data.Id)
	if !ok {
		return nil, errFakeNotFound
	}
	return &chargecomponentpb.ReadChargeComponentResponse{Data: []*chargecomponentpb.ChargeComponent{m.(*chargecomponentpb.ChargeComponent)}}, nil
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
	f.s.put(r.Data)
	return &chargeeffectpb.CreateChargeEffectResponse{Success: true}, nil
}
func (f *fakeEffects) ListChargeEffects(_ context.Context, r *chargeeffectpb.ListChargeEffectsRequest) (*chargeeffectpb.ListChargeEffectsResponse, error) {
	out := &chargeeffectpb.ListChargeEffectsResponse{Success: true}
	for _, m := range f.s.list(r.GetFilters()) {
		out.Data = append(out.Data, m.(*chargeeffectpb.ChargeEffect))
	}
	return out, nil
}

// fake transaction: releases row locks at the end (no undo).
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

type fakeTx struct{}

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

// strictAuthz: the ordinary check allows (shadow mode), the strict check denies.
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

type harness struct {
	uc                                                                  *UseCases
	cols, apps, revs, pays, docs, lines, charges, comps, posts, effects *store
	clients, methods                                                    *store
	repos                                                               Repositories
	svc                                                                 Services
}

func newHarness(deny bool) *harness {
	h := &harness{cols: newStore(), apps: newStore(), revs: newStore(), pays: newStore(), docs: newStore(), lines: newStore(),
		charges: newStore(), comps: newStore(), posts: newStore(), effects: newStore(), clients: newStore(), methods: newStore()}
	gate := actiongate.NewActionGatekeeper(&authz{deny: deny}, ports.NewNoOpTranslator())
	h.repos = Repositories{
		Collection: &fakeCollections{s: h.cols}, CollectionApplication: &fakeApps{s: h.apps}, Revenue: &fakeRevenues{s: h.revs},
		RevenuePayment: &fakePayments{s: h.pays}, RecoveryDocument: &fakeDocs{s: h.docs}, RecoveryDocumentLine: &fakeLines{s: h.lines},
		BillableCharge: &fakeCharges{s: h.charges}, ChargeComponent: &fakeComps{s: h.comps}, ChargePolicyPosting: &fakePostings{s: h.posts},
		ChargeEffect: &fakeEffects{s: h.effects}, Client: &fakeClients{s: h.clients}, CollectionMethod: &fakeMethods{s: h.methods},
	}
	h.svc = Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, Transactor: fakeTx{}, IDGenerator: &seqIDs{}}
	h.uc = NewUseCases(h.repos, h.svc)
	for _, c := range []string{"c1", "c2"} {
		h.clients.put(&clientpb.Client{Id: c})
	}
	h.methods.put(&collectionmethodpb.CollectionMethod{Id: "m1"})
	// policy version v1: APPLICATION mapping (DR CASH / CR RECEIVABLE)
	for i, p := range []struct {
		role enumspb.ChargePostingRole
		acct string
	}{
		{enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH, "acc-cash"},
		{enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE, "acc-recv"},
	} {
		h.posts.put(&postingpb.ChargePolicyPosting{Id: fmt.Sprintf("p%d", i), ChargePolicyVersionId: "v1", Event: enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION, PostingRole: p.role, AccountId: p.acct, Active: true})
	}
	return h
}

func (h *harness) ctx() context.Context {
	return contextutil.WithWorkspaceID(contextutil.WithUserID(context.Background(), "u1"), "ws1")
}

func (h *harness) revenue(id, client string, total int64, due, cur string) {
	h.revs.put(&revenuepb.Revenue{Id: id, ClientId: client, TotalAmount: total, Currency: cur, Status: "complete", Active: true, DueDate: &due, ReferenceNumber: &id})
}

// statement seeds an ISSUED STATEMENT document with one line -> component -> charge on version v1.
func (h *harness) statement(id, client string, total int64, due, cur string) {
	dd := due
	h.docs.put(&recoverydocumentpb.RecoveryDocument{
		Id: id, ClientId: client, DocumentNumber: id, DocumentType: recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_STATEMENT,
		TotalAmount: total, Currency: cur, Status: recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED, Active: true, DueDate: &dd,
	})
	h.lines.put(&recoverydocumentlinepb.RecoveryDocumentLine{Id: "ln-" + id, RecoveryDocumentId: id, ChargeComponentId: "cc-" + id, Amount: total})
	h.comps.put(&chargecomponentpb.ChargeComponent{Id: "cc-" + id, BillableChargeId: "ch-" + id, Amount: total, Currency: cur})
	h.charges.put(&billablechargepb.BillableCharge{Id: "ch-" + id, ChargePolicyVersionId: proto.String("v1"), Amount: total, Currency: cur})
}
