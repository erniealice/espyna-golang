//go:build postgresql

package expenditure

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	ledgeradapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/ledger"
	subscriptionadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/subscription"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	allocationbatchuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/allocation_batch"
	expenserecognition "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expense_recognition"
	"github.com/erniealice/espyna-golang/registry/entityid"
	txbridge "github.com/erniealice/espyna-golang/shared/database/transactions"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// F3 conformance tests on the REAL adapters and a real database (C6, C7, C16, R1 blast radius).
// Rolled-back scenarios use scopetest.Harness.Run; the recognition and race scenarios need
// COMMITTED rows (the legacy recognition path is non-transactional; two connections must see each
// other) so they use scopetest.NewReplica + SeedCommitted, which delete their rows again by id prefix.

type prefixSeq struct {
	prefix string
	n      atomic.Int64
}

func (s *prefixSeq) GenerateID() string {
	return fmt.Sprintf("%s%04d", s.prefix, s.n.Add(1))
}
func (s *prefixSeq) IsEnabled() bool                           { return true }
func (s *prefixSeq) GetProviderInfo() string                   { return "seq" }
func (s *prefixSeq) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

func errCode(err error) string {
	var ce interface{ ErrorCode() string }
	if errors.As(err, &ce) {
		return ce.ErrorCode()
	}
	return ""
}

// ---- C7 zero weight + C6 immutable deletes ---------------------------------------------------

