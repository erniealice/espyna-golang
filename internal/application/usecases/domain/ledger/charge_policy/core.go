// Package charge_policy holds the use cases of the charge policy feature
// (20260927-usage-and-pass-through-charges, Slice A; build-spec.md §2-3):
// versioned charge policies (policy header + immutable-once-approved versions with
// components and postings), the approval workflow (permission, separation of duties,
// checklist, S1 matrix), the retire/delete lifecycle and the policy resolver.
//
// Entities: charge_policy, charge_policy_version, charge_policy_component,
// charge_policy_posting, charge_policy_version_editor (domain ledger). Permission codes:
// charge_policy:{list,read,create,update,delete,retire,approve,approve_own}; child rows are
// governed by the parent's codes (no charge_policy_version:* codes).
//
// Every use case takes and returns esqyma proto messages (C1). Named refusals are usecaseerr.Error values
// whose message is translated from `charge_policy.errors.<code>` and whose ErrorCode() is the
// stable <code> (C2); repository failures are logged and wrapped, never reported as not_found (C9).
package charge_policy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	editorpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version_editor"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
)

// Stable refusal codes (Lyngua `charge_policy.errors.<code>`; ErrorCode() returns the bare code).
const (
	CodeValidation             = "validation"
	CodeNotFound               = "not_found"
	CodeNotDraft               = "not_draft"
	CodeDraftExists            = "draft_exists"
	CodeNoApprovedVersion      = "no_approved_version"
	CodeRetired                = "retired"
	CodeSelfApproval           = "self_approval"
	CodeChecklistFailed        = "checklist_failed"
	CodeUnsupportedCombination = "unsupported_combination"
	CodeInUse                  = "in_use"
	CodeTransactionRequired    = "transaction_required"
	CodeTaxTreatmentNotFound   = "tax_treatment_not_found"
	CodeUnverifiable           = "unverifiable"
)

// defaultMessages are the English fallbacks used when the Translator has no value for the key.
var defaultMessages = map[string]string{
	CodeValidation:             "The charge policy request is invalid",
	CodeNotFound:               "Charge policy record not found",
	CodeNotDraft:               "Only a draft version can be changed",
	CodeDraftExists:            "The policy already has a draft version",
	CodeNoApprovedVersion:      "The policy has no approved version",
	CodeRetired:                "The charge policy is retired",
	CodeSelfApproval:           "A draft you prepared or edited needs the approve-own permission to approve",
	CodeChecklistFailed:        "The approval checklist has failed items",
	CodeUnsupportedCombination: "This classification is not supported yet",
	CodeInUse:                  "The charge policy was approved or is referenced and cannot be deleted",
	CodeTransactionRequired:    "A database transaction is required for this operation",
	CodeTaxTreatmentNotFound:   "The tax treatment does not exist",
	CodeUnverifiable:           "The request cannot be verified because a dependency is not wired",
}

// Named refusals are usecaseerr.Error values (build-spec §7c C29); ErrorCode() maps to Lyngua
// `charge_policy.errors.<code>`.

// checklistError is the approval refusal that carries the readiness report so the view can render
// the per-item result (`interface{ Checklist() *versionpb.ChargePolicyApprovalChecklist }`).
type checklistError struct {
	err       *usecaseerr.Error
	checklist *versionpb.ChargePolicyApprovalChecklist
}

func (e *checklistError) Error() string     { return e.err.Error() }
func (e *checklistError) ErrorCode() string { return e.err.ErrorCode() }

func (e *checklistError) Checklist() *versionpb.ChargePolicyApprovalChecklist { return e.checklist }

// Repositories groups the repository dependencies of the charge policy use cases.
type Repositories struct {
	ChargePolicy          policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion   versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyPosting   postingpb.ChargePolicyPostingDomainServiceServer
	// ChargePolicyVersionEditor records every user who edited a draft (C12 separation of duties);
	// approval fails closed when it is not wired.
	ChargePolicyVersionEditor editorpb.ChargePolicyVersionEditorDomainServiceServer
	// Account backs the approval checklist ("every posting account exists in the workspace
	// and is active"). When nil the account check fails closed.
	Account accountpb.AccountDomainServiceServer
	// TaxTreatment backs the existence check of a version's tax_treatment_id.
	TaxTreatment taxtreatmentpb.TaxTreatmentDomainServiceServer
}

// Services groups the shared service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// core is the shared collaborator of every charge policy use case (gates, typed reads, locks).
type core struct {
	repos Repositories
	svc   Services
}

