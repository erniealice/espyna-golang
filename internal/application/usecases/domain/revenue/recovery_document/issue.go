package recovery_document

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/charge_effect"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// IssueRecoveryDocumentsRepositories groups repository dependencies.
type IssueRecoveryDocumentsRepositories struct {
	RecoveryDocument     recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
	DocumentSeries       documentseriespb.DocumentSeriesDomainServiceServer
	BillableCharge       billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent      chargecomponentpb.ChargeComponentDomainServiceServer
	ChargePolicyPosting  postingpb.ChargePolicyPostingDomainServiceServer
	ChargeEffect         chargeeffectpb.ChargeEffectDomainServiceServer
}

// IssueRecoveryDocumentsServices groups service dependencies.
type IssueRecoveryDocumentsServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	IDGenerator      ports.IDGenerator
}

// IssueRecoveryDocumentsUseCase issues STATEMENT / CREDIT_NOTE documents from OPEN charges in one
// transaction (build-spec §6.3). Permission recovery_document:issue, strict gate (C3).
//
// Lock order: document series row, then charge rows by ascending id (all workspace-scoped, fail
// closed). Documents are grouped by (client, subscription, document type, corrected document, pinned
// policy version); a document number is allocated per document from the series counter, gapless
// because the series row is held FOR UPDATE until commit. Per-document idempotency key is
// "<issuance_key>#<n>". A retry with the same key returns the ORIGINAL documents when the request
// names the same series and charges (the request fingerprint), and refuses issuance_conflict
// otherwise (C12). The series must be a RECOVERY_DOCUMENT series and every component a
// RECOVERY_DOCUMENT component (C12).
type IssueRecoveryDocumentsUseCase struct {
	repositories IssueRecoveryDocumentsRepositories
	services     IssueRecoveryDocumentsServices
}

// NewIssueRecoveryDocumentsUseCase creates the use case with grouped dependencies.
func NewIssueRecoveryDocumentsUseCase(r IssueRecoveryDocumentsRepositories, s IssueRecoveryDocumentsServices) *IssueRecoveryDocumentsUseCase {
	return &IssueRecoveryDocumentsUseCase{repositories: r, services: s}
}

var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type groupKey struct {
	client, subscription, corrects, version string
	credit                                  bool
}

type chargeBundle struct {
	charge     *billablechargepb.BillableCharge
	components []*chargecomponentpb.ChargeComponent
}

func (uc *IssueRecoveryDocumentsUseCase) deps() documentDeps {
	r := uc.repositories
	return documentDeps{RecoveryDocument: r.RecoveryDocument, RecoveryDocumentLine: r.RecoveryDocumentLine, BillableCharge: r.BillableCharge,
		ChargeComponent: r.ChargeComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargeEffect: r.ChargeEffect}
}

func (uc *IssueRecoveryDocumentsUseCase) newID() (string, error) {
	if uc.services.IDGenerator == nil {
		return "", fmt.Errorf("recovery_document: id generator unavailable")
	}
	return uc.services.IDGenerator.GenerateID(), nil
}

// Execute issues the documents.
func (uc *IssueRecoveryDocumentsUseCase) Execute(ctx context.Context, req *recoverydocumentpb.IssueRecoveryDocumentsRequest) (*recoverydocumentpb.IssueRecoveryDocumentsResponse, error) {
	if err := gateStrict(ctx, uc.services.ActionGatekeeper, entityid.ActionIssue); err != nil {
		return nil, err
	}
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "recovery_document", err)
	}
	return out, nil
}