func TestZeroWeightShareAndImmutableDeletesOnPostgres(t *testing.T) {
	h := scopetest.New(t, "cost_source_component", "allocation_batch", "allocation_share", "agreement_line_term", "billable_charge", "charge_component", "charge_policy_version", "charge_policy_component")
	comps := NewPostgresCostSourceComponentRepository(h.Ops, entityid.CostSourceComponent)
	batches := NewPostgresAllocationBatchRepository(h.Ops, entityid.AllocationBatch)
	shares := NewPostgresAllocationShareRepository(h.Ops, entityid.AllocationShare)
	terms := subscriptionadapter.NewPostgresAgreementLineTermRepository(h.Ops, entityid.AgreementLineTerm)
	charges := subscriptionadapter.NewPostgresBillableChargeRepository(h.Ops, entityid.BillableCharge)
	chargeComps := subscriptionadapter.NewPostgresChargeComponentRepository(h.Ops, entityid.ChargeComponent)
	versions := ledgeradapter.NewPostgresChargePolicyVersionRepository(h.Ops, entityid.ChargePolicyVersion)
	policyComps := ledgeradapter.NewPostgresChargePolicyComponentRepository(h.Ops, entityid.ChargePolicyComponent)
	tr := replicaTransactor{inner: txbridge.NewTransactionServiceAdapter(h.Tm), h: h}
	gate := actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
	uc := allocationbatchuc.NewUseCases(allocationbatchuc.Repositories{
		AllocationBatch: batches, AllocationShare: shares, CostSourceComponent: comps, AgreementLineTerm: terms,
		ChargePolicyVersion: versions, ChargePolicyComponent: policyComps, BillableCharge: charges, ChargeComponent: chargeComps,
	}, allocationbatchuc.Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, Transactor: tr, IDGenerator: &prefixSeq{prefix: "f3z-"}})

	h.Run(t, func(base, ctxA, ctxB, _ context.Context) {
		exec := postgresCore.TxExecutor(base, h.Ops)
		must := func(q string, args ...any) {
			if _, err := exec.ExecContext(base, q, args...); err != nil {
				t.Fatalf("fixture %q: %v", q, err)
			}
		}
		ws := scopetest.WsA
		must(`INSERT INTO cost_source_component (id, workspace_id, expenditure_id, component_kind, amount, currency, source_version, active, service_from, service_to)
		      VALUES ('f3z-comp-1',$1,'f3z-exp-1','COST_SOURCE_COMPONENT_KIND_ENERGY',10000,'PHP',1,true,'2026-01-01','2026-02-01')`, ws)
		must(`INSERT INTO charge_policy_version (id, workspace_id, charge_policy_id, version_number, status, self_approved, active, tax_position)
		      VALUES ('f3z-ver-1',$1,'f3z-pol-1',1,'CHARGE_POLICY_VERSION_STATUS_APPROVED',false,true,'TAX_POSITION_EXCLUDED_REIMBURSEMENT')`, ws)
		must(`INSERT INTO charge_policy_component (id, workspace_id, charge_policy_version_id, component_role, document_kind, book_presentation, sequence_order, active)
		      VALUES ('f3z-pc-1',$1,'f3z-ver-1','CHARGE_COMPONENT_ROLE_RECOVERY_COST','CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT','BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT',0,true)`, ws)
		for i, s := range []string{"f3z-sub-1", "f3z-sub-2"} {
			must(`INSERT INTO agreement_line_term (id, workspace_id, subscription_id, product_price_plan_id, client_id, charge_policy_version_id, effective_from, origin, active)
			      VALUES ($1,$2,$3,'f3z-ppp-1',$4,'f3z-ver-1','2026-01-01','AGREEMENT_LINE_TERM_ORIGIN_COPIED',true)`,
				"f3z-term-"+string(rune('1'+i)), ws, s, "f3z-cli-"+string(rune('1'+i)))
		}
		ctx := contextutil.WithUserID(ctxA, "f3z-user")

		// C7: a legal zero-weight share (own use 0/4, recoverable sub-2 0/4) is stored as 0, not NULL.
		in := []*allocationsharepb.AllocationShare{
			{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, BasisNumerator: 0, BasisDenominator: 4},
			{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: scopetest.Str("f3z-sub-1"), ClientId: scopetest.Str("f3z-cli-1"), BasisNumerator: 4, BasisDenominator: 4},
			{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: scopetest.Str("f3z-sub-2"), ClientId: scopetest.Str("f3z-cli-2"), BasisNumerator: 0, BasisDenominator: 4},
		}
		dr, err := uc.CreateAllocationBatch.Execute(ctx, createReq("f3z-comp-1", in))
		if err != nil {
			t.Fatalf("a zero-weight share must be storable on real postgres: %v", err)
		}
		batch := dr.Data[0]
		var zeroRows, nullRows int
		if err := exec.QueryRowContext(base, `SELECT count(*) FILTER (WHERE basis_numerator = 0 AND amount = 0), count(*) FILTER (WHERE basis_numerator IS NULL OR amount IS NULL)
		                                        FROM allocation_share WHERE allocation_batch_id = $1`, batch.Id).Scan(&zeroRows, &nullRows); err != nil {
			t.Fatal(err)
		}
		if zeroRows != 2 || nullRows != 0 {
			t.Fatalf("zero-weight shares must persist as 0: zero=%d null=%d", zeroRows, nullRows)
		}

		// C6: a DRAFT batch can be deleted at the port, but not its shares once PUBLISHED.
		draftShare := dr.Shares[0]
		pub, err := uc.PublishAllocationBatch.Execute(ctx, publishReq(batch.Id))
		if err != nil {
			t.Fatalf("publish with zero-weight shares: %v", err)
		}
		if len(pub.Charges) != 1 || pub.Charges[0].Amount != 10000 {
			t.Fatalf("only the positive recoverable share may produce a charge: %d", len(pub.Charges))
		}
		if _, err := batches.DeleteAllocationBatch(ctx, &allocationbatchpb.DeleteAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{Id: batch.Id}}); !errors.Is(err, domainports.ErrImmutableRow) {
			t.Fatalf("delete of a PUBLISHED batch must be immutable, got %v", err)
		}
		if _, err := shares.DeleteAllocationShare(ctx, &allocationsharepb.DeleteAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: draftShare.Id}}); !errors.Is(err, domainports.ErrImmutableRow) {
			t.Fatalf("delete of a share of a PUBLISHED batch must be immutable, got %v", err)
		}
		if _, err := charges.DeleteBillableCharge(ctx, &billablechargepb.DeleteBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: pub.Charges[0].Id}}); !errors.Is(err, domainports.ErrImmutableRow) {
			t.Fatalf("billable_charge delete must be immutable, got %v", err)
		}
		var compID string
		if err := exec.QueryRowContext(base, `SELECT id FROM charge_component WHERE billable_charge_id = $1`, pub.Charges[0].Id).Scan(&compID); err != nil {
			t.Fatal(err)
		}
		if _, err := chargeComps.DeleteChargeComponent(ctx, &chargecomponentpb.DeleteChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: compID}}); !errors.Is(err, domainports.ErrImmutableRow) {
			t.Fatalf("charge_component delete must be immutable, got %v", err)
		}
		if _, err := comps.DeleteCostSourceComponent(ctx, &costsourcecomponentpb.DeleteCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: "f3z-comp-1"}}); !errors.Is(err, domainports.ErrImmutableRow) {
			t.Fatalf("delete of a claimed cost source component must be immutable, got %v", err)
		}
		var left int
		if err := exec.QueryRowContext(base, `SELECT (SELECT count(*) FROM allocation_batch WHERE id=$1) + (SELECT count(*) FROM billable_charge WHERE id=$2) + (SELECT count(*) FROM charge_component WHERE id=$3)`,
			batch.Id, pub.Charges[0].Id, compID).Scan(&left); err != nil || left != 3 {
			t.Fatalf("refused deletes must leave the rows: %v left=%d", err, left)
		}

		// a DRAFT batch is deletable (working row): second component, draft only.
		must(`INSERT INTO cost_source_component (id, workspace_id, expenditure_id, component_kind, amount, currency, source_version, active, service_from, service_to)
		      VALUES ('f3z-comp-2',$1,'f3z-exp-2','COST_SOURCE_COMPONENT_KIND_ENERGY',5000,'PHP',1,true,'2026-01-01','2026-02-01')`, ws)
		d2, err := uc.CreateAllocationBatch.Execute(ctx, createReq("f3z-comp-2", in))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := shares.DeleteAllocationShare(ctx, &allocationsharepb.DeleteAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: d2.Shares[0].Id}}); err != nil {
			t.Fatalf("a share of a DRAFT batch is a working row and must be deletable: %v", err)
		}
		if _, err := comps.DeleteCostSourceComponent(ctx, &costsourcecomponentpb.DeleteCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: "f3z-comp-2"}}); err != nil {
			t.Fatalf("an unclaimed component stays deletable at the port: %v", err)
		}
		_ = ctxB
	})
}

