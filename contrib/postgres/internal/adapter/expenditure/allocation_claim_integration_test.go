//go:build postgresql

package expenditure

import (
	"context"
	"errors"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	ledgeradapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/ledger"
	subscriptionadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/subscription"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	allocationbatchuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/allocation_batch"
	expenserecognition "github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure/expense_recognition"
	"github.com/erniealice/espyna-golang/registry/entityid"
	txbridge "github.com/erniealice/espyna-golang/shared/database/transactions"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
)

// replicaTransactor joins/opens the real transaction and disables FK triggers for it so the
// behaviour is exercised without seeding the parent graph (subscriptions, clients, plans ...).
// Same trade-off as the scopetest harness: this proves claim/lock/unique-key/filter behaviour against the
// REAL adapters, not referential integrity.
type replicaTransactor struct {
	inner ports.Transactor
	h     *scopetest.Harness
}

func (r replicaTransactor) SupportsTransactions() bool { return true }
func (r replicaTransactor) IsTransactionActive(ctx context.Context) bool {
	return r.inner.IsTransactionActive(ctx)
}
func (r replicaTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return r.inner.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		if exec := postgresCore.TxExecutor(txCtx, r.h.Ops); exec != nil {
			if _, err := exec.ExecContext(txCtx, "SET LOCAL session_replication_role = replica"); err != nil {
				return err
			}
		}
		return fn(txCtx)
	})
}

type draft struct {
	Batch  *allocationbatchpb.AllocationBatch
	Shares []*allocationsharepb.AllocationShare
}

func createReq(componentID string, shares []*allocationsharepb.AllocationShare) *allocationbatchpb.CreateAllocationBatchRequest {
	return &allocationbatchpb.CreateAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{CostSourceComponentId: componentID}, Shares: shares}
}
func publishReq(id string) *allocationbatchpb.PublishAllocationBatchRequest {
	return &allocationbatchpb.PublishAllocationBatchRequest{AllocationBatchId: id}
}

type intSeq struct{ n int }

func (s *intSeq) GenerateID() string {
	s.n++
	return "w3a-" + string(rune('a'+s.n/26)) + string(rune('a'+s.n%26))
}
func (s *intSeq) IsEnabled() bool                           { return true }
func (s *intSeq) GetProviderInfo() string                   { return "seq" }
func (s *intSeq) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

type pgRecogRepo struct {
	expenserecognitionpb.UnimplementedExpenseRecognitionDomainServiceServer
	n int
}

func (p *pgRecogRepo) CreateExpenseRecognition(_ context.Context, r *expenserecognitionpb.CreateExpenseRecognitionRequest) (*expenserecognitionpb.CreateExpenseRecognitionResponse, error) {
	p.n++
	return &expenserecognitionpb.CreateExpenseRecognitionResponse{Data: []*expenserecognitionpb.ExpenseRecognition{r.Data}, Success: true}, nil
}

type pgExpRepo struct {
	expenditurepb.UnimplementedExpenditureDomainServiceServer
}

func (pgExpRepo) ReadExpenditure(_ context.Context, r *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	return &expenditurepb.ReadExpenditureResponse{Data: []*expenditurepb.Expenditure{{Id: r.Data.Id}}}, nil
}