func (uc *IssueRecoveryDocumentsUseCase) execute(ctx context.Context, req *recoverydocumentpb.IssueRecoveryDocumentsRequest) (*recoverydocumentpb.IssueRecoveryDocumentsResponse, error) {
	r := uc.repositories
	if r.RecoveryDocument == nil || r.RecoveryDocumentLine == nil || r.DocumentSeries == nil || r.BillableCharge == nil ||
		r.ChargeComponent == nil || r.ChargePolicyPosting == nil || r.ChargeEffect == nil {
		return nil, fmt.Errorf("recovery_document: issuance repositories unavailable")
	}
	if req == nil {
		return nil, errNothingSelected
	}
	ids := sortedUnique(req.GetBillableChargeIds())
	if len(ids) == 0 {
		return nil, errNothingSelected
	}
	if blank(req.GetDocumentSeriesId()) || blank(req.GetIssuanceKey()) {
		return nil, invalid("document series and issuance key are required")
	}
	if strings.Contains(req.GetIssuanceKey(), "#") {
		return nil, invalid("issuance key must not contain '#'")
	}
	issueDate := strings.TrimSpace(req.GetIssueDate())
	if issueDate == "" {
		issueDate = time.Now().UTC().Format("2006-01-02")
	}
	if !isoDate.MatchString(issueDate) || (req.GetDueDate() != "" && !isoDate.MatchString(req.GetDueDate())) {
		return nil, invalid("dates must be YYYY-MM-DD")
	}
	userID := contextutil.ExtractUserIDFromContext(ctx)
	deps := uc.deps()

	var out *recoverydocumentpb.IssueRecoveryDocumentsResponse
	err := inTx(ctx, uc.services.Transactor, func(tx context.Context) error {
		// 1. Series first (fixed lock order).
		series, err := lockSeries(tx, r.DocumentSeries, req.GetDocumentSeriesId())
		if err != nil {
			return err
		}
		// 2. Retry with the same key returns the original documents (before any refusal that
		// could differ on retry, e.g. the charges are no longer OPEN or the series is retired) -
		// but only for the same request (series + charge set); anything else is a conflict.
		prev, err := uc.originalDocuments(tx, req.GetIssuanceKey())
		if err != nil {
			return err
		}
		if len(prev) > 0 {
			if err := uc.matchesFingerprint(tx, deps, prev, series.GetId(), ids); err != nil {
				return err
			}
			out = &recoverydocumentpb.IssueRecoveryDocumentsResponse{Data: prev, Replayed: true, Success: true}
			return nil
		}
		if series.GetDocumentKind() != enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT {
			return errSeriesKindMismatch
		}
		if series.GetStatus() != documentseriespb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_ACTIVE {
			return errSeriesRetired
		}

		// 3. Charges, ascending id.
		var bundles []chargeBundle
		currency := ""
		for _, id := range ids {
			c, err := deps.lockCharge(tx, id)
			if err != nil {
				return err
			}
			if c.GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN {
				return errNotOpen
			}
			if currency == "" {
				currency = c.GetCurrency()
			} else if c.GetCurrency() != currency {
				return errCurrencyMismatch
			}
			if blank(c.GetClientId()) {
				return invalid("charge %s has no client", c.GetId())
			}
			comps, err := deps.componentsOf(tx, c.GetId())
			if err != nil {
				return err
			}
			if len(comps) == 0 {
				return invalid("charge %s has no components", c.GetId())
			}
			for _, comp := range comps {
				if comp.GetDocumentKind() != enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT {
					return errComponentKindMismatch
				}
			}
			bundles = append(bundles, chargeBundle{c, comps})
		}

		// 4. Group.
		groups := map[groupKey][]chargeBundle{}
		for _, b := range bundles {
			k := groupKey{client: b.charge.GetClientId(), subscription: b.charge.GetSubscriptionId(), version: b.charge.GetChargePolicyVersionId()}
			if b.charge.GetChargeKind() == billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_CORRECTION {
				k.credit = true
				doc, err := uc.correctedDocument(tx, deps, b.charge)
				if err != nil {
					return err
				}
				k.corrects = doc
			}
			groups[k] = append(groups[k], b)
		}
		keys := make([]groupKey, 0, len(groups))
		for k := range groups {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			a, b := keys[i], keys[j]
			switch {
			case a.client != b.client:
				return a.client < b.client
			case a.subscription != b.subscription:
				return a.subscription < b.subscription
			case a.credit != b.credit:
				return !a.credit
			case a.corrects != b.corrects:
				return a.corrects < b.corrects
			}
			return a.version < b.version
		})

		// 5. Issue.
		next := series.GetNextNumber()
		resp := &recoverydocumentpb.IssueRecoveryDocumentsResponse{Success: true}
		for i, k := range keys {
			doc, err := uc.issueGroup(tx, deps, series, next, i+1, k, groups[k], req, issueDate, currency, userID)
			if err != nil {
				return err
			}
			next++
			resp.Data = append(resp.Data, doc)
		}
		if _, err := r.DocumentSeries.UpdateDocumentSeries(tx, &documentseriespb.UpdateDocumentSeriesRequest{
			Data: &documentseriespb.DocumentSeries{Id: series.GetId(), NextNumber: next},
		}); err != nil {
			return usecaseerr.RepoErr("recovery_document", "advance document_series", err, series.GetId())
		}
		out = resp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// originalDocuments returns documents previously issued under the key ("<key>#<n>").
func (uc *IssueRecoveryDocumentsUseCase) originalDocuments(ctx context.Context, key string) ([]*recoverydocumentpb.RecoveryDocument, error) {
	rows, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*recoverydocumentpb.RecoveryDocument, error) {
		resp, err := uc.repositories.RecoveryDocument.ListRecoveryDocuments(ctx, &recoverydocumentpb.ListRecoveryDocumentsRequest{Filters: prefixFilter("issuance_key", key+"#"), Sort: s, Pagination: p})
		return resp.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "lookup issuance key", err)
	}
	var out []*recoverydocumentpb.RecoveryDocument
	for _, d := range rows {
		if strings.HasPrefix(d.GetIssuanceKey(), key+"#") {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetSequenceNumber() < out[j].GetSequenceNumber() })
	return out, nil
}

