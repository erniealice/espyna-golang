package collection_application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	recoverydocumentuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/recovery_document"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// receiveParams is the shared input of the preview and receive use cases (built from the proto
// requests of either).
type receiveParams struct {
	ClientID           string
	Amount             int64  // centavos, > 0
	Currency           string // required; only open items in this currency are eligible (N15)
	PaymentDate        string // ISO date; blank = today (UTC)
	CollectionMethodID string
	ReferenceNumber    string
	Name               string
}

func paramsFromReceive(r *collectionapplicationpb.ReceiveAndApplyCollectionRequest) *receiveParams {
	if r == nil {
		return nil
	}
	return &receiveParams{ClientID: r.GetClientId(), Amount: r.GetAmount(), Currency: r.GetCurrency(), PaymentDate: r.GetPaymentDate(),
		CollectionMethodID: r.GetCollectionMethodId(), ReferenceNumber: r.GetReferenceNumber(), Name: r.GetName()}
}

func paramsFromPreview(r *collectionapplicationpb.PreviewCollectionApplicationRequest) *receiveParams {
	if r == nil {
		return nil
	}
	return &receiveParams{ClientID: r.GetClientId(), Amount: r.GetAmount(), Currency: r.GetCurrency(), PaymentDate: r.GetPaymentDate(),
		CollectionMethodID: r.GetCollectionMethodId(), ReferenceNumber: r.GetReferenceNumber(), Name: r.GetName()}
}

type openItem struct {
	kind    collectionapplicationpb.ApplicationTargetKind
	id      string
	number  string
	due     string
	balance int64
	cur     string
}

func kindRank(k collectionapplicationpb.ApplicationTargetKind) int {
	if k == collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE {
		return 0 // rent before recovery at the same due date
	}
	return 1
}

// sortN9 orders open items: oldest due date first (blank last); same due date -> revenue before
// recovery document; then by document number, then id.
func sortN9(items []openItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.due != b.due {
			if a.due == "" {
				return false
			}
			if b.due == "" {
				return true
			}
			return a.due < b.due
		}
		if kindRank(a.kind) != kindRank(b.kind) {
			return kindRank(a.kind) < kindRank(b.kind)
		}
		if a.number != b.number {
			return a.number < b.number
		}
		return a.id < b.id
	})
}

func (l ledger) readWired() bool {
	return l.Collection != nil && l.CollectionApplication != nil && l.Revenue != nil &&
		l.RecoveryDocument != nil
}

func validate(req *receiveParams) error {
	if req == nil || blank(req.ClientID) {
		return errClientRequired
	}
	if req.Amount <= 0 {
		return errAmountInvalid
	}
	if blank(req.Currency) {
		return errCurrencyMismatch
	}
	// A1 m3: payment_date is stored as text and cast to a date by the collection summary; a
	// malformed one would break that report for the whole workspace. Blank = today.
	if !blank(req.PaymentDate) {
		d, err := time.Parse("2006-01-02", req.PaymentDate)
		if err != nil || d.Year() < minPaymentYear || d.Year() > maxPaymentYear {
			return errDateInvalid
		}
	}
	return nil
}

// The sane range of a receipt date (A1 m3).
const (
	minPaymentYear = 1900
	maxPaymentYear = 2999
)

func (l ledger) balanceRepos() recoverydocumentuc.BalanceRepos {
	return recoverydocumentuc.BalanceRepos{RecoveryDocument: l.RecoveryDocument, CollectionApplication: l.CollectionApplication}
}

