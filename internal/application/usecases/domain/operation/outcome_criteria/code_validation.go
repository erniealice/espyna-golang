package outcome_criteria

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

// CODE CONTRACT (matches the DB anchor's MATCH SIMPLE mechanics — esqyma
// migrations 20260718000003/4 and the proto comment on OutcomeCriteria.code):
//   - Lineage stability: every CODED version of a criteria_group carries the
//     SAME code. NULL/absent-code versions are exempt BY DESIGN and remain
//     permitted inside anchored groups (draft-first flows may version an
//     uncoded row alongside coded siblings).
//   - Domain ownership: a normalized (scope, workspace_id, industry_code, code)
//     is claimed by exactly ONE criteria_group across ALL version statuses and
//     both active states.
// The criteria_group anchor (composite FK + populate trigger + domain unique
// index) is the AUTHORITATIVE enforcement; the helpers here are friendly,
// fail-closed pre-checks layered above it.

const (
	// ocReadTimeout bounds one lineage/collision validation read (purpose-built
	// point lookup or the fallback paginated scan) so a code-bearing write can
	// never hang a request on an unbounded read.
	ocReadTimeout = 5 * time.Second
	// ocReadPageSize is the per-page window for the FALLBACK paginated read. The
	// generic List adapter caps its limit at 100, so this is the effective
	// maximum page size regardless of a larger request.
	ocReadPageSize = 100
	// ocReadMaxPages bounds the fallback pagination loop: 50 pages x 100 rows =
	// a 5,000-version evidence bound per group/code key. Reaching it means a
	// single group/domain holds an implausible number of versions; that is
	// treated as a truncation error (fail-closed, operationally useful message)
	// rather than a short-but-authoritative answer. The bound exists only on the
	// fallback path — the preferred purpose-built path is a single point lookup.
	ocReadMaxPages = 50
)

// lineageAnchorReader is the OPTIONAL purpose-built read seam. The PostgreSQL
// repository implements it with single indexed point lookups against the
// criteria_group anchor (contrib/postgres/.../operation/outcome_criteria.go);
// providers that do not are served by the bounded paginated fallback below.
// Both methods are single-statement reads (inherently snapshot-consistent) and
// fail closed on any error.
type lineageAnchorReader interface {
	// LineageEstablishedCode returns the code anchored for a lineage ("" when
	// the lineage has no coded version yet).
	LineageEstablishedCode(ctx context.Context, criteriaGroupID string) (string, error)
	// CodeOwnerGroup returns the criteria_group_id owning the normalized
	// (scopeKey, workspaceKey, industryKey, code) claim ("" when unclaimed).
	CodeOwnerGroup(ctx context.Context, scopeKey, workspaceKey, industryKey, code string) (string, error)
	// LineageClaimedDomain returns the normalized (scopeKey, workspaceKey,
	// industryKey) domain the lineage's anchor row claims, with claimed=false
	// when the lineage has no anchor yet (no coded version).
	LineageClaimedDomain(ctx context.Context, criteriaGroupID string) (scopeKey, workspaceKey, industryKey string, claimed bool, err error)
}

// codePathPattern mirrors the DB CHECK on outcome_criteria.code: a normalized
// path segment — lower(btrim(code)) matching ^[a-z][a-z0-9_]*$. Validating in the
// use case turns an opaque database constraint violation into a friendly, early
// rejection.
var codePathPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// normalizeCriteriaCode trims + lowercases a supplied code (mirroring the DB's
// lower(btrim(...))) and reports whether the result is a valid path segment.
func normalizeCriteriaCode(raw string) (string, bool) {
	n := strings.ToLower(strings.TrimSpace(raw))
	if !codePathPattern.MatchString(n) {
		return "", false
	}
	return n, true
}

// criteriaScopeKey returns the normalized scope key the anchor stores: "" for
// the unspecified zero value (protojson omits zero enums, so the DB column is
// NULL and the anchor normalizes it to ''), else the enum name exactly as
// protojson persists it (e.g. "CRITERIA_SCOPE_WORKSPACE").
func criteriaScopeKey(c *pb.OutcomeCriteria) string {
	if c.GetScope() == 0 {
		return ""
	}
	return c.GetScope().String()
}