// matchesFingerprint refuses issuance_conflict unless the documents previously issued under the key
// came from the same series and cover exactly the requested charge set (the request fingerprint is
// derived from the stored documents, so no extra column is needed).
func (uc *IssueRecoveryDocumentsUseCase) matchesFingerprint(ctx context.Context, deps documentDeps, prev []*recoverydocumentpb.RecoveryDocument, seriesID string, wantCharges []string) error {
	have := map[string]bool{}
	for _, d := range prev {
		if d.GetDocumentSeriesId() != seriesID {
			return errIssuanceConflict
		}
		lines, err := linesOf(ctx, deps.RecoveryDocumentLine, d.GetId())
		if err != nil {
			return err
		}
		for _, ln := range lines {
			cr, err := deps.ChargeComponent.ReadChargeComponent(ctx, &chargecomponentpb.ReadChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: ln.GetChargeComponentId()}})
			if err != nil && !usecaseerr.IsNotFound(err) {
				return usecaseerr.RepoErr("recovery_document", "read charge_component", err, ln.GetChargeComponentId())
			}
			if err != nil || len(cr.GetData()) == 0 {
				return errIssuanceConflict
			}
			have[cr.Data[0].GetBillableChargeId()] = true
		}
	}
	if len(have) != len(wantCharges) {
		return errIssuanceConflict
	}
	for _, id := range wantCharges {
		if !have[id] {
			return errIssuanceConflict
		}
	}
	return nil
}

// correctedDocument finds the ISSUED document that carries the predecessor charge.
func (uc *IssueRecoveryDocumentsUseCase) correctedDocument(ctx context.Context, deps documentDeps, corr *billablechargepb.BillableCharge) (string, error) {
	if blank(corr.GetPredecessorId()) {
		return "", invalid("correction %s has no predecessor", corr.GetId())
	}
	comps, err := deps.componentsOf(ctx, corr.GetPredecessorId())
	if err != nil {
		return "", err
	}
	for _, comp := range comps {
		lines, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*recoverydocumentlinepb.RecoveryDocumentLine, error) {
			resp, err := deps.RecoveryDocumentLine.ListRecoveryDocumentLines(ctx, &recoverydocumentlinepb.ListRecoveryDocumentLinesRequest{Filters: listdata.EqFilter("charge_component_id", comp.GetId()), Sort: s, Pagination: p})
			return resp.GetData(), err
		})
		if err != nil {
			return "", usecaseerr.RepoErr("recovery_document", "find original line", err, comp.GetId())
		}
		for _, ln := range lines {
			// Lock the original document: a concurrent void serialises here and is seen as not ISSUED.
			d, err := lockDocument(ctx, deps.RecoveryDocument, ln.GetRecoveryDocumentId())
			if err != nil {
				return "", err
			}
			if d.GetStatus() != recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED {
				return "", errNotIssued
			}
			return d.GetId(), nil
		}
	}
	return "", errNotIssued
}