// openItems lists the client's open items (any currency) with balance > 0.
func (l ledger) openItems(ctx context.Context, clientID string) ([]openItem, error) {
	var items []openItem

	// Revenues: total - legacy receipts - legacy payments - CASH APPLIED applications.
	revs, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*revenuepb.Revenue, error) {
		rr, err := l.Revenue.ListRevenues(ctx, &revenuepb.ListRevenuesRequest{Filters: listdata.EqFilter("client_id", clientID), Sort: s, Pagination: p})
		return rr.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("collection_application", "list revenues", err, clientID)
	}
	apps, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*collectionapplicationpb.CollectionApplication, error) {
		ar, err := l.CollectionApplication.ListCollectionApplications(ctx, &collectionapplicationpb.ListCollectionApplicationsRequest{Filters: listdata.EqFilter("client_id", clientID), Sort: s, Pagination: p})
		return ar.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("collection_application", "list collection_applications", err, clientID)
	}
	appliedToRevenue := map[string]int64{}
	for _, a := range apps {
		if a.GetActive() &&
			a.GetStatus() == collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED &&
			a.GetApplicationKind() == collectionapplicationpb.ApplicationKind_APPLICATION_KIND_CASH &&
			a.GetTargetKind() == collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE {
			appliedToRevenue[a.GetRevenueId()] += a.GetAmount()
		}
	}
	for _, rev := range revs {
		if !rev.GetActive() || rev.GetStatus() == "cancelled" || rev.GetStatus() == "draft" {
			continue
		}
		paid := appliedToRevenue[rev.GetId()]
		if l.Collection != nil {
			cols, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*collectionpb.Collection, error) {
				cr, err := l.Collection.ListCollections(ctx, &collectionpb.ListCollectionsRequest{Filters: listdata.EqFilter("revenue_id", rev.GetId()), Sort: s, Pagination: p})
				return cr.GetData(), err
			})
			if err != nil {
				return nil, usecaseerr.RepoErr("collection_application", "list treasury_collections", err, rev.GetId())
			}
			for _, c := range cols {
				if c.GetActive() && (c.GetStatus() == "paid" || c.GetStatus() == "completed") {
					paid += c.GetAmount()
				}
			}
		}
		if l.RevenuePayment != nil {
			pays, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*revenuepaymentpb.RevenuePayment, error) {
				pr, err := l.RevenuePayment.ListRevenuePayments(ctx, &revenuepaymentpb.ListRevenuePaymentsRequest{Filters: listdata.EqFilter("revenue_id", rev.GetId()), Sort: s, Pagination: p})
				return pr.GetData(), err
			})
			if err != nil {
				return nil, usecaseerr.RepoErr("collection_application", "list revenue_payments", err, rev.GetId())
			}
			for _, p := range pays {
				if p.GetActive() {
					paid += p.GetAmount()
				}
			}
		}
		if bal := rev.GetTotalAmount() - paid; bal > 0 {
			due := rev.GetDueDate()
			if due == "" {
				due = rev.GetRevenueDate()
			}
			num := rev.GetReferenceNumber()
			if num == "" {
				num = rev.GetName()
			}
			items = append(items, openItem{kind: collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE, id: rev.GetId(), number: num, due: due, balance: bal, cur: rev.GetCurrency()})
		}
	}

	// Recovery documents: ISSUED STATEMENTs net of credit notes and CASH applications.
	snap, err := recoverydocumentuc.LoadSnapshot(ctx, l.balanceRepos(), clientID, "")
	if err != nil {
		return nil, err
	}
	for _, s := range snap.Statements {
		if s.Balance <= 0 {
			continue
		}
		d := s.Document
		due := d.GetDueDate()
		if due == "" {
			due = d.GetIssueDate()
		}
		items = append(items, openItem{kind: collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_RECOVERY_DOCUMENT, id: d.GetId(), number: d.GetDocumentNumber(), due: due, balance: s.Balance, cur: d.GetCurrency()})
	}
	return items, nil
}