// refuse builds a named refusal: translated message from `charge_policy.errors.<code>` plus detail.
func (c *core) refuse(ctx context.Context, code, detailFormat string, a ...any) error {
	msg := contextutil.GetTranslatedMessageWithContext(ctx, c.svc.Translator, "charge_policy.errors."+code, defaultMessages[code])
	if detailFormat != "" {
		msg += ": " + fmt.Sprintf(detailFormat, a...)
	}
	return usecaseerr.New("", code, msg)
}

func (c *core) gate(ctx context.Context, action string) error {
	return c.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ChargePolicy, Action: action})
}

// gateStrict is the shadow-immune check used by approval verbs (C3).
func (c *core) gateStrict(ctx context.Context, action string) error {
	return c.svc.ActionGatekeeper.CheckStrict(ctx, &actiongate.CheckActionRequest{Entity: entityid.ChargePolicy, Action: action})
}

func (c *core) actor(ctx context.Context) (string, error) {
	uid, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return "", c.refuse(ctx, CodeValidation, "authenticated user required")
	}
	return uid, nil
}

func (c *core) newID() string {
	if c.svc.IDGenerator == nil {
		return ""
	}
	return c.svc.IDGenerator.GenerateID()
}

// inTx runs fn in one transaction. Multi-write lifecycle operations never fall back to a
// non-transactional path (fail closed).
func (c *core) inTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if c.svc.Transactor == nil || !c.svc.Transactor.SupportsTransactions() {
		return c.refuse(ctx, CodeTransactionRequired, "")
	}
	return c.svc.Transactor.ExecuteInTransaction(ctx, fn)
}

func stamp() (ms int64, s string) {
	now := time.Now()
	return now.UnixMilli(), now.Format(time.RFC3339)
}

func strp(s string) *string { return &s }
func i64p(i int64) *int64   { return &i }
func blank(s string) bool   { return strings.TrimSpace(s) == "" }

// --- typed reads -------------------------------------------------------------

func (c *core) readPolicy(ctx context.Context, id string) (*policypb.ChargePolicy, error) {
	if blank(id) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy id is required")
	}
	resp, err := c.repos.ChargePolicy.ReadChargePolicy(ctx, &policypb.ReadChargePolicyRequest{Data: &policypb.ChargePolicy{Id: id}})
	if err != nil {
		if usecaseerr.IsNotFound(err) {
			return nil, c.refuse(ctx, CodeNotFound, "charge policy %s", id)
		}
		return nil, usecaseerr.RepoErr("charge_policy", "read charge_policy", err, id)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, c.refuse(ctx, CodeNotFound, "charge policy %s", id)
	}
	return resp.Data[0], nil
}

func (c *core) readVersion(ctx context.Context, id string) (*versionpb.ChargePolicyVersion, error) {
	if blank(id) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy version id is required")
	}
	resp, err := c.repos.ChargePolicyVersion.ReadChargePolicyVersion(ctx, &versionpb.ReadChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: id}})
	if err != nil {
		if usecaseerr.IsNotFound(err) {
			return nil, c.refuse(ctx, CodeNotFound, "charge policy version %s", id)
		}
		return nil, usecaseerr.RepoErr("charge_policy", "read charge_policy_version", err, id)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, c.refuse(ctx, CodeNotFound, "charge policy version %s", id)
	}
	return resp.Data[0], nil
}

// lockPolicy takes the policy row lock. The postgres adapter implements ChargePolicyLocker; a
// repository without the capability is refused (C4: fail closed, no unlocked-read fallback).
func (c *core) lockPolicy(ctx context.Context, id string) (*policypb.ChargePolicy, error) {
	l, ok := c.repos.ChargePolicy.(domainports.ChargePolicyLocker)
	if !ok {
		return nil, c.refuse(ctx, CodeUnverifiable, "charge policy repository %T cannot lock rows", c.repos.ChargePolicy)
	}
	p, err := l.LockChargePolicyForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, domainports.ErrLockedRowNotFound) {
			return nil, c.refuse(ctx, CodeNotFound, "charge policy %s", id)
		}
		return nil, usecaseerr.RepoErr("charge_policy", "lock charge_policy", err, id)
	}
	return p, nil
}

func (c *core) lockVersion(ctx context.Context, id string) (*versionpb.ChargePolicyVersion, error) {
	l, ok := c.repos.ChargePolicyVersion.(domainports.ChargePolicyVersionLocker)
	if !ok {
		return nil, c.refuse(ctx, CodeUnverifiable, "charge policy version repository %T cannot lock rows", c.repos.ChargePolicyVersion)
	}
	v, err := l.LockChargePolicyVersionForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, domainports.ErrLockedRowNotFound) {
			return nil, c.refuse(ctx, CodeNotFound, "charge policy version %s", id)
		}
		return nil, usecaseerr.RepoErr("charge_policy", "lock charge_policy_version", err, id)
	}
	return v, nil
}