func formatNumber(s *documentseriespb.DocumentSeries, seq int64) string {
	return fmt.Sprintf("%s%0*d", s.GetPrefix(), int(s.GetNumberPadding()), seq)
}

func (uc *IssueRecoveryDocumentsUseCase) issueGroup(ctx context.Context, deps documentDeps, series *documentseriespb.DocumentSeries, seq int64, ordinal int,
	k groupKey, items []chargeBundle, req *recoverydocumentpb.IssueRecoveryDocumentsRequest, issueDate, currency, userID string) (*recoverydocumentpb.RecoveryDocument, error) {
	docID, err := uc.newID()
	if err != nil {
		return nil, err
	}
	var total int64
	for _, b := range items {
		total += b.charge.GetAmount()
	}
	dtype := recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_STATEMENT
	if k.credit {
		dtype = recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_CREDIT_NOTE
	}
	now := time.Now()
	doc := &recoverydocumentpb.RecoveryDocument{
		Id:               docID,
		DocumentSeriesId: series.GetId(),
		SequenceNumber:   seq,
		DocumentNumber:   formatNumber(series, seq),
		DocumentType:     dtype,
		ClientId:         k.client,
		IssueDate:        strp(issueDate),
		TotalAmount:      total,
		Currency:         currency,
		Status:           recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED,
		IssuanceKey:      fmt.Sprintf("%s#%d", req.GetIssuanceKey(), ordinal),
		IssuedAt:         i64p(now.UnixMilli()),
		Active:           true,
		DateCreated:      i64p(now.UnixMilli()),
		DateModified:     i64p(now.UnixMilli()),
	}
	if k.subscription != "" {
		doc.SubscriptionId = strp(k.subscription)
	}
	if k.corrects != "" {
		doc.CorrectsDocumentId = strp(k.corrects)
	}
	if req.GetDueDate() != "" {
		doc.DueDate = strp(req.GetDueDate())
	}
	if userID != "" {
		doc.IssuedBy = strp(userID)
	}
	created, err := deps.RecoveryDocument.CreateRecoveryDocument(ctx, &recoverydocumentpb.CreateRecoveryDocumentRequest{Data: doc})
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "create recovery_document", err, docID)
	}
	if len(created.GetData()) > 0 {
		doc = created.Data[0]
	}
	for _, b := range items {
		for _, comp := range b.components {
			lineID, err := uc.newID()
			if err != nil {
				return nil, err
			}
			line := &recoverydocumentlinepb.RecoveryDocumentLine{
				Id: lineID, RecoveryDocumentId: docID, ChargeComponentId: comp.GetId(),
				Amount: comp.GetAmount(), Currency: comp.GetCurrency(), Active: true,
				DateCreated: i64p(now.UnixMilli()), DateModified: i64p(now.UnixMilli()),
			}
			if b.charge.ServiceFrom != nil {
				line.ServiceFrom = b.charge.ServiceFrom
			}
			if b.charge.ServiceTo != nil {
				line.ServiceTo = b.charge.ServiceTo
			}
			if _, err := deps.RecoveryDocumentLine.CreateRecoveryDocumentLine(ctx, &recoverydocumentlinepb.CreateRecoveryDocumentLineRequest{Data: line}); err != nil {
				return nil, usecaseerr.RepoErr("recovery_document", "create recovery_document_line", err, lineID)
			}
		}
		if _, err := deps.BillableCharge.UpdateBillableCharge(ctx, &billablechargepb.UpdateBillableChargeRequest{Data: &billablechargepb.BillableCharge{
			Id: b.charge.GetId(), Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED, DateModified: i64p(now.UnixMilli()),
		}}); err != nil {
			return nil, usecaseerr.RepoErr("recovery_document", "mark billable_charge issued", err, b.charge.GetId())
		}
	}
	// ISSUE effects (D9): DR RECEIVABLE / CR CLEARING from the pinned version's postings.
	if blank(k.version) {
		return nil, errMissingPosting
	}
	if err := charge_effect.RecordChargeEffects(ctx, deps.effectRepos(uc.services.IDGenerator.GenerateID, uc.services.Translator), k.version, enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE,
		docID, issueDate, currency, total, doc.GetDocumentNumber()); err != nil {
		if isMissingPosting(err) {
			return nil, errMissingPosting
		}
		return nil, err
	}
	return doc, nil
}