// sameCriteriaDomain reports whether two criteria share the (scope, workspace_id,
// industry_code) domain the code-uniqueness invariant is scoped to. NULL/absent
// optionals collapse to "" here, matching the DB's COALESCE normalization.
// Callers MUST have stamped the proposal's workspace from the trusted request
// context before comparing (never trust a client-supplied tenant key).
func sameCriteriaDomain(a, b *pb.OutcomeCriteria) bool {
	return a.GetScope() == b.GetScope() &&
		a.GetWorkspaceId() == b.GetWorkspaceId() &&
		a.GetIndustryCode() == b.GetIndustryCode()
}

// listAllVersionsMatching is the FALLBACK complete read for providers without
// the purpose-built anchor seam: EVERY outcome_criteria row matching a single
// case-sensitive equality filter — across ALL version_statuses AND both active
// states — by pagination-exhausting the list RPC (the PostgreSQL adapter now
// forwards Pagination into the generic List; see ListOutcomeCriterias).
//
// Two passes are required because the generic List adapter defaults to
// active-only UNLESS an explicit `active` boolean filter is supplied: an
// active=true pass (drafts + published + deprecated rows that stay active) plus
// an active=false pass (soft-deleted versions) together cover the lineage's
// ENTIRE history. Completeness is load-bearing and fails closed: a page-bound
// overrun returns an error rather than silently truncating, and adapter decode
// failures propagate (the adapter returns an error instead of
// logging-and-dropping) rather than yielding a short, wrong answer.
func listAllVersionsMatching(ctx context.Context, repo pb.OutcomeCriteriaDomainServiceServer, field, value string) ([]*pb.OutcomeCriteria, error) {
	var out []*pb.OutcomeCriteria
	for _, active := range []bool{true, false} {
		for page := int32(1); ; page++ {
			if page > ocReadMaxPages {
				return nil, fmt.Errorf("outcome_criteria %s=%q read exceeded the %d-page (%d-row) evidence bound — refusing to answer from a truncated read; investigate the row volume for this key", field, value, ocReadMaxPages, ocReadMaxPages*ocReadPageSize)
			}
			resp, err := repo.ListOutcomeCriterias(ctx, &pb.ListOutcomeCriteriasRequest{
				Filters:    equalsActiveFilter(field, value, active),
				Pagination: offsetPage(page, ocReadPageSize),
			})
			if err != nil {
				return nil, err
			}
			if resp == nil {
				break
			}
			out = append(out, resp.GetData()...)
			if len(resp.GetData()) < int(ocReadPageSize) {
				break
			}
		}
	}
	return out, nil
}

// lineageEstablishedCode returns the code already anchored on a criteria_group's
// lineage, or "" when the lineage carries no code yet. Every CODED version in a
// group shares the SAME code (DB-enforced by the criteria_group anchor), so a
// single value is authoritative; NULL-code versions are exempt and ignored.
// The code stays reserved for its lineage even when its versions are inactive
// (soft-deleted), matching the anchor's all-version scope.
//
// Preferred path: one indexed point lookup against the anchor (the
// lineageAnchorReader seam). Fallback: the bounded, complete paginated scan.
// Both are capped by ocReadTimeout and fail closed.
func lineageEstablishedCode(ctx context.Context, repo pb.OutcomeCriteriaDomainServiceServer, criteriaGroupID string) (string, error) {
	if criteriaGroupID == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, ocReadTimeout)
	defer cancel()

	if ar, ok := repo.(lineageAnchorReader); ok {
		return ar.LineageEstablishedCode(ctx, criteriaGroupID)
	}

	rows, err := listAllVersionsMatching(ctx, repo, "criteria_group_id", criteriaGroupID)
	if err != nil {
		return "", err
	}
	for _, r := range rows {
		if r == nil {
			continue
		}
		if c := r.GetCode(); c != "" {
			return c, nil
		}
	}
	return "", nil
}