// W3-A on the real adapters: allocation publish claims the source with a real row lock and writes
// the claim through the postgres adapter; recognition is then refused; and the reverse order
// refuses publish. Runs in ONE rolled-back transaction on the leasing_usage1 clone.
func TestAllocationClaimOnPostgres(t *testing.T) {
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
	}, allocationbatchuc.Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, Transactor: tr, IDGenerator: &intSeq{}})

	recs := &pgRecogRepo{}
	recognize := expenserecognition.NewRecognizeFromExpenditureUseCase(
		expenserecognition.RecognizeFromExpenditureRepositories{ExpenseRecognition: recs, Expenditure: pgExpRepo{}, CostSourceComponent: comps},
		expenserecognition.RecognizeFromExpenditureServices{Authorizer: ports.NewNoOpAuthorizer(), Transactor: tr, Translator: ports.NewNoOpTranslator(),
			ActionGatekeeper: gate, IDGenerator: &intSeq{n: 400}})

	code := func(err error) string {
		var ce interface{ ErrorCode() string }
		if errors.As(err, &ce) {
			return ce.ErrorCode()
		}
		return ""
	}
	shareInputs := []*allocationsharepb.AllocationShare{
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, BasisNumerator: 1, BasisDenominator: 3},
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: scopetest.Str("w3a-sub-1"), ClientId: scopetest.Str("w3a-cli-1"), BasisNumerator: 1, BasisDenominator: 3},
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: scopetest.Str("w3a-sub-2"), ClientId: scopetest.Str("w3a-cli-2"), BasisNumerator: 1, BasisDenominator: 3},
	}

	h.Run(t, func(base, ctxA, _, _ context.Context) {
		exec := postgresCore.TxExecutor(base, h.Ops)
		must := func(q string, args ...any) {
			if _, err := exec.ExecContext(base, q, args...); err != nil {
				t.Fatalf("fixture %q: %v", q, err)
			}
		}
		ws := scopetest.WsA
		for _, c := range []string{"w3a-comp-1", "w3a-comp-2"} {
			must(`INSERT INTO cost_source_component (id, workspace_id, expenditure_id, component_kind, amount, currency, source_version, active, service_from, service_to)
			      VALUES ($1,$2,$3,'COST_SOURCE_COMPONENT_KIND_ENERGY',10000,'PHP',1,true,'2026-01-01','2026-02-01')`, c, ws, "w3a-exp-"+c[len(c)-1:])
		}
		must(`INSERT INTO charge_policy_version (id, workspace_id, charge_policy_id, version_number, status, self_approved, active, tax_position)
		      VALUES ('w3a-ver-1',$1,'w3a-pol-1',1,'CHARGE_POLICY_VERSION_STATUS_APPROVED',false,true,'TAX_POSITION_EXCLUDED_REIMBURSEMENT')`, ws)
		must(`INSERT INTO charge_policy_component (id, workspace_id, charge_policy_version_id, component_role, document_kind, book_presentation, sequence_order, active)
		      VALUES ('w3a-pc-1',$1,'w3a-ver-1','CHARGE_COMPONENT_ROLE_RECOVERY_COST','CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT','BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT',0,true)`, ws)
		for i, s := range []string{"w3a-sub-1", "w3a-sub-2"} {
			must(`INSERT INTO agreement_line_term (id, workspace_id, subscription_id, product_price_plan_id, client_id, charge_policy_version_id, effective_from, origin, active)
			      VALUES ($1,$2,$3,'w3a-ppp-1',$4,'w3a-ver-1','2026-01-01','AGREEMENT_LINE_TERM_ORIGIN_COPIED',true)`,
				"w3a-term-"+string(rune('1'+i)), ws, s, "w3a-cli-"+string(rune('1'+i)))
		}
		ctx := contextutil.WithUserID(ctxA, "w3a-user")

		// ---- order 1: allocation first (component 1 / expenditure 1) ----
		dr, err := uc.CreateAllocationBatch.Execute(ctx, createReq("w3a-comp-1", shareInputs))
		if err != nil {
			t.Fatalf("create draft: %v", err)
		}
		d := draft{Batch: dr.Data[0], Shares: dr.Shares}
		var sum int64
		for _, s := range d.Shares {
			sum += s.Amount
		}
		if sum != 10000 || d.Batch.Revision != 1 {
			t.Fatalf("draft wrong: sum=%d rev=%d", sum, d.Batch.Revision)
		}
		pub, err := uc.PublishAllocationBatch.Execute(ctx, publishReq(d.Batch.Id))
		if err != nil {
			t.Fatalf("publish: %v", err)
		}
		if len(pub.Charges) != 2 {
			t.Fatalf("charges: %d", len(pub.Charges))
		}
		got, err := comps.ReadCostSourceComponent(ctx, &costsourcecomponentpb.ReadCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: "w3a-comp-1"}})
		if err != nil || got.Data[0].GetClaimKind() != costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_ALLOCATION || got.Data[0].GetClaimRefId() != d.Batch.Id || got.Data[0].GetClaimedAt() == 0 {
			t.Fatalf("claim not persisted by the adapter: %v %+v", err, got)
		}
		listed, err := charges.ListBillableCharges(ctx, &billablechargepb.ListBillableChargesRequest{Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "obligation_key", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: pub.Charges[0].ObligationKey, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}}}}}})
		if err != nil || len(listed.Data) != 1 || listed.Data[0].ContentHash == "" {
			t.Fatalf("obligation-key filter on the real adapter: %v %+v", err, listed)
		}
		if _, err := recognize.Execute(ctx, &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "w3a-exp-1"}); code(err) != "source_claimed_by_allocation" {
			t.Fatalf("recognition after allocation: %v", err)
		}
		if _, err := uc.PublishAllocationBatch.Execute(ctx, publishReq(d.Batch.Id)); code(err) != "already_published" {
			t.Fatalf("republish: %v", err)
		}

		// ---- order 2: recognition first (component 2 / expenditure 2) ----
		dr2, err := uc.CreateAllocationBatch.Execute(ctx, createReq("w3a-comp-2", shareInputs))
		if err != nil {
			t.Fatal(err)
		}
		d2 := draft{Batch: dr2.Data[0], Shares: dr2.Shares}
		if _, err := recognize.Execute(ctx, &expenserecognitionpb.RecognizeFromExpenditureRequest{ExpenditureId: "w3a-exp-2"}); err != nil {
			t.Fatalf("recognition of an unclaimed source: %v", err)
		}
		got2, _ := comps.ReadCostSourceComponent(ctx, &costsourcecomponentpb.ReadCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: "w3a-comp-2"}})
		if got2.Data[0].GetClaimKind() != costsourcecomponentpb.SourceClaimKind_SOURCE_CLAIM_KIND_RECOGNITION {
			t.Fatalf("recognition claim not persisted: %+v", got2.Data[0])
		}
		if _, err := uc.PublishAllocationBatch.Execute(ctx, publishReq(d2.Batch.Id)); code(err) != "source_claimed_by_recognition" {
			t.Fatalf("publish after recognition: %v", err)
		}
	})
}
