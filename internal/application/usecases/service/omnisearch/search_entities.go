package omnisearch

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"

	omnisearchpb "github.com/erniealice/esqyma/pkg/schema/v1/service/omni_search"
)

const (
	// minQueryLen is the minimum query length the palette searches on. A shorter
	// query yields an empty (successful) response — the palette shows nothing
	// until the user has typed enough to be selective. Matches the client-side
	// debounce contract.
	minQueryLen = 2
	// maxQueryLen caps the server-side query length (S5): a longer query is
	// truncated before it reaches the adapter, bounding the LIKE-scan cost. The
	// debounce + hx-sync are UX controls, not abuse controls — this is the
	// server-side backstop.
	maxQueryLen = 64
	// defaultLimitPerCategory is the per-category result cap when the request
	// omits limit_per_category.
	defaultLimitPerCategory = 5
	// maxLimitPerCategory is the server-side hard cap on limit_per_category (a
	// larger request value is clamped down).
	maxLimitPerCategory = 10
)

// SearchEntitiesRepositories groups infrastructure dependencies.
type SearchEntitiesRepositories struct {
	Query port
}

// SearchEntitiesServices groups application services.
type SearchEntitiesServices struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SearchEntitiesUseCase runs the cross-entity palette search. It validates the
// query, resolves the requested category set against the compile-time registry,
// gates each category on its "<entity>:list" code (fail-closed), and delegates
// ONLY the permitted subset to the adapter port — so a denied category is absent
// from the response, never enumerated.
type SearchEntitiesUseCase struct {
	repositories SearchEntitiesRepositories
	services     SearchEntitiesServices
}

// NewSearchEntitiesUseCase wires the use case. Any dep may be nil; Execute
// degrades to an empty, successful response when the Query port is missing, and
// fails closed (zero categories) when the ActionGatekeeper is nil.
func NewSearchEntitiesUseCase(
	repositories SearchEntitiesRepositories,
	services SearchEntitiesServices,
) *SearchEntitiesUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &SearchEntitiesUseCase{repositories: repositories, services: services}
}

// Execute runs the palette search.
//
//	(a) validate the query (>= minQueryLen after trim-agnostic length check;
//	    server-side length cap). A too-short query is an empty, successful
//	    response (not an error) — the palette simply shows nothing.
//	(b) resolve the requested category set: an empty request list means "all
//	    registered categories"; an explicit list is validated against the
//	    compile-time registry (an unknown key is INVALID_ARGUMENT).
//	(c) per requested category, ActionGatekeeper.Check("<entity>:list") — a
//	    denied category is silently dropped (fail-closed: nil gatekeeper denies
//	    all). The permitted keys REPLACE req.Categories so the adapter queries
//	    exactly the gated allow-list.
//	(d) clamp limit_per_category, then delegate to the adapter port. A nil port
//	    (mock / non-postgres) yields an empty, successful response.
func (uc *SearchEntitiesUseCase) Execute(
	ctx context.Context,
	req *omnisearchpb.OmniSearchRequest,
) (*omnisearchpb.OmniSearchResponse, error) {
	if req == nil {
		return &omnisearchpb.OmniSearchResponse{Success: true}, nil
	}

	// (a) query validation. Length cap first (bounds the scan), then min-length
	// gate. Below the minimum → empty success (never an error).
	query := req.GetQuery()
	if len(query) > maxQueryLen {
		query = query[:maxQueryLen]
	}
	if len(query) < minQueryLen {
		return &omnisearchpb.OmniSearchResponse{Success: true}, nil
	}

	// (b) resolve requested categories against the compile-time registry.
	requested := req.GetCategories()
	if len(requested) == 0 {
		for _, c := range Registry() {
			requested = append(requested, c.Key)
		}
	} else {
		for _, key := range requested {
			if _, ok := categoryByKey(key); !ok {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
					ctx, uc.services.Translator,
					"omni_search.validation.unknown_category",
					"unknown omni-search category"))
			}
		}
	}

	// (c) per-category ":list" gate. Check is nil-receiver-safe (DENIES), so a
	// mis-wired nil gatekeeper fails closed — zero categories — instead of
	// skipping the gate. A denied category is dropped, never enumerated.
	allowed := make([]string, 0, len(requested))
	for _, key := range requested {
		cat, ok := categoryByKey(key)
		if !ok {
			continue
		}
		if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
			Entity: cat.GateEntity,
			Action: entityid.ActionList,
		}); err != nil {
			continue
		}
		allowed = append(allowed, key)
	}

	// Fail-closed: no permitted category (nil/empty perms, empty registry) →
	// empty panel, never an error that would leak category names.
	if len(allowed) == 0 {
		return &omnisearchpb.OmniSearchResponse{Success: true}, nil
	}

	if uc.repositories.Query == nil {
		// No provider registered (mock / non-postgres) — empty, successful.
		return &omnisearchpb.OmniSearchResponse{Success: true}, nil
	}

	// (d) clamp the per-category limit, then hand the adapter the gated
	// allow-list (identity/workspace scope is ctx-derived inside the adapter).
	limit := defaultLimitPerCategory
	if req.LimitPerCategory != nil {
		if v := int(req.GetLimitPerCategory()); v > 0 {
			limit = v
		}
	}
	if limit > maxLimitPerCategory {
		limit = maxLimitPerCategory
	}
	limit32 := int32(limit)

	adapterReq := &omnisearchpb.OmniSearchRequest{
		Query:            query,
		LimitPerCategory: &limit32,
		Categories:       allowed,
	}

	return uc.repositories.Query.SearchEntities(ctx, adapterReq)
}