// lockOpenItems locks every open item of the client - invoices first, then recovery documents, each
// in ascending id - so concurrent receipts, voids and reversals serialise on the rows they change
// before any balance is read for real. The open set is recomputed after each locking round and any
// item that became open meanwhile (a concurrent reversal committed) is locked too, until the set is
// stable (A1 m2: every open item is locked before the plan). The plan is built from fresh reads.
func (l ledger) lockOpenItems(ctx context.Context, clientID string) error {
	locked := map[string]bool{}
	for round := 0; round < maxLockRounds; round++ {
		items, err := l.openItems(ctx, clientID)
		if err != nil {
			return err
		}
		var revenues, documents []string
		for _, it := range items {
			key := it.kind.String() + "/" + it.id
			if locked[key] {
				continue
			}
			locked[key] = true
			if it.kind == collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE {
				revenues = append(revenues, it.id)
			} else {
				documents = append(documents, it.id)
			}
		}
		if len(revenues) == 0 && len(documents) == 0 {
			return nil
		}
		sort.Strings(revenues)
		sort.Strings(documents)
		for _, id := range revenues {
			if err := l.lockRevenue(ctx, id); err != nil {
				return err
			}
		}
		for _, id := range documents {
			if err := l.lockDocument(ctx, id); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("collection_application: the open items kept changing while locking; refusing to apply")
}

// maxLockRounds bounds lockOpenItems' recompute loop; a set that is still growing after it fails
// closed instead of applying against an unlocked item.
const maxLockRounds = 5

// buildPlan allocates req.Amount across the client's open items in N9 order (same-currency items
// only). Items in other currencies are never applied; if the client has open items but none in the
// receipt's currency the receipt is refused with currency_mismatch (N15).
func (l ledger) buildPlan(ctx context.Context, req *receiveParams) (*collectionapplicationpb.CollectionApplicationPlan, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	if !l.readWired() {
		return nil, fmt.Errorf("collection_application: repositories not wired")
	}
	all, err := l.openItems(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}
	var eligible []openItem
	for _, it := range all {
		if it.cur == req.Currency {
			eligible = append(eligible, it)
		}
	}
	if len(all) > 0 && len(eligible) == 0 {
		return nil, errCurrencyMismatch
	}
	sortN9(eligible)
	plan := &collectionapplicationpb.CollectionApplicationPlan{Currency: req.Currency}
	remaining := req.Amount
	for _, it := range eligible {
		if remaining == 0 {
			break
		}
		apply := it.balance
		if remaining < apply {
			apply = remaining
		}
		plan.Allocations = append(plan.Allocations, &collectionapplicationpb.CollectionApplicationPlanAllocation{
			TargetKind: it.kind, TargetId: it.id, Number: it.number, DueDate: it.due,
			Balance: it.balance, Apply: apply, Rank: int32(len(plan.Allocations) + 1),
		})
		plan.Applied += apply
		remaining -= apply
	}
	plan.Unapplied = remaining
	return plan, nil
}

// verifyReferences reads the request's client (and collection method, when named) through the
// workspace-scoped repositories: a foreign or missing id is refused before anything is written.
func (l ledger) verifyReferences(ctx context.Context, req *receiveParams) error {
	if l.Client == nil {
		return fmt.Errorf("collection_application: client repository not wired")
	}
	cr, err := l.Client.ReadClient(ctx, &clientpb.ReadClientRequest{Data: &clientpb.Client{Id: req.ClientID}})
	if err != nil && !usecaseerr.IsNotFound(err) {
		return usecaseerr.RepoErr("collection_application", "read client", err, req.ClientID)
	}
	if err != nil || len(cr.GetData()) == 0 {
		return errClientRequired
	}
	if !blank(req.CollectionMethodID) {
		if l.CollectionMethod == nil {
			return fmt.Errorf("collection_application: collection method repository not wired")
		}
		mr, err := l.CollectionMethod.ReadCollectionMethod(ctx, &collectionmethodpb.ReadCollectionMethodRequest{Data: &collectionmethodpb.CollectionMethod{Id: req.CollectionMethodID}})
		if err != nil && !usecaseerr.IsNotFound(err) {
			return usecaseerr.RepoErr("collection_application", "read collection_method", err, req.CollectionMethodID)
		}
		if err != nil || len(mr.GetData()) == 0 {
			return errNotFound
		}
	}
	return nil
}
