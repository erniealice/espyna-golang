package collection_application

// The S1 "Receive & apply" cash behaviour of the collection_application entity, per
// 20260927-usage-and-pass-through-charges (build-spec §6.1/§6.3, AC-UC-10/-31/-33, N9, N15, D10):
//
//   - ReceiveAndApplyCollection: one transaction creates the receipt (treasury_collection, revenue_id
//     NULL) and CASH collection_application rows against the client's open items in N9 order;
//   - PreviewCollectionApplication: the same allocation with no writes (drawer preview);
//   - ReverseCollectionApplication: a reversing row plus REVERSAL effects (D10).
//
// Every Execute takes and returns esqyma proto messages (build-spec §7 C1); refusals are *Error values
// with ErrorCode() (C2); receive and reverse use the strict permission gate (C3). The receipt is the
// treasury_collection row; applications carry the targets. Money is int64 centavos. Actor and
// workspace come from the context, never from the request.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/charge_effect"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// ReceiptCollectionType marks treasury_collection rows created by ReceiveAndApplyCollection. The
// collection summary report's application leg keys on it (revenue_id is NULL on these receipts).
const ReceiptCollectionType = "receipt"

// fallbackMessages are the English defaults of the general-tier Lyngua keys collection_application.errors.<code>
// (labels-s1.md); vertical wording comes only from Lyngua overlays.
var fallbackMessages = map[string]string{
	"currency_mismatch":    "The currencies do not match.",
	"not_found":            "Application not found.",
	"amount_invalid":       "Enter an amount above zero.",
	"client_required":      "Select a customer.",
	"already_reversed":     "This application is already reversed.",
	"missing_posting":      "The charge policy has no account mapping for this step.",
	"transaction_required": "This action needs a database transaction, which is not available.",
	"date_invalid":         "Enter the payment date as YYYY-MM-DD.",
}

// newErr builds a named refusal (usecaseerr, C2/C29): Error() = "collection_application: " + the English
// fallback, ErrorCode() = code; use cases translate it with usecaseerr.Localize before returning.
func newErr(code string) *usecaseerr.Error {
	return usecaseerr.New("collection_application: ", code, fallbackMessages[code])
}

// Refusal codes (Lyngua collection_application.errors.<code>). The values are unexported so no other
// package couples to them; callers branch with usecaseerr.IsCode or the ErrorCode() method.
var (
	errCurrencyMismatch    = newErr("currency_mismatch")
	errNotFound            = newErr("not_found")
	errAmountInvalid       = newErr("amount_invalid")
	errClientRequired      = newErr("client_required")
	errAlreadyReversed     = newErr("already_reversed")
	errMissingPosting      = newErr("missing_posting")
	errTransactionRequired = newErr("transaction_required")
	errDateInvalid         = newErr("date_invalid")
)

func blank(s string) bool   { return strings.TrimSpace(s) == "" }
func strp(s string) *string { return &s }
func i64p(i int64) *int64   { return &i }
func i32p(i int32) *int32   { return &i }

func inTx(ctx context.Context, tx ports.Transactor, fn func(context.Context) error) error {
	if tx == nil || !tx.SupportsTransactions() {
		return errTransactionRequired
	}
	return tx.ExecuteInTransaction(ctx, fn)
}

func newID(g ports.IDGenerator) (string, error) {
	if g == nil {
		return "", fmt.Errorf("collection_application: id generator unavailable")
	}
	return g.GenerateID(), nil
}

// gateStrict is the security-sensitive gate (C3): a strict-capable authorizer's fresh verdict.
func gateStrict(ctx context.Context, gk *actiongate.ActionGatekeeper, action string) error {
	return gk.CheckStrict(ctx, &actiongate.CheckActionRequest{Entity: entityid.CollectionApplication, Action: action})
}

func gateAction(ctx context.Context, gk *actiongate.ActionGatekeeper, action string) error {
	return gk.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.CollectionApplication, Action: action})
}

func today() string { return time.Now().UTC().Format("2006-01-02") }

// ledger is the internal working set of repositories the use cases share; each use case fills only
// the subset it needs from its own <Op>Repositories.
type ledger struct {
	Collection            collectionpb.CollectionDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	Revenue               revenuepb.RevenueDomainServiceServer
	RevenuePayment        revenuepaymentpb.RevenuePaymentDomainServiceServer
	RecoveryDocument      recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine  recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
	BillableCharge        billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent       chargecomponentpb.ChargeComponentDomainServiceServer
	ChargePolicyPosting   postingpb.ChargePolicyPostingDomainServiceServer
	ChargeEffect          chargeeffectpb.ChargeEffectDomainServiceServer
	Client                clientpb.ClientDomainServiceServer
	CollectionMethod      collectionmethodpb.CollectionMethodDomainServiceServer
}

