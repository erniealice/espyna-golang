package charge_policy

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// --- generic in-memory store (partial-update semantics = proto.Merge, like the DB adapter) ---

type memStore struct {
	rows  map[string]proto.Message
	order []string
	newFn func() proto.Message
}

func newMemStore(newFn func() proto.Message) *memStore {
	return &memStore{rows: map[string]proto.Message{}, newFn: newFn}
}

func idOf(m proto.Message) string {
	return m.ProtoReflect().Get(m.ProtoReflect().Descriptor().Fields().ByName("id")).String()
}

func (s *memStore) create(m proto.Message) proto.Message {
	c := proto.Clone(m)
	id := idOf(c)
	s.rows[id] = c
	s.order = append(s.order, id)
	return proto.Clone(c)
}

func (s *memStore) read(id string) (proto.Message, bool) {
	m, ok := s.rows[id]
	if !ok {
		return nil, false
	}
	return proto.Clone(m), true
}

func (s *memStore) update(patch proto.Message) (proto.Message, bool) {
	id := idOf(patch)
	cur, ok := s.rows[id]
	if !ok {
		return nil, false
	}
	proto.Merge(cur, patch)
	return proto.Clone(cur), true
}

func (s *memStore) del(id string) bool {
	if _, ok := s.rows[id]; !ok {
		return false
	}
	delete(s.rows, id)
	for i, o := range s.order {
		if o == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// list applies a single string-equals filter (enum fields compared by name).
func (s *memStore) list(f *commonpb.FilterRequest) []proto.Message {
	var out []proto.Message
	for _, id := range s.order {
		m := s.rows[id]
		if f != nil {
			match := true
			for _, tf := range f.GetFilters() {
				fd := m.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(tf.GetField()))
				if fd == nil {
					match = false
					break
				}
				v := m.ProtoReflect().Get(fd)
				var got string
				if fd.Kind() == protoreflect.EnumKind {
					got = string(fd.Enum().Values().ByNumber(v.Enum()).Name())
				} else {
					got = v.String()
				}
				if got != tf.GetStringFilter().GetValue() {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		out = append(out, proto.Clone(m))
	}
	return out
}

// --- typed fake repositories ---

type fakePolicyRepo struct {
	policypb.UnimplementedChargePolicyDomainServiceServer
	s          *memStore
	referenced map[string]bool // policies "referenced by price plans"
}

func (r *fakePolicyRepo) CreateChargePolicy(_ context.Context, req *policypb.CreateChargePolicyRequest) (*policypb.CreateChargePolicyResponse, error) {
	return &policypb.CreateChargePolicyResponse{Data: []*policypb.ChargePolicy{r.s.create(req.Data).(*policypb.ChargePolicy)}, Success: true}, nil
}
func (r *fakePolicyRepo) ReadChargePolicy(_ context.Context, req *policypb.ReadChargePolicyRequest) (*policypb.ReadChargePolicyResponse, error) {
	m, ok := r.s.read(req.Data.Id)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &policypb.ReadChargePolicyResponse{Data: []*policypb.ChargePolicy{m.(*policypb.ChargePolicy)}, Success: true}, nil
}
func (r *fakePolicyRepo) UpdateChargePolicy(_ context.Context, req *policypb.UpdateChargePolicyRequest) (*policypb.UpdateChargePolicyResponse, error) {
	m, ok := r.s.update(req.Data)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &policypb.UpdateChargePolicyResponse{Data: []*policypb.ChargePolicy{m.(*policypb.ChargePolicy)}, Success: true}, nil
}
func (r *fakePolicyRepo) DeleteChargePolicy(_ context.Context, req *policypb.DeleteChargePolicyRequest) (*policypb.DeleteChargePolicyResponse, error) {
	if !r.s.del(req.Data.Id) {
		return nil, fmt.Errorf("record not found")
	}
	return &policypb.DeleteChargePolicyResponse{Success: true}, nil
}
func (r *fakePolicyRepo) ListChargePolicies(_ context.Context, req *policypb.ListChargePoliciesRequest) (*policypb.ListChargePoliciesResponse, error) {
	out := &policypb.ListChargePoliciesResponse{Success: true}
	for _, m := range r.s.list(req.GetFilters()) {
		out.Data = append(out.Data, m.(*policypb.ChargePolicy))
	}
	return out, nil
}
func (r *fakePolicyRepo) LockChargePolicyForUpdate(_ context.Context, id string) (*policypb.ChargePolicy, error) {
	m, ok := r.s.read(id)
	if !ok {
		return nil, fmt.Errorf("charge_policy lock: %w", domainports.ErrLockedRowNotFound)
	}
	return m.(*policypb.ChargePolicy), nil
}
func (r *fakePolicyRepo) GetChargePolicyListPageData(_ context.Context, req *policypb.GetChargePolicyListPageDataRequest) (*policypb.GetChargePolicyListPageDataResponse, error) {
	out := &policypb.GetChargePolicyListPageDataResponse{Success: true}
	for _, m := range r.s.list(req.GetFilters()) {
		out.ChargePolicyList = append(out.ChargePolicyList, m.(*policypb.ChargePolicy))
	}
	return out, nil
}
func (r *fakePolicyRepo) ChargePolicyIDsReferencedByPricePlans(_ context.Context, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, id := range ids {
		if r.referenced[id] {
			out[id] = true
		}
	}
	return out, nil
}

type fakeVersionRepo struct {
	versionpb.UnimplementedChargePolicyVersionDomainServiceServer
	s *memStore
}

func (r *fakeVersionRepo) CreateChargePolicyVersion(_ context.Context, req *versionpb.CreateChargePolicyVersionRequest) (*versionpb.CreateChargePolicyVersionResponse, error) {
	// mirror the DB one-draft partial unique index
	if req.Data.GetStatus() == enumspbVersionDraft {
		for _, m := range r.s.rows {
			v := m.(*versionpb.ChargePolicyVersion)
			if v.GetChargePolicyId() == req.Data.GetChargePolicyId() && v.GetStatus() == enumspbVersionDraft {
				return nil, fmt.Errorf("duplicate key value violates unique constraint uq_charge_policy_version_one_draft")
			}
		}
	}
	return &versionpb.CreateChargePolicyVersionResponse{Data: []*versionpb.ChargePolicyVersion{r.s.create(req.Data).(*versionpb.ChargePolicyVersion)}, Success: true}, nil
}
func (r *fakeVersionRepo) ReadChargePolicyVersion(_ context.Context, req *versionpb.ReadChargePolicyVersionRequest) (*versionpb.ReadChargePolicyVersionResponse, error) {
	m, ok := r.s.read(req.Data.Id)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &versionpb.ReadChargePolicyVersionResponse{Data: []*versionpb.ChargePolicyVersion{m.(*versionpb.ChargePolicyVersion)}, Success: true}, nil
}
func (r *fakeVersionRepo) UpdateChargePolicyVersion(_ context.Context, req *versionpb.UpdateChargePolicyVersionRequest) (*versionpb.UpdateChargePolicyVersionResponse, error) {
	m, ok := r.s.update(req.Data)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &versionpb.UpdateChargePolicyVersionResponse{Data: []*versionpb.ChargePolicyVersion{m.(*versionpb.ChargePolicyVersion)}, Success: true}, nil
}
func (r *fakeVersionRepo) DeleteChargePolicyVersion(_ context.Context, req *versionpb.DeleteChargePolicyVersionRequest) (*versionpb.DeleteChargePolicyVersionResponse, error) {
	if !r.s.del(req.Data.Id) {
		return nil, fmt.Errorf("record not found")
	}
	return &versionpb.DeleteChargePolicyVersionResponse{Success: true}, nil
}
func (r *fakeVersionRepo) ListChargePolicyVersions(_ context.Context, req *versionpb.ListChargePolicyVersionsRequest) (*versionpb.ListChargePolicyVersionsResponse, error) {
	out := &versionpb.ListChargePolicyVersionsResponse{Success: true}
	for _, m := range r.s.list(req.GetFilters()) {
		out.Data = append(out.Data, m.(*versionpb.ChargePolicyVersion))
	}
	return out, nil
}

func (r *fakeVersionRepo) LockChargePolicyVersionForUpdate(_ context.Context, id string) (*versionpb.ChargePolicyVersion, error) {
	m, ok := r.s.read(id)
	if !ok {
		return nil, fmt.Errorf("charge_policy_version lock: %w", domainports.ErrLockedRowNotFound)
	}
	return m.(*versionpb.ChargePolicyVersion), nil
}

// fakeEditorRepo mirrors the charge_policy_version_editor table incl. UNIQUE(version,user).
type fakeEditorRepo struct {
	editorpb.UnimplementedChargePolicyVersionEditorDomainServiceServer
	s *memStore
}

func (r *fakeEditorRepo) CreateChargePolicyVersionEditor(_ context.Context, req *editorpb.CreateChargePolicyVersionEditorRequest) (*editorpb.CreateChargePolicyVersionEditorResponse, error) {
	for _, m := range r.s.rows {
		e := m.(*editorpb.ChargePolicyVersionEditor)
		if e.GetChargePolicyVersionId() == req.Data.GetChargePolicyVersionId() && e.GetUserId() == req.Data.GetUserId() {
			return nil, fmt.Errorf("duplicate key value violates unique constraint")
		}
	}
	return &editorpb.CreateChargePolicyVersionEditorResponse{Data: []*editorpb.ChargePolicyVersionEditor{r.s.create(req.Data).(*editorpb.ChargePolicyVersionEditor)}, Success: true}, nil
}
func (r *fakeEditorRepo) DeleteChargePolicyVersionEditor(_ context.Context, req *editorpb.DeleteChargePolicyVersionEditorRequest) (*editorpb.DeleteChargePolicyVersionEditorResponse, error) {
	if !r.s.del(req.Data.Id) {
		return nil, fmt.Errorf("record not found")
	}
	return &editorpb.DeleteChargePolicyVersionEditorResponse{Success: true}, nil
}
func (r *fakeEditorRepo) ListChargePolicyVersionEditors(_ context.Context, req *editorpb.ListChargePolicyVersionEditorsRequest) (*editorpb.ListChargePolicyVersionEditorsResponse, error) {
	out := &editorpb.ListChargePolicyVersionEditorsResponse{Success: true}
	for _, m := range r.s.list(req.GetFilters()) {
		out.Data = append(out.Data, m.(*editorpb.ChargePolicyVersionEditor))
	}
	return out, nil
}

type fakeTaxRepo struct {
	taxtreatmentpb.UnimplementedTaxTreatmentDomainServiceServer
	ids map[string]bool
}

func (r *fakeTaxRepo) ReadTaxTreatment(_ context.Context, req *taxtreatmentpb.ReadTaxTreatmentRequest) (*taxtreatmentpb.ReadTaxTreatmentResponse, error) {
	if !r.ids[req.Data.Id] {
		return nil, fmt.Errorf("record not found")
	}
	return &taxtreatmentpb.ReadTaxTreatmentResponse{Data: []*taxtreatmentpb.TaxTreatment{{Id: req.Data.Id}}, Success: true}, nil
}

type fakeComponentRepo struct {
	componentpb.UnimplementedChargePolicyComponentDomainServiceServer
	s *memStore
}

func (r *fakeComponentRepo) CreateChargePolicyComponent(_ context.Context, req *componentpb.CreateChargePolicyComponentRequest) (*componentpb.CreateChargePolicyComponentResponse, error) {
	return &componentpb.CreateChargePolicyComponentResponse{Data: []*componentpb.ChargePolicyComponent{r.s.create(req.Data).(*componentpb.ChargePolicyComponent)}, Success: true}, nil
}
func (r *fakeComponentRepo) ReadChargePolicyComponent(_ context.Context, req *componentpb.ReadChargePolicyComponentRequest) (*componentpb.ReadChargePolicyComponentResponse, error) {
	m, ok := r.s.read(req.Data.Id)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &componentpb.ReadChargePolicyComponentResponse{Data: []*componentpb.ChargePolicyComponent{m.(*componentpb.ChargePolicyComponent)}, Success: true}, nil
}
func (r *fakeComponentRepo) UpdateChargePolicyComponent(_ context.Context, req *componentpb.UpdateChargePolicyComponentRequest) (*componentpb.UpdateChargePolicyComponentResponse, error) {
	m, ok := r.s.update(req.Data)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &componentpb.UpdateChargePolicyComponentResponse{Data: []*componentpb.ChargePolicyComponent{m.(*componentpb.ChargePolicyComponent)}, Success: true}, nil
}
func (r *fakeComponentRepo) DeleteChargePolicyComponent(_ context.Context, req *componentpb.DeleteChargePolicyComponentRequest) (*componentpb.DeleteChargePolicyComponentResponse, error) {
	if !r.s.del(req.Data.Id) {
		return nil, fmt.Errorf("record not found")
	}
	return &componentpb.DeleteChargePolicyComponentResponse{Success: true}, nil
}
func (r *fakeComponentRepo) ListChargePolicyComponents(_ context.Context, req *componentpb.ListChargePolicyComponentsRequest) (*componentpb.ListChargePolicyComponentsResponse, error) {
	out := &componentpb.ListChargePolicyComponentsResponse{Success: true}
	for _, m := range r.s.list(req.GetFilters()) {
		out.Data = append(out.Data, m.(*componentpb.ChargePolicyComponent))
	}
	return out, nil
}

type fakePostingRepo struct {
	postingpb.UnimplementedChargePolicyPostingDomainServiceServer
	s *memStore
}

func (r *fakePostingRepo) CreateChargePolicyPosting(_ context.Context, req *postingpb.CreateChargePolicyPostingRequest) (*postingpb.CreateChargePolicyPostingResponse, error) {
	return &postingpb.CreateChargePolicyPostingResponse{Data: []*postingpb.ChargePolicyPosting{r.s.create(req.Data).(*postingpb.ChargePolicyPosting)}, Success: true}, nil
}
func (r *fakePostingRepo) ReadChargePolicyPosting(_ context.Context, req *postingpb.ReadChargePolicyPostingRequest) (*postingpb.ReadChargePolicyPostingResponse, error) {
	m, ok := r.s.read(req.Data.Id)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &postingpb.ReadChargePolicyPostingResponse{Data: []*postingpb.ChargePolicyPosting{m.(*postingpb.ChargePolicyPosting)}, Success: true}, nil
}
func (r *fakePostingRepo) UpdateChargePolicyPosting(_ context.Context, req *postingpb.UpdateChargePolicyPostingRequest) (*postingpb.UpdateChargePolicyPostingResponse, error) {
	m, ok := r.s.update(req.Data)
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &postingpb.UpdateChargePolicyPostingResponse{Data: []*postingpb.ChargePolicyPosting{m.(*postingpb.ChargePolicyPosting)}, Success: true}, nil
}
func (r *fakePostingRepo) DeleteChargePolicyPosting(_ context.Context, req *postingpb.DeleteChargePolicyPostingRequest) (*postingpb.DeleteChargePolicyPostingResponse, error) {
	if !r.s.del(req.Data.Id) {
		return nil, fmt.Errorf("record not found")
	}
	return &postingpb.DeleteChargePolicyPostingResponse{Success: true}, nil
}
func (r *fakePostingRepo) ListChargePolicyPostings(_ context.Context, req *postingpb.ListChargePolicyPostingsRequest) (*postingpb.ListChargePolicyPostingsResponse, error) {
	out := &postingpb.ListChargePolicyPostingsResponse{Success: true}
	for _, m := range r.s.list(req.GetFilters()) {
		out.Data = append(out.Data, m.(*postingpb.ChargePolicyPosting))
	}
	return out, nil
}

type fakeAccountRepo struct {
	accountpb.UnimplementedAccountDomainServiceServer
	accounts map[string]bool // id -> active
}

func (r *fakeAccountRepo) ReadAccount(_ context.Context, req *accountpb.ReadAccountRequest) (*accountpb.ReadAccountResponse, error) {
	active, ok := r.accounts[req.Data.Id]
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	return &accountpb.ReadAccountResponse{Data: []*accountpb.Account{{Id: req.Data.Id, Active: active}}, Success: true}, nil
}

// --- services doubles ---

type testAuthorizer struct{ perms map[string]bool }

func (a *testAuthorizer) IsEnabled() bool { return true }
func (a *testAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return a.perms[permission], nil
}

type testTransactor struct{ calls, rollbacks int }

func (t *testTransactor) SupportsTransactions() bool               { return true }
func (t *testTransactor) IsTransactionActive(context.Context) bool { return false }
func (t *testTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.calls++
	if err := fn(ctx); err != nil {
		t.rollbacks++
		return err
	}
	return nil
}

type seqIDs struct{ n atomic.Int64 }

func (s *seqIDs) GenerateID() string                        { return fmt.Sprintf("id-%03d", s.n.Add(1)) }
func (s *seqIDs) IsEnabled() bool                           { return true }
func (s *seqIDs) GetProviderInfo() string                   { return "seq" }
func (s *seqIDs) GenerateIDWithPrefix(prefix string) string { return prefix + "-" + s.GenerateID() }

// harness bundles the use cases with their fakes.
type harness struct {
	uc         *UseCases
	policies   *fakePolicyRepo
	versions   *fakeVersionRepo
	components *fakeComponentRepo
	postings   *fakePostingRepo
	editors    *fakeEditorRepo
	taxes      *fakeTaxRepo
	accounts   *fakeAccountRepo
	authz      *testAuthorizer
	tx         *testTransactor
	ids        *seqIDs
}

var allPerms = []string{"list", "read", "create", "update", "delete", "retire", "approve"}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{
		policies:   &fakePolicyRepo{s: newMemStore(nil), referenced: map[string]bool{}},
		versions:   &fakeVersionRepo{s: newMemStore(nil)},
		components: &fakeComponentRepo{s: newMemStore(nil)},
		postings:   &fakePostingRepo{s: newMemStore(nil)},
		editors:    &fakeEditorRepo{s: newMemStore(nil)},
		taxes:      &fakeTaxRepo{ids: map[string]bool{"tax-ok": true}},
		accounts:   &fakeAccountRepo{accounts: map[string]bool{"acct-recv": true, "acct-clear": true, "acct-cash": true, "acct-inactive": false}},
		authz:      &testAuthorizer{perms: map[string]bool{}},
		tx:         &testTransactor{},
		ids:        &seqIDs{},
	}
	for _, p := range allPerms {
		h.authz.perms["charge_policy:"+p] = true
	}
	h.uc = NewUseCases(Repositories{
		ChargePolicy: h.policies, ChargePolicyVersion: h.versions, ChargePolicyComponent: h.components,
		ChargePolicyPosting: h.postings, ChargePolicyVersionEditor: h.editors, Account: h.accounts, TaxTreatment: h.taxes,
	}, Services{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       h.tx,
		Translator:       ports.NewNoOpTranslator(),
		IDGenerator:      h.ids,
		ActionGatekeeper: actiongate.NewActionGatekeeper(h.authz, ports.NewNoOpTranslator()),
	})
	return h
}

func (h *harness) repos() Repositories {
	return Repositories{ChargePolicy: h.policies, ChargePolicyVersion: h.versions, ChargePolicyComponent: h.components,
		ChargePolicyPosting: h.postings, ChargePolicyVersionEditor: h.editors, Account: h.accounts, TaxTreatment: h.taxes}
}

func (h *harness) svc() Services {
	return Services{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       h.tx,
		Translator:       ports.NewNoOpTranslator(),
		IDGenerator:      h.ids,
		ActionGatekeeper: actiongate.NewActionGatekeeper(h.authz, ports.NewNoOpTranslator()),
	}
}

func (h *harness) comps(ctx context.Context, versionID string) []*componentpb.ChargePolicyComponent {
	r, _ := h.uc.ListChargePolicyComponents.Execute(ctx, &componentpb.ListChargePolicyComponentsRequest{ChargePolicyVersionId: strp(versionID)})
	return r.GetData()
}

func (h *harness) posts(ctx context.Context, versionID string) []*postingpb.ChargePolicyPosting {
	r, _ := h.uc.ListChargePolicyPostings.Execute(ctx, &postingpb.ListChargePolicyPostingsRequest{ChargePolicyVersionId: strp(versionID)})
	return r.GetData()
}

func ctxAs(user string) context.Context { return contextutil.WithUserID(context.Background(), user) }

// seedApprovableDraft creates a policy (prepared by preparer) and configures its draft v1 to
// satisfy the S1 matrix and checklist. Returns the policy and draft version.
func (h *harness) seedApprovableDraft(t *testing.T, preparer, code string) (*policypb.ChargePolicy, *versionpb.ChargePolicyVersion) {
	t.Helper()
	ctx := ctxAs(preparer)
	created, err := h.uc.CreateChargePolicy.Execute(ctx, &policypb.CreateChargePolicyRequest{Data: &policypb.ChargePolicy{Code: code, Name: "Utility recovery"}})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	vid := created.Draft.GetId()
	role := enumspb.AccountingRole_ACCOUNTING_ROLE_AGENT
	tax := enumspb.TaxPosition_TAX_POSITION_EXCLUDED_REIMBURSEMENT
	if _, err := h.uc.UpdateChargePolicyVersion.Execute(ctx, &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{
		Id: vid, AccountingRole: &role, TaxPosition: &tax, AssessmentScope: strp("Utility pass-through to tenants"),
	}}); err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if _, err := h.uc.CreateChargePolicyComponent.Execute(ctx, &componentpb.CreateChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{
		ChargePolicyVersionId: vid,
		ComponentRole:         enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST,
		DocumentKind:          enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT,
		BookPresentation:      enumspb.BookPresentation_BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT,
	}}); err != nil {
		t.Fatalf("create component: %v", err)
	}
	for _, p := range []struct {
		e    enumspb.ChargePostingEvent
		r    enumspb.ChargePostingRole
		acct string
	}{
		{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE, "acct-recv"},
		{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CLEARING, "acct-clear"},
		{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH, "acct-cash"},
		{enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE, "acct-recv"},
	} {
		if _, err := h.uc.CreateChargePolicyPosting.Execute(ctx, &postingpb.CreateChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{
			ChargePolicyVersionId: vid, Event: p.e, PostingRole: p.r, AccountId: p.acct,
		}}); err != nil {
			t.Fatalf("create posting: %v", err)
		}
	}
	return created.Data[0], created.Draft
}
