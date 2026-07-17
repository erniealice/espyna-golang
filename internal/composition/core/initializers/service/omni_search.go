package service

import (
	"database/sql"

	omnisearchpb "github.com/erniealice/esqyma/pkg/schema/v1/service/omni_search"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	omnisearchusecases "github.com/erniealice/espyna-golang/internal/application/usecases/service/omnisearch"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// initServiceOmniSearch wires the service-layer OmniSearch use case
// (service/omni_search). It threads the ActionGatekeeper through because the
// palette gates every category on its "<entity>:list" code (Q-OMNI-4), and the
// registered postgres query port (nil on non-postgres builds → empty response).
func initServiceOmniSearch(db *sql.DB, i18nSvc ports.Translator, actionGate *actiongate.ActionGatekeeper) *omnisearchusecases.UseCases {
	query := omniSearchQueryFromDB(db)
	return omnisearchusecases.NewUseCases(
		omnisearchusecases.Repositories{Query: query},
		omnisearchusecases.Services{Translator: i18nSvc, ActionGatekeeper: actionGate},
	)
}

// omniSearchQueryFromDB returns the registered omni-search query port backed by
// the provided raw connection, or nil when no provider has been registered (e.g.
// non-postgres / non-mock builds). The factory takes `any` to dodge the cyclic
// import — see registry/omni_search.go.
func omniSearchQueryFromDB(db *sql.DB) omnisearchpb.OmniSearchServiceServer {
	factory, ok := internalregistry.GetOmniSearchFactory()
	if !ok || factory == nil {
		return nil
	}
	result := factory(db)
	if result == nil {
		return nil
	}
	if q, ok := result.(omnisearchpb.OmniSearchServiceServer); ok {
		return q
	}
	return nil
}