func (l ledger) effectRepos(g ports.IDGenerator, tr ports.Translator) charge_effect.Repos {
	er := charge_effect.Repos{ChargeEffect: l.ChargeEffect, ChargePolicyPosting: l.ChargePolicyPosting, Translator: tr}
	if g != nil {
		er.NewID = g.GenerateID
	}
	return er
}

// lockDocument locks a recovery document FOR UPDATE. It fails closed (C4): a repository that cannot
// lock refuses the operation instead of reading unlocked. A row the lock cannot find is not_found;
// any other repository error is logged and returned, never reported as not_found (C9).
func (l ledger) lockDocument(ctx context.Context, id string) error {
	lk, ok := l.RecoveryDocument.(domainports.RecoveryDocumentLocker)
	if !ok {
		return fmt.Errorf("collection_application: recovery_document repository cannot lock rows")
	}
	d, err := lk.LockRecoveryDocumentForUpdate(ctx, id)
	switch {
	case err != nil && usecaseerr.IsNotFound(err):
		return errNotFound
	case err != nil:
		return usecaseerr.RepoErr("collection_application", "lock recovery_document", err, id)
	case d == nil:
		return errNotFound
	}
	return nil
}

// lockApplication locks a collection_application row FOR UPDATE (fail closed, same contract as
// lockDocument).
func (l ledger) lockApplication(ctx context.Context, id string) error {
	lk, ok := l.CollectionApplication.(domainports.CollectionApplicationLocker)
	if !ok {
		return fmt.Errorf("collection_application: collection_application repository cannot lock rows")
	}
	a, err := lk.LockCollectionApplicationForUpdate(ctx, id)
	switch {
	case err != nil && usecaseerr.IsNotFound(err):
		return errNotFound
	case err != nil:
		return usecaseerr.RepoErr("collection_application", "lock collection_application", err, id)
	case a == nil:
		return errNotFound
	}
	return nil
}

// lockRevenue locks an invoice FOR UPDATE (fail closed, same contract as lockDocument): two
// concurrent receipts for the same invoice serialise here and the second sees the first's applications.
func (l ledger) lockRevenue(ctx context.Context, id string) error {
	lk, ok := l.Revenue.(domainports.RevenueLocker)
	if !ok {
		return fmt.Errorf("collection_application: revenue repository cannot lock rows")
	}
	rv, err := lk.LockRevenueForUpdate(ctx, id)
	switch {
	case err != nil && usecaseerr.IsNotFound(err):
		return errNotFound
	case err != nil:
		return usecaseerr.RepoErr("collection_application", "lock revenue", err, id)
	case rv == nil:
		return errNotFound
	}
	return nil
}

// versionOfDocument resolves the pinned charge-policy version of a document through its lines ->
// components -> charges. Issuance groups by version, so a document has exactly one; a document
// with none (or several) cannot post effects.
func (l ledger) versionOfDocument(ctx context.Context, docID string) (string, error) {
	lines, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*recoverydocumentlinepb.RecoveryDocumentLine, error) {
		lr, err := l.RecoveryDocumentLine.ListRecoveryDocumentLines(ctx, &recoverydocumentlinepb.ListRecoveryDocumentLinesRequest{Filters: listdata.EqFilter("recovery_document_id", docID), Sort: s, Pagination: p})
		return lr.GetData(), err
	})
	if err != nil {
		return "", usecaseerr.RepoErr("collection_application", "list recovery_document_lines", err, docID)
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].GetId() < lines[j].GetId() })
	version := ""
	for _, ln := range lines {
		cr, err := l.ChargeComponent.ReadChargeComponent(ctx, &chargecomponentpb.ReadChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: ln.GetChargeComponentId()}})
		if err != nil && !usecaseerr.IsNotFound(err) {
			return "", usecaseerr.RepoErr("collection_application", "read charge_component", err, ln.GetChargeComponentId())
		}
		if err != nil || len(cr.GetData()) == 0 {
			return "", errMissingPosting
		}
		br, err := l.BillableCharge.ReadBillableCharge(ctx, &billablechargepb.ReadBillableChargeRequest{Data: &billablechargepb.BillableCharge{Id: cr.Data[0].GetBillableChargeId()}})
		if err != nil && !usecaseerr.IsNotFound(err) {
			return "", usecaseerr.RepoErr("collection_application", "read billable_charge", err, cr.Data[0].GetBillableChargeId())
		}
		if err != nil || len(br.GetData()) == 0 {
			return "", errMissingPosting
		}
		v := br.Data[0].GetChargePolicyVersionId()
		if v == "" || (version != "" && v != version) {
			return "", errMissingPosting
		}
		version = v
	}
	if version == "" {
		return "", errMissingPosting
	}
	return version, nil
}

func mapEffectErr(err error) error {
	if usecaseerr.IsCode(err, "missing_posting") {
		return errMissingPosting
	}
	return err
}