// scopeFilter ANDs a scope equality onto the caller's filters (OR logic is refused: the scope
// must never be OR-ed away).
func (c *core) scopeFilter(ctx context.Context, existing *commonpb.FilterRequest, field, value string) (*commonpb.FilterRequest, error) {
	if existing == nil || len(existing.GetFilters()) == 0 {
		return listdata.EqFilter(field, value), nil
	}
	if existing.GetLogic() == commonpb.FilterLogic_OR {
		return nil, c.refuse(ctx, CodeValidation, "OR filter logic is not supported on a scoped list")
	}
	out := &commonpb.FilterRequest{Logic: commonpb.FilterLogic_AND, Filters: append([]*commonpb.TypedFilter{}, existing.GetFilters()...)}
	out.Filters = append(out.Filters, listdata.EqFilter(field, value).Filters...)
	return out, nil
}

func (c *core) listVersions(ctx context.Context, policyID string) ([]*versionpb.ChargePolicyVersion, error) {
	return listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*versionpb.ChargePolicyVersion, error) {
		resp, err := c.repos.ChargePolicyVersion.ListChargePolicyVersions(ctx, &versionpb.ListChargePolicyVersionsRequest{Filters: listdata.EqFilter("charge_policy_id", policyID), Sort: s, Pagination: p})
		if err != nil {
			return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_version", err, policyID)
		}
		return resp.GetData(), nil
	})
}

func (c *core) listComponents(ctx context.Context, versionID string) ([]*componentpb.ChargePolicyComponent, error) {
	return listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*componentpb.ChargePolicyComponent, error) {
		resp, err := c.repos.ChargePolicyComponent.ListChargePolicyComponents(ctx, &componentpb.ListChargePolicyComponentsRequest{Filters: listdata.EqFilter("charge_policy_version_id", versionID), Sort: s, Pagination: p})
		if err != nil {
			return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_component", err, versionID)
		}
		return resp.GetData(), nil
	})
}

func (c *core) listPostings(ctx context.Context, versionID string) ([]*postingpb.ChargePolicyPosting, error) {
	return listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*postingpb.ChargePolicyPosting, error) {
		resp, err := c.repos.ChargePolicyPosting.ListChargePolicyPostings(ctx, &postingpb.ListChargePolicyPostingsRequest{Filters: listdata.EqFilter("charge_policy_version_id", versionID), Sort: s, Pagination: p})
		if err != nil {
			return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_posting", err, versionID)
		}
		return resp.GetData(), nil
	})
}

// listEditors returns every recorded editor of a version. Fails closed when the editor
// repository is not wired (approval must never assume "no editors").
func (c *core) listEditors(ctx context.Context, versionID string) ([]*editorpb.ChargePolicyVersionEditor, error) {
	if c.repos.ChargePolicyVersionEditor == nil {
		return nil, c.refuse(ctx, CodeUnverifiable, "draft editors cannot be read")
	}
	return listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*editorpb.ChargePolicyVersionEditor, error) {
		resp, err := c.repos.ChargePolicyVersionEditor.ListChargePolicyVersionEditors(ctx, &editorpb.ListChargePolicyVersionEditorsRequest{ChargePolicyVersionId: strp(versionID), Filters: listdata.EqFilter("charge_policy_version_id", versionID), Sort: s, Pagination: p})
		if err != nil {
			return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_version_editor", err, versionID)
		}
		return resp.GetData(), nil
	})
}

// recordEditor records the actor as an editor of the draft (insert-if-absent; UNIQUE(version,user)).
// The caller holds the version row lock, so two edits by one user cannot race the insert.
func (c *core) recordEditor(ctx context.Context, versionID, actor string) error {
	eds, err := c.listEditors(ctx, versionID)
	if err != nil {
		return err
	}
	for _, e := range eds {
		if e.GetUserId() == actor {
			return nil
		}
	}
	ms, ts := stamp()
	_, err = c.repos.ChargePolicyVersionEditor.CreateChargePolicyVersionEditor(ctx, &editorpb.CreateChargePolicyVersionEditorRequest{Data: &editorpb.ChargePolicyVersionEditor{
		Id:                    c.newID(),
		ChargePolicyVersionId: versionID,
		UserId:                actor,
		FirstEditedAt:         i64p(ms),
		Active:                true,
		DateCreated:           i64p(ms),
		DateCreatedString:     strp(ts),
		DateModified:          i64p(ms),
		DateModifiedString:    strp(ts),
	}})
	if err != nil {
		return usecaseerr.RepoErr("charge_policy", "create charge_policy_version_editor", err, versionID)
	}
	return nil
}

