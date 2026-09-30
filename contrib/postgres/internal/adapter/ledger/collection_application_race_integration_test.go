//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	entityadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/entity"
	revenueadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/revenue"
	subscriptionadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/subscription"
	treasuryadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/treasury"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/recovery_document"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/collection_application"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	"google.golang.org/protobuf/proto"
)

// C16 (build-spec §7): two-connection real-database proofs of the financial-document locking claims.
// Every use-case call opens its own transaction (its own pooled connection); the workspace-predicated
// row locks (LockRecoveryDocumentForUpdate / LockRevenueForUpdate / LockDocumentSeriesForUpdate /
// LockBillableChargeForUpdate / LockCollectionApplicationForUpdate) must serialise them. Connections
// cannot see each other's uncommitted rows, so the seed rows are COMMITTED under a dedicated
// workspace id and removed again afterwards (also on failure). The tests need a superuser (FK
// triggers are disabled only on the seed/cleanup sessions) on a disposable usage clone
// (TEST_DATABASE_URL); an unset/unreachable database FAILS the test (core/scopetest).

type dbTransactor struct{ tm interfaces.TransactionManager }

func (t dbTransactor) SupportsTransactions() bool               { return true }
func (t dbTransactor) IsTransactionActive(context.Context) bool { return false }
func (t dbTransactor) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return t.tm.RunInTransaction(ctx, fn)
}

type raceAllow struct{}

func (raceAllow) IsEnabled() bool { return true }
func (raceAllow) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}

// raceIDs hands out unique ids for one race environment.
type raceIDs struct {
	prefix string
	n      atomic.Int64
}

func (r *raceIDs) GenerateID() string {
	return r.prefix + "-gen-" + itoa(r.n.Add(1))
}
func (r *raceIDs) IsEnabled() bool                           { return true }
func (r *raceIDs) GetProviderInfo() string                   { return "c16" }
func (r *raceIDs) GenerateIDWithPrefix(prefix string) string { return prefix + r.GenerateID() }

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// raceEnv is one committed-seed race environment under its own workspace id.
type raceEnv struct {
	t    *testing.T
	ws   string
	db   *sql.DB
	docs *recovery_document.UseCases
	cash *collection_application.UseCases
}

func (e *raceEnv) exec(q string, args ...any) {
	e.t.Helper()
	if _, err := e.db.ExecContext(context.Background(), q, args...); err != nil {
		e.t.Fatalf("race seed/cleanup: %v\n%s", err, q)
	}
}

func (e *raceEnv) cleanup() {
	for _, q := range []string{
		`DELETE FROM collection_application WHERE workspace_id = $1`,
		`DELETE FROM charge_effect WHERE workspace_id = $1`,
		`DELETE FROM treasury_collection WHERE workspace_id = $1`,
		`DELETE FROM recovery_document_line WHERE workspace_id = $1`,
		`DELETE FROM recovery_document WHERE workspace_id = $1`,
		`DELETE FROM charge_component WHERE workspace_id = $1`,
		`DELETE FROM billable_charge WHERE workspace_id = $1`,
		`DELETE FROM charge_policy_posting WHERE workspace_id = $1`,
		`DELETE FROM charge_policy_version WHERE workspace_id = $1`,
		`DELETE FROM charge_policy WHERE workspace_id = $1`,
		`DELETE FROM document_series WHERE workspace_id = $1`,
		`DELETE FROM revenue WHERE workspace_id = $1`,
		`DELETE FROM account WHERE workspace_id = $1`,
		`DELETE FROM client WHERE workspace_id = $1`,
	} {
		_, _ = e.db.ExecContext(context.Background(), q, e.ws)
	}
	_, _ = e.db.ExecContext(context.Background(), `DELETE FROM workspace WHERE id = $1`, e.ws)
}

