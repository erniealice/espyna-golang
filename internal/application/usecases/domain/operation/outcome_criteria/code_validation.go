package outcome_criteria

import (
	"context"
	"regexp"
	"strings"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

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

// sameCriteriaDomain reports whether two criteria share the (scope, workspace_id,
// industry_code) domain the code-uniqueness invariant is scoped to. NULL/absent
// optionals collapse to "" here, matching the DB partial index's COALESCE.
func sameCriteriaDomain(a, b *pb.OutcomeCriteria) bool {
	return a.GetScope() == b.GetScope() &&
		a.GetWorkspaceId() == b.GetWorkspaceId() &&
		a.GetIndustryCode() == b.GetIndustryCode()
}

// lineageEstablishedCode returns the code already anchored on a criteria_group's
// lineage (the first non-empty code across its active versions), or "" when the
// lineage carries no code yet. All versions in a group must share the SAME code,
// so a single non-empty value is authoritative.
func lineageEstablishedCode(ctx context.Context, repo pb.OutcomeCriteriaDomainServiceServer, criteriaGroupID string) (string, error) {
	resp, err := repo.ListOutcomeCriterias(ctx, &pb.ListOutcomeCriteriasRequest{
		Filters: equalsStringFilter("criteria_group_id", criteriaGroupID),
	})
	if err != nil {
		return "", err
	}
	for _, r := range resp.GetData() {
		if r == nil {
			continue
		}
		if c := r.GetCode(); c != "" {
			return c, nil
		}
	}
	return "", nil
}

// codeTakenByAnotherGroup reports whether an ACTIVE criterion in a DIFFERENT
// criteria_group already holds `code` within self's domain.
//
// This is a friendly pre-check LAYERED ABOVE the DB partial unique index. The
// index (uq_outcome_criteria_published_domain_code) remains the authoritative
// backstop against check-then-write races; this read is stricter than the index
// (it also flags active DRAFT rows the published-only index ignores) but is not a
// substitute for it.
func codeTakenByAnotherGroup(ctx context.Context, repo pb.OutcomeCriteriaDomainServiceServer, code string, self *pb.OutcomeCriteria) (bool, error) {
	resp, err := repo.ListOutcomeCriterias(ctx, &pb.ListOutcomeCriteriasRequest{
		Filters: equalsStringFilter("code", code),
	})
	if err != nil {
		return false, err
	}
	for _, r := range resp.GetData() {
		if r == nil || r.GetCode() != code {
			continue
		}
		if r.CriteriaGroupId == self.CriteriaGroupId {
			continue // the criterion's own lineage legitimately shares the code
		}
		if sameCriteriaDomain(r, self) {
			return true, nil
		}
	}
	return false, nil
}

// equalsStringFilter builds a single-field, case-sensitive equality FilterRequest.
func equalsStringFilter(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{{
			Field: field,
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Value:         value,
					Operator:      commonpb.StringOperator_STRING_EQUALS,
					CaseSensitive: true,
				},
			},
		}},
	}
}
