package expenserecognition

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

type recRepo struct {
	expenserecognitionpb.UnimplementedExpenseRecognitionDomainServiceServer
	created []*expenserecognitionpb.ExpenseRecognition
}

func (r *recRepo) CreateExpenseRecognition(_ context.Context, q *expenserecognitionpb.CreateExpenseRecognitionRequest) (*expenserecognitionpb.CreateExpenseRecognitionResponse, error) {
	r.created = append(r.created, q.Data)
	return &expenserecognitionpb.CreateExpenseRecognitionResponse{Data: []*expenserecognitionpb.ExpenseRecognition{q.Data}, Success: true}, nil
}

type lineRepo struct {
	expenserecognitionlinepb.UnimplementedExpenseRecognitionLineDomainServiceServer
	created []*expenserecognitionlinepb.ExpenseRecognitionLine
	failErr error // injected insert failure
}

func (r *lineRepo) CreateExpenseRecognitionLine(_ context.Context, q *expenserecognitionlinepb.CreateExpenseRecognitionLineRequest) (*expenserecognitionlinepb.CreateExpenseRecognitionLineResponse, error) {
	if r.failErr != nil {
		return nil, r.failErr
	}
	r.created = append(r.created, q.Data)
	return &expenserecognitionlinepb.CreateExpenseRecognitionLineResponse{Success: true}, nil
}

type expRepo struct {
	expenditurepb.UnimplementedExpenditureDomainServiceServer
}

func (expRepo) ReadExpenditure(_ context.Context, q *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	return &expenditurepb.ReadExpenditureResponse{Data: []*expenditurepb.Expenditure{{Id: q.Data.Id}}}, nil
}

type eliRepo struct {
	expenditurelineitempb.UnimplementedExpenditureLineItemDomainServiceServer
}

func (eliRepo) ListExpenditureLineItems(context.Context, *expenditurelineitempb.ListExpenditureLineItemsRequest) (*expenditurelineitempb.ListExpenditureLineItemsResponse, error) {
	return &expenditurelineitempb.ListExpenditureLineItemsResponse{Data: []*expenditurelineitempb.ExpenditureLineItem{
		{Id: "eli-1", Description: "Power", Quantity: 2, UnitPrice: 500, TotalPrice: 1000},
		{Id: "eli-2", Description: "Water", Quantity: 1, UnitPrice: 300, TotalPrice: 300},
	}}, nil
}

// componentRepo spies: it records any call so the no-components case can prove nothing was touched.
type componentRepo struct {
	costsourcecomponentpb.UnimplementedCostSourceComponentDomainServiceServer
	mu      sync.Mutex
	rows    []*costsourcecomponentpb.CostSourceComponent
	updates int
	lists   int
	locks   int
	listErr error
}

func (c *componentRepo) ListCostSourceComponents(context.Context, *costsourcecomponentpb.ListCostSourceComponentsRequest) (*costsourcecomponentpb.ListCostSourceComponentsResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.listErr != nil {
		return nil, c.listErr
	}
	c.lists++
	return &costsourcecomponentpb.ListCostSourceComponentsResponse{Data: c.rows, Success: true}, nil
}

// lockingComponentRepo adds the postgres locker capability (C4: fakes implement it).
type lockingComponentRepo struct{ *componentRepo }

func (c *lockingComponentRepo) LockCostSourceComponentForUpdate(context.Context, string) (*costsourcecomponentpb.CostSourceComponent, error) {
	return nil, domainports.ErrLockedRowNotFound
}
func (c *lockingComponentRepo) LockCostSourceComponentsByExpenditure(context.Context, string) ([]*costsourcecomponentpb.CostSourceComponent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.locks++
	return c.rows, nil
}
func (c *componentRepo) UpdateCostSourceComponent(context.Context, *costsourcecomponentpb.UpdateCostSourceComponentRequest) (*costsourcecomponentpb.UpdateCostSourceComponentResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updates++
	return &costsourcecomponentpb.UpdateCostSourceComponentResponse{Success: true}, nil
}

type txRunner struct{ runs int }

func (t *txRunner) SupportsTransactions() bool               { return true }
func (t *txRunner) IsTransactionActive(context.Context) bool { return false }
func (t *txRunner) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.runs++
	return fn(ctx)
}

type seq struct{ n int }