// newRaceEnv opens the pool, seeds the shared graph (workspace, accounts, an approved policy version
// with ISSUE + APPLICATION postings) and wires real adapters + use cases. Cleanup is registered.
func newRaceEnv(t *testing.T, ws string) *raceEnv {
	t.Helper()
	// A replica pool: every session runs with session_replication_role = replica (superuser), so the
	// committed fixtures need no parent graph; it names the disposable leasing_usage1 clone or FAILS.
	h := scopetest.NewReplica(t, "collection_application", "recovery_document", "document_series")
	db := h.DB
	db.SetMaxOpenConns(12)
	e := &raceEnv{t: t, ws: ws, db: db}
	e.cleanup() // a previous aborted run
	t.Cleanup(e.cleanup)

	e.exec(`INSERT INTO workspace (id) VALUES ($1)`, ws)
	e.exec(`INSERT INTO account (id, workspace_id) VALUES ($2, $1), ($3, $1), ($4, $1)`, ws, ws+"-acc-cash", ws+"-acc-recv", ws+"-acc-clear")
	e.exec(`INSERT INTO charge_policy_version (id, workspace_id, charge_policy_id, version_number, status) VALUES ($2, $1, $3, 1, 'CHARGE_POLICY_VERSION_STATUS_APPROVED')`, ws, ws+"-ver", ws+"-pol")
	e.exec(`INSERT INTO charge_policy_posting (id, workspace_id, charge_policy_version_id, event, posting_role, account_id, active) VALUES
		($2, $1, $3, 'CHARGE_POSTING_EVENT_APPLICATION', 'CHARGE_POSTING_ROLE_CASH', $4, true),
		($5, $1, $3, 'CHARGE_POSTING_EVENT_APPLICATION', 'CHARGE_POSTING_ROLE_RECEIVABLE', $6, true),
		($7, $1, $3, 'CHARGE_POSTING_EVENT_ISSUE', 'CHARGE_POSTING_ROLE_RECEIVABLE', $6, true),
		($8, $1, $3, 'CHARGE_POSTING_EVENT_ISSUE', 'CHARGE_POSTING_ROLE_CLEARING', $9, true)`,
		ws, ws+"-p1", ws+"-ver", ws+"-acc-cash", ws+"-p2", ws+"-acc-recv", ws+"-p3", ws+"-p4", ws+"-acc-clear")

	// Real adapters over the pool (unaudited: audit partitions are not what is under test).
	ops := postgresCore.NewWorkspaceAwareOperations(db)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	gate := actiongate.NewActionGatekeeper(raceAllow{}, ports.NewNoOpTranslator())
	ids := &raceIDs{prefix: ws}
	docRepo := revenueadapter.NewPostgresRecoveryDocumentRepository(ops, "recovery_document")
	lineRepo := revenueadapter.NewPostgresRecoveryDocumentLineRepository(ops, "recovery_document_line")
	appRepo := treasuryadapter.NewPostgresCollectionApplicationRepository(ops, "collection_application")
	chargeRepo := subscriptionadapter.NewPostgresBillableChargeRepository(ops, "billable_charge")
	compRepo := subscriptionadapter.NewPostgresChargeComponentRepository(ops, "charge_component")
	postingRepo := NewPostgresChargePolicyPostingRepository(ops, "charge_policy_posting")
	effectRepo := NewPostgresChargeEffectRepository(ops, "charge_effect")
	svcTx := dbTransactor{tm}
	e.cash = collection_application.NewUseCases(collection_application.Repositories{
		Collection:            treasuryadapter.NewPostgresCollectionRepository(ops, "treasury_collection"),
		CollectionApplication: appRepo,
		Revenue:               revenueadapter.NewPostgresRevenueRepository(ops, "revenue"),
		RecoveryDocument:      docRepo, RecoveryDocumentLine: lineRepo, BillableCharge: chargeRepo, ChargeComponent: compRepo,
		ChargePolicyPosting: postingRepo, ChargeEffect: effectRepo,
		Client: entityadapter.NewPostgresClientRepository(ops, "client"),
	}, collection_application.Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, Transactor: svcTx, IDGenerator: ids})
	e.docs = recovery_document.NewUseCases(recovery_document.Repositories{
		RecoveryDocument: docRepo, RecoveryDocumentLine: lineRepo, DocumentSeries: revenueadapter.NewPostgresDocumentSeriesRepository(ops, "document_series"),
		BillableCharge: chargeRepo, ChargeComponent: compRepo, ChargePolicyPosting: postingRepo, ChargeEffect: effectRepo, CollectionApplication: appRepo,
	}, recovery_document.Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: gate, Transactor: svcTx, IDGenerator: ids})
	return e
}

