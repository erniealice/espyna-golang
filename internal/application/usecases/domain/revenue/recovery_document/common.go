package recovery_document

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/charge_effect"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// gate is the ordinary permission gate (read/list); gateStrict is the security-sensitive gate for
// status verbs issue/void (C3: a strict-capable authorizer's fresh verdict, never shadow-allow).
func gate(ctx context.Context, gk *actiongate.ActionGatekeeper, action string) error {
	return gk.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RecoveryDocument, Action: action})
}

func gateStrict(ctx context.Context, gk *actiongate.ActionGatekeeper, action string) error {
	return gk.CheckStrict(ctx, &actiongate.CheckActionRequest{Entity: entityid.RecoveryDocument, Action: action})
}

func stringEq(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
		}},
	}
}

func prefixFilter(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: value, Operator: commonpb.StringOperator_STRING_STARTS_WITH, CaseSensitive: true,
		}},
	}}}
}

func blank(s string) bool   { return strings.TrimSpace(s) == "" }
func strp(s string) *string { return &s }
func i64p(i int64) *int64   { return &i }

func sortedUnique(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if !blank(id) && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func inTx(ctx context.Context, tx ports.Transactor, fn func(context.Context) error) error {
	if tx == nil || !tx.SupportsTransactions() {
		return errTransactionRequired
	}
	return tx.ExecuteInTransaction(ctx, fn)
}

// documentDeps are the repositories the issuance / void transactions share.
type documentDeps struct {
	RecoveryDocument     recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
	BillableCharge       billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent      chargecomponentpb.ChargeComponentDomainServiceServer
	ChargePolicyPosting  postingpb.ChargePolicyPostingDomainServiceServer
	ChargeEffect         chargeeffectpb.ChargeEffectDomainServiceServer
}

// lockCharge locks a billable charge FOR UPDATE and returns it. It fails closed (C4): a repository
// that cannot lock refuses the operation instead of reading unlocked. A row the lock cannot find is
// not_found; any other repository error is logged and returned, never reported as not_found (C9).
func (d documentDeps) lockCharge(ctx context.Context, id string) (*billablechargepb.BillableCharge, error) {
	l, ok := d.BillableCharge.(domainports.BillableChargeLocker)
	if !ok {
		return nil, fmt.Errorf("recovery_document: billable_charge repository cannot lock rows")
	}
	c, err := l.LockBillableChargeForUpdate(ctx, id)
	switch {
	case err != nil && usecaseerr.IsNotFound(err):
		return nil, errNotFound
	case err != nil:
		return nil, usecaseerr.RepoErr("recovery_document", "lock billable_charge", err, id)
	case c == nil:
		return nil, errNotFound
	}
	return c, nil
}

// lockDocument locks a recovery document FOR UPDATE (fail closed, see lockCharge).
func lockDocument(ctx context.Context, repo recoverydocumentpb.RecoveryDocumentDomainServiceServer, id string) (*recoverydocumentpb.RecoveryDocument, error) {
	l, ok := repo.(domainports.RecoveryDocumentLocker)
	if !ok {
		return nil, fmt.Errorf("recovery_document: recovery_document repository cannot lock rows")
	}
	doc, err := l.LockRecoveryDocumentForUpdate(ctx, id)
	switch {
	case err != nil && usecaseerr.IsNotFound(err):
		return nil, errNotFound
	case err != nil:
		return nil, usecaseerr.RepoErr("recovery_document", "lock recovery_document", err, id)
	case doc == nil:
		return nil, errNotFound
	}
	return doc, nil
}

// lockSeries locks a document series FOR UPDATE (fail closed, see lockCharge).
func lockSeries(ctx context.Context, repo documentseriespb.DocumentSeriesDomainServiceServer, id string) (*documentseriespb.DocumentSeries, error) {
	l, ok := repo.(domainports.DocumentSeriesLocker)
	if !ok {
		return nil, fmt.Errorf("recovery_document: document_series repository cannot lock rows")
	}
	s, err := l.LockDocumentSeriesForUpdate(ctx, id)
	switch {
	case err != nil && usecaseerr.IsNotFound(err):
		return nil, errNotFound
	case err != nil:
		return nil, usecaseerr.RepoErr("recovery_document", "lock document_series", err, id)
	case s == nil:
		return nil, errNotFound
	}
	return s, nil
}

func (d documentDeps) componentsOf(ctx context.Context, chargeID string) ([]*chargecomponentpb.ChargeComponent, error) {
	rows, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*chargecomponentpb.ChargeComponent, error) {
		resp, err := d.ChargeComponent.ListChargeComponents(ctx, &chargecomponentpb.ListChargeComponentsRequest{Filters: listdata.EqFilter("billable_charge_id", chargeID), Sort: s, Pagination: p})
		return resp.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "list charge_components", err, chargeID)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].GetId() < rows[j].GetId() })
	return rows, nil
}

func linesOf(ctx context.Context, repo recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer, docID string) ([]*recoverydocumentlinepb.RecoveryDocumentLine, error) {
	rows, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*recoverydocumentlinepb.RecoveryDocumentLine, error) {
		resp, err := repo.ListRecoveryDocumentLines(ctx, &recoverydocumentlinepb.ListRecoveryDocumentLinesRequest{Filters: listdata.EqFilter("recovery_document_id", docID), Sort: s, Pagination: p})
		return resp.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "list recovery_document_lines", err, docID)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].GetId() < rows[j].GetId() })
	return rows, nil
}

func (d documentDeps) effectRepos(newID func() string, tr ports.Translator) charge_effect.Repos {
	return charge_effect.Repos{ChargeEffect: d.ChargeEffect, ChargePolicyPosting: d.ChargePolicyPosting, NewID: newID, Translator: tr}
}

func isMissingPosting(err error) bool { return usecaseerr.IsCode(err, "missing_posting") }