func (s *seq) GenerateID() string {
	s.n++
	return "id-" + string(rune('a'+s.n))
}
func (s *seq) IsEnabled() bool                           { return true }
func (s *seq) GetProviderInfo() string                   { return "seq" }
func (s *seq) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

func runRecognize(t *testing.T, comps *componentRepo, tx *txRunner) (*recRepo, *lineRepo, *expenserecognitionpb.RecognizeFromExpenditureResponse) {
	t.Helper()
	recs, lines := &recRepo{}, &lineRepo{}
	resp, err := recognize(comps, tx, recs, lines)
	if err != nil {
		t.Fatal(err)
	}
	return recs, lines, resp
}

func recognize(comps *componentRepo, tx *txRunner, recs *recRepo, lines *lineRepo) (*expenserecognitionpb.RecognizeFromExpenditureResponse, error) {
	repos := RecognizeFromExpenditureRepositories{ExpenseRecognition: recs, ExpenseRecognitionLine: lines, Expenditure: expRepo{}, ExpenditureLineItem: eliRepo{}}
	if comps != nil {
		repos.CostSourceComponent = &lockingComponentRepo{comps}
	}
	uc := NewRecognizeFromExpenditureUseCase(repos, RecognizeFromExpenditureServices{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx, Translator: ports.NewNoOpTranslator(), IDGenerator: &seq{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	})
	return uc.Execute(contextutil.WithUserID(context.Background(), "u1"), &expenserecognitionpb.RecognizeFromExpenditureRequest{
		ExpenditureId: "exp-1", RecognitionPeriod: strp("2026-01")})
}

func strp(s string) *string { return &s }

// AC-VERT-03: an expenditure without cost source components recognises exactly as before, with or
// without the claim collaborator wired, and never writes a claim.
func TestRecognizeFromExpenditureUnchangedWithoutComponents(t *testing.T) {
	baseRecs, baseLines, baseResp := runRecognize(t, nil, &txRunner{})
	comps := &componentRepo{}
	tx := &txRunner{}
	recs, lines, resp := runRecognize(t, comps, tx)

	if comps.updates != 0 || comps.locks != 0 || tx.runs != 0 {
		t.Fatalf("no components: no claim, no lock and no transaction may be used (updates=%d locks=%d tx=%d)", comps.updates, comps.locks, tx.runs)
	}
	if len(recs.created) != 1 || len(baseRecs.created) != 1 || len(lines.created) != 2 || len(baseLines.created) != 2 {
		t.Fatalf("rows: recs %d/%d lines %d/%d", len(recs.created), len(baseRecs.created), len(lines.created), len(baseLines.created))
	}
	a, b := recs.created[0], baseRecs.created[0]
	if a.GetIdempotencyKey() != b.GetIdempotencyKey() || a.GetStatus() != b.GetStatus() || a.GetExpenditureId() != b.GetExpenditureId() || !resp.GetSuccess() || !baseResp.GetSuccess() {
		t.Fatalf("recognition differs: %+v vs %+v", a, b)
	}
	for i := range lines.created {
		x, y := lines.created[i], baseLines.created[i]
		if x.GetDescription() != y.GetDescription() || x.GetAmount() != y.GetAmount() || x.GetQuantity() != y.GetQuantity() || x.GetUnitAmount() != y.GetUnitAmount() {
			t.Fatalf("line %d differs: %+v vs %+v", i, x, y)
		}
	}
	if a.GetIdempotencyKey() != "EXPENDITURE:exp-1:2026-01" {
		t.Fatalf("idempotency key changed: %s", a.GetIdempotencyKey())
	}
}

// Components present: the claim is written once per unclaimed component, in the transaction.
func TestRecognizeFromExpenditureClaimsComponents(t *testing.T) {
	comps := &componentRepo{rows: []*costsourcecomponentpb.CostSourceComponent{
		{Id: "c1", ExpenditureId: "exp-1"},
		{Id: "c2", ExpenditureId: "exp-1", ClaimKind: costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_RECOGNITION.Enum()},
	}}
	tx := &txRunner{}
	runRecognize(t, comps, tx)
	if comps.updates != 1 || tx.runs != 1 {
		t.Fatalf("claims=%d tx=%d", comps.updates, tx.runs)
	}
	comps.rows[1].ClaimKind = costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_ALLOCATION.Enum()
	uc := NewRecognizeFromExpenditureUseCase(RecognizeFromExpenditureRepositories{ExpenseRecognition: &recRepo{}, CostSourceComponent: comps},
		RecognizeFromExpenditureServices{Transactor: tx, Translator: ports.NewNoOpTranslator(), IDGenerator: &seq{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())})
	uc = NewRecognizeFromExpenditureUseCase(RecognizeFromExpenditureRepositories{ExpenseRecognition: &recRepo{}, CostSourceComponent: &lockingComponentRepo{comps}},
		RecognizeFromExpenditureServices{Transactor: tx, Translator: ports.NewNoOpTranslator(), IDGenerator: &seq{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())})
	if _, err := uc.Execute(contextutil.WithUserID(context.Background(), "u1"), &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "exp-1"}); !usecaseerr.IsCode(err, "source_claimed_by_allocation") {
		t.Fatalf("an allocation-claimed component must refuse recognition, got %v", err)
	}
}