func (e *raceEnv) ctx() context.Context {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: e.ws})
	return contextutil.WithWorkspaceID(contextutil.WithUserID(ctx, e.ws+"-user"), e.ws)
}

func (e *raceEnv) client(id string) {
	e.exec(`INSERT INTO client (id, name, workspace_id, active) VALUES ($2, 'C16', $1, true)`, e.ws, id)
}

// charge seeds a charge (status) with one recovery-document component of the same amount.
func (e *raceEnv) charge(id, client, status string, amount int64) {
	e.exec(`INSERT INTO billable_charge (id, workspace_id, obligation_key, content_hash, client_id, charge_kind, amount, currency, status, charge_policy_version_id, active)
		VALUES ($2, $1, $3, 'h', $4, 'BILLABLE_CHARGE_KIND_ORIGINAL', $5, 'PHP', $6, $7, true)`, e.ws, id, "key-"+id, client, amount, status, e.ws+"-ver")
	e.exec(`INSERT INTO charge_component (id, workspace_id, billable_charge_id, component_role, document_kind, book_presentation, tax_position, amount, currency, active)
		VALUES ($2, $1, $3, 'CHARGE_COMPONENT_ROLE_RECOVERY_COST', 'CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT', 'BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT', 'TAX_POSITION_EXCLUDED_REIMBURSEMENT', $4, 'PHP', true)`, e.ws, "cc-"+id, id, amount)
}

// issuedDocument seeds an ISSUED statement of `amount` for the client with its charge/component/line.
func (e *raceEnv) issuedDocument(id, client string, amount int64, seq int64) {
	e.charge("ch-"+id, client, "BILLABLE_CHARGE_STATUS_ISSUED", amount)
	e.exec(`INSERT INTO recovery_document (id, workspace_id, document_series_id, sequence_number, document_number, document_type, client_id, issue_date, due_date, total_amount, currency, status, issuance_key, active)
		VALUES ($2, $1, $3, $4, $5, 'RECOVERY_DOCUMENT_TYPE_STATEMENT', $6, '2026-09-01', '2026-09-30', $7, 'PHP', 'RECOVERY_DOCUMENT_STATUS_ISSUED', $8, true)`,
		e.ws, id, e.ws+"-ser-seed", seq, "D-"+id, client, amount, "issue-"+id)
	e.exec(`INSERT INTO recovery_document_line (id, workspace_id, recovery_document_id, charge_component_id, amount, currency, active) VALUES ($2, $1, $3, $4, $5, 'PHP', true)`,
		e.ws, "ln-"+id, id, "cc-ch-"+id, amount)
}

func (e *raceEnv) scalar(q string, args ...any) int64 {
	e.t.Helper()
	var v int64
	if err := e.db.QueryRow(q, args...).Scan(&v); err != nil {
		e.t.Fatalf("%v\n%s", err, q)
	}
	return v
}

func receiveReq(client string, amount int64) *collectionapplicationpb.ReceiveAndApplyCollectionRequest {
	return &collectionapplicationpb.ReceiveAndApplyCollectionRequest{
		ClientId: client, Amount: amount, Currency: "PHP", PaymentDate: proto.String("2026-09-10"), ReferenceNumber: proto.String("OR-C16"),
	}
}