// ---- R1 blast radius: recognition without components on real postgres ------------------------

type countingTx struct {
	inner ports.Transactor
	runs  atomic.Int64
}

func (c *countingTx) SupportsTransactions() bool { return true }
func (c *countingTx) IsTransactionActive(ctx context.Context) bool {
	return c.inner.IsTransactionActive(ctx)
}
func (c *countingTx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	c.runs.Add(1)
	return c.inner.ExecuteInTransaction(ctx, fn)
}

type failingLines struct {
	expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

func (failingLines) CreateExpenseRecognitionLine(context.Context, *expenserecognitionlinepb.CreateExpenseRecognitionLineRequest) (*expenserecognitionlinepb.CreateExpenseRecognitionLineResponse, error) {
	return nil, errors.New("simulated line insert failure")
}

func recognitionStack(h *scopetest.Harness, ids *prefixSeq, withClaim bool, lineOverride expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer, tx ports.Transactor) *expenserecognition.RecognizeFromExpenditureUseCase {
	var lines expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer = NewPostgresExpenseRecognitionLineRepository(h.Ops, "expense_recognition_line")
	if lineOverride != nil {
		lines = lineOverride
	}
	repos := expenserecognition.RecognizeFromExpenditureRepositories{
		ExpenseRecognition:     NewPostgresExpenseRecognitionRepository(h.Ops, "expense_recognition"),
		ExpenseRecognitionLine: lines,
		Expenditure:            NewPostgresExpenditureRepository(h.Ops, "expenditure"),
		ExpenditureLineItem:    NewPostgresExpenditureLineItemRepository(h.Ops, "expenditure_line_item"),
	}
	if withClaim {
		repos.CostSourceComponent = NewPostgresCostSourceComponentRepository(h.Ops, entityid.CostSourceComponent)
	}
	return expenserecognition.NewRecognizeFromExpenditureUseCase(repos, expenserecognition.RecognizeFromExpenditureServices{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx, Translator: ports.NewNoOpTranslator(), IDGenerator: ids,
		ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
	})
}

func seedExpenditure(t *testing.T, c *scopetest.Committed, id string) {
	t.Helper()
	c.Exec(t, `INSERT INTO expenditure (id, workspace_id, name, expenditure_type, total_amount, currency, status) VALUES ($1,$2,'F3 recognition',  'expense', 13.00, 'PHP', 'draft')`, id, scopetest.WsA)
	c.Exec(t, `INSERT INTO expenditure_line_item (id, expenditure_id, description, quantity, unit_price, line_amount) VALUES ($1,$2,'Power',2,500,1000)`, id+"-li1", id)
	c.Exec(t, `INSERT INTO expenditure_line_item (id, expenditure_id, description, quantity, unit_price, line_amount) VALUES ($1,$2,'Water',1,300,300)`, id+"-li2", id)
}

func countRows(t *testing.T, h *scopetest.Harness, q string, args ...any) int {
	t.Helper()
	var n int
	if err := h.DB.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// An expenditure WITHOUT cost source components recognises exactly as before whether or not the S1
// claim collaborator is wired, on the real adapters: same rows, no transaction opened, no claim; the
// formerly swallowed line-insert error keeps its old semantics for it. WITH components the claim
// path runs in one transaction and a failed line insert rolls the whole recognition back.
func TestRecognitionBlastRadiusOnPostgres(t *testing.T) {
	h := scopetest.NewReplica(t, "expense_recognition", "expense_recognition_line", "expenditure", "expenditure_line_item", "cost_source_component")
	c := scopetest.SeedCommitted(t, h, "f3rec-", "expense_recognition_line", "expense_recognition", "expenditure_line_item", "expenditure", "cost_source_component")
	for _, e := range []string{"f3rec-exp-base", "f3rec-exp-wired", "f3rec-exp-swallow", "f3rec-exp-claim"} {
		seedExpenditure(t, c, e)
	}
	c.Exec(t, `INSERT INTO cost_source_component (id, workspace_id, expenditure_id, component_kind, amount, currency, source_version, active, service_from, service_to)
	           VALUES ('f3rec-comp-1',$1,'f3rec-exp-claim','COST_SOURCE_COMPONENT_KIND_ENERGY',10000,'PHP',1,true,'2026-01-01','2026-02-01')`, scopetest.WsA)
	ctx := contextutil.WithUserID(scopetest.Ctx(context.Background(), scopetest.WsA), "f3rec-user")
	req := func(exp string) *expenserecognitionpb.RecognizeFromExpenditureRequest {
		p := "2026-01"
		return &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: exp, RecognitionPeriod: &p}
	}
	tx := &countingTx{inner: txbridge.NewTransactionServiceAdapter(h.Tm)}

	baseRes, baseErr := recognitionStack(h, &prefixSeq{prefix: "f3rec-b-"}, false, nil, tx).Execute(ctx, req("f3rec-exp-base"))
	wiredRes, wiredErr := recognitionStack(h, &prefixSeq{prefix: "f3rec-w-"}, true, nil, tx).Execute(ctx, req("f3rec-exp-wired"))
	if (baseErr == nil) != (wiredErr == nil) {
		t.Fatalf("baseline and wired-without-components must behave identically: base=%v wired=%v", baseErr, wiredErr)
	}
	if tx.runs.Load() != 0 {
		t.Fatalf("no components: no transaction may be opened, runs=%d", tx.runs.Load())
	}
	if baseErr != nil {
		// Pre-existing behaviour on the real schema (recognition header NOT NULL columns): both paths
		// fail identically; the semantics tests below need a working header insert, so report clearly.
		t.Fatalf("baseline recognition fails on the real database before any S1 code runs: %v", baseErr)
	}
	lines := func(exp string) int {
		return countRows(t, h, `SELECT count(*) FROM expense_recognition_line l JOIN expense_recognition r ON r.id = l.expense_recognition_id WHERE r.expenditure_id = $1`, exp)
	}
	heads := func(exp string) int {
		return countRows(t, h, `SELECT count(*) FROM expense_recognition WHERE expenditure_id = $1`, exp)
	}
	if heads("f3rec-exp-base") != 1 || heads("f3rec-exp-wired") != 1 || lines("f3rec-exp-base") != 2 || lines("f3rec-exp-wired") != 2 {
		t.Fatalf("same rows expected: base heads=%d lines=%d wired heads=%d lines=%d", heads("f3rec-exp-base"), lines("f3rec-exp-base"), heads("f3rec-exp-wired"), lines("f3rec-exp-wired"))
	}
	if baseRes.GetData().GetIdempotencyKey() != "EXPENDITURE:f3rec-exp-base:2026-01" || wiredRes.GetData().GetIdempotencyKey() != "EXPENDITURE:f3rec-exp-wired:2026-01" {
		t.Fatalf("idempotency key changed: %q %q", baseRes.GetData().GetIdempotencyKey(), wiredRes.GetData().GetIdempotencyKey())
	}
	if n := countRows(t, h, `SELECT count(*) FROM cost_source_component WHERE id LIKE 'f3rec-%' AND claim_kind IS NOT NULL`); n != 0 {
		t.Fatalf("no claim may exist yet: %d", n)
	}

	// Old semantics for a no-component expenditure: the failed line insert is swallowed, the header stays.
	res, err := recognitionStack(h, &prefixSeq{prefix: "f3rec-s-"}, true, failingLines{}, tx).Execute(ctx, req("f3rec-exp-swallow"))
	if err != nil || !res.GetSuccess() || heads("f3rec-exp-swallow") != 1 || lines("f3rec-exp-swallow") != 0 {
		t.Fatalf("no components: the old swallowed line-insert semantics must be kept: err=%v heads=%d lines=%d", err, heads("f3rec-exp-swallow"), lines("f3rec-exp-swallow"))
	}
	if tx.runs.Load() != 0 {
		t.Fatalf("still no transaction expected, runs=%d", tx.runs.Load())
	}

	// With components: one transaction; a failed line insert surfaces and rolls EVERYTHING back (no header, no claim).
	if _, err := recognitionStack(h, &prefixSeq{prefix: "f3rec-c-"}, true, failingLines{}, tx).Execute(ctx, req("f3rec-exp-claim")); err == nil {
		t.Fatal("with components a failed line insert must fail the recognition")
	}
	if tx.runs.Load() != 1 || heads("f3rec-exp-claim") != 0 {
		t.Fatalf("the claim path must be one rolled-back transaction: runs=%d heads=%d", tx.runs.Load(), heads("f3rec-exp-claim"))
	}
	if n := countRows(t, h, `SELECT count(*) FROM cost_source_component WHERE id = 'f3rec-comp-1' AND claim_kind IS NOT NULL`); n != 0 {
		t.Fatal("the rolled-back recognition must leave the component unclaimed")
	}
	// And the happy claim path claims once, in one transaction.
	if _, err := recognitionStack(h, &prefixSeq{prefix: "f3rec-d-"}, true, nil, tx).Execute(ctx, req("f3rec-exp-claim")); err != nil {
		t.Fatalf("recognition of an expenditure with components: %v", err)
	}
	if n := countRows(t, h, `SELECT count(*) FROM cost_source_component WHERE id = 'f3rec-comp-1' AND claim_kind = 'SOURCE_CLAIM_KIND_RECOGNITION'`); n != 1 {
		t.Fatalf("the recognition claim must be persisted: %d", n)
	}
}

// ---- C16 two-connection claim race: allocation publish vs recognition -------------------------

// Two independent pools (real, separate connections) run PublishAllocationBatch and
// RecognizeFromExpenditure on the SAME committed component at the same time; the component row
// lock serialises them and exactly one wins, the loser gets the matching named refusal, and the
// persisted claim / charges / recognition agree with the winner. Repeated over fresh components.
func TestAllocationClaimRaceTwoConnectionsOnPostgres(t *testing.T) {
	tables := []string{"cost_source_component", "allocation_batch", "allocation_share", "agreement_line_term", "billable_charge", "charge_component",
		"charge_policy_version", "charge_policy_component", "expense_recognition", "expense_recognition_line", "expenditure", "expenditure_line_item"}
	setup := scopetest.NewReplica(t, tables...)
	pubPool := scopetest.NewReplica(t, tables...)
	recPool := scopetest.NewReplica(t, tables...)
	c := scopetest.SeedCommitted(t, setup, "f3race-", "charge_component", "billable_charge", "allocation_share", "allocation_batch", "agreement_line_term",
		"charge_policy_component", "charge_policy_version", "expense_recognition_line", "expense_recognition", "expenditure_line_item", "expenditure", "cost_source_component")
	ws := scopetest.WsA
	c.Exec(t, `INSERT INTO charge_policy_version (id, workspace_id, charge_policy_id, version_number, status, self_approved, active, tax_position)
	           VALUES ('f3race-ver-1',$1,'f3race-pol-1',1,'CHARGE_POLICY_VERSION_STATUS_APPROVED',false,true,'TAX_POSITION_EXCLUDED_REIMBURSEMENT')`, ws)
	c.Exec(t, `INSERT INTO charge_policy_component (id, workspace_id, charge_policy_version_id, component_role, document_kind, book_presentation, sequence_order, active)
	           VALUES ('f3race-pc-1',$1,'f3race-ver-1','CHARGE_COMPONENT_ROLE_RECOVERY_COST','CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT','BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT',0,true)`, ws)
	for i, s := range []string{"f3race-sub-1", "f3race-sub-2"} {
		c.Exec(t, `INSERT INTO agreement_line_term (id, workspace_id, subscription_id, product_price_plan_id, client_id, charge_policy_version_id, effective_from, origin, active)
		           VALUES ($1,$2,$3,'f3race-ppp-1',$4,'f3race-ver-1','2026-01-01','AGREEMENT_LINE_TERM_ORIGIN_COPIED',true)`,
			"f3race-term-"+string(rune('1'+i)), ws, s, "f3race-cli-"+string(rune('1'+i)))
	}
	ctx := contextutil.WithUserID(scopetest.Ctx(context.Background(), ws), "f3race-user")

	stack := func(h *scopetest.Harness, prefix string) (*allocationbatchuc.UseCases, *expenserecognition.RecognizeFromExpenditureUseCase) {
		tr := txbridge.NewTransactionServiceAdapter(h.Tm)
		gate := actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
		ids := &prefixSeq{prefix: prefix}
		batchUC := allocationbatchuc.NewUseCases(allocationbatchuc.Repositories{
			AllocationBatch:       NewPostgresAllocationBatchRepository(h.Ops, entityid.AllocationBatch),
			AllocationShare:       NewPostgresAllocationShareRepository(h.Ops, entityid.AllocationShare),
			CostSourceComponent:   NewPostgresCostSourceComponentRepository(h.Ops, entityid.CostSourceComponent),
			AgreementLineTerm:     subscriptionadapter.NewPostgresAgreementLineTermRepository(h.Ops, entityid.AgreementLineTerm),
			ChargePolicyVersion:   ledgeradapter.NewPostgresChargePolicyVersionRepository(h.Ops, entityid.ChargePolicyVersion),
			ChargePolicyComponent: ledgeradapter.NewPostgresChargePolicyComponentRepository(h.Ops, entityid.ChargePolicyComponent),
			BillableCharge:        subscriptionadapter.NewPostgresBillableChargeRepository(h.Ops, entityid.BillableCharge),
			ChargeComponent:       subscriptionadapter.NewPostgresChargeComponentRepository(h.Ops, entityid.ChargeComponent),
		}, allocationbatchuc.Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, Transactor: tr, IDGenerator: ids})
		return batchUC, recognitionStack(h, &prefixSeq{prefix: prefix + "r-"}, true, nil, tr)
	}
	pubUC, _ := stack(pubPool, "f3race-p-")
	_, recUC := stack(recPool, "f3race-q-")
	setupUC, _ := stack(setup, "f3race-s-")

	const iterations = 12
	var pubWins, recWins int
	for i := 0; i < iterations; i++ {
		comp, exp := fmt.Sprintf("f3race-comp-%02d", i), fmt.Sprintf("f3race-exp-%02d", i)
		seedExpenditure(t, c, exp)
		c.Exec(t, `INSERT INTO cost_source_component (id, workspace_id, expenditure_id, component_kind, amount, currency, source_version, active, service_from, service_to)
		           VALUES ($1,$2,$3,'COST_SOURCE_COMPONENT_KIND_ENERGY',10000,'PHP',1,true,'2026-01-01','2026-02-01')`, comp, ws, exp)
		dr, err := setupUC.CreateAllocationBatch.Execute(ctx, createReq(comp, []*allocationsharepb.AllocationShare{
			{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, BasisNumerator: 1, BasisDenominator: 3},
			{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: scopetest.Str("f3race-sub-1"), ClientId: scopetest.Str("f3race-cli-1"), BasisNumerator: 1, BasisDenominator: 3},
			{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: scopetest.Str("f3race-sub-2"), ClientId: scopetest.Str("f3race-cli-2"), BasisNumerator: 1, BasisDenominator: 3},
		}))
		if err != nil {
			t.Fatalf("iteration %d: draft: %v", i, err)
		}
		batchID := dr.Data[0].Id

		var wg sync.WaitGroup
		start := make(chan struct{})
		var pubErr, recErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, pubErr = pubUC.PublishAllocationBatch.Execute(ctx, publishReq(batchID))
		}()
		go func() {
			defer wg.Done()
			<-start
			p := "2026-01"
			_, recErr = recUC.Execute(ctx, &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: exp, RecognitionPeriod: &p})
		}()
		close(start)
		wg.Wait()

		var claim *string
		if err := setup.DB.QueryRow(`SELECT claim_kind FROM cost_source_component WHERE id = $1`, comp).Scan(&claim); err != nil {
			t.Fatal(err)
		}
		charges := countRows(t, setup, `SELECT count(*) FROM billable_charge WHERE obligation_key LIKE $1`, "SRC:"+comp+":%")
		recs := countRows(t, setup, `SELECT count(*) FROM expense_recognition WHERE expenditure_id = $1`, exp)
		switch {
		case pubErr == nil && recErr != nil:
			pubWins++
			if errCode(recErr) != "source_claimed_by_allocation" || claim == nil || *claim != "SOURCE_CLAIM_KIND_ALLOCATION" || charges != 2 || recs != 0 {
				t.Fatalf("iter %d: publish won but state wrong: recErr=%v claim=%v charges=%d recs=%d", i, recErr, claim, charges, recs)
			}
		case recErr == nil && pubErr != nil:
			recWins++
			if errCode(pubErr) != "source_claimed_by_recognition" || claim == nil || *claim != "SOURCE_CLAIM_KIND_RECOGNITION" || charges != 0 || recs != 1 {
				t.Fatalf("iter %d: recognition won but state wrong: pubErr=%v claim=%v charges=%d recs=%d", i, pubErr, claim, charges, recs)
			}
		default:
			t.Fatalf("iter %d: exactly one must win: pubErr=%v recErr=%v", i, pubErr, recErr)
		}
	}
	t.Logf("two-connection race over %d components: publish won %d, recognition won %d", iterations, pubWins, recWins)
}