// C4: with components but a repository that cannot lock, recognition fails closed (no unlocked fallback).
func TestRecognizeFromExpenditureFailsClosedWithoutLocker(t *testing.T) {
	comps := &componentRepo{rows: []*costsourcecomponentpb.CostSourceComponent{{Id: "c1", ExpenditureId: "exp-1"}}}
	recs := &recRepo{}
	uc := NewRecognizeFromExpenditureUseCase(RecognizeFromExpenditureRepositories{ExpenseRecognition: recs, Expenditure: expRepo{}, ExpenditureLineItem: eliRepo{}, ExpenseRecognitionLine: &lineRepo{}, CostSourceComponent: comps},
		RecognizeFromExpenditureServices{Transactor: &txRunner{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &seq{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())})
	if _, err := uc.Execute(contextutil.WithUserID(context.Background(), "u1"), &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "exp-1"}); !usecaseerr.IsCode(err, "lock_unavailable") {
		t.Fatalf("want lock_unavailable, got %v", err)
	}
	if len(recs.created) != 0 {
		t.Fatal("a refused recognition must not write")
	}
}

// A failing component read is not treated as "no components": recognition refuses (fail closed).
func TestRecognizeFromExpenditureComponentReadFailureRefuses(t *testing.T) {
	comps := &componentRepo{listErr: errors.New("connection reset")}
	recs, lines := &recRepo{}, &lineRepo{}
	if _, err := recognize(comps, &txRunner{}, recs, lines); err == nil {
		t.Fatal("a failing component existence read must refuse recognition")
	}
	if len(recs.created) != 0 {
		t.Fatal("nothing may be written")
	}
}

// Old semantics stay for an expenditure WITHOUT components: a failed line insert is still swallowed
// and the recognition succeeds. WITH components the failed insert aborts the whole operation.
func TestRecognizeFromExpenditureLineInsertFailureSemantics(t *testing.T) {
	boom := errors.New("insert failed")
	recs, lines := &recRepo{}, &lineRepo{failErr: boom}
	resp, err := recognize(&componentRepo{}, &txRunner{}, recs, lines)
	if err != nil || !resp.GetSuccess() || len(recs.created) != 1 {
		t.Fatalf("no components: the old swallowed-insert behaviour must be kept, got resp=%v err=%v", resp, err)
	}
	comps := &componentRepo{rows: []*costsourcecomponentpb.CostSourceComponent{{Id: "c1", ExpenditureId: "exp-1"}}}
	recs, lines = &recRepo{}, &lineRepo{failErr: boom}
	if _, err := recognize(comps, &txRunner{}, recs, lines); !errors.Is(err, boom) {
		t.Fatalf("with components the failed insert must surface, got %v", err)
	}
	if comps.updates != 0 {
		t.Fatal("no claim may be written when the recognition failed")
	}
}

// F7 (pre-existing bug A): the header carries the NOT NULL identity columns of the postgres schema
// (internal_id UNIQUE, name, recognition_date) so the insert succeeds on a real database.
func TestRecognizeFromExpenditureHeaderCarriesRequiredIdentity(t *testing.T) {
	recs, _, _ := runRecognize(t, nil, &txRunner{})
	h := recs.created[0]
	if h.GetInternalId() == "" || h.GetName() == "" || h.GetRecognitionDate() == nil {
		t.Fatalf("header misses internal_id/name/recognition_date: %+v", h)
	}
	if got := h.GetRecognitionDate().AsTime().Format("2006-01-02"); got != "2026-01-31" {
		t.Fatalf("recognition_date = %s, want the period end 2026-01-31", got)
	}
}