func (e *raceEnv) scalarString(q string, args ...any) string {
	e.t.Helper()
	var v string
	if err := e.db.QueryRow(q, args...).Scan(&v); err != nil {
		e.t.Fatalf("%v\n%s", err, q)
	}
	return v
}

func runConcurrently(workers int, fn func(i int)) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
}

// Receipts racing on ONE open recovery document never over-apply it and the effects stay balanced.
func TestReceiveApplyRaceTwoConnectionsNeverOverApplies(t *testing.T) {
	e := newRaceEnv(t, "c16-race-ws")
	e.client("c16-client")
	e.issuedDocument("c16-doc", "c16-client", 100, 1)

	const workers = 5
	errs := make([]error, workers)
	runConcurrently(workers, func(i int) {
		_, errs[i] = e.cash.ReceiveAndApplyCollection.Execute(e.ctx(), receiveReq("c16-client", 60))
	})
	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d: %v", i, err)
		}
	}
	applied := e.scalar(`SELECT COALESCE(SUM(amount),0) FROM collection_application WHERE workspace_id=$1 AND status='APPLICATION_STATUS_APPLIED'`, e.ws)
	effects := e.scalar(`SELECT COALESCE(SUM(amount),0) FROM charge_effect WHERE workspace_id=$1 AND direction='EFFECT_DIRECTION_DEBIT'`, e.ws)
	receipts := e.scalar(`SELECT COUNT(*) FROM treasury_collection WHERE workspace_id=$1`, e.ws)
	if applied != 100 || effects != 100 || receipts != workers {
		t.Fatalf("applied=%d effects(DR)=%d receipts=%d; want exactly 100 applied (never over-applied), 100 DR effects and %d receipts", applied, effects, receipts, workers)
	}
}

// Receipts racing on ONE invoice (no recovery document at all) serialise on the invoice row lock.
func TestReceiveApplyRaceOnInvoiceNeverOverApplies(t *testing.T) {
	e := newRaceEnv(t, "c16-race-inv-ws")
	e.client("c16-client")
	e.exec(`INSERT INTO revenue (id, workspace_id, client_id, total_amount, currency, status, reference_number, active)
		VALUES ('c16-rent', $1, 'c16-client', 100, 'PHP', 'complete', 'RENT-1', true)`, e.ws)

	const workers = 5
	errs := make([]error, workers)
	runConcurrently(workers, func(i int) {
		_, errs[i] = e.cash.ReceiveAndApplyCollection.Execute(e.ctx(), receiveReq("c16-client", 60))
	})
	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d: %v", i, err)
		}
	}
	applied := e.scalar(`SELECT COALESCE(SUM(amount),0) FROM collection_application WHERE workspace_id=$1 AND status='APPLICATION_STATUS_APPLIED'`, e.ws)
	if applied != 100 {
		t.Fatalf("applied=%d; want exactly 100 (the invoice must never be over-applied)", applied)
	}
}

