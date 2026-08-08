package job_list_tab_support

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// ListJobListTabSupportRepositories groups infrastructure dependencies.
type ListJobListTabSupportRepositories struct {
	Query query
}

// ListJobListTabSupportServices groups application services.
type ListJobListTabSupportServices struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListJobListTabSupportUseCase serves the job-list tabstrip support read. It
// enforces TWO INDEPENDENT per-kind gates — job_category:list and
// job_template:list — and binds the results as server-derived include flags on
// the adapter request. This deliberately does NOT collapse to the page's
// job:list gate: doing so would expose category/template metadata that the
// component-level list use cases deny. A denied kind is OMITTED (its UNION branch
// never runs), matching the pre-existing component-level degradation; there is no
// silent partial and no fail-open. Workspace is taken from the session identity
// inside the adapter, never from a request param.
type ListJobListTabSupportUseCase struct {
	repositories ListJobListTabSupportRepositories
	services     ListJobListTabSupportServices
}

// NewListJobListTabSupportUseCase wires the use case. Any dep may be nil; Execute
// degrades to an empty response when the Query port is missing, and fails closed
// (both kinds omitted) when the gatekeeper is missing.
func NewListJobListTabSupportUseCase(
	repositories ListJobListTabSupportRepositories,
	services ListJobListTabSupportServices,
) *ListJobListTabSupportUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &ListJobListTabSupportUseCase{repositories: repositories, services: services}
}

// Execute runs the complete tab-support read.
//
//	(a) Two INDEPENDENT ActionGatekeeper.Check calls (job_category:list,
//	    job_template:list). Check fails closed (nil receiver DENIES), so a
//	    mis-wired nil gatekeeper omits BOTH kinds instead of exposing them. Any
//	    gate error (deny or authz-infra) omits that kind — the same fail-closed
//	    degradation the former listAllJobCategories / activeTemplatesByCategory
//	    log-and-empty produced.
//	(b) delegate to the adapter port with the server-derived include flags. QUERY
//	    errors propagate (real error state — no silent partial tabs); a nil port
//	    (mock / non-postgres) degrades to an empty response.
func (uc *ListJobListTabSupportUseCase) Execute(ctx context.Context) (*ports.JobListTabSupportResponse, error) {
	includeCategories := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobCategory,
		Action: entityid.ActionList,
	}) == nil
	includeTemplates := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplate,
		Action: entityid.ActionList,
	}) == nil
	return uc.execute(ctx, includeCategories, includeTemplates)
}

// ExecuteCategoriesOnly runs the tab-support read needed by consumers that only
// render categories. It keeps the category-list authorization gate while
// intentionally avoiding the template gate and query branch.
func (uc *ListJobListTabSupportUseCase) ExecuteCategoriesOnly(ctx context.Context) (*ports.JobListTabSupportResponse, error) {
	includeCategories := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobCategory,
		Action: entityid.ActionList,
	}) == nil
	return uc.execute(ctx, includeCategories, false)
}

func (uc *ListJobListTabSupportUseCase) execute(ctx context.Context, includeCategories, includeTemplates bool) (*ports.JobListTabSupportResponse, error) {

	if !includeCategories && !includeTemplates {
		// Both kinds denied (or a nil/mis-wired gatekeeper) — empty response, the
		// adapter port is never touched (no SQL). Fail-closed, matching the former
		// component-level "log-and-empty" for both reads.
		return &ports.JobListTabSupportResponse{}, nil
	}

	if uc.repositories.Query == nil {
		// No provider registered (mock / non-postgres) — empty response.
		return &ports.JobListTabSupportResponse{}, nil
	}

	return uc.repositories.Query.ListJobListTabSupport(ctx, &ports.JobListTabSupportRequest{
		IncludeCategories: includeCategories,
		IncludeTemplates:  includeTemplates,
	})
}