// lineageClaimedDomain returns the normalized (scope, workspace, industry)
// domain a lineage's anchor claims, or claimed=false when the lineage carries
// no coded version yet. This backs the NEW-1 domain-consistency guard: a coded
// write into an EXISTING group must carry the group's claimed domain — the DB
// cannot catch divergence there (the populate trigger's ON CONFLICT DO NOTHING
// never re-stamps an existing anchor, and uq_criteria_group_domain_code only
// arbitrates NEW anchors).
//
// Preferred path: one indexed point lookup against the anchor (the
// lineageAnchorReader seam). Fallback for providers without the seam: the
// bounded, complete paginated scan, deriving the claim from the lineage's
// first coded version (coded versions agree by contract — any divergent
// sibling would itself have been rejected by this guard). Capped by
// ocReadTimeout; fails closed on any read error.
func lineageClaimedDomain(ctx context.Context, repo pb.OutcomeCriteriaDomainServiceServer, criteriaGroupID string) (scopeKey, workspaceKey, industryKey string, claimed bool, err error) {
	if criteriaGroupID == "" {
		return "", "", "", false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ocReadTimeout)
	defer cancel()

	if ar, ok := repo.(lineageAnchorReader); ok {
		return ar.LineageClaimedDomain(ctx, criteriaGroupID)
	}

	rows, rerr := listAllVersionsMatching(ctx, repo, "criteria_group_id", criteriaGroupID)
	if rerr != nil {
		return "", "", "", false, rerr
	}
	for _, r := range rows {
		if r == nil || r.GetCode() == "" {
			continue
		}
		return criteriaScopeKey(r), r.GetWorkspaceId(), r.GetIndustryCode(), true, nil
	}
	return "", "", "", false, nil
}

// criteriaDomainDiverges reports whether a proposal's normalized (scope,
// workspace, industry) domain differs from a lineage's claimed domain keys.
// NULL/absent optionals collapse to "" (the anchor's COALESCE normalization).
// The proposal's workspace MUST already carry its trusted value (stamped from
// the request context on create; server-read existing row on update).
func criteriaDomainDiverges(proposal *pb.OutcomeCriteria, scopeKey, workspaceKey, industryKey string) bool {
	return criteriaScopeKey(proposal) != scopeKey ||
		proposal.GetWorkspaceId() != workspaceKey ||
		proposal.GetIndustryCode() != industryKey
}

// codeTakenByAnotherGroup reports whether a DIFFERENT criteria_group already
// holds `code` within self's (scope, workspace_id, industry_code) domain —
// across ALL version statuses, active or inactive. self's workspace MUST carry
// the trusted context-stamped value (see stampTrustedWorkspace) and self's
// CriteriaGroupId must be the EFFECTIVE post-write lineage (never a raw
// client-supplied tenant key against a server-returned row).
//
// This is a friendly pre-check LAYERED ABOVE the DB constraints: the anchor's
// domain unique index (uq_criteria_group_domain_code) remains the authoritative
// race backstop for concurrent cross-group drafts and direct writes. Preferred
// path: one indexed owner lookup against the anchor. Fallback: the bounded,
// complete paginated scan. Both capped by ocReadTimeout, fail closed.
func codeTakenByAnotherGroup(ctx context.Context, repo pb.OutcomeCriteriaDomainServiceServer, code string, self *pb.OutcomeCriteria) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, ocReadTimeout)
	defer cancel()

	if ar, ok := repo.(lineageAnchorReader); ok {
		owner, err := ar.CodeOwnerGroup(ctx, criteriaScopeKey(self), self.GetWorkspaceId(), self.GetIndustryCode(), code)
		if err != nil {
			return false, err
		}
		return owner != "" && owner != self.GetCriteriaGroupId(), nil
	}

	rows, err := listAllVersionsMatching(ctx, repo, "code", code)
	if err != nil {
		return false, err
	}
	for _, r := range rows {
		if r == nil || r.GetCode() != code {
			continue
		}
		if r.GetCriteriaGroupId() == self.GetCriteriaGroupId() {
			continue // the criterion's own lineage legitimately shares the code
		}
		if sameCriteriaDomain(r, self) {
			return true, nil
		}
	}
	return false, nil
}

// equalsActiveFilter builds a two-predicate FilterRequest: a case-sensitive
// equality on `field` AND an explicit `active` boolean. Supplying an explicit
// active filter is precisely what makes the generic List adapter honour the
// caller's value (returning inactive rows on the active=false pass) instead of
// applying its active-only default.
func equalsActiveFilter(field, value string, active bool) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{
			{
				Field: field,
				FilterType: &commonpb.TypedFilter_StringFilter{
					StringFilter: &commonpb.StringFilter{
						Value:         value,
						Operator:      commonpb.StringOperator_STRING_EQUALS,
						CaseSensitive: true,
					},
				},
			},
			{
				Field: "active",
				FilterType: &commonpb.TypedFilter_BooleanFilter{
					BooleanFilter: &commonpb.BooleanFilter{Value: active},
				},
			},
		},
	}
}

// offsetPage builds a 1-based offset pagination request of the given size.
func offsetPage(page, size int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{
		Limit:  size,
		Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}},
	}
}