// C16: concurrent issuances against ONE series on separate connections allocate gapless, unique
// numbers 1..N (the series row lock is held to commit) and leave next_number = N+1.
func TestIssueSeriesGaplessTwoConnections(t *testing.T) {
	e := newRaceEnv(t, "c16-race-series-ws")
	e.client("c16-client")
	e.exec(`INSERT INTO document_series (id, workspace_id, code, issuer_name, document_kind, prefix, fiscal_reset, next_number, number_padding, status, active)
		VALUES ('c16-ser', $1, 'C16', 'Acme', 'CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT', 'SOA-', 'DOCUMENT_SERIES_FISCAL_RESET_NONE', 1, 4, 'DOCUMENT_SERIES_STATUS_ACTIVE', true)`, e.ws)
	const workers = 8
	for i := 0; i < workers; i++ {
		e.charge("c16-ch"+itoa(int64(i)), "c16-client", "BILLABLE_CHARGE_STATUS_OPEN", 100+int64(i))
	}
	errs := make([]error, workers)
	runConcurrently(workers, func(i int) {
		_, errs[i] = e.docs.IssueRecoveryDocuments.Execute(e.ctx(), &recoverydocumentpb.IssueRecoveryDocumentsRequest{
			BillableChargeIds: []string{"c16-ch" + itoa(int64(i))}, DocumentSeriesId: "c16-ser", IssuanceKey: "k-" + itoa(int64(i)),
		})
	})
	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d: %v", i, err)
		}
	}
	rows, err := e.db.Query(`SELECT sequence_number, document_number FROM recovery_document WHERE workspace_id=$1 ORDER BY sequence_number`, e.ws)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var want int64 = 1
	for rows.Next() {
		var seq int64
		var number string
		if err := rows.Scan(&seq, &number); err != nil {
			t.Fatal(err)
		}
		if seq != want || number != "SOA-000"+itoa(want) {
			t.Fatalf("sequence %d / number %s at position %d: numbering has a gap or duplicate", seq, number, want)
		}
		want++
	}
	if want != workers+1 {
		t.Fatalf("issued %d documents, want %d", want-1, workers)
	}
	if next := e.scalar(`SELECT next_number FROM document_series WHERE id='c16-ser' AND workspace_id=$1`, e.ws); next != workers+1 {
		t.Fatalf("next_number = %d, want %d", next, workers+1)
	}
}

// Void racing apply on the same document: whatever the interleaving, a document is never both VOID
// and carrying an APPLIED application, and an applied document is never over-applied.
func TestVoidVersusApplyNeverLeavesVoidedDocumentApplied(t *testing.T) {
	e := newRaceEnv(t, "c16-race-void-ws")
	e.exec(`INSERT INTO document_series (id, workspace_id, code, issuer_name, document_kind, prefix, fiscal_reset, next_number, number_padding, status, active)
		VALUES ($2, $1, 'SEED', 'Acme', 'CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT', 'SOA-', 'DOCUMENT_SERIES_FISCAL_RESET_NONE', 100, 4, 'DOCUMENT_SERIES_STATUS_ACTIVE', true)`, e.ws, e.ws+"-ser-seed")
	const rounds = 6
	for r := 0; r < rounds; r++ {
		client := "c16-client-" + itoa(int64(r))
		doc := "c16-vdoc-" + itoa(int64(r))
		e.client(client)
		e.issuedDocument(doc, client, 100, int64(r+1))
		voidErr := make([]error, 1)
		runConcurrently(4, func(i int) {
			if i == 0 {
				_, voidErr[0] = e.docs.VoidRecoveryDocument.Execute(e.ctx(), &recoverydocumentpb.VoidRecoveryDocumentRequest{RecoveryDocumentId: doc, Reason: "race"})
				return
			}
			_, _ = e.cash.ReceiveAndApplyCollection.Execute(e.ctx(), receiveReq(client, 60))
		})
		status := e.scalarString(`SELECT status FROM recovery_document WHERE id=$1 AND workspace_id=$2`, doc, e.ws)
		applied := e.scalar(`SELECT COALESCE(SUM(amount),0) FROM collection_application WHERE workspace_id=$1 AND recovery_document_id=$2 AND status='APPLICATION_STATUS_APPLIED'`, e.ws, doc)
		if applied > 100 {
			t.Fatalf("round %d: document over-applied (%d)", r, applied)
		}
		if status == "RECOVERY_DOCUMENT_STATUS_VOID" && applied != 0 {
			t.Fatalf("round %d: document is VOID but carries %d of APPLIED cash", r, applied)
		}
		if voidErr[0] != nil && status == "RECOVERY_DOCUMENT_STATUS_VOID" {
			t.Fatalf("round %d: void reported %v but the document is VOID", r, voidErr[0])
		}
		if voidErr[0] == nil && status != "RECOVERY_DOCUMENT_STATUS_VOID" {
			t.Fatalf("round %d: void reported success but the document is %s", r, status)
		}
	}
}
