// Package omnisearch hosts the service-driven omni-search read use case
// (proto: service/omni_search). It backs the ⌘K command palette: one query fans
// out across the wave-1 entity categories (client, subscription,
// subscription_group, plan, price_schedule, product) and returns results split
// by category. It is a cross-aggregate derived read with no aggregate root of
// its own — the same service-driven-domain rationale as service/operation/
// outcome_matrix and service/security/permission_query.
//
// Per Q-PROTO-MODE (service{rpc}) the port IS the GENERATED
// omni_searchv1.OmniSearchServiceServer interface — there is no hand-written
// port. The postgres adapter (contrib/postgres/.../omnisearch/omni_search_query.go)
// implements it and self-registers via the omni-search registry factory; the
// composition initializer resolves that factory into this aggregate. On
// mock/non-postgres builds the port is nil and Execute degrades to an empty,
// successful response.
//
// Authorization is fail-closed and lives entirely in the use case (the L4
// backstop): each requested category is gated on the entity's ":list" code via
// ActionGatekeeper; only the permitted subset is passed to the adapter, so a
// denied category is absent from the response (never enumerated). A nil
// gatekeeper denies every category (Check has a nil-receiver guard), and an
// empty registry yields zero categories — both produce an empty panel, never an
// error that leaks category names.
//
// Apps reach it via uc.Service.OmniSearch.SearchEntities.Execute(ctx, req).
package omnisearch

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

	omnisearchpb "github.com/erniealice/esqyma/pkg/schema/v1/service/omni_search"
)

// port is the read port the omni-search use case delegates to. It is exactly the
// GENERATED omni_searchv1.OmniSearchServiceServer interface (Q-PROTO-MODE:
// service{rpc}) — no hand-written port. The postgres adapter embeds
// UnimplementedOmniSearchServiceServer and satisfies this directly; on
// mock/non-postgres builds the port is nil and Execute degrades to an empty,
// successful response.
type port = omnisearchpb.OmniSearchServiceServer

// UseCases aggregates every omni-search service use case.
type UseCases struct {
	SearchEntities *SearchEntitiesUseCase
}

// Repositories groups infrastructure dependencies. Query may be nil when no
// provider is registered — the use case degrades gracefully (empty response).
type Repositories struct {
	Query port
}

// Services groups application services. ActionGatekeeper is REQUIRED for the
// per-category "<entity>:list" gate; a nil gatekeeper fails closed (every
// category denied).
type Services struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires every omni-search use case from shared dependencies.
func NewUseCases(repositories Repositories, services Services) *UseCases {
	return &UseCases{
		SearchEntities: NewSearchEntitiesUseCase(
			SearchEntitiesRepositories{Query: repositories.Query},
			SearchEntitiesServices{
				Translator:       services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