// currentApproved returns the highest-numbered APPROVED version of a policy, or nil.
func currentApproved(versions []*versionpb.ChargePolicyVersion) *versionpb.ChargePolicyVersion {
	var best *versionpb.ChargePolicyVersion
	for _, v := range versions {
		if v.GetStatus() != enumspbVersionApproved {
			continue
		}
		if best == nil || v.GetVersionNumber() > best.GetVersionNumber() {
			best = v
		}
	}
	return best
}

func draftOf(versions []*versionpb.ChargePolicyVersion) *versionpb.ChargePolicyVersion {
	for _, v := range versions {
		if v.GetStatus() == enumspbVersionDraft {
			return v
		}
	}
	return nil
}

func maxVersionNumber(versions []*versionpb.ChargePolicyVersion) int32 {
	var m int32
	for _, v := range versions {
		if v.GetVersionNumber() > m {
			m = v.GetVersionNumber()
		}
	}
	return m
}

// requireDraftVersion locks the version row and refuses unless it is DRAFT (AC-CP-02).
// Callers must already be inside a transaction.
func (c *core) requireDraftVersion(ctx context.Context, versionID string) (*versionpb.ChargePolicyVersion, error) {
	v, err := c.lockVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if v.GetStatus() != enumspbVersionDraft {
		return nil, c.refuse(ctx, CodeNotDraft, "version %s is %s", v.GetId(), v.GetStatus())
	}
	return v, nil
}

// editDraftVersion is requireDraftVersion plus the C12 editor stamp: every draft, component and
// posting edit records the actor, and approval treats any recorded editor as the preparer.
func (c *core) editDraftVersion(ctx context.Context, versionID, actor string) (*versionpb.ChargePolicyVersion, error) {
	v, err := c.requireDraftVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if err := c.recordEditor(ctx, versionID, actor); err != nil {
		return nil, err
	}
	return v, nil
}

// deleteVersionTree deletes a version's editors, postings, components and the version row. The
// caller has established that the version may be deleted (DRAFT, or part of a never-approved policy).
func (c *core) deleteVersionTree(ctx context.Context, versionID string) error {
	eds, err := c.listEditors(ctx, versionID)
	if err != nil {
		return err
	}
	for _, x := range eds {
		if _, err := c.repos.ChargePolicyVersionEditor.DeleteChargePolicyVersionEditor(ctx, &editorpb.DeleteChargePolicyVersionEditorRequest{Data: &editorpb.ChargePolicyVersionEditor{Id: x.GetId()}}); err != nil {
			return usecaseerr.RepoErr("charge_policy", "delete charge_policy_version_editor", err, x.GetId())
		}
	}
	comps, err := c.listComponents(ctx, versionID)
	if err != nil {
		return err
	}
	for _, x := range comps {
		if _, err := c.repos.ChargePolicyComponent.DeleteChargePolicyComponent(ctx, &componentpb.DeleteChargePolicyComponentRequest{Data: &componentpb.ChargePolicyComponent{Id: x.GetId()}}); err != nil {
			return usecaseerr.RepoErr("charge_policy", "delete charge_policy_component", err, x.GetId())
		}
	}
	posts, err := c.listPostings(ctx, versionID)
	if err != nil {
		return err
	}
	for _, x := range posts {
		if _, err := c.repos.ChargePolicyPosting.DeleteChargePolicyPosting(ctx, &postingpb.DeleteChargePolicyPostingRequest{Data: &postingpb.ChargePolicyPosting{Id: x.GetId()}}); err != nil {
			return usecaseerr.RepoErr("charge_policy", "delete charge_policy_posting", err, x.GetId())
		}
	}
	if _, err := c.repos.ChargePolicyVersion.DeleteChargePolicyVersion(ctx, &versionpb.DeleteChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: versionID}}); err != nil {
		return usecaseerr.RepoErr("charge_policy", "delete charge_policy_version", err, versionID)
	}
	return nil
}

// inUse reports, of the given policy ids, those that cannot be deleted: some version was ever
// APPROVED/SUPERSEDED, or a product_price_plan references the policy.
func (c *core) inUse(ctx context.Context, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	reader, ok := c.repos.ChargePolicy.(domainports.ChargePolicyReferenceReader)
	if !ok {
		return nil, c.refuse(ctx, CodeUnverifiable, "charge policy repository %T cannot report price plan references", c.repos.ChargePolicy)
	}
	refs, err := reader.ChargePolicyIDsReferencedByPricePlans(ctx, ids)
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "read charge_policy references", err, ids...)
	}
	for _, id := range ids {
		if refs[id] {
			out[id] = true
			continue
		}
		vs, err := c.listVersions(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, v := range vs {
			if v.GetStatus() != enumspbVersionDraft {
				out[id] = true
				break
			}
		}
	}
	return out, nil
}
